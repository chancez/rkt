// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

type ImageId struct {
	path string
	hash string
}

func (imgId *ImageId) getShortHash(length int) (string, error) {
	if length >= len(imgId.hash) {
		return "", fmt.Errorf("getShortHash: Hash %s is shorter than %d chars", imgId.hash, length)
	}

	return imgId.hash[:length], nil
}

// containsConflictingHash returns an ImageId pair if a conflicting short hash is found. The minimum
// hash of 2 chars is used for comparisons.
func (imgId *ImageId) containsConflictingHash(imgIds []ImageId) (imgIdPair []ImageId, found bool) {
	shortHash, err := imgId.getShortHash(2)
	if err != nil {
		panic(fmt.Sprintf("containsConflictingHash: %s", err))
	}

	for _, iId := range imgIds {
		if strings.HasPrefix(iId.hash, shortHash) {
			imgIdPair = []ImageId{*imgId, iId}
			found = true
			break
		}
	}
	return
}

// TestShortHash tests that the short hash generated by the rkt image list
// command is usable by the commands that accept image hashes.
func TestShortHash(t *testing.T) {
	var (
		imageIds []ImageId
		iter     int
	)

	// Generate unique images until we get a collision of the first 2 hash chars
	for {
		image := patchTestACI(fmt.Sprintf("rkt-shorthash-%d.aci", iter), fmt.Sprintf("--name=shorthash--%d", iter))
		defer os.Remove(image)

		imageHash := getHashOrPanic(image)
		imageId := ImageId{image, imageHash}

		imageIdPair, isMatch := imageId.containsConflictingHash(imageIds)
		if isMatch {
			imageIds = imageIdPair
			break
		}

		imageIds = append(imageIds, imageId)
		iter++
	}
	ctx := newRktRunCtx()
	defer ctx.cleanup()

	// Pull the 2 images with matching first 2 hash chars into cas
	for _, imageId := range imageIds {
		cmd := fmt.Sprintf("%s --insecure-skip-verify fetch %s", ctx.cmd(), imageId.path)
		t.Logf("Fetching %s: %v", imageId.path, cmd)
		spawnAndWaitOrFail(t, cmd, true)
	}

	// Get hash from 'rkt image list'
	hash0 := fmt.Sprintf("sha512-%s", imageIds[0].hash[:12])
	hash1 := fmt.Sprintf("sha512-%s", imageIds[1].hash[:12])
	for _, hash := range []string{hash0, hash1} {
		imageListCmd := fmt.Sprintf("%s image list --fields=id --no-legend", ctx.cmd())
		runRktAndCheckOutput(t, imageListCmd, hash, false)
	}

	tmpDir := createTempDirOrPanic("rkt_image_list_test")
	defer os.RemoveAll(tmpDir)

	// Define tests
	tests := []struct {
		cmd        string
		shouldFail bool
		expect     string
	}{
		// Try invalid ID
		{
			"image cat-manifest sha512-12341234",
			true,
			"no image IDs found",
		},
		// Try using one char hash
		{
			fmt.Sprintf("image cat-manifest %s", hash0[:len("sha512-")+1]),
			true,
			"image ID too short",
		},
		// Try short hash that collides
		{
			fmt.Sprintf("image cat-manifest %s", hash0[:len("sha512-")+2]),
			true,
			"ambiguous image ID",
		},
		// Test that 12-char hash works with image cat-manifest
		{
			fmt.Sprintf("image cat-manifest %s", hash0),
			false,
			"ImageManifest",
		},
		// Test that 12-char hash works with image export
		{
			fmt.Sprintf("image export --overwrite %s %s/export.aci", hash0, tmpDir),
			false,
			"",
		},
		// Test that 12-char hash works with image render
		{
			fmt.Sprintf("image render --overwrite %s %s", hash0, tmpDir),
			false,
			"",
		},
		// Test that 12-char hash works with image extract
		{
			fmt.Sprintf("image extract --overwrite %s %s", hash0, tmpDir),
			false,
			"",
		},
		// Test that 12-char hash works with prepare
		{
			fmt.Sprintf("prepare --debug %s", hash0),
			false,
			"Writing pod manifest",
		},
		// Test that 12-char hash works with image rm
		{
			fmt.Sprintf("image rm %s", hash1),
			false,
			"successfully removed aci",
		},
	}

	// Run tests
	for i, tt := range tests {
		runCmd := fmt.Sprintf("%s %s", ctx.cmd(), tt.cmd)
		t.Logf("Running test #%d", i)
		runRktAndCheckOutput(t, runCmd, tt.expect, tt.shouldFail)
	}
}
