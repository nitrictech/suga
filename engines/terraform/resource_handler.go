package terraform

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
	app_spec_schema "github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/nitrictech/suga/server/plugin"
)

func (e *TerraformEngine) resolvePluginsForService(servicePlugin *ResourcePluginManifest) (*plugin.PluginDefinition, error) {
	gets := []string{}

	// Check if Runtime is nil to prevent panic
	if servicePlugin.Runtime == nil {
		return nil, fmt.Errorf("service plugin %s has no runtime configuration", servicePlugin.Name)
	}

	pluginDef := &plugin.PluginDefinition{
		Service: plugin.GoPlugin{
			Alias:  "svcPlugin",
			Name:   "default",
			Import: strings.Split(servicePlugin.Runtime.GoModule, "@")[0],
		},
	}
	gets = append(gets, servicePlugin.Runtime.GoModule)

	storagePlugins, err := e.GetPluginManifestsForType("bucket")
	if err != nil {
		return nil, err
	}

	for name, plug := range storagePlugins {
		pluginDef.Storage = append(pluginDef.Storage, plugin.GoPlugin{
			Alias:  fmt.Sprintf("storage_%s", name),
			Name:   name,
			Import: strings.Split(plug.Runtime.GoModule, "@")[0],
		})
		gets = append(gets, plug.Runtime.GoModule)
	}

	pluginDef.Gets = gets

	// Collect all local server proxies from platform libraries
	goproxies := []string{}
	for _, libVersion := range e.platform.Libraries {
		// Check if this is an HTTP URL (local server)
		if strings.HasPrefix(string(libVersion), "http://") || strings.HasPrefix(string(libVersion), "https://") {
			// Replace localhost with host.docker.internal for Docker builds
			proxy := strings.Replace(string(libVersion), "localhost", "host.docker.internal", 1)
			proxy = strings.Replace(proxy, "127.0.0.1", "host.docker.internal", 1)
			if !slices.Contains(goproxies, proxy) {
				goproxies = append(goproxies, proxy)
			}
		}
	}

	// Convert proxy map to slice
	if len(goproxies) > 0 {
		pluginDef.Goproxies = goproxies
	}

	return pluginDef, nil
}

func (td *TerraformDeployment) processServiceIdentities(appSpec *app_spec_schema.Application) (map[string]*SugaServiceVariables, error) {
	serviceInputs := map[string]*SugaServiceVariables{}

	for intentName, serviceIntent := range appSpec.ServiceIntents {
		spec, err := td.engine.platform.GetServiceBlueprint(serviceIntent.GetSubType())
		if err != nil {
			return nil, fmt.Errorf("could not find platform type for %s.%s: %w", serviceIntent.GetType(), serviceIntent.GetSubType(), err)
		}
		if spec == nil {
			return nil, fmt.Errorf("platform returned nil blueprint for service %s with type %s.%s", intentName, serviceIntent.GetType(), serviceIntent.GetSubType())
		}
		plug, err := td.engine.resolvePlugin(spec.ResourceBlueprint)
		if err != nil {
			return nil, fmt.Errorf("could not resolve plugin for service %s: %w", intentName, err)
		}

		sugaVar, err := td.resolveService(intentName, serviceIntent, spec, plug)
		if err != nil {
			return nil, err
		}

		serviceInputs[intentName] = sugaVar
	}

	return serviceInputs, nil
}

func (td *TerraformDeployment) collectServiceAccessors(appSpec *app_spec_schema.Application) map[string]map[string]interface{} {
	serviceAccessors := make(map[string]map[string]interface{})

	for targetServiceName, targetServiceIntent := range appSpec.ServiceIntents {
		if access, ok := targetServiceIntent.GetAccess(); ok {
			accessors := map[string]interface{}{}

			for accessorServiceName, actions := range access {
				expandedActions := app_spec_schema.ExpandActions(actions, app_spec_schema.Service)
				idMap := td.serviceIdentities[accessorServiceName]

				accessors[accessorServiceName] = map[string]interface{}{
					"actions":    jsii.Strings(expandedActions...),
					"identities": idMap,
				}
			}

			if len(accessors) > 0 {
				serviceAccessors[targetServiceName] = accessors
			}
		}
	}

	return serviceAccessors
}

