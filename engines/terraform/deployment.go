package terraform

import (
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/aws/jsii-runtime-go"
	random "github.com/cdktf/cdktf-provider-random-go/random/v11/provider"
	"github.com/cdktf/cdktf-provider-random-go/random/v11/stringresource"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
	app_spec_schema "github.com/nitrictech/suga/cli/pkg/schema"
)

type TerraformDeployment struct {
	app     cdktf.App
	stack   cdktf.TerraformStack
	stackId stringresource.StringResource
	engine  *TerraformEngine

	serviceIdentities map[string]map[string]interface{}

	terraformResources          map[string]cdktf.TerraformHclModule
	terraformInfraResources     map[string]cdktf.TerraformHclModule
	terraformIdentityResources  map[string]cdktf.TerraformHclModule
	identityBlueprints          map[string]*ResourceBlueprint
	terraformVariables          map[string]cdktf.TerraformVariable
	instancedTerraformVariables map[string]map[string]cdktf.TerraformVariable
}

func NewTerraformDeployment(engine *TerraformEngine, stackName string) *TerraformDeployment {
	app := cdktf.NewApp(&cdktf.AppConfig{
		Outdir: jsii.String(engine.outputDir),
	})
	stack := cdktf.NewTerraformStack(app, jsii.String(stackName))

	NewNilTerraformBackend(stack, jsii.String("nil_backend"))

	random.NewRandomProvider(stack, jsii.String("random"), &random.RandomProviderConfig{})

	stackId := stringresource.NewStringResource(stack, jsii.String("stack_id"), &stringresource.StringResourceConfig{
		Length:  jsii.Number(8),
		Upper:   jsii.Bool(false),
		Lower:   jsii.Bool(true),
		Numeric: jsii.Bool(false),
		Special: jsii.Bool(false),
	})

	return &TerraformDeployment{
		app:                         app,
		stack:                       stack,
		stackId:                     stackId,
		engine:                      engine,
		terraformResources:          map[string]cdktf.TerraformHclModule{},
		terraformInfraResources:     map[string]cdktf.TerraformHclModule{},
		terraformIdentityResources:  map[string]cdktf.TerraformHclModule{},
		identityBlueprints:          map[string]*ResourceBlueprint{},
		terraformVariables:          map[string]cdktf.TerraformVariable{},
		instancedTerraformVariables: map[string]map[string]cdktf.TerraformVariable{},
		serviceIdentities:           map[string]map[string]interface{}{},
	}
}

func (td *TerraformDeployment) resolveInfraResource(infraName string) (cdktf.TerraformHclModule, error) {
	resource, ok := td.engine.platform.InfraSpecs[infraName]
	if !ok {
		availableResources := slices.Collect(maps.Keys(td.engine.platform.InfraSpecs))
		return nil, fmt.Errorf("referenced infra resource '%s' is not defined in the platform. Available infra resources are: %v", infraName, availableResources)
	}

	if _, ok := td.terraformInfraResources[infraName]; !ok {
		pluginRef, err := td.engine.resolvePlugin(resource)
		if err != nil {
			return nil, fmt.Errorf("could not resolve plugin for infra resource %s: %w", infraName, err)
		}

		td.createVariablesForIntent(infraName, resource)

		td.terraformInfraResources[infraName] = cdktf.NewTerraformHclModule(td.stack, jsii.String(infraName), &cdktf.TerraformHclModuleConfig{
			Source: jsii.String(pluginRef.Deployment.Terraform),
		})
	}

	return td.terraformInfraResources[infraName], nil
}

