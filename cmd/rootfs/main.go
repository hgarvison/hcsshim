// --------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE file in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Microsoft/hcsshim/ext4/tar2ext4"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func main() {
	image := flag.String("i", "", "image")
	tag := flag.String("t", "latest", "tag")
	destination := flag.String("d", "local", "destination")

	flag.Parse()
	if flag.NArg() != 0 || len(*image) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	layers, err := getRootfsLayerHashes(*image, *tag, *destination)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for i, layer := range layers {
		fmt.Printf("[%d]: %s\n", i, layer)
	}
}

func getRootfsLayerHashes(imageName string, imageTag string, destination string) ([]string, error) {
	imageString := imageName + ":" + imageTag

	if len(imageName) == 0 || len(imageTag) == 0 {
		return nil, fmt.Errorf("'%s' is not a valid image name", imageString)
	}

	validDestinations := map[string]bool{
		"local":  true,
		"remote": true,
	}
	if !validDestinations[destination] {
		fmt.Print("Destination value should fall in following values:[")
		for k := range validDestinations {
			fmt.Print(k + ",")
		}
		fmt.Println("]")
		return nil, errors.New("invalid destination")
	}

	ref, err := name.ParseReference(imageString)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a valid image name", imageString)
	}

	// by default, using local as destination
	var image v1.Image
	if strings.ToLower(destination) == "remote" {
		image, err = remote.Image(ref)
	} else {
		image, err = daemon.Image(ref)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to fetch image '%s': %w", imageString, err)
	}

	layers, err := image.Layers()
	if err != nil {
		return nil, err
	}

	hashes := []string{}
	for _, layer := range layers {
		hashString, err := getRoothash(layer)
		if err != nil {
			return nil, err
		}

		hashes = append(hashes, hashString)
	}

	return hashes, nil
}

func getRoothash(layer v1.Layer) (string, error) {
	r, err := layer.Uncompressed()
	if err != nil {
		return "", err
	}

	return tar2ext4.ConvertAndComputeRootDigest(r)
}
