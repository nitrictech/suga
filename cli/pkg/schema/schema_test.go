package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplicationFromYaml_ValidBasic(t *testing.T) {
	yaml := `
name: test-app
description: A test application
target: team/platform@1
services:
  api:
    container:
      docker:
        dockerfile: Dockerfile
        context: .
buckets:
  storage:
    access:
      api:
        - read
        - write
`

	app, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.True(t, result.Valid(), "Expected valid result, got validation errors: %v", result.Errors())
	assert.Equal(t, "test-app", app.Name)
	assert.Equal(t, "team/platform@1", app.Target)
	assert.Len(t, app.ServiceIntents, 1)
	assert.Len(t, app.BucketIntents, 1)
}

func TestApplicationFromYaml_MissingRequiredFields(t *testing.T) {
	yaml := `
description: A test application without required fields
services:
  api:
    container:
      docker:
        dockerfile: Dockerfile
`

	_, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.False(t, result.Valid(), "Expected invalid result due to missing required fields")

	validationErrs := GetSchemaValidationErrors(result.Errors())
	assert.Len(t, validationErrs, 2)

	errString := FormatValidationErrors(validationErrs)
	assert.Contains(t, errString, "name:    # <-- The name property is required")
	assert.Contains(t, errString, "target:    # <-- The target property is required")
}

func TestApplicationFromYaml_InvalidTargetFormat(t *testing.T) {
	yaml := `
name: test-app
description: A test application
target: invalid-target-format
services:
  api:
    container:
      docker:
        dockerfile: Dockerfile
`

	_, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.False(t, result.Valid(), "Expected invalid result due to invalid target format")

	validationErrs := GetSchemaValidationErrors(result.Errors())
	assert.Len(t, validationErrs, 1)

	errString := FormatValidationErrors(validationErrs)
	assert.Contains(t, errString, "Must be in the format: `<team>/<platform>@<revision>` or `file:<path>`")
}

func TestApplicationFromYaml_ValidHyphenatedTargets(t *testing.T) {
	yaml := `
name: test-app
description: A test application with hyphenated target
target: suga/aws-fargate@1
services:
  api:
    container:
      docker:
        dockerfile: Dockerfile
`

	app, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.True(t, result.Valid(), "Expected valid result, got validation errors: %v", result.Errors())
	assert.Equal(t, "test-app", app.Name)
	assert.Equal(t, "suga/aws-fargate@1", app.Target)
}

func TestApplicationFromYaml_InvalidHyphenatedTargets(t *testing.T) {
	yaml := `
name: test-app
description: A test application with invalid hyphenated target
target: -team/platform@1
services:
  api:
    container:
      docker:
        dockerfile: Dockerfile
`

	_, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.False(t, result.Valid(), "Expected invalid result due to invalid hyphenated target format")

	validationErrs := GetSchemaValidationErrors(result.Errors())
	assert.Len(t, validationErrs, 1)

	errString := FormatValidationErrors(validationErrs)
	assert.Contains(t, errString, "Must be in the format: `<team>/<platform>@<revision>` or `file:<path>`")
}

func TestApplicationFromYaml_ServiceWithImage(t *testing.T) {
	yaml := `
name: test-app
description: test
target: team/platform@1
services:
  api:
    container:
      image:
        id: nginx:latest
`

	app, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.True(t, result.Valid(), "Expected valid result, got validation errors: %v", result.Errors())
	assert.Len(t, app.ServiceIntents, 1)

	service, exists := app.ServiceIntents["api"]
	assert.True(t, exists, "Expected service 'api' to exist")
	assert.NotNil(t, service.Container.Image, "Expected service to have image configuration")
	assert.Equal(t, "nginx:latest", service.Container.Image.ID)
}

func TestApplicationFromYaml_ServiceWithTriggers(t *testing.T) {
	yaml := `
name: test-app
description: test
target: team/platform@1
services:
  worker:
    container:
      docker:
        dockerfile: Dockerfile
    triggers:
      scheduled:
        cron: "0 0 * * *"
        path: /scheduled
`

	app, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.True(t, result.Valid(), "Expected valid result, got validation errors: %v", result.Errors())

	service, exists := app.ServiceIntents["worker"]
	assert.True(t, exists, "Expected service 'worker' to exist")
	assert.Len(t, service.Triggers, 1)

	trigger, exists := service.Triggers["scheduled"]
	assert.True(t, exists, "Expected trigger 'scheduled' to exist")
	assert.Equal(t, "0 0 * * *", trigger.Cron)
	assert.Equal(t, "/scheduled", trigger.Path)
}

