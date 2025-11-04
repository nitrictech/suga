package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceBlueprint_ValidateServiceAccess(t *testing.T) {
	tests := []struct {
		name            string
		usableBy        []string
		serviceSubtype  string
		resourceName    string
		resourceType    string
		expectError     bool
		errorContains   string
	}{
		{
			name:           "empty usable_by allows all service types",
			usableBy:       []string{},
			serviceSubtype: "lambda",
			resourceName:   "my_bucket",
			resourceType:   "bucket",
			expectError:    false,
		},
		{
			name:           "nil usable_by allows all service types",
			usableBy:       nil,
			serviceSubtype: "fargate",
			resourceName:   "my_database",
			resourceType:   "database",
			expectError:    false,
		},
		{
			name:           "service subtype in allowed list succeeds",
			usableBy:       []string{"web", "worker", "api"},
			serviceSubtype: "web",
			resourceName:   "users_db",
			resourceType:   "database",
			expectError:    false,
		},
		{
			name:           "service subtype not in allowed list fails",
			usableBy:       []string{"web", "worker"},
			serviceSubtype: "lambda",
			resourceName:   "users_db",
			resourceType:   "database",
			expectError:    true,
			errorContains:  "database 'users_db' cannot be accessed by service subtype 'lambda'",
		},
		{
			name:           "single allowed service type succeeds",
			usableBy:       []string{"fargate"},
			serviceSubtype: "fargate",
			resourceName:   "assets",
			resourceType:   "bucket",
			expectError:    false,
		},
		{
			name:           "single allowed service type with different subtype fails",
			usableBy:       []string{"fargate"},
			serviceSubtype: "lambda",
			resourceName:   "assets",
			resourceType:   "bucket",
			expectError:    true,
			errorContains:  "bucket 'assets' cannot be accessed by service subtype 'lambda'",
		},
		{
			name:           "error message includes allowed types list",
			usableBy:       []string{"web", "worker", "cron"},
			serviceSubtype: "lambda",
			resourceName:   "cache_db",
			resourceType:   "database",
			expectError:    true,
			errorContains:  "only usable by service types: [web worker cron]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprint := &ResourceBlueprint{
				UsableBy: tt.usableBy,
			}

			err := blueprint.ValidateServiceAccess(tt.serviceSubtype, tt.resourceName, tt.resourceType)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
