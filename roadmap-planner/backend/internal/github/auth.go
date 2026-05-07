/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenSource is the auth boundary for the GitHub client.
//
// Two implementations ship in this package:
//
//   - StaticTokenSource — wraps a personal access token (PAT). What the
//     local-dev / one-off testing path uses.
//
//   - AppTokenSource — mints and refreshes installation tokens for a
//     GitHub App. Each installation token lives ~60 min; we refresh
//     proactively a few minutes before expiry. Rate-limit pool is per
//     installation, scales with repos+users, and doesn't compete with
//     human PATs — see PROPOSAL.md §6.7.
//
// Token() must be safe for concurrent use; both built-ins are.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// StaticTokenSource returns the same string every time. Use for PATs
// and for tests.
type StaticTokenSource struct{ Value string }

func (s StaticTokenSource) Token(_ context.Context) (string, error) {
	if s.Value == "" {
		return "", nil
	}
	return s.Value, nil
}

// ----------------------------------------------------------------------
// AppTokenSource
// ----------------------------------------------------------------------

// AppTokenSource mints GitHub-App installation tokens.
//
// Flow per refresh cycle:
//  1. Build a short-lived JWT signed with the App's RSA private key
//     (10-minute lifetime, the GitHub maximum).
//  2. POST /app/installations/{id}/access_tokens with that JWT as the
//     bearer.
//  3. Response carries the installation token (1h lifetime) and an
//     `expires_at` ISO-8601 timestamp.
//
// Tokens are cached and refreshed lazily on Token() when within
// `refreshSkew` of expiry — no background goroutine, so a long-idle
// process doesn't spin needlessly.
type AppTokenSource struct {
	appID          int64
	installationID int64
	key            *rsa.PrivateKey
	baseURL        string
	http           *http.Client

	// refreshSkew controls when we proactively refresh; default 5 min.
	refreshSkew time.Duration

	mu        sync.Mutex
	cached    string
	expiresAt time.Time
}

// AppCredentials groups the inputs to NewAppTokenSource.
//
// Either PrivateKeyPEM (raw bytes) or PrivateKeyPath (path to a .pem
// file on disk) must be set; PrivateKeyPEM wins if both are.
type AppCredentials struct {
	AppID          int64
	InstallationID int64
	PrivateKeyPEM  []byte
	PrivateKeyPath string
	BaseURL        string       // empty -> https://api.github.com
	HTTPClient     *http.Client // optional
}

// NewAppTokenSource validates inputs, parses the PEM, and returns a
// ready-to-use TokenSource. Does not mint a token yet — that happens on
// the first Token() call.
func NewAppTokenSource(c AppCredentials) (*AppTokenSource, error) {
	if c.AppID <= 0 {
		return nil, fmt.Errorf("app credentials: AppID is required")
	}
	if c.InstallationID <= 0 {
		return nil, fmt.Errorf("app credentials: InstallationID is required")
	}
	pemBytes := c.PrivateKeyPEM
	if len(pemBytes) == 0 && c.PrivateKeyPath != "" {
		b, err := os.ReadFile(c.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key %s: %w", c.PrivateKeyPath, err)
		}
		pemBytes = b
	}
	if len(pemBytes) == 0 {
		return nil, fmt.Errorf("app credentials: private key is required (PEM or path)")
	}
	key, err := parseRSAPrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse app private key: %w", err)
	}
	base := c.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	httpc := c.HTTPClient
	if httpc == nil {
		httpc = &http.Client{Timeout: 30 * time.Second}
	}
	return &AppTokenSource{
		appID:          c.AppID,
		installationID: c.InstallationID,
		key:            key,
		baseURL:        base,
		http:           httpc,
		refreshSkew:    5 * time.Minute,
	}, nil
}

// Token returns a valid installation access token, refreshing if the
// cached one is within refreshSkew of expiry (or absent).
func (a *AppTokenSource) Token(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cached != "" && time.Until(a.expiresAt) > a.refreshSkew {
		return a.cached, nil
	}
	if err := a.refreshLocked(ctx); err != nil {
		return "", err
	}
	return a.cached, nil
}

func (a *AppTokenSource) refreshLocked(ctx context.Context) error {
	jwtToken, err := a.signJWT(time.Now())
	if err != nil {
		return fmt.Errorf("sign app jwt: %w", err)
	}
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", a.baseURL, a.installationID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("github app installation token: %s — %s", resp.Status, string(body))
	}

	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode installation token: %w", err)
	}
	if out.Token == "" {
		return fmt.Errorf("installation token response missing token")
	}
	a.cached = out.Token
	a.expiresAt = out.ExpiresAt
	return nil
}

// signJWT builds the App-level JWT used to authenticate the
// /app/installations/.../access_tokens call. iat is set 60s in the past
// to absorb minor clock skew between us and GitHub.
func (a *AppTokenSource) signJWT(now time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
		Issuer:    strconv.FormatInt(a.appID, 10),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return tok.SignedString(a.key)
}

// parseRSAPrivateKey accepts either PKCS#1 ("RSA PRIVATE KEY") or
// PKCS#8 ("PRIVATE KEY") blocks. GitHub's downloaded PEMs are PKCS#1;
// converted PEMs (e.g., via openssl pkcs8) are PKCS#8. We support both
// so operators don't have to worry about the format.
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("pem decode failed (no PEM block found)")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not RSA (got %T)", k)
		}
		return rk, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type %q (want RSA PRIVATE KEY or PRIVATE KEY)", block.Type)
	}
}
