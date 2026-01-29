// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package storage provides functions for accessing Google Cloud Storge buckets.
package storage

import (
	"context"
	"errors"
	"os"

	"cloud.google.com/go/storage"
	"firebase.google.com/go/v4/internal"
)

// Client is the interface for the Firebase Storage service.
type Client struct {
	client *storage.Client
	bucket string
}

// NewClient creates a new instance of the Firebase Storage Client.
//
// This function can only be invoked from within the SDK. Client applications should access the
// the Storage service through firebase.App.
func NewClient(ctx context.Context, c *internal.StorageConfig) (*Client, error) {
	if os.Getenv("STORAGE_EMULATOR_HOST") == "" && os.Getenv("FIREBASE_STORAGE_EMULATOR_HOST") != "" {
		os.Setenv("STORAGE_EMULATOR_HOST", os.Getenv("FIREBASE_STORAGE_EMULATOR_HOST"))
	}
	client, err := storage.NewClient(ctx, c.Opts...)
	if err != nil {
		return nil, err
	}
	return &Client{client: client, bucket: c.Bucket}, nil
}

// DefaultBucket returns a handle to the default Cloud Storage bucket.
//
// To use this method, the default bucket name must be specified via firebase.Config when
// initializing the App.
func (c *Client) DefaultBucket() (*storage.BucketHandle, error) {
	return c.Bucket(c.bucket)
}

// Bucket returns a handle to the specified Cloud Storage bucket.
func (c *Client) Bucket(name string) (*storage.BucketHandle, error) {
	if name == "" {
		return nil, errors.New("bucket name not specified")
	}
	return c.client.Bucket(name), nil
}
