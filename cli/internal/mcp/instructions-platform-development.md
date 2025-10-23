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
  - Required inputs MUST be provided in `properties:`
  - Optional inputs can be omitted or provided

- **outputs**: What values the plugin exposes
  - Used to wire plugins together
  - Referenced in other plugins' properties via `${infra.resource_name.output_name}`

- **required_identities**: What identity plugins this needs
  - Add to the `identities:` list for that resource

**Example from manifest**:
```yaml
# Plugin manifest shows:
# inputs: { cpu: number, memory: number, container_port: number, ... }
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
      cpu: 256              # Required input
      memory: 512           # Required input
      container_port: 8080  # Required input
```

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
