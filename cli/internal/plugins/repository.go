package plugins

import (
	"errors"
	"fmt"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/engines/terraform"
)

type PluginRepository struct {
	apiClient   *api.SugaApiClient
	currentTeam string // Current user's team slug for smart API ordering
}

func (r *PluginRepository) GetResourcePlugin(team, libname, version, name string) (*terraform.ResourcePluginManifest, error) {
	pluginManifest, err := r.getPluginManifest(team, libname, version, name)
	if err != nil {
		return nil, err
	}

	resourcePluginManifest, ok := pluginManifest.(*terraform.ResourcePluginManifest)
	if !ok {
		return nil, fmt.Errorf("encountered malformed manifest for plugin %s/%s/%s@%s", team, libname, name, version)
	}

	return resourcePluginManifest, nil
}

func (r *PluginRepository) GetIdentityPlugin(team, libname, version, name string) (*terraform.IdentityPluginManifest, error) {
	pluginManifest, err := r.getPluginManifest(team, libname, version, name)
	if err != nil {
		return nil, err
	}

	identityPluginManifest, ok := pluginManifest.(*terraform.IdentityPluginManifest)
	if !ok {
		return nil, fmt.Errorf("encountered malformed manifest for plugin %s/%s/%s@%s", team, libname, name, version)
	}

	return identityPluginManifest, nil
}

// getPluginManifest handles the smart ordering logic for both resource and identity plugins
func (r *PluginRepository) getPluginManifest(team, libname, version, name string) (interface{}, error) {
	// Smart ordering: try public first if the plugin team doesn't match current user's team
	if team != r.currentTeam {
		// Try public access first
		publicPluginManifest, publicErr := r.apiClient.GetPublicPluginManifest(team, libname, version, name)
		if publicErr == nil {
			return publicPluginManifest, nil
		}

		// If public fails with 404, it's definitely not found
		if errors.Is(publicErr, api.ErrNotFound) {
			return nil, fmt.Errorf("plugin %s/%s/%s@%s not found", team, libname, name, version)
		}

		// If public fails for other reasons, try authenticated access
		pluginManifest, err := r.apiClient.GetPluginManifest(team, libname, version, name)
		if err != nil {
			if errors.Is(err, api.ErrNotFound) {
				return nil, fmt.Errorf("plugin %s/%s/%s@%s not found", team, libname, name, version)
			}
			return nil, err
		}
		return pluginManifest, nil
	}

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
			return publicPluginManifest, nil
		} else {
			// Return the original error for other cases
			return nil, err
		}
	}

	return pluginManifest, nil
}


func NewPluginRepository(apiClient *api.SugaApiClient, currentTeam string) *PluginRepository {
	return &PluginRepository{
		apiClient:   apiClient,
		currentTeam: currentTeam,
	}
}