func (td *TerraformDeployment) resolveEntrypointSugaVar(name string, appSpec *app_spec_schema.Application, spec *app_spec_schema.EntrypointIntent) (interface{}, error) {
	origins := map[string]interface{}{}
	for path, route := range spec.Routes {
		intentTarget, ok := appSpec.GetResourceIntent(route.TargetName)
		if !ok {
			return nil, fmt.Errorf("entrypoint '%s' has route target with name '%s', but no resources found with that name", name, route.TargetName)
		}

		var intentTargetType string
		switch intentTarget.(type) {
		case *app_spec_schema.ServiceIntent:
			intentTargetType = "service"
		case *app_spec_schema.BucketIntent:
			intentTargetType = "bucket"
		default:
			return nil, fmt.Errorf("entrypoint '%s' has target '%s', which is not a service or bucket", name, route.TargetName)
		}

		hclTarget, ok := td.terraformResources[route.TargetName]
		if !ok {
			return nil, fmt.Errorf("target %s not found", route.TargetName)
		}

		domainNameSugaVar := hclTarget.Get(jsii.String("suga.domain_name"))
		idSugaVar := hclTarget.Get(jsii.String("suga.id"))
		resourcesSugaVar := hclTarget.Get(jsii.String("suga.exports.resources"))

		if origin, exists := origins[route.TargetName]; exists {
			origin, ok := origin.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("failed to get origin for target %s, value is not a map", route.TargetName)
			}

			existingRoutes := origin["routes"].([]map[string]interface{})

			newRoute := map[string]interface{}{
				"path":      jsii.String(path),
				"base_path": jsii.String(route.BasePath),
			}
			newRoutes := append(existingRoutes, newRoute)

			origin["routes"] = newRoutes
		} else {
			origins[route.TargetName] = map[string]interface{}{
				"routes": []map[string]interface{}{
					{
						"path":      jsii.String(path),
						"base_path": jsii.String(route.BasePath),
					},
				},
				"type":        jsii.String(intentTargetType),
				"id":          idSugaVar,
				"domain_name": domainNameSugaVar,
				"resources":   resourcesSugaVar,
			}
		}
	}

	sugaVar := map[string]interface{}{
		"name":     jsii.String(name),
		"stack_id": td.stackId.Result(),
		"origins":  origins,
	}

	return sugaVar, nil
}

