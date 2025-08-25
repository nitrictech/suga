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
	apiClient *api.SugaApiClient
}

var _ terraform.PlatformRepository = (*PlatformRepository)(nil)

func (r *PlatformRepository) GetPlatform(name string) (*terraform.PlatformSpec, error) {
	// Split the name into team, lib, and revision using a regex <team>/<lib>@<revision>
	re := regexp.MustCompile(`^(?P<team>[^/]+)/(?P<platform>[^@]+)@(?P<revision>\d+)$`)
	matches := re.FindStringSubmatch(name)

	if matches == nil {
		return nil, fmt.Errorf("invalid platform name format: %s. Expected format: <team>/<lib>@<revision> e.g. %s/aws@1", version.CommandName, name)
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

func NewPlatformRepository(apiClient *api.SugaApiClient) *PlatformRepository {
	return &PlatformRepository{
		apiClient: apiClient,
	}
}
