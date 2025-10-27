package pluginserver

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nitrictech/suga/engines/terraform"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// LocalPluginRepository implements terraform.PluginRepository
// It reads plugins directly from the local filesystem for development
type LocalPluginRepository struct {
	fs          afero.Fs
	pluginPaths []string
}

func NewLocalPluginRepository(fs afero.Fs, pluginPaths []string) *LocalPluginRepository {
	return &LocalPluginRepository{
		fs:          fs,
		pluginPaths: pluginPaths,
	}
}

func (r *LocalPluginRepository) GetResourcePlugin(team, libname, version, name string) (*terraform.ResourcePluginManifest, error) {
	manifestPath, err := r.findPluginManifest(team, libname, version, name)
	if err != nil {
		return nil, err
	}

	manifest, err := r.readManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	resourceManifest, ok := manifest.(*terraform.ResourcePluginManifest)
	if !ok {
		return nil, fmt.Errorf("plugin %s/%s/%s@%s is not a resource plugin", team, libname, name, version)
	}

	return resourceManifest, nil
}

func (r *LocalPluginRepository) GetIdentityPlugin(team, libname, version, name string) (*terraform.IdentityPluginManifest, error) {
	manifestPath, err := r.findPluginManifest(team, libname, version, name)
	if err != nil {
		return nil, err
	}

	manifest, err := r.readManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	identityManifest, ok := manifest.(*terraform.IdentityPluginManifest)
	if !ok {
		return nil, fmt.Errorf("plugin %s/%s/%s@%s is not an identity plugin", team, libname, name, version)
	}

	return identityManifest, nil
}

// findPluginManifest searches for a plugin manifest in the configured paths
// Expected structure: {pluginPath}/{team}/{lib}/{version}/{name}/manifest.{yaml,yml,json}
func (r *LocalPluginRepository) findPluginManifest(team, lib, version, name string) (string, error) {
	for _, basePath := range r.pluginPaths {
		// Try different manifest file extensions
		for _, ext := range []string{"yaml", "yml", "json"} {
			manifestPath := filepath.Join(basePath, team, lib, version, name, fmt.Sprintf("manifest.%s", ext))
			if exists, _ := afero.Exists(r.fs, manifestPath); exists {
				return manifestPath, nil
			}
		}
	}

	return "", fmt.Errorf("plugin %s/%s/%s@%s not found in local paths: %v", team, lib, name, version, r.pluginPaths)
}

// readManifest reads and parses a plugin manifest file
func (r *LocalPluginRepository) readManifest(path string) (interface{}, error) {
	data, err := afero.ReadFile(r.fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest map[string]interface{}

	// Parse based on file extension
	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse manifest YAML: %w", err)
		}
	}

	// Convert back to JSON for proper unmarshaling into typed structs
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Try to unmarshal as ResourcePluginManifest first
	var pluginManifest terraform.ResourcePluginManifest
	if err := json.Unmarshal(manifestBytes, &pluginManifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// Check if it's an identity plugin
	if pluginManifest.Type == "identity" {
		var identityManifest terraform.IdentityPluginManifest
		if err := json.Unmarshal(manifestBytes, &identityManifest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal identity manifest: %w", err)
		}
		return &identityManifest, nil
	}

	return &pluginManifest, nil
}
