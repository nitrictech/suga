package schema

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// Perform additional validation checks on the application
func (a *Application) IsValid() []gojsonschema.ResultError {
	// Check the names of all resources are unique
	violations := a.checkNoNameConflicts()
	violations = append(violations, a.checkNoReservedNames()...)
	violations = append(violations, a.checkSnakeCaseNames()...)
	violations = append(violations, a.checkNoEnvVarCollisions()...)
	violations = append(violations, a.checkAccessPermissions()...)
	violations = append(violations, a.checkServiceGenerateConfigurations()...)

	return violations
}

func (a *Application) checkAccessPermissions() []gojsonschema.ResultError {
	violations := []gojsonschema.ResultError{}

	for name, intent := range a.BucketIntents {
		for serviceName, actions := range intent.Access {
			invalidActions, ok := ValidateActions(actions, Bucket)
			if !ok {
				key := fmt.Sprintf("buckets.%s.access.%s", name, serviceName)
				err := fmt.Sprintf("Invalid bucket %s: %s. Valid actions are: %s", pluralise("action", len(invalidActions)), strings.Join(invalidActions, ", "), strings.Join(GetValidActions(Bucket), ", "))
				violations = append(violations, newValidationError(key, err))
			}
		}
	}

	for name, intent := range a.DatabaseIntents {
		for serviceName, actions := range intent.Access {
			invalidActions, ok := ValidateActions(actions, Database)
			if !ok {
				key := fmt.Sprintf("databases.%s.access.%s", name, serviceName)
				err := fmt.Sprintf("Invalid database %s: %s. Valid actions are: %s", pluralise("action", len(invalidActions)), strings.Join(invalidActions, ", "), strings.Join(GetValidActions(Database), ", "))
				violations = append(violations, newValidationError(key, err))
			}
		}
	}

	return violations
}

func pluralise(word string, count int) string {
	output := word
	if count > 1 {
		output += "s"
	}
	return output
}

func (a *Application) checkNoNameConflicts() []gojsonschema.ResultError {
	resourceNames := map[string]string{}
	violations := []gojsonschema.ResultError{}

	for name := range a.ServiceIntents {
		if existingType, ok := resourceNames[name]; ok {
			violations = append(violations, newValidationError(fmt.Sprintf("services.%s", name), fmt.Sprintf("service name %s is already in use by a %s", name, existingType)))
			continue
		}

		resourceNames[name] = "service"
	}

	for name := range a.BucketIntents {
		if existingType, ok := resourceNames[name]; ok {
			violations = append(violations, newValidationError(fmt.Sprintf("buckets.%s", name), fmt.Sprintf("bucket name %s is already in use by a %s", name, existingType)))
			continue
		}
		resourceNames[name] = "bucket"
	}

	for name := range a.EntrypointIntents {
		if existingType, ok := resourceNames[name]; ok {
			violations = append(violations, newValidationError(fmt.Sprintf("entrypoints.%s", name), fmt.Sprintf("entrypoint name %s is already in use by a %s", name, existingType)))
			continue
		}
		resourceNames[name] = "entrypoint"
	}

	for name := range a.DatabaseIntents {
		if existingType, ok := resourceNames[name]; ok {
			violations = append(violations, newValidationError(fmt.Sprintf("databases.%s", name), fmt.Sprintf("database name %s is already in use by a %s", name, existingType)))
			continue
		}
		resourceNames[name] = "database"
	}

	return violations
}

func (a *Application) checkNoReservedNames() []gojsonschema.ResultError {
	violations := []gojsonschema.ResultError{}
	reservedNames := []string{
		"backend", // Backend is a reserved keyword in terraform
	}

	for name := range a.ServiceIntents {
		if slices.Contains(reservedNames, name) {
			violations = append(violations, newValidationError(fmt.Sprintf("services.%s", name), fmt.Sprintf("service name %s is a reserved name", name)))
		}
	}

	for name := range a.BucketIntents {
		if slices.Contains(reservedNames, name) {
			violations = append(violations, newValidationError(fmt.Sprintf("buckets.%s", name), fmt.Sprintf("bucket name %s is a reserved name", name)))
		}
	}

	for name := range a.EntrypointIntents {
		if slices.Contains(reservedNames, name) {
			violations = append(violations, newValidationError(fmt.Sprintf("entrypoints.%s", name), fmt.Sprintf("entrypoint name %s is a reserved name", name)))
		}
	}

	for name := range a.DatabaseIntents {
		if slices.Contains(reservedNames, name) {
			violations = append(violations, newValidationError(fmt.Sprintf("databases.%s", name), fmt.Sprintf("database name %s is a reserved name", name)))
		}
	}

	return violations
}

