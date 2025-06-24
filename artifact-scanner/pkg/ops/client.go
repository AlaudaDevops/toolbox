/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/models"
)

// APIPathVulnerability is the API path for vulnerability scanning
const APIPathVulnerability = "image/vulnerability/all"

// APIPathComponentOwner is the API path for component ownership information
const APIPathComponentOwner = "/components/image"

// Client represents a client for interacting with the OPS API
// BaseURL: The base URL of the OPS API
type Client struct {
	BaseURL string
}

// NewClient creates a new OPS API client
// baseURL: The base URL of the OPS API
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

// Vulnerability retrieves vulnerability scan results for an image
// image: The full address of the image to scan
// Returns the scan results and any error that occurred
func (s *Client) Vulnerability(image string) (*models.ScanResult, error) {
	query := &url.Values{}
	query.Add("image_full_address", image)
	result := &models.ScanResult{}
	if err := s.get(APIPathVulnerability, query, result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetOwner retrieves the owner information for an image
// image: The full address of the image
// Returns the owner information and any error that occurred
func (s *Client) GetOwner(image string) (*models.Owner, error) {
	imageURL, err := url.Parse(image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image URL: %w", err)
	}

	query := &url.Values{
		"harbor_repository_slug": []string{imageURL.Path},
	}
	result := make([]models.Owner, 0)
	if err := s.get(APIPathComponentOwner, query, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return &result[0], nil
}

// get performs a GET request to the OPS API
// path: The API path to request
// query: The query parameters to include
// result: The struct to unmarshal the response into
// Returns any error that occurred
func (s *Client) get(path string, query *url.Values, result interface{}) error {
	url := fmt.Sprintf("%s/%s", s.BaseURL, strings.TrimPrefix(path, "/"))
	if query != nil {
		url = url + "?" + query.Encode()
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to request %s, body: %s, status: %s", url, string(body), resp.Status)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response, error: %w, body: %s", err, string(body))
	}

	return nil
}
