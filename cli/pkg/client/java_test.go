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

	// Check if the file was created
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
		"package com.example.test;",
		"public class GeneratedSugaClient extends SugaClient",
		"public final Bucket myTestBucket;",
		"this.myTestBucket = createBucket(\"my-test-bucket\");",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated content does not contain expected string: %s", expected)
		}
	}
}

func TestAppSpecToJavaTemplateData(t *testing.T) {
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

	data, err := AppSpecToJavaTemplateData(appSpec, "com.test")
	if err != nil {
		t.Fatalf("AppSpecToJavaTemplateData() error = %v", err)
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

	// Should create file in default location
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
	if !strings.Contains(contentStr, "package com.nitric.suga.client;") {
		t.Error("Expected default package com.nitric.suga.client")
	}
}
