package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/engines/terraform"
)

func (c *SugaApiClient) parsePluginManifest(body []byte, endpointType string) (interface{}, error) {
	// First, unmarshal the response wrapper
	var manifestResponse GetPluginManifestResponse
	err := json.Unmarshal(body, &manifestResponse)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from %s %s plugin details endpoint: %v", version.ProductName, endpointType, err)
	}

	// Convert the manifest map back to JSON for proper unmarshaling
	manifestBytes, err := json.Marshal(manifestResponse.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest from %s %s plugin details endpoint: %v", version.ProductName, endpointType, err)
	}

	// Try to unmarshal as ResourcePluginManifest first
	var pluginManifest terraform.ResourcePluginManifest
	err = json.Unmarshal(manifestBytes, &pluginManifest)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from %s %s plugin details endpoint: %v", version.ProductName, endpointType, err)
	}

	if pluginManifest.Type == "identity" {
		var identityPluginManifest terraform.IdentityPluginManifest
		err = json.Unmarshal(manifestBytes, &identityPluginManifest)
		if err != nil {
			return nil, fmt.Errorf("unexpected response from %s %s plugin details endpoint: %v", version.ProductName, endpointType, err)
		}

		return &identityPluginManifest, nil
	}

	return &pluginManifest, nil
}

// FIXME: Because of the difference in fields between identity and resource plugins we need to return an interface
func (c *SugaApiClient) GetPluginManifest(team, lib, libVersion, name string) (interface{}, error) {
	response, err := c.get(fmt.Sprintf("/api/teams/%s/plugin_libraries/%s/versions/%s/plugins/%s", url.PathEscape(team), url.PathEscape(lib), url.PathEscape(libVersion), url.PathEscape(name)), true)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s plugin details endpoint: %v", version.ProductName, err)
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if response.StatusCode == 401 {
		return nil, ErrUnauthenticated
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("received non 200 response from %s plugin details endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s plugin details endpoint: %v", version.ProductName, err)
	}

	return c.parsePluginManifest(body, "")
}

func (c *SugaApiClient) GetPublicPluginManifest(team, lib, libVersion, name string) (interface{}, error) {
	response, err := c.get(fmt.Sprintf("/api/public/plugin_libraries/%s/%s/versions/%s/plugins/%s", url.PathEscape(team), url.PathEscape(lib), url.PathEscape(libVersion), url.PathEscape(name)), true)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s public plugin details endpoint: %v", version.ProductName, err)
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("received non 200 response from %s public plugin details endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s public plugin details endpoint: %v", version.ProductName, err)
	}

	return c.parsePluginManifest(body, "public")
}

func (c *SugaApiClient) ListPluginLibraries(team string) ([]PluginLibraryWithVersions, error) {
	response, err := c.get(fmt.Sprintf("/api/teams/%s/plugin_libraries", url.PathEscape(team)), true)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		if response.StatusCode == 404 {
			return nil, ErrNotFound
		}

		if response.StatusCode == 401 {
			return nil, ErrUnauthenticated
		}

		return nil, fmt.Errorf("received non 200 response from %s plugin libraries list endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s plugin libraries list endpoint: %v", version.ProductName, err)
	}

	var librariesResponse ListPluginLibrariesResponse
	err = json.Unmarshal(body, &librariesResponse)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from %s plugin libraries list endpoint: %v", version.ProductName, err)
	}
	return librariesResponse.Libraries, nil
}

func (c *SugaApiClient) ListPublicPluginLibraries() ([]PluginLibraryWithVersions, error) {
	response, err := c.get("/api/public/plugin_libraries", true)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		if response.StatusCode == 404 {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("received non 200 response from %s public plugin libraries list endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s public plugin libraries list endpoint: %v", version.ProductName, err)
	}

	var librariesResponse ListPluginLibrariesResponse
	err = json.Unmarshal(body, &librariesResponse)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from %s public plugin libraries list endpoint: %v", version.ProductName, err)
	}
	return librariesResponse.Libraries, nil
}

func (c *SugaApiClient) GetPluginLibraryVersion(team, lib, libVersion string) (*PluginLibraryVersion, error) {
	response, err := c.get(fmt.Sprintf("/api/teams/%s/plugin_libraries/%s/versions/%s", url.PathEscape(team), url.PathEscape(lib), url.PathEscape(libVersion)), true)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		if response.StatusCode == 404 {
			return nil, ErrNotFound
		}

		if response.StatusCode == 401 {
			return nil, ErrUnauthenticated
		}

		return nil, fmt.Errorf("received non 200 response from %s plugin library version endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s plugin library version endpoint: %v", version.ProductName, err)
	}

	var versionResponse GetPluginLibraryVersionResponse
	err = json.Unmarshal(body, &versionResponse)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from %s plugin library version endpoint: %v", version.ProductName, err)
	}
	return versionResponse.Version, nil
}

func (c *SugaApiClient) GetPublicPluginLibraryVersion(team, lib, libVersion string) (*PluginLibraryVersion, error) {
	response, err := c.get(fmt.Sprintf("/api/public/plugin_libraries/%s/%s/versions/%s", url.PathEscape(team), url.PathEscape(lib), url.PathEscape(libVersion)), true)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		if response.StatusCode == 404 {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("received non 200 response from %s public plugin library version endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s public plugin library version endpoint: %v", version.ProductName, err)
	}

	var versionResponse GetPluginLibraryVersionResponse
	err = json.Unmarshal(body, &versionResponse)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from %s public plugin library version endpoint: %v", version.ProductName, err)
	}
	return versionResponse.Version, nil
}
