package pluginserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/engines/terraform"
)

// HTTPPluginRepository fetches plugins from an HTTP server (local or remote)
// This is used when a library has a ServerURL specified
type HTTPPluginRepository struct {
	serverURL string
	client    *http.Client
}

func NewHTTPPluginRepository(serverURL string) *HTTPPluginRepository {
	base := strings.TrimRight(serverURL, "/")
	return &HTTPPluginRepository{
		serverURL: base,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *HTTPPluginRepository) GetResourcePlugin(team, libname, version, name string) (*terraform.ResourcePluginManifest, error) {
	pluginManifest, err := r.getPluginManifest(team, libname, version, name)
	if err != nil {
		return nil, err
	}

	resourcePluginManifest, ok := pluginManifest.(*terraform.ResourcePluginManifest)
	if !ok {
		return nil, fmt.Errorf("encountered malformed manifest for plugin %s/%s/%s@%s", team, libname, name, version)
	}

	// Transform relative Terraform module paths to HTTP URLs
	r.transformTerraformModulePath(&resourcePluginManifest.Deployment, name)

	return resourcePluginManifest, nil
}

func (r *HTTPPluginRepository) GetIdentityPlugin(team, libname, version, name string) (*terraform.IdentityPluginManifest, error) {
	pluginManifest, err := r.getPluginManifest(team, libname, version, name)
	if err != nil {
		return nil, err
	}

	identityPluginManifest, ok := pluginManifest.(*terraform.IdentityPluginManifest)
	if !ok {
		return nil, fmt.Errorf("encountered malformed manifest for plugin %s/%s/%s@%s", team, libname, name, version)
	}

	// Transform relative Terraform module paths to HTTP URLs
	r.transformTerraformModulePath(&identityPluginManifest.Deployment, name)

	return identityPluginManifest, nil
}

func (r *HTTPPluginRepository) getPluginManifest(team, libname, version, name string) (interface{}, error) {
	// Try public endpoint (matches suga server behavior)
	publicURL := fmt.Sprintf("%s/api/public/plugin_libraries/%s/%s/versions/%s/plugins/%s",
		r.serverURL,
		url.PathEscape(team),
		url.PathEscape(libname),
		url.PathEscape(version),
		url.PathEscape(name))

	resp, err := r.client.Get(publicURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to plugin server at %s: %v", r.serverURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("plugin %s/%s/%s@%s not found on server %s", team, libname, name, version, r.serverURL)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-200 response from plugin server: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from plugin server: %v", err)
	}

	return parsePluginManifest(body)
}

func parsePluginManifest(body []byte) (interface{}, error) {
	// First, unmarshal the response wrapper
	var manifestResponse api.GetPluginManifestResponse
	err := json.Unmarshal(body, &manifestResponse)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from plugin server: %v", err)
	}

	// Convert the manifest map back to JSON for proper unmarshaling
	manifestBytes, err := json.Marshal(manifestResponse.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest from plugin server: %v", err)
	}

	// Try to unmarshal as ResourcePluginManifest first
	var pluginManifest terraform.ResourcePluginManifest
	err = json.Unmarshal(manifestBytes, &pluginManifest)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from plugin server: %v", err)
	}

	if pluginManifest.Type == "identity" {
		var identityPluginManifest terraform.IdentityPluginManifest
		err = json.Unmarshal(manifestBytes, &identityPluginManifest)
		if err != nil {
			return nil, fmt.Errorf("unexpected response from plugin server: %v", err)
		}

		return &identityPluginManifest, nil
	}

	return &pluginManifest, nil
}

// transformTerraformModulePath converts relative Terraform module paths to HTTP URLs
// The pluginName is included in the URL so the server can find the plugin directory and resolve the module
func (r *HTTPPluginRepository) transformTerraformModulePath(deployment *terraform.DeploymentModule, pluginName string) {
	// Check if the path is relative (starts with ./ or ../)
	if strings.HasPrefix(deployment.Terraform, "./") || strings.HasPrefix(deployment.Terraform, "../") {
		// Convert to HTTP URL with just the plugin name
		// The server will use the manifest to find the actual module path
		deployment.Terraform = fmt.Sprintf("%s/terraform-modules/%s.zip", r.serverURL, pluginName)
	}
}