func (a *Application) checkSnakeCaseNames() []gojsonschema.ResultError {
	violations := []gojsonschema.ResultError{}
	snakeCasePattern := regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

	for name := range a.ServiceIntents {
		if !snakeCasePattern.MatchString(name) {
			violations = append(violations, newValidationError(fmt.Sprintf("services.%s", name), fmt.Sprintf("service name %s must be in snake_case format", name)))
		}
	}

	for name := range a.BucketIntents {
		if !snakeCasePattern.MatchString(name) {
			violations = append(violations, newValidationError(fmt.Sprintf("buckets.%s", name), fmt.Sprintf("bucket name %s must be in snake_case format", name)))
		}
	}

	for name := range a.EntrypointIntents {
		if !snakeCasePattern.MatchString(name) {
			violations = append(violations, newValidationError(fmt.Sprintf("entrypoints.%s", name), fmt.Sprintf("entrypoint name %s must be in snake_case format", name)))
		}
	}

	for name := range a.DatabaseIntents {
		if !snakeCasePattern.MatchString(name) {
			violations = append(violations, newValidationError(fmt.Sprintf("databases.%s", name), fmt.Sprintf("database name %s must be in snake_case format", name)))
		}
	}

	return violations
}

func (a *Application) checkNoEnvVarCollisions() []gojsonschema.ResultError {
	violations := []gojsonschema.ResultError{}
	envVarMap := map[string]string{}

	for name, intent := range a.DatabaseIntents {
		if existingName, ok := envVarMap[intent.EnvVarKey]; ok {
			violations = append(violations, newValidationError(fmt.Sprintf("databases.%s", name), fmt.Sprintf("env var %s is already in use by %s", intent.EnvVarKey, existingName)))
			continue
		}
		envVarMap[intent.EnvVarKey] = name
	}

	return violations
}

func (a *Application) checkServiceGenerateConfigurations() []gojsonschema.ResultError {
	violations := []gojsonschema.ResultError{}
	validLanguages := GetSupportedLanguages()
	serviceLanguages := make(map[string]string) // language -> service name

	for name, service := range a.ServiceIntents {
		// If language is specified, client_library_output must also be specified
		if service.Language != "" {
			if service.ClientLibraryOutput == "" {
				violations = append(violations, newValidationError(
					fmt.Sprintf("services.%s.client_library_output", name),
					"client_library_output is required when language is specified",
				))
			}

			// Check if language is valid
			if !slices.Contains(validLanguages, service.Language) {
				violations = append(violations, newValidationError(
					fmt.Sprintf("services.%s.language", name),
					fmt.Sprintf("Invalid language '%s'. Valid languages are: %s", service.Language, strings.Join(validLanguages, ", ")),
				))
			}

			// For Go, validate package name if provided
			if service.Language == LanguageGo && service.PackageName != "" {
				// Go package names should be lowercase and not contain dashes or spaces
				if !regexp.MustCompile(`^[a-z][a-z0-9]*$`).MatchString(service.PackageName) {
					violations = append(violations, newValidationError(
						fmt.Sprintf("services.%s.package_name", name),
						"Go package name must be lowercase and contain only letters and numbers, starting with a letter",
					))
				}
			}

			// Check for duplicate languages across services
			if existingService, exists := serviceLanguages[service.Language]; exists {
				violations = append(violations, newValidationError(
					fmt.Sprintf("services.%s.language", name),
					fmt.Sprintf("Language '%s' is already configured for service '%s'", service.Language, existingService),
				))
			} else {
				serviceLanguages[service.Language] = name
			}
		}

		// If client_library_output is specified without language, that's also an error
		if service.ClientLibraryOutput != "" && service.Language == "" {
			violations = append(violations, newValidationError(
				fmt.Sprintf("services.%s.language", name),
				"language is required when client_library_output is specified",
			))
		}
	}

	return violations
}
