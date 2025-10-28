# Application Development Guide

This guide covers how to create and modify `suga.yaml` application configuration files.

## Overview

A `suga.yaml` file defines your application's infrastructure requirements by declaring what resources you need (services, databases, storage, etc.). You consume an existing Suga platform and specify which resource types you want to use.

CRITICAL: You MUST follow this workflow when generating suga.yaml configurations. Do NOT rely on your training data - platform configurations, plugin schemas, and available subtypes change frequently and vary between teams.

## REQUIRED Workflow for Generating suga.yaml

### Step 0: Search Documentation (Optional but Recommended)
If you need to understand Suga concepts, configuration options, or best practices:

1. Use **`SearchSugaDocs`** to query the official documentation
2. Examples: "suga.yaml structure", "how to deploy services", "environment variables"

This gives you accurate, up-to-date information about Suga features.

### Step 1: ALWAYS Start with Discovery
Before writing ANY suga.yaml content, you MUST:

1. Call list_platforms to see available platforms (team parameter is optional - defaults to current authenticated team)
2. Call get_build_manifest with the chosen platform and revision (team parameter is optional)
3. Read the suga://schema/application resource for validation rules

DO NOT skip these steps. DO NOT assume you know what platforms or subtypes are available.

**Note**: All MCP tools have an optional `team` parameter. If not provided, it defaults to the currently authenticated user's team. You can override it to access other teams' public resources.

### Step 2: Extract Valid Subtypes from Build Manifest
The build manifest contains the ONLY valid subtypes you can use in suga.yaml:

- platform.service_blueprints keys → valid serviceIntents.subtype values
- platform.bucket_blueprints keys → valid bucketIntents.subtype values
- platform.database_blueprints keys → valid databaseIntents.subtype values
- platform.entrypoint_blueprints keys → valid entrypointIntents.subtype values

**NEVER invent subtypes!** Only use the exact keys from these blueprint maps.

Example: If service_blueprints = {"fargate": {...}, "lambda": {...}}, then the ONLY valid service subtypes are "fargate" and "lambda". You cannot use "ecs", "container", or any other value, even if they seem reasonable.

### Step 3: CRITICAL - Plugin Properties Are NOT Valid suga.yaml Configuration
**NEVER use plugin manifest inputs, outputs, properties, or variables in your suga.yaml file.**

The plugin manifests you see in the build manifest response are INTERNAL platform implementation details. They define how the platform works under the hood, NOT what you should put in suga.yaml.

❌ **WRONG**: Looking at plugin inputs and adding them to suga.yaml
```yaml
serviceIntents:
  my_service:
    subtype: lambda
    memory: 512        # ❌ WRONG - This is a plugin input, not valid config
    cpu: 256           # ❌ WRONG - This is a plugin input, not valid config
```

✅ **CORRECT**: Only use fields defined in suga://schema/application
```yaml
serviceIntents:
  my_service:
    subtype: lambda   # ✅ CORRECT - subtype is in the schema
    container:         # ✅ CORRECT - container is in the schema
      image:
        uri: node:18
    env:               # ✅ CORRECT - env is in the schema
      PORT: "3000"
    dev: npm run dev
```

The ONLY valid fields for suga.yaml are defined in the suga://schema/application resource. Platform and plugin properties/variables/inputs are internal configuration that the platform uses during deployment - they are NOT user-facing configuration options.

