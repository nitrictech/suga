package client

import (
	"fmt"

	"github.com/nitrictech/suga/cli/pkg/schema"
)

// Processes permissions into a format suitable for client generation
func ExtractPermissionsForBuckets(appSpec schema.Application) ([]BucketWithPermissions, error) {
	buckets := []BucketWithPermissions{}

	for name, bucketIntent := range appSpec.BucketIntents {
		normalized, err := NewResourceNameNormalizer(name)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize resource name: %w", err)
		}

		// Get all unique permissions across all services
		permissionsMap := make(map[string]bool)
		for _, actions := range bucketIntent.Access {
			expandedActions := schema.ExpandActions(actions, schema.Bucket)
			for _, action := range expandedActions {
				permissionsMap[action] = true
			}
		}

		permissions := make([]string, 0, len(permissionsMap))
		for perm := range permissionsMap {
			permissions = append(permissions, perm)
		}

		// If no permissions are specified, the bucket has no accessible methods
		// This ensures secure by default behavior
		buckets = append(buckets, BucketWithPermissions{
			ResourceNameNormalizer: normalized,
			Permissions:            permissions,
		})
	}

	return buckets, nil
}