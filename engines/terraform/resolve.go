package terraform

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

func SpecReferenceFromToken(token string) (*SpecReference, error) {
	contents, ok := extractTokenContents(token)
	if !ok {
		return nil, fmt.Errorf("invalid reference format for token: %s", token)
	}

	parts := strings.Split(contents, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid reference format for token: %s", token)
	}

	// Validate that all path components are non-empty
	for _, part := range parts[1:] {
		if part == "" {
			return nil, fmt.Errorf("invalid reference format for token: %s", token)
		}
	}

	return &SpecReference{
		Source: parts[0],
		Path:   parts[1:],
	}, nil
}

func (td *TerraformDeployment) resolveValue(intentName string, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return td.resolveTokenValue(intentName, v)

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

func (td *TerraformDeployment) resolveToken(intentName string, specRef *SpecReference) (interface{}, error) {
	switch specRef.Source {
	case "infra":
		if len(specRef.Path) < 2 {
			return nil, fmt.Errorf("infra reference requires at least 2 path components")
		}

		refName := specRef.Path[0]
		attribute := strings.Join(specRef.Path[1:], ".")

		infraResource, err := td.resolveInfraResource(refName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve infrastructure resource %s: %w", refName, err)
		}

		result := infraResource.Get(jsii.String(attribute))
		if result == nil {
			return nil, fmt.Errorf("attribute '%s' not found on infrastructure resource '%s' (type: %s)", attribute, refName, *infraResource.Node().Id())
		}
		return result, nil

	case "self":
		if len(specRef.Path) == 0 {
			return nil, fmt.Errorf("references 'self' without specifying a variable. All references must include a variable name e.g. self.my_var")
		}

		varName := specRef.Path[0]

		availableVars := slices.Collect(maps.Keys(td.instancedTerraformVariables[intentName]))
		tfVariable, ok := td.instancedTerraformVariables[intentName][varName]
		if !ok {
			return nil, fmt.Errorf("variable %s does not exist for provided blueprint available variables are: %v", varName, availableVars)
		}

		if len(specRef.Path) > 1 {
			attribute := strings.Join(specRef.Path[1:], ".")
			return cdktf.Fn_Lookup(tfVariable.Value(), jsii.String(attribute), nil), nil
		}
		return tfVariable.Value(), nil

	case "var":
		if len(specRef.Path) < 1 {
			return nil, fmt.Errorf("var `%s` doesn't contain a valid variable reference", specRef.Source)
		}

		varName := specRef.Path[0]

		tfVariable, ok := td.getPlatformVariable(varName)
		if !ok {
			return nil, fmt.Errorf("variable %s does not exist for this platform", varName)
		}

		if len(specRef.Path) > 1 {
			attribute := strings.Join(specRef.Path[1:], ".")
			return cdktf.Fn_Lookup(tfVariable.Value(), jsii.String(attribute), nil), nil
		}
		return tfVariable.Value(), nil

	default:
		return nil, fmt.Errorf("unknown reference source '%s'", specRef.Source)
	}
}

func (td *TerraformDeployment) resolveTokenValue(intentName string, input string) (interface{}, error) {
	tokens := findAllTokens(input)

	if len(tokens) == 0 {
		return input, nil
	}

	if len(tokens) == 1 && isOnlyToken(input) {
		specRef, err := SpecReferenceFromToken(tokens[0].Token)
		if err != nil {
			return input, nil
		}

		return td.resolveToken(intentName, specRef)
	}

	return td.resolveStringInterpolation(intentName, input, tokens)
}

func (td *TerraformDeployment) resolveStringInterpolation(intentName string, input string, tokens []TokenMatch) (string, error) {
	result := input
	for i := len(tokens) - 1; i >= 0; i-- {
		token := tokens[i]

		specRef, err := SpecReferenceFromToken(token.Token)
		if err != nil {
			continue
		}

		tokenValue, err := td.resolveToken(intentName, specRef)
		if err != nil {
			return "", err
		}

		replacement := cdktf.Token_AsString(tokenValue, nil)
		if replacement == nil {
			return "", fmt.Errorf("cannot use reference '%s' in string interpolation: the resolved value is not string-compatible (likely an object, array, or complex type). Use the reference alone without surrounding text to preserve its type", token.Token)
		}

		result = result[:token.Start] + *replacement + result[token.End:]
	}

	return result, nil
}

func (td *TerraformDeployment) resolveTokensForModule(intentName string, resource *ResourceBlueprint, module cdktf.TerraformHclModule) error {
	for property, value := range resource.Properties {
		resolvedValue, err := td.resolveValue(intentName, value)
		if err != nil {
			return fmt.Errorf("failed to resolve property %s for %s: %w", property, intentName, err)
		}
		if resolvedValue != nil {
			module.Set(jsii.String(property), resolvedValue)
		}
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
