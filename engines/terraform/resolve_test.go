package terraform

import (
	"testing"

	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
	"github.com/stretchr/testify/assert"
)

func TestSpecReferenceFromToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expected    *SpecReference
		expectError bool
		errorMsg    string
	}{
		{
			name:  "valid infra reference",
			token: "${infra.resource.property}",
			expected: &SpecReference{
				Source: "infra",
				Path:   []string{"resource", "property"},
			},
			expectError: false,
		},
		{
			name:  "valid self reference",
			token: "${self.my_var}",
			expected: &SpecReference{
				Source: "self",
				Path:   []string{"my_var"},
			},
			expectError: false,
		},
		{
			name:  "valid self reference with nested path",
			token: "${self.my_var.nested.value}",
			expected: &SpecReference{
				Source: "self",
				Path:   []string{"my_var", "nested", "value"},
			},
			expectError: false,
		},
		{
			name:  "valid var reference",
			token: "${var.platform_var}",
			expected: &SpecReference{
				Source: "var",
				Path:   []string{"platform_var"},
			},
			expectError: false,
		},
		{
			name:        "invalid token format - no closing brace",
			token:       "${infra.resource",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - not enough parts",
			token:       "${infra}",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - only source",
			token:       "${self}",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - only var source",
			token:       "${var}",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - self with dot but no path",
			token:       "${self.}",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - var with dot but no path",
			token:       "${var.}",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SpecReferenceFromToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Source, result.Source)
				assert.Equal(t, tt.expected.Path, result.Path)
			}
		})
	}
}

func TestResolveToken_SelfReference(t *testing.T) {
	tests := []struct {
		name        string
		specRef     *SpecReference
		setupVar    bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "self reference without variable name",
			specRef: &SpecReference{
				Source: "self",
				Path:   []string{},
			},
			expectError: true,
			errorMsg:    "references 'self' without specifying a variable",
		},
		{
			name: "self reference with single path",
			specRef: &SpecReference{
				Source: "self",
				Path:   []string{"my_var"},
			},
			setupVar:    true,
			expectError: false,
		},
		{
			name: "self reference with nested path",
			specRef: &SpecReference{
				Source: "self",
				Path:   []string{"my_var", "nested", "value"},
			},
			setupVar:    true,
			expectError: false,
		},
		{
			name: "self reference with non-existent variable",
			specRef: &SpecReference{
				Source: "self",
				Path:   []string{"non_existent"},
			},
			expectError: true,
			errorMsg:    "does not exist for provided blueprint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cdktf.NewApp(nil)
			stack := cdktf.NewTerraformStack(app, jsii.String("test"))

			td := &TerraformDeployment{
				app:                         app,
				stack:                       stack,
				instancedTerraformVariables: make(map[string]map[string]cdktf.TerraformVariable),
			}

			if tt.setupVar {
				td.instancedTerraformVariables["test_intent"] = make(map[string]cdktf.TerraformVariable)
				td.instancedTerraformVariables["test_intent"]["my_var"] = cdktf.NewTerraformVariable(stack, jsii.String("test_var"), &cdktf.TerraformVariableConfig{
					Default: map[string]interface{}{
						"nested": map[string]interface{}{
							"value": "test_value",
						},
					},
				})
			}

			_, err := td.resolveToken("test_intent", tt.specRef)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResolveToken_UnknownSource(t *testing.T) {
	app := cdktf.NewApp(nil)
	stack := cdktf.NewTerraformStack(app, jsii.String("test"))

	td := &TerraformDeployment{
		app:   app,
		stack: stack,
	}

	specRef := &SpecReference{
		Source: "unknown",
		Path:   []string{"something"},
	}

	_, err := td.resolveToken("test_intent", specRef)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown reference source 'unknown'")
}
