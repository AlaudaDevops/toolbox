/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// genTestKey produces an RSA-2048 private key + its PKCS#1 PEM. We
// never persist this; it lives the lifetime of one test.
func genTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return key, pemBytes
}

// TestParsePrivateKeyAcceptsBothFormats locks down PKCS#1 + PKCS#8.
func TestParsePrivateKeyAcceptsBothFormats(t *testing.T) {
	key, pkcs1 := genTestKey(t)

	if _, err := parseRSAPrivateKey(pkcs1); err != nil {
		t.Fatalf("PKCS#1 parse: %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8 := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if _, err := parseRSAPrivateKey(pkcs8); err != nil {
		t.Fatalf("PKCS#8 parse: %v", err)
	}

	if _, err := parseRSAPrivateKey([]byte("not a pem")); err == nil {
		t.Fatal("expected garbage to fail")
	}
}

// TestAppTokenSource_FetchAndCache verifies the full mint → cache →
// reuse flow. The fake GitHub server validates the JWT (signature,
// issuer, expiry window) before issuing an installation token.
func TestAppTokenSource_FetchAndCache(t *testing.T) {
	key, pemBytes := genTestKey(t)

	var (
		mu       sync.Mutex
		mintHits int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/installations/777/access_tokens" {
			http.NotFound(w, r)
			return
		}
		// Validate the App JWT.
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing bearer", http.StatusUnauthorized)
			return
		}
		raw := strings.TrimPrefix(auth, "Bearer ")
		parsed, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{}, func(*jwt.Token) (interface{}, error) {
			return &key.PublicKey, nil
		})
		if err != nil || !parsed.Valid {
			http.Error(w, fmt.Sprintf("jwt: %v", err), http.StatusUnauthorized)
			return
		}
		claims := parsed.Claims.(*jwt.RegisteredClaims)
		if claims.Issuer != "12345" {
			http.Error(w, "wrong issuer", http.StatusUnauthorized)
			return
		}
		mu.Lock()
		mintHits++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      fmt.Sprintf("ghs_test_%d", mintHits),
			"expires_at": time.Now().Add(time.Hour),
		})
	}))
	t.Cleanup(srv.Close)

	src, err := NewAppTokenSource(AppCredentials{
		AppID:          12345,
		InstallationID: 777,
		PrivateKeyPEM:  pemBytes,
		BaseURL:        srv.URL,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	got1, err := src.Token(ctx)
	if err != nil {
		t.Fatalf("token1: %v", err)
	}
	if got1 != "ghs_test_1" {
		t.Fatalf("token1 = %q, want ghs_test_1", got1)
	}

	// Second call within refreshSkew of expiry returns the cached token
	// without hitting the server.
	got2, err := src.Token(ctx)
	if err != nil {
		t.Fatalf("token2: %v", err)
	}
	if got2 != got1 {
		t.Fatalf("token2 = %q, want cached %q", got2, got1)
	}

	mu.Lock()
	if mintHits != 1 {
		t.Fatalf("mintHits = %d, expected 1 (caching broke)", mintHits)
	}
	mu.Unlock()

	// Forcing the cache near expiry triggers a re-mint.
	src.mu.Lock()
	src.expiresAt = time.Now().Add(2 * time.Minute) // < refreshSkew (5m)
	src.mu.Unlock()

	got3, err := src.Token(ctx)
	if err != nil {
		t.Fatalf("token3: %v", err)
	}
	if got3 == got1 {
		t.Fatal("expected re-mint, got the cached token")
	}
	mu.Lock()
	if mintHits != 2 {
		t.Fatalf("mintHits after refresh = %d, want 2", mintHits)
	}
	mu.Unlock()
}

// TestAppTokenSource_PropagatesErrors makes sure a 4xx from /access_tokens
// surfaces as a Token() error rather than empty + nil.
func TestAppTokenSource_PropagatesErrors(t *testing.T) {
	_, pemBytes := genTestKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad app", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	src, err := NewAppTokenSource(AppCredentials{
		AppID:          1, InstallationID: 1, PrivateKeyPEM: pemBytes, BaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := src.Token(context.Background()); err == nil {
		t.Fatal("expected error from 401, got nil")
	}
}
