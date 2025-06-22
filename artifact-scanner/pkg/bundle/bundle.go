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
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/models"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

// extract extracts the contents of a bundle image to a local directory
// ctx: The context for the operation
// bundle: The bundle image to extract
// Returns the path to the extracted bundle directory and any error that occurred
func extract(ctx context.Context, bundle models.Image) (string, error) {
	logger := logging.FromContext(ctx).With("bundle", bundle)

	tmpDir, err := os.MkdirTemp("", "artifact-scanner_")
	if err != nil {
		return "", err
	}
	imageName := strings.ReplaceAll(bundle.Repository, "/", "-")
	bundleDir := path.Join(tmpDir, imageName)
	manifestsDir := path.Join(bundleDir, "manifests")
	if _, err := os.Stat(manifestsDir); err == nil {
		logger.Debugw("bundle already extracted", "path", bundleDir)
		return bundleDir, nil
	}

	logger.Debugw("create bundle dir", "path", bundleDir)

	if err := os.Mkdir(bundleDir, fs.ModePerm); err != nil {
		return "", err
	}

	// Use a containerd registry instead of shelling out to a container tool.
	reg, err := containerdregistry.NewRegistry(
		containerdregistry.WithLog(newEntry(logger.Desugar())),
		containerdregistry.SkipTLSVerify(true))

	if err != nil {
		return "", fmt.Errorf("failed to create registry client: %w", err)
	}

	defer func() {
		if err := reg.Destroy(); err != nil {
			logger.Errorw("Error destroying local cache", "error", err)
		}
	}()

	logger.Debugw("pull bundle image")

	imageURL := bundle.URL()
	if err := reg.Pull(ctx, image.SimpleReference(imageURL)); err != nil {
		return "", fmt.Errorf("error pulling image %s: %v", imageURL, err)
	}

	logger.Debugw("unpack bundle image")

	// Unpack the image's contents.
	if err := reg.Unpack(ctx, image.SimpleReference(imageURL), bundleDir); err != nil {
		return "", fmt.Errorf("error unpacking image %s: %v", imageURL, err)
	}

	return bundleDir, nil
}

// getRelatedImages retrieves all related images from a bundle's CSV
// ctx: The context for the operation
// bundle: The bundle image to analyze
// Returns a list of related images and any error that occurred
func getRelatedImages(ctx context.Context, bundle models.Image) ([]models.Image, error) {
	logger := logging.FromContext(ctx).With("image", bundle)

	dir, err := extract(ctx, bundle)
	if err != nil {
		logger.Errorw("failed to extract image", zap.Error(err))
		return nil, err
	}

	logger.Debugw("read csv from manifest")

	csv, err := registry.ReadCSVFromBundleDirectory(path.Join(dir, "manifests"))
	if err != nil {
		logger.Errorw("failed to read csv", zap.Error(err))
		return nil, err
	}

	images, err := csv.GetRelatedImages()
	if err != nil {
		logger.Errorw("failed to get related images", zap.Error(err))
		return nil, err
	}
	if len(images) == 0 {
		logger.Errorw("failed to get related images", zap.Error(err))
		return nil, fmt.Errorf("no images found in csv")
	}

	results := make([]models.Image, 0, len(images))
	for key, _ := range images {
		relatedImage, err := models.ImageFromURL(key)
		relatedImage.Owner = bundle.Owner
		if err != nil {
			return nil, err
		}
		results = append(results, relatedImage)
	}

	logger.Debugw("found related images", "images", results)

	return results, nil
}