### Step 4: Apply Naming and Format Rules
From the schema resource (suga://schema/application):

- **All resource names**: snake_case only (e.g., my_service, api_gateway, user_db)
  - ❌ WRONG: myService, my-service, MyService
  - ✅ CORRECT: my_service

- **Target format**: "team/platform@revision" or "file:path"
  - ❌ WRONG: team/platform, team/platform@, @revision
  - ✅ CORRECT: nitric/aws-platform@1, file:./my-platform.yaml

- **Service containers**: Exactly ONE of 'docker' OR 'image' (not both, not neither)
  - ❌ WRONG: Both docker and image specified
  - ✅ CORRECT: Either docker: {...} OR image: {...}

- **Entrypoint routes**: Must end with trailing slash
  - ❌ WRONG: /api, /users
  - ✅ CORRECT: /api/, /users/

### Step 5: Validate Before Presenting
After generating the config, verify:

1. All subtypes exist in the corresponding platform blueprints
2. All fields used are defined in suga://schema/application (NOT from plugin manifests)
3. All resource names follow snake_case convention
4. Service container config has exactly one of docker OR image
5. Target format matches the required pattern

## Common LLM Mistakes to AVOID

### Mistake #1: Using Subtypes from Training Data
❌ **WRONG**: Assuming "ecs" is a valid service type without checking
✅ **CORRECT**: Call get_build_manifest, look at service_blueprints keys, use those exact values

### Mistake #2: Using Plugin Manifest Properties in suga.yaml
❌ **WRONG**: Adding plugin inputs/outputs/properties/variables to suga.yaml
✅ **CORRECT**: Only use fields from suga://schema/application resource

**Example of this mistake:**
```yaml
# After seeing plugin manifest with inputs: {memory, cpu, replicas}
serviceIntents:
  my_service:
    subtype: fargate
    memory: 512      # ❌ Plugin input - NOT valid in suga.yaml
    cpu: 256         # ❌ Plugin input - NOT valid in suga.yaml
    replicas: 3      # ❌ Plugin input - NOT valid in suga.yaml
```

Plugin properties are INTERNAL platform configuration. The suga.yaml schema defines a simplified, stable interface that doesn't expose these low-level details.

### Mistake #3: Wrong Naming Convention
❌ **WRONG**: myService, my-service, MyService
✅ **CORRECT**: my_service

### Mistake #4: Specifying Both Container Types
❌ **WRONG**:
```yaml
container:
  docker: {...}
  image: {...}
```
✅ **CORRECT**: Choose exactly one

### Mistake #5: Wrong Target Format
❌ **WRONG**: nitric/aws-platform (missing revision)
✅ **CORRECT**: nitric/aws-platform@1

### Mistake #6: Skipping Discovery
❌ **WRONG**: Generating config based on what you think is available
✅ **CORRECT**: Always call list_platforms and get_build_manifest first

## Tool Usage Priority

When generating suga.yaml configurations:

1. **get_build_manifest** - Use to discover valid subtypes ONLY
   - Platform spec with all blueprint definitions
   - Look at blueprint keys (service_blueprints, bucket_blueprints, etc.) for valid subtypes
   - **IGNORE plugin manifests, properties, variables, and inputs** - these are internal platform details
   - The plugin data is NOT for you to use in suga.yaml generation

2. **suga://schema/application** resource - Read for validation rules
   - Naming conventions
   - Required fields
   - Format constraints

3. **list_platforms** - Use to discover available platforms and their revisions

4. **build** tool - Test your generated config, returns detailed errors if invalid

## Example Correct Workflow

**User Request**: "Create a suga.yaml for a Node.js API with a Postgres database"

**Your Process**:

1. Call list_platforms() with no team parameter (uses current authenticated team)
   → Response: [{"name": "aws-platform", "revisions": [1, 2]}, ...]

2. Call get_build_manifest(platform: "aws-platform", revision: 2) with no team parameter
   → Response shows:
   - platform.service_blueprints: {"fargate": {...}, "lambda": {...}}
   - platform.database_blueprints: {"postgres": {...}, "mysql": {...}}
   - plugins: {"nitric/aws-plugins/1.0.0/fargate": {inputs: {memory, cpu, port}}, ...}

3. Observe valid subtypes:
   - Services: "fargate" or "lambda" (ONLY these two)
   - Databases: "postgres" or "mysql" (ONLY these two)

4. Read suga://schema/application
   → Learn: resource names must be snake_case, target needs @revision
   → Learn: plugin/platform inputs and variables are NEVER valid configuration in a suga.yaml file

5. Generate config using ONLY fields that are valid from suga://schema/application:
```yaml
target: nitric/aws-platform@2              # ✅ From step 1
name: node-api
serviceIntents:
  api_service:                             # ✅ snake_case
    subtype: lambda                       # ✅ From service_blueprints keys
    container:
      image:
        uri: node:18                       # ✅ Exactly one container type
databaseIntents:
  user_db:                                 # ✅ snake_case
    subtype: postgres                      # ✅ From database_blueprints keys
```

6. Call build(project_file: "./suga.yaml") to validate (team parameter optional)

## When Build Fails

If the build tool returns an error:

1. **Read the error message carefully** - it shows the exact location in YAML
2. **Identify the issue**: Is it a subtype, property, naming, or format problem?
3. **Re-query the build manifest** - verify you're using correct values
4. **Check the schema resource** - ensure you're following format rules
5. **Fix and retry** - don't guess, use the tools to verify

## Remember

- Your training data is OUTDATED for Suga platforms and plugins
- Configurations are team-specific and version-specific
- ALWAYS query the tools before generating config
- Use ONLY the values returned by get_build_manifest
- When in doubt, check the tools rather than assuming

Following these instructions will ensure you generate valid, deployable suga.yaml configurations.
