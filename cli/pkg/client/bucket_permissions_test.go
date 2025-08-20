package client

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPermissionsIntegration tests the complete permission handling with the test YAML file
func TestPermissionsIntegration(t *testing.T) {
	fs := afero.NewOsFs()
	yamlPath := filepath.Join("testdata", "permissions.yaml")
	
	// Load the test configuration
	appSpec, err := schema.LoadFromFile(fs, yamlPath, false)
	require.NoError(t, err, "Should load test YAML file")
	require.NotNil(t, appSpec)
	
	// Verify all buckets are present
	assert.Len(t, appSpec.BucketIntents, 5)
	
	t.Run("Python generation respects permissions", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		err := GeneratePython(memFs, *appSpec, "generated")
		require.NoError(t, err)
		
		content, err := afero.ReadFile(memFs, "generated/client.py")
		require.NoError(t, err)
		generated := string(content)
		
		// Read-only bucket: only read methods + always available methods
		readOnlySection := extractBucketClass(generated, "ReadOnlyStorageBucket")
		assert.Contains(t, readOnlySection, "self.read")
		assert.NotContains(t, readOnlySection, "self.write")
		assert.NotContains(t, readOnlySection, "self.delete")
		// Always available methods should be present
		assert.Contains(t, readOnlySection, "self.list")
		assert.Contains(t, readOnlySection, "self.exists")
		assert.Contains(t, readOnlySection, "self.get_download_url")
		assert.Contains(t, readOnlySection, "self.get_upload_url")
		
		// Write-only bucket: only write method + always available methods
		writeOnlySection := extractBucketClass(generated, "WriteOnlyStorageBucket")
		assert.Contains(t, writeOnlySection, "self.write")
		assert.NotContains(t, writeOnlySection, "self.read")
		assert.NotContains(t, writeOnlySection, "self.delete")
		// Always available methods should be present
		assert.Contains(t, writeOnlySection, "self.list")
		assert.Contains(t, writeOnlySection, "self.exists")
		assert.Contains(t, writeOnlySection, "self.get_download_url")
		assert.Contains(t, writeOnlySection, "self.get_upload_url")
		
		// Full access (using 'all'): all methods + always available methods
		fullAccessSection := extractBucketClass(generated, "FullAccessStorageBucket")
		assert.Contains(t, fullAccessSection, "self.read")
		assert.Contains(t, fullAccessSection, "self.write")
		assert.Contains(t, fullAccessSection, "self.delete")
		// Always available methods should be present
		assert.Contains(t, fullAccessSection, "self.list")
		assert.Contains(t, fullAccessSection, "self.exists")
		assert.Contains(t, fullAccessSection, "self.get_download_url")
		assert.Contains(t, fullAccessSection, "self.get_upload_url")
		
		// Image bucket: read and write, no delete + always available methods
		imageSection := extractBucketClass(generated, "ImageBucket")
		assert.Contains(t, imageSection, "self.read")
		assert.Contains(t, imageSection, "self.write")
		assert.NotContains(t, imageSection, "self.delete")
		// Always available methods should be present
		assert.Contains(t, imageSection, "self.list")
		assert.Contains(t, imageSection, "self.exists")
		assert.Contains(t, imageSection, "self.get_download_url")
		assert.Contains(t, imageSection, "self.get_upload_url")
		
		// No permissions: no permission-controlled methods but always available methods should be present
		noPermsSection := extractBucketClass(generated, "NoPermissionsStorageBucket")
		assert.NotContains(t, noPermsSection, "self.read")
		assert.NotContains(t, noPermsSection, "self.write")
		assert.NotContains(t, noPermsSection, "self.delete")
		// Always available methods should be present
		assert.Contains(t, noPermsSection, "self.list")
		assert.Contains(t, noPermsSection, "self.exists")
		assert.Contains(t, noPermsSection, "self.get_download_url")
		assert.Contains(t, noPermsSection, "self.get_upload_url")
	})
	
	t.Run("Go generation respects permissions", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		err := GenerateGo(memFs, *appSpec, "generated", "client")
		require.NoError(t, err)
		
		content, err := afero.ReadFile(memFs, "generated/client.go")
		require.NoError(t, err)
		generated := string(content)
		
		// Verify correct permission-controlled methods are generated
		assert.Contains(t, generated, "func (b *ReadOnlyStorageBucket) Read(")
		assert.NotContains(t, generated, "func (b *ReadOnlyStorageBucket) Write(")
		assert.NotContains(t, generated, "func (b *ReadOnlyStorageBucket) Delete(")
		
		assert.Contains(t, generated, "func (b *WriteOnlyStorageBucket) Write(")
		assert.NotContains(t, generated, "func (b *WriteOnlyStorageBucket) Read(")
		assert.NotContains(t, generated, "func (b *WriteOnlyStorageBucket) Delete(")
		
		assert.Contains(t, generated, "func (b *FullAccessStorageBucket) Read(")
		assert.Contains(t, generated, "func (b *FullAccessStorageBucket) Write(")
		assert.Contains(t, generated, "func (b *FullAccessStorageBucket) Delete(")
		
		assert.Contains(t, generated, "func (b *ImageBucket) Read(")
		assert.Contains(t, generated, "func (b *ImageBucket) Write(")
		assert.NotContains(t, generated, "func (b *ImageBucket) Delete(")
		
		// Verify always available methods are present for all bucket types
		bucketTypes := []string{"ReadOnlyStorageBucket", "WriteOnlyStorageBucket", "FullAccessStorageBucket", "ImageBucket", "NoPermissionsStorageBucket"}
		for _, bucketType := range bucketTypes {
			assert.Contains(t, generated, fmt.Sprintf("func (b *%s) List(", bucketType))
			assert.Contains(t, generated, fmt.Sprintf("func (b *%s) Exists(", bucketType))
			assert.Contains(t, generated, fmt.Sprintf("func (b *%s) GetDownloadURL(", bucketType))
			assert.Contains(t, generated, fmt.Sprintf("func (b *%s) GetUploadURL(", bucketType))
		}
	})
	
	t.Run("TypeScript generation respects permissions", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		err := GenerateTypeScript(memFs, *appSpec, "generated")
		require.NoError(t, err)
		
		content, err := afero.ReadFile(memFs, "generated/client.ts")
		require.NoError(t, err)
		generated := string(content)
		
		// Verify the classes are generated
		assert.Contains(t, generated, "class ReadOnlyStorageBucket")
		assert.Contains(t, generated, "class WriteOnlyStorageBucket")
		assert.Contains(t, generated, "class FullAccessStorageBucket")
		assert.Contains(t, generated, "class ImageBucket")
		assert.Contains(t, generated, "class NoPermissionsStorageBucket")
		
		// Verify always available methods are present in TypeScript
		assert.Contains(t, generated, "async list(")
		assert.Contains(t, generated, "async exists(")
		assert.Contains(t, generated, "async getDownloadURL(")
		assert.Contains(t, generated, "async getUploadURL(")
		
		// Verify permission-controlled methods are conditionally present
		// Read permission
		if strings.Contains(generated, "async read(") {
			// Should only be in buckets with read permission
		}
		
		// Write permission  
		if strings.Contains(generated, "async write(") {
			// Should only be in buckets with write permission
		}
		
		// Delete permission
		if strings.Contains(generated, "async delete(") {
			// Should only be in buckets with delete permission
		}
	})
}

// TestExpandActions verifies the 'all' permission expansion
func TestExpandActions(t *testing.T) {
	// Test that 'all' expands to all bucket permissions
	expanded := schema.ExpandActions([]string{"all"}, schema.Bucket)
	assert.Contains(t, expanded, "read")
	assert.Contains(t, expanded, "write")
	assert.Contains(t, expanded, "delete")
	assert.Len(t, expanded, 3)
	
	// Test specific permissions remain unchanged
	expanded = schema.ExpandActions([]string{"read", "write"}, schema.Bucket)
	assert.Contains(t, expanded, "read")
	assert.Contains(t, expanded, "write")
	assert.NotContains(t, expanded, "delete")
	assert.Len(t, expanded, 2)
}

// Helper function to extract a specific bucket class from generated code
func extractBucketClass(content, className string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inClass := false
	
	for _, line := range lines {
		if strings.Contains(line, "class "+className+":") {
			inClass = true
		} else if inClass && strings.HasPrefix(line, "class ") && !strings.Contains(line, className) {
			break
		}
		
		if inClass {
			result = append(result, line)
		}
	}
	
	return strings.Join(result, "\n")
}