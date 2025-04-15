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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewClient(t *testing.T) {
	g := NewWithT(t)

	baseURL := "http://test-api"
	client := NewClient(baseURL)

	g.Expect(client).NotTo(BeNil())
	g.Expect(client.BaseURL).To(Equal(baseURL))
}

func TestClient_Vulnerability(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name           string
		image          string
		responseStatus int
		responseBody   string
		wantErr        bool
	}{
		{
			name:           "should get vulnerability successfully",
			image:          "test-registry/test-image:latest",
			responseStatus: http.StatusOK,
			responseBody:   `{"vulnerabilities": []}`,
			wantErr:        false,
		},
		{
			name:           "should handle API error",
			image:          "test-registry/test-image:latest",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error": "internal server error"}`,
			wantErr:        true,
		},
		{
			name:           "should handle invalid response",
			image:          "test-registry/test-image:latest",
			responseStatus: http.StatusOK,
			responseBody:   `invalid json`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			result, err := client.Vulnerability(tt.image)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(result).To(BeNil())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result).NotTo(BeNil())
			}
		})
	}
}

func TestClient_GetOwner(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name           string
		image          string
		responseStatus int
		responseBody   string
		wantErr        bool
	}{
		{
			name:           "should get owner successfully",
			image:          "test-registry/test-image:latest",
			responseStatus: http.StatusOK,
			responseBody:   `[{"name": "test-owner"}]`,
			wantErr:        false,
		},
		{
			name:           "should handle no owner found",
			image:          "test-registry/test-image:latest",
			responseStatus: http.StatusOK,
			responseBody:   `[]`,
			wantErr:        false,
		},
		{
			name:           "should handle API error",
			image:          "test-registry/test-image:latest",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error": "internal server error"}`,
			wantErr:        true,
		},
		{
			name:           "should handle invalid response",
			image:          "test-registry/test-image:latest",
			responseStatus: http.StatusOK,
			responseBody:   `invalid json`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			result, err := client.GetOwner(tt.image)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(result).To(BeNil())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				if tt.responseBody == `[]` {
					g.Expect(result).To(BeNil())
				} else {
					g.Expect(result).NotTo(BeNil())
				}
			}
		})
	}
}

func TestClient_get(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name           string
		path           string
		query          *url.Values
		responseStatus int
		responseBody   string
		wantErr        bool
	}{
		{
			name:           "should get successfully",
			path:           "test",
			query:          &url.Values{"key": []string{"value"}},
			responseStatus: http.StatusOK,
			responseBody:   `{"result": "success"}`,
			wantErr:        false,
		},
		{
			name:           "should handle API error",
			path:           "test",
			query:          &url.Values{},
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error": "internal server error"}`,
			wantErr:        true,
		},
		{
			name:           "should handle invalid response",
			path:           "test",
			query:          &url.Values{},
			responseStatus: http.StatusOK,
			responseBody:   `invalid json`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			var result map[string]interface{}
			err := client.get(tt.path, tt.query, &result)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result).NotTo(BeNil())
			}
		})
	}
}
