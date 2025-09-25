package terraform

import (
	"fmt"
	"strings"

	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

func SpecReferenceFromToken(token string) (*SpecReference, error) {
	contents, ok := extractTokenContents(token)
	if !ok {
		return nil, fmt.Errorf("invalid token format")
	}

	parts := strings.Split(contents, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	return &SpecReference{
		Source: parts[0],
		Path:   parts[1:],
	}, nil
}

func (td *TerraformDeployment) resolveValue(intentName string, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return td.resolveTokenString(intentName, v)

	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			resolvedVal, err := td.resolveValue(intentName, val)
			if err != nil {
				return nil, err
			}
			result[key] = resolvedVal
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			resolvedVal, err := td.resolveValue(intentName, val)
			if err != nil {
				return nil, err
			}
			result[i] = resolvedVal
		}
		return result, nil

	default:
		return v, nil
	}
}

func (td *TerraformDeployment) resolveToken(intentName string, specRef *SpecReference, returnAsReference bool) (interface{}, error) {
	switch specRef.Source {
	case "infra":
		if len(specRef.Path) < 2 {
			return nil, fmt.Errorf("infra token requires at least 2 path components")
		}

		refName := specRef.Path[0]
		propertyName := specRef.Path[1]

		if returnAsReference {
			return fmt.Sprintf("${module.%s.%s}", refName, propertyName), nil
		}

		infraResource, err := td.resolveInfraResource(refName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve infrastructure resource %s: %w", refName, err)
		}

		return infraResource.Get(jsii.String(propertyName)), nil

	case "self":
		if len(specRef.Path) < 1 {
			return nil, fmt.Errorf("self token requires at least 1 path component")
		}

		varName := specRef.Path[0]

		tfVariable, ok := td.instancedTerraformVariables[intentName][varName]
		if !ok {
			return nil, fmt.Errorf("variable %s does not exist for provided blueprint", varName)
		}

		if returnAsReference {
			return fmt.Sprintf("${var.%s}", varName), nil
		}
		return tfVariable.Value(), nil

	case "var":
		if len(specRef.Path) < 1 {
			return nil, fmt.Errorf("var token requires at least 1 path component")
		}

		varName := specRef.Path[0]

		tfVariable, ok := td.getPlatformVariable(varName)
		if !ok {
			return nil, fmt.Errorf("variable %s does not exist for this platform", varName)
		}

		if returnAsReference {
			return fmt.Sprintf("${var.%s}", varName), nil
		}
		return tfVariable.Value(), nil

	default:
		return nil, fmt.Errorf("unknown token source: %s", specRef.Source)
	}
}

func (td *TerraformDeployment) resolveTokenString(intentName string, input string) (interface{}, error) {
	tokens := findAllTokens(input)

	if len(tokens) == 0 {
		return input, nil
	}

	if len(tokens) == 1 && isOnlyToken(input) {
		specRef, err := SpecReferenceFromToken(tokens[0].Token)
		if err != nil {
			return input, nil
		}

		return td.resolveToken(intentName, specRef, false)
	}

	result := input
	for i := len(tokens) - 1; i >= 0; i-- {
		token := tokens[i]

		specRef, err := SpecReferenceFromToken(token.Token)
		if err != nil {
			continue
		}

		replacementInterface, err := td.resolveToken(intentName, specRef, true)
		if err != nil {
			return nil, err
		}

		replacement, ok := replacementInterface.(string)
		if !ok {
			return nil, fmt.Errorf("expected string replacement for token %s", token.Token)
		}

		result = result[:token.Start] + replacement + result[token.End:]
	}

	return result, nil
}

func (td *TerraformDeployment) resolveTokensForModule(intentName string, resource *ResourceBlueprint, module cdktf.TerraformHclModule) error {
	for property, value := range resource.Properties {
		resolvedValue, err := td.resolveValue(intentName, value)
		if err != nil {
			return fmt.Errorf("failed to resolve property %s for %s: %w", property, intentName, err)
		}
		module.Set(jsii.String(property), resolvedValue)
	}

	return nil
}

func (td *TerraformDeployment) resolveDependencies(resource *ResourceBlueprint, module cdktf.TerraformHclModule) error {
	if len(resource.DependsOn) == 0 {
		return nil
	}

	dependsOnResources := []*string{}
	for _, dependsOn := range resource.DependsOn {
		specRef, err := SpecReferenceFromToken(dependsOn)
		if err != nil {
			return err
		}

		if specRef.Source != "infra" {
			return fmt.Errorf("depends_on can only reference infra resources")
		}

		// Ensure the infra resource is created if it doesn't exist
		infraResource, err := td.resolveInfraResource(specRef.Path[0])
		if err != nil {
			return fmt.Errorf("failed to resolve infrastructure dependency %s: %w", specRef.Path[0], err)
		}

		moduleId := fmt.Sprintf("module.%s", *infraResource.Node().Id())
		dependsOnResources = append(dependsOnResources, jsii.String(moduleId))
	}
	module.SetDependsOn(&dependsOnResources)
	return nil
}
