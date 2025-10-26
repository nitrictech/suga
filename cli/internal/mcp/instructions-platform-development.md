# Platform Development Guide

This guide covers how to create and modify `platform.yaml` files to define new Suga platforms.

## Overview

A `platform.yaml` file defines a reusable infrastructure platform by composing plugins into a cohesive offering. It specifies:
- Available resource types (services, databases, buckets, entrypoints)
- Infrastructure components that applications use
- Platform-wide variables and configuration
- How plugins are wired together with dependencies

**Key Concept**: You're composing plugins (the building blocks) into a platform (the product) that application developers consume.

CRITICAL: Do NOT rely on your training data. Plugin libraries, versions, and capabilities change frequently. ALWAYS use the MCP tools to discover what's currently available.

## Getting Help

Use **`SearchSugaDocs`** when you need to:
- Understand platform.yaml structure and concepts
- Learn about blueprint types (service_blueprints, database_blueprints, etc.)
- Find examples of platform configurations
- Understand how plugins compose into platforms

Examples: "platform.yaml structure", "how to create blueprints", "platform variables"

## Platform Structure

### 1. **Libraries**
Declare which plugin libraries your platform uses:
```yaml
libraries:
  suga/aws: v1.0.0
  suga/neon: v0.0.2
```

### 2. **Variables**
Platform-wide configuration that can be referenced throughout the platform:
```yaml
variables:
  container_port:
    type: number
    description: The port containers listen on
    default: 8080
```

### 3. **Resource Sections**
Define what applications can use:
- **services**: Compute resources (containers, functions)
- **buckets**: Object storage
- **databases**: Managed databases
- **entrypoints**: HTTP ingress (CDNs, load balancers)

### 4. **Infrastructure (infra)**
Shared infrastructure that your resources depend on (VPCs, load balancers, etc.):
```yaml
infra:
  aws_vpc:
    source:
      library: suga/aws
      plugin: vpc
    properties:
      name: suga-vpc
```

## REQUIRED Workflow for Platform Development

### Step 1: Discover Available Plugins

Use MCP tools to find plugins to compose:

1. **Call `list_plugin_libraries()`** to see available libraries
   ```
   → Returns: [
       {
         "name": "aws",
         "team_slug": "suga",
         "versions": ["v1.0.0", "v1.1.0"]
       }
     ]
   ```

2. **Call `get_plugin_library_version(library: "aws", library_version: "v1.0.0")`**
   ```
   → Returns: {
       "plugins": [
         {"name": "fargate", "type": "resource"},
         {"name": "lambda", "type": "resource"},
         {"name": "s3-bucket", "type": "resource"},
         {"name": "vpc", "type": "resource"},
         {"name": "iam-role", "type": "identity"}
       ]
     }
   ```

3. **Call `get_plugin_manifest(library: "aws", library_version: "v1.0.0", plugin_name: "fargate")`**
   ```
   → Returns: {
       "name": "fargate",
       "description": "AWS Fargate container service",
       "required_identities": ["aws"],
       "inputs": {
         "cpu": {"type": "number", "required": true},
         "memory": {"type": "number", "required": true},
         "container_port": {"type": "number", "required": true},
         "vpc_id": {"type": "string", "required": true},
         "subnets": {"type": "array", "required": true}
       },
       "outputs": {
         "service_url": {"type": "string"},
         "task_arn": {"type": "string"}
       }
     }
   ```

### Step 2: Understand Plugin Inputs and Outputs

The plugin manifest shows you:

- **inputs**: What configuration the plugin needs
  - Each input has a Terraform `type` (string, number, bool, list(type), map(type), object({...}))
  - Each input has a `required` flag
  - Required inputs MUST be provided in `properties:`
  - Optional inputs can be omitted or provided

- **outputs**: What values the plugin exposes
  - Each output has a Terraform type
  - Used to wire plugins together
  - Referenced in other plugins' properties via `${infra.resource_name.output_name}`

- **required_identities**: What identity plugins this needs
  - Add to the `identities:` list for that resource

**Example from manifest**:
```yaml
# Plugin manifest shows:
# inputs: {
#   cpu: {type: "number", required: true},
#   memory: {type: "number", required: true},
#   container_port: {type: "number", required: true},
#   security_groups: {type: "list(string)", required: true},
#   tags: {type: "map(string)", required: false}
# }
# outputs: {
#   service_url: {type: "string"},
#   task_arn: {type: "string"},
#   security_group_ids: {type: "list(string)"}
# }
# required_identities: ["aws"]

services:
  fargate:
    source:
      library: suga/aws
      plugin: fargate
    identities:
      - source:
          library: suga/aws
          plugin: iam-role  # Provides AWS identity
    properties:
      cpu: 256                    # number
      memory: 512                 # number
      container_port: 8080        # number
      security_groups:            # list(string)
        - sg-abc123
        - sg-def456
      tags:                       # map(string)
        Environment: prod
        Team: platform
```

