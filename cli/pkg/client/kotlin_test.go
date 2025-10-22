package client

import (
	"strings"
	"testing"

	"github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/spf13/afero"
)

func TestGenerateKotlin(t *testing.T) {
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

	// Test Kotlin generation
	err := GenerateKotlin(fs, appSpec, "test-output", "com.example.test")
	if err != nil {
		t.Fatalf("GenerateKotlin() error = %v", err)
	}

	// Check if the Kotlin file was created
	exists, err := afero.Exists(fs, "test-output/GeneratedSugaClient.kt")
	if err != nil {
		t.Fatalf("Failed to check if file exists: %v", err)
	}
	if !exists {
		t.Error("Expected GeneratedSugaClient.kt to be created, but it doesn't exist")
	}

	// Read and verify the generated content
	content, err := afero.ReadFile(fs, "test-output/GeneratedSugaClient.kt")
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify key parts of the generated Kotlin code
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

func TestAppSpecToKotlinTemplateData(t *testing.T) {
	appSpec := schema.Application{
		Name:   "test-app",
		Target: "team/platform@1",
		BucketIntents: map[string]*schema.BucketIntent{
			"my-bucket": {},
		},
		ServiceIntents: map[string]*schema.ServiceIntent{
			"api": {}, // Should be ignored for bucket generation
		},
	}

	data, err := AppSpecToKotlinTemplateData(appSpec, "com.test")
	if err != nil {
		t.Fatalf("AppSpecToKotlinTemplateData() error = %v", err)
	}

	if data.Package != "com.test" {
		t.Errorf("Expected package 'com.test', got '%s'", data.Package)
	}

	if len(data.Buckets) != 1 {
		t.Errorf("Expected 1 bucket, got %d", len(data.Buckets))
	}

	if len(data.Buckets) > 0 && data.Buckets[0].Unmodified() != "my-bucket" {
		t.Errorf("Expected bucket name 'my-bucket', got '%s'", data.Buckets[0].Unmodified())
	}
}

func TestGenerateKotlinDefaultValues(t *testing.T) {
	appSpec := schema.Application{
		Name:          "test-app",
		Target:        "team/platform@1",
		BucketIntents: map[string]*schema.BucketIntent{},
	}

	fs := afero.NewMemMapFs()

	// Test with default values
	err := GenerateKotlin(fs, appSpec, "", "")
	if err != nil {
		t.Fatalf("GenerateKotlin() with defaults error = %v", err)
	}

	// Should create Kotlin file in default location
	exists, err := afero.Exists(fs, "suga/kotlin/client/GeneratedSugaClient.kt")
	if err != nil {
		t.Fatalf("Failed to check if default file exists: %v", err)
	}
	if !exists {
		t.Error("Expected GeneratedSugaClient.kt to be created in default location")
	}

	// Read and verify default package
	content, err := afero.ReadFile(fs, "suga/kotlin/client/GeneratedSugaClient.kt")
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "package com.addsuga.client") {
		t.Error("Expected default package com.addsuga.client")
	}
}

func TestValidateKotlinPackageName(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple package",
			packageName: "com.example",
			expectError: false,
		},
		{
			name:        "valid complex package",
			packageName: "com.example.client.v1",
			expectError: false,
		},
		{
			name:        "valid with underscores",
			packageName: "com.example_client.test_package",
			expectError: false,
		},
		{
			name:        "empty package name",
			packageName: "",
			expectError: true,
			errorMsg:    "Kotlin package name cannot be empty",
		},
		{
			name:        "starts with digit",
			packageName: "com.1example",
			expectError: true,
			errorMsg:    "invalid Kotlin package name",
		},
		{
			name:        "contains spaces",
			packageName: "com.example client",
			expectError: true,
			errorMsg:    "invalid Kotlin package name",
		},
		{
			name:        "contains special characters",
			packageName: "com.example@client",
			expectError: true,
			errorMsg:    "invalid Kotlin package name",
		},
		{
			name:        "Java reserved word",
			packageName: "com.class.client",
			expectError: true,
			errorMsg:    "segment 'class' is a Kotlin reserved keyword",
		},
		{
			name:        "Kotlin reserved word - fun",
			packageName: "com.fun.client",
			expectError: true,
			errorMsg:    "segment 'fun' is a Kotlin reserved keyword",
		},
		{
			name:        "Kotlin reserved word - object",
			packageName: "com.object.test",
			expectError: true,
			errorMsg:    "segment 'object' is a Kotlin reserved keyword",
		},
		{
			name:        "Kotlin reserved word - when",
			packageName: "com.when.example",
			expectError: true,
			errorMsg:    "segment 'when' is a Kotlin reserved keyword",
		},
		{
			name:        "multiple reserved words",
			packageName: "com.class.fun.test",
			expectError: true,
			errorMsg:    "segment 'class' is a Kotlin reserved keyword",
		},
		{
			name:        "ends with dot",
			packageName: "com.example.",
			expectError: true,
			errorMsg:    "invalid Kotlin package name",
		},
		{
			name:        "starts with dot",
			packageName: ".com.example",
			expectError: true,
			errorMsg:    "invalid Kotlin package name",
		},
		{
			name:        "consecutive dots",
			packageName: "com..example",
			expectError: true,
			errorMsg:    "invalid Kotlin package name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKotlinPackageName(tt.packageName)

			if tt.expectError {
				if err == nil {
					t.Errorf("validateKotlinPackageName() expected error for package '%s', but got none", tt.packageName)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateKotlinPackageName() expected error message to contain '%s', but got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("validateKotlinPackageName() unexpected error for package '%s': %v", tt.packageName, err)
				}
			}
		})
	}
}

func TestGenerateKotlinWithInvalidPackage(t *testing.T) {
	appSpec := schema.Application{
		Name:          "test-app",
		Target:        "team/platform@1",
		BucketIntents: map[string]*schema.BucketIntent{},
	}

	fs := afero.NewMemMapFs()

	// Test with invalid package name
	err := GenerateKotlin(fs, appSpec, "test-output", "invalid.class.package")
	if err == nil {
		t.Error("GenerateKotlin() expected error for invalid package name, but got none")
	}

	expectedErrMsg := "invalid Kotlin package name"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("GenerateKotlin() expected error message to contain '%s', but got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGenerateKotlinWithMultipleBuckets(t *testing.T) {
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

	err := GenerateKotlin(fs, appSpec, "test-output", "com.example.test")
	if err != nil {
		t.Fatalf("GenerateKotlin() error = %v", err)
	}

	// Read and verify the generated content contains all buckets
	content, err := afero.ReadFile(fs, "test-output/GeneratedSugaClient.kt")
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
