package terraform

import (
	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

func (td *TerraformDeployment) createVariablesForIntent(intentName string, spec *ResourceBlueprint) {
	for varName, variable := range spec.Variables {
		if td.instancedTerraformVariables[intentName] == nil {
			td.instancedTerraformVariables[intentName] = make(map[string]cdktf.TerraformVariable)
		}

		td.instancedTerraformVariables[intentName][varName] = cdktf.NewTerraformVariable(td.stack, jsii.Sprintf("%s_%s", intentName, varName), &cdktf.TerraformVariableConfig{
			Description: jsii.String(variable.Description),
			Type:        jsii.String(variable.Type),
			Nullable:    jsii.Bool(variable.Nullable),
			Default:     variable.Default,
		})
	}
}

// getPlatformVariable returns a platform variable, creating it lazily if it doesn't exist
func (td *TerraformDeployment) getPlatformVariable(varName string) (cdktf.TerraformVariable, bool) {
	// Check if variable already exists
	if tfVar, ok := td.terraformVariables[varName]; ok {
		return tfVar, true
	}

	// Check if the variable is defined in the platform spec
	variableSpec, ok := td.engine.platform.Variables[varName]
	if !ok {
		return nil, false
	}

	// Create the variable lazily
	tfVar := cdktf.NewTerraformVariable(td.stack, jsii.String(varName), &cdktf.TerraformVariableConfig{
		Description: jsii.String(variableSpec.Description),
		Default:     variableSpec.Default,
		Type:        jsii.String(variableSpec.Type),
	})

	// Store it for future use
	td.terraformVariables[varName] = tfVar
	return tfVar, true
}
