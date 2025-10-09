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
			token: "infra.resource.property",
			expected: &SpecReference{
				Source: "infra",
				Path:   []string{"resource", "property"},
			},
			expectError: false,
		},
		{
			name:  "valid infra reference with ${} wrapping",
			token: "${infra.resource.property}",
			expected: &SpecReference{
				Source: "infra",
				Path:   []string{"resource", "property"},
			},
			expectError: false,
		},
		{
			name:  "valid self reference",
			token: "self.my_var",
			expected: &SpecReference{
				Source: "self",
				Path:   []string{"my_var"},
			},
			expectError: false,
		},
		{
			name:  "valid self reference with ${} wrapping",
			token: "${self.my_var}",
			expected: &SpecReference{
				Source: "self",
				Path:   []string{"my_var"},
			},
			expectError: false,
		},
		{
			name:  "valid self reference with nested path",
			token: "self.my_var.nested.value",
			expected: &SpecReference{
				Source: "self",
				Path:   []string{"my_var", "nested", "value"},
			},
			expectError: false,
		},
		{
			name:  "valid var reference",
			token: "var.platform_var",
			expected: &SpecReference{
				Source: "var",
				Path:   []string{"platform_var"},
			},
			expectError: false,
		},
		{
			name:  "valid var reference with nested path",
			token: "var.platform_var.nested.value",
			expected: &SpecReference{
				Source: "var",
				Path:   []string{"platform_var", "nested", "value"},
			},
			expectError: false,
		},
		{
			name:  "valid token with unclosed brace extracts token",
			token: "${infra.resource",
			expected: &SpecReference{
				Source: "infra",
				Path:   []string{"resource"},
			},
			expectError: false,
		},
		{
			name:        "invalid token format - not enough parts",
			token:       "infra",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - only source",
			token:       "self",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - only var source",
			token:       "var",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - self with dot but no path",
			token:       "self.",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid reference format",
		},
		{
			name:        "invalid token format - var with dot but no path",
			token:       "var.",
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

func TestResolveToken_VarReference(t *testing.T) {
	tests := []struct {
		name        string
		specRef     *SpecReference
		setupVar    bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "var reference without variable name",
			specRef: &SpecReference{
				Source: "var",
				Path:   []string{},
			},
			expectError: true,
			errorMsg:    "doesn't contain a valid variable reference",
		},
		{
			name: "var reference with single path",
			specRef: &SpecReference{
				Source: "var",
				Path:   []string{"platform_var"},
			},
			setupVar:    true,
			expectError: false,
		},
		{
			name: "var reference with nested path",
			specRef: &SpecReference{
				Source: "var",
				Path:   []string{"platform_var", "nested", "value"},
			},
			setupVar:    true,
			expectError: false,
		},
		{
			name: "var reference with non-existent variable",
			specRef: &SpecReference{
				Source: "var",
				Path:   []string{"non_existent"},
			},
			expectError: true,
			errorMsg:    "does not exist for this platform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cdktf.NewApp(nil)
			stack := cdktf.NewTerraformStack(app, jsii.String("test"))

			engine := &TerraformEngine{
				platform: &PlatformSpec{
					Variables: make(map[string]Variable),
				},
			}

			td := &TerraformDeployment{
				app:                app,
				stack:              stack,
				engine:             engine,
				terraformVariables: make(map[string]cdktf.TerraformVariable),
			}

			if tt.setupVar {
				engine.platform.Variables["platform_var"] = Variable{
					Type:        "object",
					Description: "Test platform variable",
					Default: map[string]interface{}{
						"nested": map[string]interface{}{
							"value": "test_value",
						},
					},
				}
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

func TestResolveTokenValue(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		setupVar    bool
		expectToken bool
		expectError bool
	}{
		{
			name:        "plain string without tokens",
			input:       "just a regular string",
			expectToken: false,
			expectError: false,
		},
		{
			name:        "single token only",
			input:       "self.my_var",
			setupVar:    true,
			expectToken: true,
			expectError: false,
		},
		{
			name:        "single token only with ${} wrapping",
			input:       "${self.my_var}",
			setupVar:    true,
			expectToken: true,
			expectError: false,
		},
		{
			name:        "string with multiple tokens",
			input:       "${self.var1}-${self.var2}",
			setupVar:    true,
			expectToken: true,
			expectError: false,
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
					Default: "test_value",
				})
				td.instancedTerraformVariables["test_intent"]["var1"] = cdktf.NewTerraformVariable(stack, jsii.String("test_var1"), &cdktf.TerraformVariableConfig{
					Default: "value1",
				})
				td.instancedTerraformVariables["test_intent"]["var2"] = cdktf.NewTerraformVariable(stack, jsii.String("test_var2"), &cdktf.TerraformVariableConfig{
					Default: "value2",
				})
			}

			result, err := td.resolveTokenValue("test_intent", tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.expectToken {
					assert.Equal(t, tt.input, result)
				} else {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestResolveStringInterpolation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		setupVars   map[string]string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:  "string with two tokens",
			input: "prefix-${self.var1-middle}-${self.var2-suffix}",
			setupVars: map[string]string{
				"var1-middle": "value1",
				"var2-suffix": "value2",
			},
			expectError: false,
		},
		{
			name:  "string with adjacent tokens",
			input: "${self.var1}${self.var2}",
			setupVars: map[string]string{
				"var1": "value1",
				"var2": "value2",
			},
			expectError: false,
		},
		{
			name:  "string with token at start",
			input: "self.var1-suffix",
			setupVars: map[string]string{
				"var1-suffix": "value1",
			},
			expectError: false,
		},
		{
			name:  "string with token at end",
			input: "prefix-self.var1",
			setupVars: map[string]string{
				"var1": "value1",
			},
			expectError: false,
		},
		{
			name:  "string with ${} wrapped tokens",
			input: "prefix-${self.var1}-middle-${self.var2}-suffix",
			setupVars: map[string]string{
				"var1": "value1",
				"var2": "value2",
			},
			expectError: false,
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

			td.instancedTerraformVariables["test_intent"] = make(map[string]cdktf.TerraformVariable)
			for varName, varValue := range tt.setupVars {
				td.instancedTerraformVariables["test_intent"][varName] = cdktf.NewTerraformVariable(stack, jsii.String("test_"+varName), &cdktf.TerraformVariableConfig{
					Default: varValue,
				})
			}

			tokens := findAllTokens(tt.input)
			result, err := td.resolveStringInterpolation("test_intent", tt.input, tokens)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
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
