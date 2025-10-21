package client

import (
	"strings"
	"testing"

	"github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/spf13/afero"
)

func TestGenerateJava(t *testing.T) {
	// Create a test application spec with a bucket resource
	appSpec := schema.Application{
		Name:   "test-app",
		Target: "team/platform@1",
		BucketIntents: map[string]*schema.BucketIntent{
			"my-test-bucket": {},
		},
	}

	// Create an in-memory filesystem
	fs := afero.NewMemMapFs()

	// Test Java generation
	err := GenerateJava(fs, appSpec, "test-output", "com.example.test")
	if err != nil {
		t.Fatalf("GenerateJava() error = %v", err)
	}

	// Check if the Java file was created
	exists, err := afero.Exists(fs, "test-output/GeneratedSugaClient.java")
	if err != nil {
		t.Fatalf("Failed to check if file exists: %v", err)
	}
	if !exists {
		t.Error("Expected GeneratedSugaClient.java to be created, but it doesn't exist")
	}

	// Read and verify the generated content
	content, err := afero.ReadFile(fs, "test-output/GeneratedSugaClient.java")
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify key parts of the generated Java code
	expectedParts := []string{
		"package com.example.test",
		"GeneratedSugaClient",
		"myTestBucket",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated content does not contain expected string: %s", expected)
		}
	}
}

func TestGenerateJavaDefaultValues(t *testing.T) {
	appSpec := schema.Application{
		Name:          "test-app",
		Target:        "team/platform@1",
		BucketIntents: map[string]*schema.BucketIntent{},
	}

	fs := afero.NewMemMapFs()

	// Test with default values
	err := GenerateJava(fs, appSpec, "", "")
	if err != nil {
		t.Fatalf("GenerateJava() with defaults error = %v", err)
	}

	// Should create Java file in default location
	exists, err := afero.Exists(fs, "suga/java/client/GeneratedSugaClient.java")
	if err != nil {
		t.Fatalf("Failed to check if default file exists: %v", err)
	}
	if !exists {
		t.Error("Expected GeneratedSugaClient.java to be created in default location")
	}

	// Read and verify default package
	content, err := afero.ReadFile(fs, "suga/java/client/GeneratedSugaClient.java")
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "package com.addsuga.client") {
		t.Error("Expected default package com.addsuga.client")
	}
}

func TestGenerateJavaWithInvalidPackage(t *testing.T) {
	appSpec := schema.Application{
		Name:          "test-app",
		Target:        "team/platform@1",
		BucketIntents: map[string]*schema.BucketIntent{},
	}

	fs := afero.NewMemMapFs()

	// Test with invalid package name
	err := GenerateJava(fs, appSpec, "test-output", "invalid.fun.package")
	if err == nil {
		t.Error("GenerateJava() expected error for invalid package name, but got none")
	}

	expectedErrMsg := "invalid Java package name"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("GenerateJava() expected error message to contain '%s', but got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGenerateJavaWithMultipleBuckets(t *testing.T) {
	appSpec := schema.Application{
		Name:   "test-app",
		Target: "team/platform@1",
		BucketIntents: map[string]*schema.BucketIntent{
			"bucket-one":   {},
			"bucket_two":   {},
			"bucket-three": {},
		},
	}

	fs := afero.NewMemMapFs()

	err := GenerateJava(fs, appSpec, "test-output", "com.example.test")
	if err != nil {
		t.Fatalf("GenerateJava() error = %v", err)
	}

	// Read and verify the generated content contains all buckets
	content, err := afero.ReadFile(fs, "test-output/GeneratedSugaClient.java")
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Check that all bucket names are properly normalized and included
	expectedBuckets := []string{"bucketOne", "bucketTwo", "bucketThree"}
	for _, bucket := range expectedBuckets {
		if !strings.Contains(contentStr, bucket) {
			t.Errorf("Generated content does not contain expected bucket: %s", bucket)
		}
	}
}