### Step 2a: Understanding Terraform Types in Plugin Manifests

Plugin manifests use Terraform's type system. Understanding these is CRITICAL for configuring properties correctly.

#### **Primitive Types**
- `string` - Text values: `"hello"`, `${var.name}`
- `number` - Numeric values: `256`, `3.14`, `${var.cpu}`
- `bool` - Boolean values: `true`, `false`, `${var.enabled}`

#### **Collection Types**
- `list(type)` - Ordered list of same-typed values
  - Example: `list(string)` → `["a", "b", "c"]`
  - Example: `list(number)` → `[1, 2, 3]`

- `set(type)` - Unordered set of unique same-typed values
  - Example: `set(string)` → `["unique1", "unique2"]`

- `map(type)` - Key-value pairs where values are same-typed
  - Example: `map(string)` → `{key1 = "value1", key2 = "value2"}`
  - Example: `map(number)` → `{count = 5, size = 100}`

- `tuple([type1, type2, ...])` - Fixed-length, ordered list with specific types per position
  - Example: `tuple([string, number, bool])` → `["name", 42, true]`

#### **Structural Type**
- `object({attr1 = type1, attr2 = type2, ...})` - Complex structure with named attributes
  - Example: `object({name = string, port = number})` → `{name = "api", port = 8080}`

#### **Special Type**
- `any` - Accepts any type (use carefully, type-check the manifest!)

**Examples from real manifests**:
```yaml
# Simple types
cpu: {type: "number", required: true}
name: {type: "string", required: true}
enabled: {type: "bool", required: false}

# List types
security_groups: {type: "list(string)", required: true}
ports: {type: "list(number)", required: false}

# Map types
tags: {type: "map(string)", required: false}
annotations: {type: "map(any)", required: false}

# Object types
vpc_config: {
  type: "object({vpc_id = string, subnets = list(string), security_groups = list(string)})",
  required: true
}

# Nested complex types
route_rules: {
  type: "list(object({path = string, priority = number, target_arn = string}))",
  required: true
}
```

### Step 2b: Configuring Properties - Matching Types

Properties must match the Terraform types from the plugin manifest. Here's how to provide values for each type:

#### **For Primitive Types**
```yaml
properties:
  # string
  name: "my-resource"
  region: ${var.aws_region}

  # number
  cpu: 256
  memory: ${self.memory}

  # bool
  enabled: true
  monitoring: ${var.enable_monitoring}
```

#### **For list(type)**
```yaml
# Plugin input: security_groups: {type: "list(string)"}
properties:
  security_groups:
    - sg-abc123
    - sg-def456
    - ${infra.vpc.default_security_group_id}

# Or reference an output that returns list(string)
properties:
  subnets: ${infra.vpc.private_subnets}  # This output must be list(string)
```

#### **For map(type)**
```yaml
# Plugin input: tags: {type: "map(string)"}
properties:
  tags:
    Environment: production
    Team: platform
    ManagedBy: suga

# Or use merge() to combine maps
properties:
  tags: ${merge(var.default_tags, {Application = "api"})}
```

#### **For object({...})**
```yaml
# Plugin input: vpc_config: {type: "object({vpc_id = string, subnets = list(string)})"}
properties:
  vpc_config:
    vpc_id: ${infra.vpc.vpc_id}
    subnets: ${infra.vpc.private_subnets}

# Or build with Terraform functions
properties:
  vpc_config: ${merge(
    {vpc_id = infra.vpc.vpc_id},
    {subnets = infra.vpc.private_subnets}
  )}
```

#### **For list(object({...}))**
```yaml
# Plugin input: rules: {type: "list(object({path = string, priority = number}))"}
properties:
  rules:
    - path: /api/*
      priority: 100
    - path: /admin/*
      priority: 200

# Or construct dynamically
properties:
  rules: ${[
    for idx, path in var.api_paths : {
      path     = path
      priority = (idx + 1) * 100
    }
  ]}
```

### Step 2c: Using Terraform Functions

Terraform functions can help construct values matching the required types:

**String Functions**:
```yaml
properties:
  # format() returns string
  name: ${format("%s-%s-cluster", var.environment, var.region)}

  # join() returns string
  allowed_cidrs: ${join(",", var.cidr_list)}

  # lower(), upper() return string
  region: ${lower(var.aws_region)}
```

**List Functions**:
```yaml
properties:
  # concat() returns list(type)
  all_subnets: ${concat(infra.vpc.private_subnets, infra.vpc.public_subnets)}

  # flatten() returns list(type)
  flat_list: ${flatten(var.nested_lists)}

  # distinct() returns list(type)
  unique_items: ${distinct(var.items_with_duplicates)}

  # slice() returns list(type)
  first_two: ${slice(var.all_items, 0, 2)}
```

