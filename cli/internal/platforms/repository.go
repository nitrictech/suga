package platforms

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/engines/terraform"
)

type PlatformRepository struct {
	apiClient   *api.SugaApiClient
	currentTeam string // Current user's team slug for smart API ordering
}

var _ terraform.PlatformRepository = (*PlatformRepository)(nil)

func (r *PlatformRepository) GetPlatform(name string) (*terraform.PlatformSpec, error) {
	// Split the name into team, lib, and revision using a regex <team>/<lib>@<revision>
	re := regexp.MustCompile(`^(?P<team>[a-z][a-z0-9-]*)/(?P<platform>[a-z][a-z0-9-]*)@(?P<revision>\d+)$`)
	matches := re.FindStringSubmatch(name)

	if matches == nil {
		return nil, fmt.Errorf("invalid platform name format: %s. Expected format: <team>/<platform>@<revision> e.g. %s/aws@1", name, version.CommandName)
	}

	// Extract named groups
	team := matches[re.SubexpIndex("team")]
	platform := matches[re.SubexpIndex("platform")]
	revisionStr := matches[re.SubexpIndex("revision")]

	// Convert revision string to integer
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid revision format: %s. Expected integer", revisionStr)
	}

	// Smart ordering: try public first if the platform team doesn't match current user's team
	if team != r.currentTeam {
		// Try public access first
		publicPlatformSpec, publicErr := r.apiClient.GetPublicPlatform(team, platform, revision)
		if publicErr == nil {
			return publicPlatformSpec, nil
		}

		// If public fails with 404, it's definitely not found
		if errors.Is(publicErr, api.ErrNotFound) {
			return nil, terraform.ErrPlatformNotFound
		}

		// If public fails for other reasons, try authenticated access
		platformSpec, err := r.apiClient.GetPlatform(team, platform, revision)
		if err != nil {
			if errors.Is(err, api.ErrNotFound) {
				return nil, terraform.ErrPlatformNotFound
			}
			if errors.Is(err, api.ErrUnauthenticated) {
				return nil, terraform.ErrUnauthenticated
			}
			return nil, err
		}
		return platformSpec, nil
	}

	// Try authenticated access first
	platformSpec, err := r.apiClient.GetPlatform(team, platform, revision)
	if err != nil {
		// If authentication failed, try public platform access
		if errors.Is(err, api.ErrUnauthenticated) || errors.Is(err, api.ErrNotFound) {
			publicPlatformSpec, publicErr := r.apiClient.GetPublicPlatform(team, platform, revision)
			if publicErr != nil {
				// If public access also fails with 404, return platform not found
				if errors.Is(publicErr, api.ErrNotFound) {
					return nil, terraform.ErrPlatformNotFound
				}
				// Return the original authentication error for other public access failures
				return nil, terraform.ErrUnauthenticated
			}
			return publicPlatformSpec, nil
		}

		// If its a 404, then return platform not found error
		if errors.Is(err, api.ErrNotFound) {
			return nil, terraform.ErrPlatformNotFound
		}

		// return the original error to the engine
		return nil, err
	}

	return platformSpec, nil
}


func NewPlatformRepository(apiClient *api.SugaApiClient, currentTeam string) *PlatformRepository {
	return &PlatformRepository{
		apiClient:   apiClient,
		currentTeam: currentTeam,
	}
}