func (td *TerraformDeployment) resolveService(name string, spec *app_spec_schema.ServiceIntent, resourceSpec *ServiceBlueprint, plug *ResourcePluginManifest) (*SugaServiceVariables, error) {
	if resourceSpec == nil {
		return nil, fmt.Errorf("resourceSpec is nil for service %s - this indicates a platform configuration issue", name)
	}

	var imageVars *map[string]interface{} = nil

	pluginManifest, err := td.engine.resolvePluginsForService(plug)
	if err != nil {
		return nil, err
	}

	pluginManifestBytes, err := json.Marshal(pluginManifest)
	if err != nil {
		return nil, err
	}

	var schedules map[string]SugaServiceSchedule = nil
	if len(schedules) > 0 && !slices.Contains(plug.Capabilities, "schedules") {
		return nil, fmt.Errorf("service %s has schedules but the plugin %s does not support schedules", name, plug.Name)
	} else {
		schedules = map[string]SugaServiceSchedule{}
	}

	for triggerName, trigger := range spec.Triggers {
		cronExpression := strings.TrimSpace(trigger.Cron)

		if cronExpression == "" {
			continue
		}

		fmt.Printf("⚠️  This project defines a schedule for service '%s'. Schedule triggers are in preview and may change in the future.\n", name)

		schedules[triggerName] = SugaServiceSchedule{
			CronExpression: jsii.String(cronExpression),
			Path:           jsii.String(trigger.Path),
		}
	}

	if spec.Container.Image != nil {
		imageVars = &map[string]interface{}{
			"image_id": jsii.String(spec.Container.Image.ID),
			"tag":      jsii.String(name),
			"args":     map[string]*string{"PLUGIN_DEFINITION": jsii.String(string(pluginManifestBytes))},
		}
	} else if spec.Container.Docker != nil {
		args := map[string]*string{"PLUGIN_DEFINITION": jsii.String(string(pluginManifestBytes))}
		for k, v := range spec.Container.Docker.Args {
			args[k] = jsii.String(v)
		}

		imageVars = &map[string]interface{}{
			"build_context": jsii.String(path.Join(spec.WorkingDir, spec.Container.Docker.Context)),
			"dockerfile":    jsii.String(spec.Container.Docker.Dockerfile),
			"tag":           jsii.String(name),
			"args":          args,
		}
	}

	imageModule := cdktf.NewTerraformHclModule(td.stack, jsii.Sprintf("%s_image", name), &cdktf.TerraformHclModuleConfig{
		Source:    jsii.String(imageModuleRef),
		Variables: imageVars,
	})

	identityModuleOutputs := map[string]interface{}{}

	// Check if IdentitiesBlueprint is nil before accessing Identities
	if resourceSpec.IdentitiesBlueprint != nil {
		for _, id := range resourceSpec.Identities {
			identityPlugin, err := td.engine.resolveIdentityPlugin(&id)
			if err != nil {
				return nil, err
			}

			identityModuleName := fmt.Sprintf("%s_%s_role", name, identityPlugin.Name)

			// Create variables for the identity blueprint
			td.createVariablesForIntent(identityModuleName, &id)

			idModule := cdktf.NewTerraformHclModule(td.stack, jsii.String(identityModuleName), &cdktf.TerraformHclModuleConfig{
				Source:    jsii.String(identityPlugin.Deployment.Terraform),
				Variables: &map[string]interface{}{},
			})

			idModule.Set(jsii.String("suga"), map[string]interface{}{
				"name":     jsii.String(name),
				"stack_id": td.stackId.Result(),
			})

			// Store the identity module and blueprint for later token resolution
			td.terraformIdentityResources[identityModuleName] = idModule
			td.identityBlueprints[identityModuleName] = &id

			identityModuleOutputs[identityPlugin.IdentityType] = idModule.Get(jsii.String("suga"))
		}
	}

	for _, requiredIdentity := range plug.RequiredIdentities {
		providedIdentities := slices.Collect(maps.Keys(identityModuleOutputs))
		if ok := slices.Contains(providedIdentities, requiredIdentity); !ok {
			if len(providedIdentities) == 0 {
				return nil, fmt.Errorf("platform blueprint for service %s is missing identity definitions - plugin %s requires identity %s but the platform provides no identities (this is a platform configuration issue)", name, plug.Name, requiredIdentity)
			} else {
				return nil, fmt.Errorf("service %s is missing identity %s, required by plugin %s, provided identities were %s", name, requiredIdentity, plug.Name, providedIdentities)
			}
		}
	}

	sugaVar := &SugaServiceVariables{
		SugaVariables: SugaVariables{
			Name: jsii.String(name),
		},
		Schedules:  &schedules,
		ImageId:    imageModule.GetString(jsii.String("image_id")),
		Identities: &identityModuleOutputs,
		StackId:    td.stackId.Result(),
		Env:        &spec.Env,
	}

	td.serviceIdentities[name] = identityModuleOutputs

	return sugaVar, nil
}

func (td *TerraformDeployment) collectResourceSubtypes(appSpec *app_spec_schema.Application) map[string]map[string]string {
	resourceSubtypes := map[string]map[string]string{}

	// Collect bucket subtypes
	if len(appSpec.BucketIntents) > 0 {
		bucketSubtypes := map[string]string{}
		for intentName, bucketIntent := range appSpec.BucketIntents {
			subtype := bucketIntent.GetSubType()
			if subtype == "" {
				subtype = "default"
			}
			bucketSubtypes[intentName] = subtype
		}
		resourceSubtypes["bucket"] = bucketSubtypes
	}

	// Collect database subtypes
	if len(appSpec.DatabaseIntents) > 0 {
		databaseSubtypes := map[string]string{}
		for intentName, databaseIntent := range appSpec.DatabaseIntents {
			subtype := databaseIntent.GetSubType()
			if subtype == "" {
				subtype = "default"
			}
			databaseSubtypes[intentName] = subtype
		}
		resourceSubtypes["database"] = databaseSubtypes
	}

	// Collect entrypoint subtypes
	if len(appSpec.EntrypointIntents) > 0 {
		entrypointSubtypes := map[string]string{}
		for intentName, entrypointIntent := range appSpec.EntrypointIntents {
			subtype := entrypointIntent.GetSubType()
			if subtype == "" {
				subtype = "default"
			}
			entrypointSubtypes[intentName] = subtype
		}
		resourceSubtypes["entrypoint"] = entrypointSubtypes
	}

	return resourceSubtypes
}

func (td *TerraformDeployment) Synth() {
	td.app.Synth()
}