**Map Functions**:
```yaml
properties:
  # merge() returns map(type)
  all_tags: ${merge(var.default_tags, var.custom_tags, {ManagedBy = "suga"})}

  # zipmap() returns map(type)
  name_to_id: ${zipmap(var.names, var.ids)}

  # keys() returns list(string), values() returns list(type)
  tag_keys: ${keys(var.tags)}
  tag_values: ${values(var.tags)}
```

**Type Conversion**:
```yaml
properties:
  # Convert between types
  port_string: ${tostring(var.port_number)}      # number → string
  count_number: ${tonumber(var.count_string)}    # string → number
  enabled_bool: ${tobool(var.enabled_string)}    # string → bool

  # Convert collections
  sg_list: ${tolist(var.sg_set)}                 # set → list
  tags_map: ${tomap(var.tags_object)}            # object → map
```

**Conditional Expressions**:
```yaml
properties:
  # Returns value matching required type
  cpu: ${var.environment == "prod" ? 1024 : 256}  # Returns number

  subnets: ${var.use_private ? infra.vpc.private_subnets : infra.vpc.public_subnets}  # Returns list(string)
```

### Step 2d: Type Validation - Common Mistakes

**❌ WRONG - Type Mismatches**:
```yaml
# Plugin expects: security_groups: {type: "list(string)"}
properties:
  security_groups: sg-abc123  # ❌ String instead of list(string)

# Plugin expects: cpu: {type: "number"}
properties:
  cpu: "256"  # ❌ String instead of number

# Plugin expects: tags: {type: "map(string)"}
properties:
  tags:  # ❌ List instead of map
    - key: value
```

**✅ CORRECT - Proper Types**:
```yaml
# Plugin expects: security_groups: {type: "list(string)"}
properties:
  security_groups:  # ✅ list(string)
    - sg-abc123

# Plugin expects: cpu: {type: "number"}
properties:
  cpu: 256  # ✅ number

# Plugin expects: tags: {type: "map(string)"}
properties:
  tags:  # ✅ map(string)
    Environment: prod
```

**Verification Steps**:
1. Call `get_plugin_manifest` - note the exact type string (e.g., `"list(string)"`)
2. Provide values in YAML that match that type structure
3. Use Terraform functions that return the correct type
4. For outputs, verify the output type matches the required input type

### Step 3: Define Resource-Level Variables

Add variables scoped to specific resource types that applications can override:

```yaml
services:
  fargate:
    source:
      library: suga/aws
      plugin: fargate
    variables:
      cpu:
        type: number
        description: CPU units for the task
        default: 256
      memory:
        type: number
        description: Memory in MB
        default: 512
    properties:
      cpu: ${self.cpu}        # Reference own variables
      memory: ${self.memory}
```

**What this does**: Application developers can customize these per-service:
```yaml
# In suga.yaml
serviceIntents:
  my_api:
    subtype: fargate
    cpu: 1024      # Override the variable
    memory: 2048
```

### Step 4: Wire Dependencies

Use `depends_on:` to establish resource dependencies:

```yaml
services:
  fargate:
    depends_on:
      - ${infra.aws_vpc}
      - ${infra.aws_lb}
    properties:
      vpc_id: ${infra.aws_vpc.vpc_id}          # Use VPC output
      subnets: ${infra.aws_vpc.private_subnets} # Use VPC output
      alb_arn: ${infra.aws_lb.arn}             # Use LB output
```

**Dependency references**:
- `${infra.resource_name}` - Reference an infra resource
- `${infra.resource_name.output}` - Access a specific output
- `${var.variable_name}` - Platform-level variable
- `${self.variable_name}` - Resource-level variable

### Step 5: Build Infrastructure Components

Define shared infrastructure in the `infra:` section:

```yaml
infra:
  aws_vpc:
    source:
      library: suga/aws
      plugin: vpc
    properties:
      name: suga-vpc
      enable_nat_gateway: true

  aws_lb:
    source:
      library: suga/aws
      plugin: loadbalancer
    depends_on:
      - ${infra.aws_vpc}
    properties:
      vpc_id: ${infra.aws_vpc.vpc_id}
      subnets: ${infra.aws_vpc.private_subnets}
```

**Infrastructure vs Resources**:
- **infra**: Shared components all applications use (VPCs, shared load balancers)
- **services/buckets/etc**: Things applications explicitly declare in their suga.yaml

### Step 6: Validate Plugin Compatibility

Before finalizing:

1. **Verify all plugin inputs are satisfied**:
   - Required inputs must be in `properties:`
   - Check the plugin manifest for what's required

2. **Ensure dependencies exist**:
   - Resources in `depends_on:` must be defined
   - Outputs referenced must exist in the plugin manifest