func TestApplicationFromYaml_ServiceMissingContainerType(t *testing.T) {
	yaml := `
name: test-app
description: test
target: team/platform@1
services:
  api:
    container: {}
`

	_, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.False(t, result.Valid(), "Expected invalid result due to missing container type")

	validationErrs := GetSchemaValidationErrors(result.Errors())
	assert.Len(t, validationErrs, 2)

	errString := FormatValidationErrors(validationErrs)
	assert.Contains(t, errString, "container:    # <-- Must provide exactly one of: docker OR image")
	assert.Contains(t, errString, "docker:    # <-- The docker property is required")
}

func TestApplicationFromYaml_EntrypointMissingTrailingSlash(t *testing.T) {
	yaml := `
name: test-app
description: test
target: team/platform@1
entrypoints:
  api:
    routes:
      /api:
        name: api
`

	_, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.False(t, result.Valid(), "Expected invalid result due to missing trailing slash")

	validationErrs := GetSchemaValidationErrors(result.Errors())
	assert.Len(t, validationErrs, 1)

	errString := FormatValidationErrors(validationErrs)
	assert.Contains(t, errString, "routes:    # <-- Missing trailing slash for route /api")
}

func TestApplicationFromYaml_EntrypointValidTrailingSlash(t *testing.T) {
	yaml := `
name: test-app
description: test
target: team/platform@1
entrypoints:
  api:
    routes:
      /api/:
        name: api
`

	app, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)
	assert.True(t, result.Valid(), "Expected valid result, got validation errors: %v", result.Errors())

	entrypoint, exists := app.EntrypointIntents["api"]
	assert.True(t, exists, "Expected entrypoint 'api' to exist")
	assert.Len(t, entrypoint.Routes, 1)

	route, exists := entrypoint.Routes["/api/"]
	assert.True(t, exists, "Expected route '/api/' to exist")
	assert.Equal(t, "api", route.TargetName)
}

func TestApplicationFromYaml_InvalidYaml(t *testing.T) {
	yaml := `
name: test-app
description: test
target: team/platform@1
services:
  api:
    container:
      docker:
        dockerfile: Dockerfile
    invalid: [key: value
`

	app, result, err := ApplicationFromYaml(yaml)
	assert.Error(t, err, "Expected error for invalid YAML")
	assert.Nil(t, app, "Expected nil app for invalid YAML")
	assert.Nil(t, result, "Expected nil result for invalid YAML")
}

