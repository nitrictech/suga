package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/engines/terraform"
)

// Repository fetches both platform and plugins in a single API call at construction time
type Repository struct {
	apiClient   *api.SugaApiClient
	currentTeam string
	// The platform reference this repository was created for
	platformRef string
	// Cached platform spec and plugin manifests
	platformSpec    *terraform.PlatformSpec
	pluginManifests map[string]map[string]interface{}
}

var _ terraform.PlatformRepository = (*Repository)(nil)
var _ terraform.PluginRepository = (*Repository)(nil)

// NewRepository creates a repository and fetches the platform and all its plugins upfront
func NewRepository(apiClient *api.SugaApiClient, currentTeam, platformRef string) (*Repository, error) {
	repo := &Repository{
		apiClient:   apiClient,
		currentTeam: currentTeam,
		platformRef: platformRef,
	}

	// Fetch platform and plugins immediately
	if err := repo.fetchPlatformAndPlugins(platformRef); err != nil {
		return nil, err
	}

	return repo, nil
}

// GetPlatform validates that the requested platform matches the one this repository was created for
func (r *Repository) GetPlatform(name string) (*terraform.PlatformSpec, error) {
	// Validate that the requested platform matches what we fetched
	if name != r.platformRef {
		return nil, fmt.Errorf("repository was created for platform %s but %s was requested", r.platformRef, name)
	}

	return r.platformSpec, nil
}

// fetchPlatformAndPlugins fetches the platform and all its plugins in a single API call
func (r *Repository) fetchPlatformAndPlugins(name string) error {
	// Parse the platform name <team>/<platform>@<revision>
	re := regexp.MustCompile(`^(?P<team>[a-z][a-z0-9-]*)/(?P<platform>[a-z][a-z0-9-]*)@(?P<revision>\d+)$`)
	matches := re.FindStringSubmatch(name)

	if matches == nil {
		return fmt.Errorf("invalid platform name format: %s. Expected format: <team>/<platform>@<revision> e.g. %s/aws@1", name, version.CommandName)
	}

	team := matches[re.SubexpIndex("team")]
	platform := matches[re.SubexpIndex("platform")]
	revisionStr := matches[re.SubexpIndex("revision")]

	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		return fmt.Errorf("invalid revision format: %s. Expected integer", revisionStr)
	}

	// Smart ordering: try public first if the platform team doesn't match current user's team
	var platformSpec *terraform.PlatformSpec
	var plugins map[string]map[string]interface{}

	if team != r.currentTeam {
		// Try public access first
		platformSpec, plugins, err = r.apiClient.GetPublicBuildManifest(team, platform, revision)
		if err == nil {
			r.platformSpec = platformSpec
			r.pluginManifests = plugins
			return nil
		}

		// If public fails with 404, it's definitely not found
		if errors.Is(err, api.ErrNotFound) {
			return terraform.ErrPlatformNotFound
		}

		// If public fails for other reasons, try authenticated access
		platformSpec, plugins, err = r.apiClient.GetBuildManifest(team, platform, revision)
		if err != nil {
			if errors.Is(err, api.ErrNotFound) {
				return terraform.ErrPlatformNotFound
			}
			if errors.Is(err, api.ErrUnauthenticated) {
				return terraform.ErrUnauthenticated
			}
			return err
		}
		r.platformSpec = platformSpec
		r.pluginManifests = plugins
		return nil
	}

	// Try authenticated access first
	platformSpec, plugins, err = r.apiClient.GetBuildManifest(team, platform, revision)
	if err != nil {
		// If authentication failed, try public platform access
		if errors.Is(err, api.ErrUnauthenticated) || errors.Is(err, api.ErrNotFound) {
			platformSpec, plugins, err = r.apiClient.GetPublicBuildManifest(team, platform, revision)
			if err != nil {
				// If public access also fails with 404, return platform not found
				if errors.Is(err, api.ErrNotFound) {
					return terraform.ErrPlatformNotFound
				}
				// Return the original authentication error for other public access failures
				return terraform.ErrUnauthenticated
			}
			r.platformSpec = platformSpec
			r.pluginManifests = plugins
			return nil
		}

		// If its a 404, then return platform not found error
		if errors.Is(err, api.ErrNotFound) {
			return terraform.ErrPlatformNotFound
		}

		// return the original error to the engine
		return err
	}

	r.platformSpec = platformSpec
	r.pluginManifests = plugins
	return nil
}

// GetResourcePlugin retrieves a cached plugin manifest
func (r *Repository) GetResourcePlugin(team, libname, version, name string) (*terraform.ResourcePluginManifest, error) {
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

// GetIdentityPlugin retrieves a cached plugin manifest
func (r *Repository) GetIdentityPlugin(team, libname, version, name string) (*terraform.IdentityPluginManifest, error) {
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

// getPluginManifest retrieves a plugin manifest from the cache
func (r *Repository) getPluginManifest(team, libname, version, name string) (interface{}, error) {
	if r.pluginManifests == nil {
		return nil, fmt.Errorf("no plugin manifests cached. GetPlatform must be called first")
	}

	key := fmt.Sprintf("%s/%s/%s/%s", team, libname, version, name)
	manifestData, ok := r.pluginManifests[key]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found in build manifest", key)
	}

	// Check the type field to determine which manifest type to use
	pluginType, ok := manifestData["type"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin manifest missing type field for %s", key)
	}

	if pluginType == "identity" {
		var identityManifest terraform.IdentityPluginManifest
		if err := remapToStruct(manifestData, &identityManifest); err != nil {
			return nil, fmt.Errorf("failed to parse identity plugin manifest for %s: %w", key, err)
		}
		return &identityManifest, nil
	}

	var resourceManifest terraform.ResourcePluginManifest
	if err := remapToStruct(manifestData, &resourceManifest); err != nil {
		return nil, fmt.Errorf("failed to parse resource plugin manifest for %s: %w", key, err)
	}
	return &resourceManifest, nil
}

// remapToStruct is a helper to convert map[string]interface{} to a struct
// This is a simple implementation using JSON marshaling
func remapToStruct(m map[string]interface{}, target interface{}) error {
	bytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}