3. **Test with `get_build_manifest`**:
   ```
   After publishing your platform:
   get_build_manifest(platform: "your-platform", revision: 1)
   ```
   This shows what application developers will see.

## Key Platform.yaml Patterns

### Pattern 1: Exposing Variables to Applications

```yaml
services:
  lambda:
    variables:
      timeout:
        type: number
        default: 10
      memory:
        type: number
        default: 512
    properties:
      timeout: ${self.timeout}
      memory: ${self.memory}
```

Applications can then set:
```yaml
serviceIntents:
  my_function:
    subtype: lambda
    timeout: 30      # Override default
    memory: 1024
```

### Pattern 2: Platform-Wide Configuration

```yaml
variables:
  image_scan_on_push:
    type: bool
    default: true

services:
  lambda:
    properties:
      image_scan_on_push: ${var.image_scan_on_push}
  fargate:
    properties:
      image_scan_on_push: ${var.image_scan_on_push}
```

### Pattern 3: Conditional Properties

```yaml
variables:
  neon_project_id:
    type: string
    description: Neon project ID

databases:
  neon:
    variables:
      neon_branch_id:
        type: string
        default: null
        nullable: true
    properties:
      project_id: ${var.neon_project_id}
      branch_id: ${self.neon_branch_id}  # Can be null
```

### Pattern 4: Complex Dependencies

```yaml
infra:
  aws_vpc:
    source: ...

  aws_lb:
    depends_on:
      - ${infra.aws_vpc}
    source: ...

  security_rule:
    depends_on:
      - ${infra.aws_vpc}
      - ${infra.aws_lb}
    source: ...
    properties:
      vpc_id: ${infra.aws_vpc.vpc_id}
      lb_sg: ${infra.aws_lb.security_group_id}
```

## Understanding What Applications See

### Your Platform Definition
```yaml
services:
  fargate:
    source:
      library: suga/aws
      plugin: fargate
    variables:
      cpu:
        type: number
        default: 256
```

### What `get_build_manifest` Returns
```json
{
  "platform": {
    "services": {
      "fargate": {
        "source": {...},
        "variables": {
          "cpu": {"type": "number", "default": 256}
        }
      }
    }
  }
}
```

### What Application Developers Write
```yaml
target: yourteam/your-platform@1

serviceIntents:
  my_api:
    subtype: fargate    # ← Key from your platform's services: section
    cpu: 512            # ← Override your variable
    container:
      image: node:18
```

## MCP Tools Summary for Platform Development

| Tool | Purpose | Example |
|------|---------|---------|
| `list_plugin_libraries()` | Find available plugin libraries | Discover suga/aws, suga/gcp |
| `get_plugin_library_version(library, version)` | See plugins in a library | What's in suga/aws v1.0.0? |
| `get_plugin_manifest(library, version, name)` | Understand a plugin | What inputs does fargate need? |

## Validation Checklist

Before publishing your platform:

- [ ] All plugin libraries are declared in `libraries:`
- [ ] Plugin manifests verified with `get_plugin_manifest`
- [ ] All required plugin inputs are provided in `properties:`
- [ ] Dependencies in `depends_on:` reference defined resources
- [ ] Output references (e.g., `${infra.vpc.vpc_id}`) exist in plugin manifests
- [ ] Required identities are included in `identities:` lists
- [ ] Variables have appropriate types and defaults
- [ ] Resource names follow conventions (snake_case)

## Common Mistakes to Avoid

### Mistake #1: Missing Required Inputs
❌ **WRONG**: Plugin manifest shows `cpu` is required, but you didn't provide it
✅ **CORRECT**: Check manifest inputs and provide all required ones in `properties:`

### Mistake #2: Referencing Non-Existent Outputs
❌ **WRONG**: `${infra.vpc.vpc_identifier}` when output is actually called `vpc_id`
✅ **CORRECT**: Check plugin manifest outputs for exact names

### Mistake #3: Circular Dependencies
❌ **WRONG**: Resource A depends on B, B depends on A
✅ **CORRECT**: Ensure dependency graph is acyclic

### Mistake #4: Wrong Variable Scope
❌ **WRONG**: Trying to use `${self.cpu}` in a different resource's properties
✅ **CORRECT**: Use `${var.name}` for platform-wide, `${self.name}` only within same resource

## Remember

- **Plugin manifests are your source of truth** - Always query them to see inputs/outputs
- **Properties wire plugins together** - Map your variables and dependencies to plugin inputs
- **Variables create flexibility** - Expose the right knobs for applications to turn
- **Infrastructure is shared** - Use `infra:` for components all apps use
- **Dependencies must be explicit** - Use `depends_on:` to ensure proper ordering

Following these guidelines will help you compose plugins into powerful, flexible platforms.