func (td *TerraformDeployment) processServiceResources(appSpec *app_spec_schema.Application, serviceInputs map[string]*SugaServiceVariables, serviceEnvs map[string][]interface{}) error {
	serviceAccessors := td.collectServiceAccessors(appSpec)

	// Track original env values before modification
	originalEnvs := make(map[string]interface{})
	for intentName := range appSpec.ServiceIntents {
		sugaVar := serviceInputs[intentName]
		originalEnvs[intentName] = sugaVar.Env
	}

	for intentName, serviceIntent := range appSpec.ServiceIntents {
		spec, err := td.engine.platform.GetResourceBlueprint(serviceIntent.GetType(), serviceIntent.GetSubType())
		if err != nil {
			return fmt.Errorf("could not find platform type for %s.%s: %w", serviceIntent.GetType(), serviceIntent.GetSubType(), err)
		}
		plug, err := td.engine.resolvePlugin(spec)
		if err != nil {
			return fmt.Errorf("could not resolve plugin for service %s: %w", intentName, err)
		}

		sugaVar := serviceInputs[intentName]

		if accessors, ok := serviceAccessors[intentName]; ok {
			sugaVar.Services = accessors
		}

		td.createVariablesForIntent(intentName, spec)

		td.terraformResources[intentName] = cdktf.NewTerraformHclModule(td.stack, jsii.String(intentName), &cdktf.TerraformHclModuleConfig{
			Source: jsii.String(plug.Deployment.Terraform),
			Variables: &map[string]interface{}{
				"suga": sugaVar,
			},
		})
	}

	// Add service to service urls
	// Build reverse index: for each accessor service, find which targets grant it access
	for accessorServiceName := range appSpec.ServiceIntents {
		for targetServiceName, targetServiceIntent := range appSpec.ServiceIntents {
			if access, ok := targetServiceIntent.GetAccess(); ok {
				if _, hasAccess := access[accessorServiceName]; hasAccess {
					if targetResource, ok := td.terraformResources[targetServiceName]; ok {
						envVarName := fmt.Sprintf("%s_URL", strings.ToUpper(targetServiceName))
						httpEndpoint := targetResource.Get(jsii.String("suga.http_endpoint"))
						serviceEnvs[accessorServiceName] = append(serviceEnvs[accessorServiceName], map[string]interface{}{
							envVarName: httpEndpoint,
						})
					}
				}
			}
		}
	}

	// Merge environment variables for all services
	for intentName := range appSpec.ServiceIntents {
		sugaVar := serviceInputs[intentName]
		mergedEnv := serviceEnvs[intentName]
		allEnv := append(mergedEnv, originalEnvs[intentName])
		sugaVar.Env = cdktf.Fn_Merge(&allEnv)
	}

	return nil
}

func (td *TerraformDeployment) processBucketResources(appSpec *app_spec_schema.Application) (map[string][]interface{}, error) {
	serviceEnvs := map[string][]interface{}{}

	for intentName, bucketIntent := range appSpec.BucketIntents {
		contentPath := ""
		if bucketIntent != nil {
			contentPath = bucketIntent.ContentPath
		}

		spec, err := td.engine.platform.GetResourceBlueprint(bucketIntent.GetType(), bucketIntent.GetSubType())
		if err != nil {
			return nil, fmt.Errorf("could not find platform type for %s.%s: %w", bucketIntent.GetType(), bucketIntent.GetSubType(), err)
		}
		plug, err := td.engine.resolvePlugin(spec)
		if err != nil {
			return nil, fmt.Errorf("could not resolve plugin for bucket %s: %w", intentName, err)
		}

		servicesInput := map[string]any{}
		if access, ok := bucketIntent.GetAccess(); ok {
			for serviceName, actions := range access {
				expandedActions := app_spec_schema.ExpandActions(actions, app_spec_schema.Bucket)

				idMap, ok := td.serviceIdentities[serviceName]
				if !ok {
					return nil, fmt.Errorf("could not give access to bucket %s: service %s not found", intentName, serviceName)
				}

				// Validate that this service subtype is allowed to access the bucket
				serviceIntent, ok := appSpec.ServiceIntents[serviceName]
				if !ok {
					return nil, fmt.Errorf("could not validate access to bucket %s: service %s not found in application spec", intentName, serviceName)
				}
				if err := spec.ValidateServiceAccess(serviceIntent.GetSubType(), intentName, "bucket"); err != nil {
					return nil, err
				}

				servicesInput[serviceName] = map[string]interface{}{
					"actions":    jsii.Strings(expandedActions...),
					"identities": idMap,
				}
			}
		}

		sugaVar := map[string]any{
			"name":         intentName,
			"stack_id":     td.stackId.Result(),
			"content_path": contentPath,
			"services":     servicesInput,
		}

		td.createVariablesForIntent(intentName, spec)

		td.terraformResources[intentName] = cdktf.NewTerraformHclModule(td.stack, jsii.String(intentName), &cdktf.TerraformHclModuleConfig{
			Source: jsii.String(plug.Deployment.Terraform),
			Variables: &map[string]interface{}{
				"suga": sugaVar,
			},
		})

		// Collect environment variables that buckets export to services
		for serviceName := range td.serviceIdentities {
			env := cdktf.Fn_Try(&[]interface{}{td.terraformResources[intentName].Get(jsii.Sprintf("suga.exports.services.%s.env", serviceName)), map[string]interface{}{}})
			serviceEnvs[serviceName] = append(serviceEnvs[serviceName], env)
		}
	}

	return serviceEnvs, nil
}

