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

package bundle

import (
	"context"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/models"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/ops"
	"knative.dev/pkg/logging"
)

// Scanner represents a scanner for analyzing bundle images and their related images
// opsClient: The OPS API client for vulnerability scanning
type Scanner struct {
	opsClient *ops.Client
}

// NewScanner creates a new scanner instance
// client: The OPS API client to use for scanning
func NewScanner(client *ops.Client) *Scanner {
	scanner := &Scanner{
		opsClient: client,
	}

	return scanner
}

// Scan performs vulnerability scanning on an image and its related images
// ctx: The context for the operation
// image: The image to scan
// Returns a map of scan results for each image and any error that occurred
func (s *Scanner) Scan(ctx context.Context, image models.Image) (models.ScanResults, error) {
	logger := logging.FromContext(ctx)

	var images []models.Image
	if !image.IsBundle {
		images = append(images, models.Image{})
	} else {
		relatedImages, err := getRelatedImages(ctx, image)
		if err != nil {
			logger.Errorf("failed to get related images: %w", err)
			return nil, err
		}
		images = relatedImages
	}

	results := make(models.ScanResults)

	for _, relatedImage := range images {
		imageURL := relatedImage.URL()
		logger.Infof("scanning image: %s", imageURL)

		result, err := s.opsClient.Vulnerability(imageURL)
		if err != nil {
			logger.Errorf("failed to scan image: %w", err)
			return nil, err
		}

		count := result.TotalVulnerabilities()
		if count > 0 {
			results[relatedImage] = result
		}
		logger.Infof("scan image completed, vulnerability count: %d", count)
	}

	return results, nil
}
