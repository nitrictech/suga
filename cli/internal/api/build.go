package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/engines/terraform"
)

// GetBuildManifest fetches the platform and all its plugins in a single call
func (c *SugaApiClient) GetBuildManifest(team, platform string, revision int) (*terraform.PlatformSpec, map[string]map[string]any, error) {
	response, err := c.get(fmt.Sprintf("/api/teams/%s/platforms/%s/revisions/%d/build-manifest", url.PathEscape(team), url.PathEscape(platform), revision), true)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		if response.StatusCode == 404 {
			return nil, nil, ErrNotFound
		}

		if response.StatusCode == 401 {
			return nil, nil, ErrUnauthenticated
		}

		return nil, nil, fmt.Errorf("received non 200 response from %s build manifest endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response from %s build manifest endpoint: %v", version.ProductName, err)
	}

	var buildManifest GetBuildManifestResponse
	err = json.Unmarshal(body, &buildManifest)
	if err != nil {
		return nil, nil, fmt.Errorf("unexpected response from %s build manifest endpoint: %v", version.ProductName, err)
	}

	return buildManifest.Platform, buildManifest.Plugins, nil
}

// GetPublicBuildManifest fetches the platform and all its plugins for a public platform
func (c *SugaApiClient) GetPublicBuildManifest(team, platform string, revision int) (*terraform.PlatformSpec, map[string]map[string]any, error) {
	response, err := c.get(fmt.Sprintf("/api/public/platforms/%s/%s/revisions/%d/build-manifest", url.PathEscape(team), url.PathEscape(platform), revision), true)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		if response.StatusCode == 404 {
			return nil, nil, ErrNotFound
		}

		return nil, nil, fmt.Errorf("received non 200 response from %s public build manifest endpoint: %d", version.ProductName, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response from %s public build manifest endpoint: %v", version.ProductName, err)
	}

	var buildManifest GetBuildManifestResponse
	err = json.Unmarshal(body, &buildManifest)
	if err != nil {
		return nil, nil, fmt.Errorf("unexpected response from %s public build manifest endpoint: %v", version.ProductName, err)
	}

	return buildManifest.Platform, buildManifest.Plugins, nil
}