func TestApplication_IsValid_NoNameConflicts(t *testing.T) {
	app := &Application{
		Name:   "test-app",
		Target: "team/platform@1",
		ServiceIntents: map[string]*ServiceIntent{
			"api": {
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
		},
		BucketIntents: map[string]*BucketIntent{
			"storage": {},
		},
	}

	violations := app.IsValid()
	assert.Len(t, violations, 0, "Expected no violations, got: %v", violations)
}

func TestApplication_IsValid_NameConflicts(t *testing.T) {
	app := &Application{
		Name:   "test-app",
		Target: "team/platform@1",
		ServiceIntents: map[string]*ServiceIntent{
			"api": {
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
		},
		BucketIntents: map[string]*BucketIntent{
			"api": {}, // Same name as service
		},
		DatabaseIntents: map[string]*DatabaseIntent{
			"api": {}, // Same name as service
		},
		EntrypointIntents: map[string]*EntrypointIntent{
			"api": {},
		},
	}

	violations := app.IsValid()
	assert.NotEmpty(t, violations, "Expected violations for name conflicts")

	errString := FormatValidationErrors(GetSchemaValidationErrors(violations))
	assert.Contains(t, errString, "api:    # <-- bucket name api is already in use by a service")
	assert.Contains(t, errString, "api:    # <-- database name api is already in use by a service")
	assert.Contains(t, errString, "api:    # <-- entrypoint name api is already in use by a service")
}

func TestApplication_IsValid_ReservedNames(t *testing.T) {
	app := &Application{
		Name:   "test-app",
		Target: "team/platform@1",
		ServiceIntents: map[string]*ServiceIntent{
			"backend": { // Reserved name
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
		},
		DatabaseIntents: map[string]*DatabaseIntent{
			"backend": {}, // Same name as service
		},
		EntrypointIntents: map[string]*EntrypointIntent{
			"backend": {},
		},
		BucketIntents: map[string]*BucketIntent{
			"backend": {},
		},
	}

	violations := app.IsValid()
	assert.NotEmpty(t, violations, "Expected violations for reserved names")

	errString := FormatValidationErrors(GetSchemaValidationErrors(violations))
	assert.Contains(t, errString, "backend:    # <-- service name backend is a reserved name")
	assert.Contains(t, errString, "backend:    # <-- database name backend is a reserved name")
	assert.Contains(t, errString, "backend:    # <-- entrypoint name backend is a reserved name")
	assert.Contains(t, errString, "backend:    # <-- bucket name backend is a reserved name")
}

func TestApplication_IsValid_ValidSnakeCaseNames(t *testing.T) {
	app := &Application{
		Name:   "test-app",
		Target: "team/platform@1",
		ServiceIntents: map[string]*ServiceIntent{
			"user_api": {
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
			"data_processor": {
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
			"_private_service": {
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
			"service123": {
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
		},
		BucketIntents: map[string]*BucketIntent{
			"file_storage":  {},
			"user_uploads":  {},
			"temp_data_123": {},
		},
		EntrypointIntents: map[string]*EntrypointIntent{
			"main_api":        {},
			"webhook_handler": {},
		},
		DatabaseIntents: map[string]*DatabaseIntent{
			"user_db": {
				EnvVarKey: "TEST_1",
			},
			"session_store": {
				EnvVarKey: "TEST_2",
			},
		},
	}

	violations := app.IsValid()
	assert.Len(t, violations, 0, "Expected no violations for valid snake_case names, got: %v", violations)
}

func TestApplication_IsValid_InvalidSnakeCaseNames(t *testing.T) {
	app := &Application{
		Name:   "test-app",
		Target: "team/platform@1",
		ServiceIntents: map[string]*ServiceIntent{
			"user-api": { // kebab-case
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
			"UserAPI": { // PascalCase
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
			"userAPI": { // camelCase
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
			"123service": { // starts with number
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
			"service!": { // contains special character
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
		},
		BucketIntents: map[string]*BucketIntent{
			"file-storage": {}, // kebab-case
			"FileStorage":  {}, // PascalCase
		},
		EntrypointIntents: map[string]*EntrypointIntent{
			"main-api": {}, // kebab-case
			"MainAPI":  {}, // PascalCase
		},
		DatabaseIntents: map[string]*DatabaseIntent{
			"user-db": {}, // kebab-case
			"UserDB":  {}, // PascalCase
		},
	}

	violations := app.IsValid()
	assert.NotEmpty(t, violations, "Expected violations for invalid snake_case names")

	errString := FormatValidationErrors(GetSchemaValidationErrors(violations))

	// Check service violations
	assert.Contains(t, errString, "user-api:    # <-- service name user-api must be in snake_case format")
	assert.Contains(t, errString, "UserAPI:    # <-- service name UserAPI must be in snake_case format")
	assert.Contains(t, errString, "userAPI:    # <-- service name userAPI must be in snake_case format")
	assert.Contains(t, errString, "123service:    # <-- service name 123service must be in snake_case format")
	assert.Contains(t, errString, "service!:    # <-- service name service! must be in snake_case format")

	// Check bucket violations
	assert.Contains(t, errString, "file-storage:    # <-- bucket name file-storage must be in snake_case format")
	assert.Contains(t, errString, "FileStorage:    # <-- bucket name FileStorage must be in snake_case format")

	// Check entrypoint violations
	assert.Contains(t, errString, "main-api:    # <-- entrypoint name main-api must be in snake_case format")
	assert.Contains(t, errString, "MainAPI:    # <-- entrypoint name MainAPI must be in snake_case format")

	// Check database violations
	assert.Contains(t, errString, "user-db:    # <-- database name user-db must be in snake_case format")
	assert.Contains(t, errString, "UserDB:    # <-- database name UserDB must be in snake_case format")
}

func TestApplicationFromYaml_InvalidResourceNames(t *testing.T) {
	yaml := `
name: test-app
description: A test application with invalid resource names
target: team/platform@1
services:
  user-api:
    container:
      docker:
        dockerfile: Dockerfile
  UserService:
    container:
      docker:
        dockerfile: Dockerfile
buckets:
  file-storage:
    access:
      user-api:
        - read
        - write
entrypoints:
  main-api:
    routes:
      /api/:
        name: user-api
databases:
  user-db: {}
websites:
  public-site: {}
`

	app, result, err := ApplicationFromYaml(yaml)
	assert.NoError(t, err)

	// First check JSON schema validation
	if !result.Valid() {
		schemaErrors := GetSchemaValidationErrors(result.Errors())
		t.Logf("Schema validation errors: %s", FormatValidationErrors(schemaErrors))
	}

	// Then check custom application validation
	violations := app.IsValid()
	assert.NotEmpty(t, violations, "Expected violations for invalid snake_case names")

	errString := FormatValidationErrors(GetSchemaValidationErrors(violations))
	assert.Contains(t, errString, "user-api:    # <-- service name user-api must be in snake_case format")
	assert.Contains(t, errString, "UserService:    # <-- service name UserService must be in snake_case format")
	assert.Contains(t, errString, "file-storage:    # <-- bucket name file-storage must be in snake_case format")
	assert.Contains(t, errString, "main-api:    # <-- entrypoint name main-api must be in snake_case format")
	assert.Contains(t, errString, "user-db:    # <-- database name user-db must be in snake_case format")
}

func TestApplication_IsValid_SubtypesOptional(t *testing.T) {
	app := &Application{
		Name:   "test-app",
		Target: "team/platform@1",
		ServiceIntents: map[string]*ServiceIntent{
			"api": {
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
		},
		BucketIntents: map[string]*BucketIntent{
			"storage": {},
		},
		EntrypointIntents: map[string]*EntrypointIntent{
			"main": {
				Routes: map[string]Route{
					"/": {TargetName: "api"},
				},
			},
		},
		DatabaseIntents: map[string]*DatabaseIntent{
			"users": {EnvVarKey: "DATABASE_URL"},
		},
	}

	// Without WithRequireSubtypes, validation should pass
	violations := app.IsValid()
	assert.Len(t, violations, 0, "Expected no violations without RequireSubtypes option, got: %v", violations)

	// With WithRequireSubtypes, validation should fail
	violations = app.IsValid(WithRequireSubtypes())
	assert.NotEmpty(t, violations, "Expected violations with RequireSubtypes option")

	errString := FormatValidationErrors(GetSchemaValidationErrors(violations))
	assert.Contains(t, errString, "api:    # <-- service must have a subtype specified for build")
	assert.Contains(t, errString, "storage:    # <-- bucket must have a subtype specified for build")
	assert.Contains(t, errString, "main:    # <-- entrypoint must have a subtype specified for build")
	assert.Contains(t, errString, "users:    # <-- database must have a subtype specified for build")
}

func TestApplication_IsValid_WithSubtypes(t *testing.T) {
	app := &Application{
		Name:   "test-app",
		Target: "team/platform@1",
		ServiceIntents: map[string]*ServiceIntent{
			"api": {
				Resource: Resource{SubType: "fargate"},
				Container: Container{
					Docker: &Docker{Dockerfile: "Dockerfile"},
				},
			},
		},
		BucketIntents: map[string]*BucketIntent{
			"storage": {
				Resource: Resource{SubType: "s3"},
			},
		},
		EntrypointIntents: map[string]*EntrypointIntent{
			"main": {
				Resource: Resource{SubType: "alb"},
				Routes: map[string]Route{
					"/": {TargetName: "api"},
				},
			},
		},
		DatabaseIntents: map[string]*DatabaseIntent{
			"users": {
				Resource:  Resource{SubType: "postgres"},
				EnvVarKey: "DATABASE_URL",
			},
		},
	}

	// With subtypes specified, validation should pass with or without RequireSubtypes
	violations := app.IsValid()
	assert.Len(t, violations, 0, "Expected no violations without RequireSubtypes option, got: %v", violations)

	violations = app.IsValid(WithRequireSubtypes())
	assert.Len(t, violations, 0, "Expected no violations with RequireSubtypes option, got: %v", violations)
}
