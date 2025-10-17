package pluginserver

import (
	"fmt"

	"github.com/nitrictech/suga/engines/terraform"
)

// CompositePluginRepository routes plugin requests to the appropriate repository
// based on whether the library has a ServerURL (local development) or not (API)
type CompositePluginRepository struct {
	platform      *terraform.PlatformSpec
	defaultRepo   terraform.PluginRepository
	serverRepos   map[string]terraform.PluginRepository // keyed by ServerURL
}

func NewCompositePluginRepository(platform *terraform.PlatformSpec, defaultRepo terraform.PluginRepository) *CompositePluginRepository {
	return &CompositePluginRepository{
		platform:    platform,
		defaultRepo: defaultRepo,
		serverRepos: make(map[string]terraform.PluginRepository),
	}
}

func (r *CompositePluginRepository) GetResourcePlugin(team, libname, version, name string) (*terraform.ResourcePluginManifest, error) {
	repo, err := r.getRepositoryForLibrary(team, libname)
	if err != nil {
		return nil, err
	}

	return repo.GetResourcePlugin(team, libname, version, name)
}

func (r *CompositePluginRepository) GetIdentityPlugin(team, libname, version, name string) (*terraform.IdentityPluginManifest, error) {
	repo, err := r.getRepositoryForLibrary(team, libname)
	if err != nil {
		return nil, err
	}

	return repo.GetIdentityPlugin(team, libname, version, name)
}

func (r *CompositePluginRepository) getRepositoryForLibrary(team, libname string) (terraform.PluginRepository, error) {
	// Find the library in the platform spec
	libID := terraform.NewLibraryID(team, libname)
	lib, err := r.platform.GetLibrary(libID)
	if err != nil {
		return nil, fmt.Errorf("library %s not found in platform: %w", libID, err)
	}

	// If the library has a ServerURL, use HTTP repository
	if lib.ServerURL != "" {
		// Check if we already have a repository for this server
		if repo, ok := r.serverRepos[lib.ServerURL]; ok {
			return repo, nil
		}

		// Create a new HTTP repository for this server
		repo := NewHTTPPluginRepository(lib.ServerURL)
		r.serverRepos[lib.ServerURL] = repo
		return repo, nil
	}

	// Otherwise use the default repository (API)
	return r.defaultRepo, nil
}