func (td *TerraformDeployment) processEntrypointResources(appSpec *app_spec_schema.Application) error {
	for intentName, entrypointIntent := range appSpec.EntrypointIntents {
		spec, err := td.engine.platform.GetResourceBlueprint(entrypointIntent.GetType(), entrypointIntent.GetSubType())
		if err != nil {
			return fmt.Errorf("could not find platform type for %s.%s: %w", entrypointIntent.GetType(), entrypointIntent.GetSubType(), err)
		}
		plug, err := td.engine.resolvePlugin(spec)
		if err != nil {
			return fmt.Errorf("could not resolve plugin for entrypoint %s: %w", intentName, err)
		}

		sugaVar, err := td.resolveEntrypointSugaVar(intentName, appSpec, entrypointIntent)
		if err != nil {
			return err
		}

		td.createVariablesForIntent(intentName, spec)

		td.terraformResources[intentName] = cdktf.NewTerraformHclModule(td.stack, jsii.String(intentName), &cdktf.TerraformHclModuleConfig{
			Source: jsii.String(plug.Deployment.Terraform),
			Variables: &map[string]interface{}{
				"suga": sugaVar,
			},
		})
	}

	return nil
}

func (td *TerraformDeployment) processDatabaseResources(appSpec *app_spec_schema.Application) (map[string][]interface{}, error) {
	serviceEnvs := map[string][]interface{}{}

	for intentName, databaseIntent := range appSpec.DatabaseIntents {
		spec, err := td.engine.platform.GetResourceBlueprint(databaseIntent.GetType(), databaseIntent.GetSubType())
		if err != nil {
			return nil, fmt.Errorf("could not find platform type for %s.%s: %w", databaseIntent.GetType(), databaseIntent.GetSubType(), err)
		}
		plug, err := td.engine.resolvePlugin(spec)
		if err != nil {
			return nil, fmt.Errorf("could not resolve plugin for database %s: %w", intentName, err)
		}

		servicesInput := map[string]any{}
		if access, ok := databaseIntent.GetAccess(); ok {
			for serviceName, actions := range access {
				expandedActions := app_spec_schema.ExpandActions(actions, app_spec_schema.Database)

				idMap, ok := td.serviceIdentities[serviceName]
				if !ok {
					return nil, fmt.Errorf("could not give access to database %s: service %s not found", intentName, serviceName)
				}

				// Validate that this service subtype is allowed to access the database
				serviceIntent, ok := appSpec.ServiceIntents[serviceName]
				if !ok {
					return nil, fmt.Errorf("could not validate access to database %s: service %s not found in application spec", intentName, serviceName)
				}
				if err := spec.ValidateServiceAccess(serviceIntent.GetSubType(), intentName, "database"); err != nil {
					return nil, err
				}

				servicesInput[serviceName] = map[string]interface{}{
					"actions":    jsii.Strings(expandedActions...),
					"identities": idMap,
				}
			}
		}

		sugaVar := map[string]any{
			"name":        intentName,
			"stack_id":    td.stackId.Result(),
			"services":    servicesInput,
			"env_var_key": databaseIntent.EnvVarKey,
		}

		td.createVariablesForIntent(intentName, spec)

		td.terraformResources[intentName] = cdktf.NewTerraformHclModule(td.stack, jsii.String(intentName), &cdktf.TerraformHclModuleConfig{
			Source: jsii.String(plug.Deployment.Terraform),
			Variables: &map[string]interface{}{
				"suga": sugaVar,
			},
		})

		// Collect environment variables that databases export to services
		for serviceName := range td.serviceIdentities {
			env := cdktf.Fn_Try(&[]interface{}{td.terraformResources[intentName].Get(jsii.Sprintf("suga.exports.services.%s.env", serviceName)), map[string]interface{}{}})
			serviceEnvs[serviceName] = append(serviceEnvs[serviceName], env)
		}
	}

	return serviceEnvs, nil
}
