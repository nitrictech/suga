package plugins

import (
	"errors"
	"fmt"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/engines/terraform"
)

type PluginRepository struct {
	apiClient *api.SugaApiClient
}

func (r *PluginRepository) GetResourcePlugin(team, libname, version, name string) (*terraform.ResourcePluginManifest, error) {
	// Try authenticated access first
	pluginManifest, err := r.apiClient.GetPluginManifest(team, libname, version, name)
	if err != nil {
		// If authentication failed or not found, try public plugin access
		if errors.Is(err, api.ErrUnauthenticated) || errors.Is(err, api.ErrNotFound) {
			publicPluginManifest, publicErr := r.apiClient.GetPublicPluginManifest(team, libname, version, name)
			if publicErr != nil {
				// If public access also fails with 404, return plugin not found
				if errors.Is(publicErr, api.ErrNotFound) {
					return nil, fmt.Errorf("plugin %s/%s/%s@%s not found", team, libname, name, version)
				}
				// Return the public error for other failures
				return nil, publicErr
			}
			pluginManifest = publicPluginManifest
		} else {
			// Return the original error for other cases
			return nil, err
		}
	}

	resourcePluginManifest, ok := pluginManifest.(*terraform.ResourcePluginManifest)
	if !ok {
		return nil, fmt.Errorf("encountered malformed manifest for plugin %s/%s/%s@%s: %v", team, libname, name, version, err)
	}

	return resourcePluginManifest, nil
}

func (r *PluginRepository) GetIdentityPlugin(team, libname, version, name string) (*terraform.IdentityPluginManifest, error) {
	// Try authenticated access first
	pluginManifest, err := r.apiClient.GetPluginManifest(team, libname, version, name)
	if err != nil {
		// If authentication failed or not found, try public plugin access
		if errors.Is(err, api.ErrUnauthenticated) || errors.Is(err, api.ErrNotFound) {
			publicPluginManifest, publicErr := r.apiClient.GetPublicPluginManifest(team, libname, version, name)
			if publicErr != nil {
				// If public access also fails with 404, return plugin not found
				if errors.Is(publicErr, api.ErrNotFound) {
					return nil, fmt.Errorf("plugin %s/%s/%s@%s not found", team, libname, name, version)
				}
				// Return the public error for other failures
				return nil, publicErr
			}
			pluginManifest = publicPluginManifest
		} else {
			// Return the original error for other cases
			return nil, err
		}
	}

	identityPluginManifest, ok := pluginManifest.(*terraform.IdentityPluginManifest)
	if !ok {
		return nil, fmt.Errorf("encountered malformed manifest for plugin %s/%s/%s@%s: %v", team, libname, name, version, err)
	}

	return identityPluginManifest, nil
}

func NewPluginRepository(apiClient *api.SugaApiClient) *PluginRepository {
	return &PluginRepository{
		apiClient: apiClient,
	}
}
