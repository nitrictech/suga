package terraform

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

// isUnset checks if a value should be considered unset/empty
func isUnset(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface:
		if rv.IsNil() {
			return true
		}
		// Treat *string("") as empty
		if rv.Kind() == reflect.Ptr && rv.Elem().IsValid() && rv.Elem().Kind() == reflect.String && rv.Elem().Len() == 0 {
			return true
		}
	case reflect.String:
		return rv.Len() == 0
	}
	return false
}

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
		specRef, err := SpecReferenceFromToken(v)
		if err != nil {
			return v, nil
		}

		if specRef.Source == "infra" {
			refName := specRef.Path[0]
			propertyName := specRef.Path[1]

			infraResource, err := td.resolveInfraResource(refName)
			if err != nil {
				return nil, err
			}
			return infraResource.Get(jsii.String(propertyName)), nil
		} else if specRef.Source == "self" {
			tfVariable, ok := td.instancedTerraformVariables[intentName][specRef.Path[0]]
			if !ok {
				return nil, fmt.Errorf("Variable %s does not exist for provided blueprint", specRef.Path[0])
			}
			return tfVariable.Value(), nil
		} else if specRef.Source == "var" {
			tfVariable, ok := td.getPlatformVariable(specRef.Path[0])
			if !ok {
				return nil, fmt.Errorf("Variable %s does not exist for this platform", specRef.Path[0])
			}
			return tfVariable.Value(), nil
		}
		return v, nil

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

func (td *TerraformDeployment) resolveTokensForModule(intentName string, resource *ResourceBlueprint, module cdktf.TerraformHclModule) error {
	for property, value := range resource.Properties {
		resolvedValue, err := td.resolveValue(intentName, value)
		if err != nil {
			return fmt.Errorf("failed to resolve property '%s' for intent '%s': %w", property, intentName, err)
		}
		// Skip nil values and empty strings
		if isUnset(resolvedValue) {
			continue
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

		if _, ok := td.terraformInfraResources[specRef.Path[0]]; !ok {
			continue
		}

		moduleId := fmt.Sprintf("module.%s", *td.terraformInfraResources[specRef.Path[0]].Node().Id())
		dependsOnResources = append(dependsOnResources, jsii.String(moduleId))
	}
	module.SetDependsOn(&dependsOnResources)
	return nil
}
