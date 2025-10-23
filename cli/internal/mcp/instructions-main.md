# Suga MCP Server Usage Instructions

Welcome to the Suga MCP Server. This server provides tools and resources to help you work with Suga infrastructure.

## What Are You Trying To Do?

Before using the MCP tools, identify your scenario:

### Scenario 1: Application Development
**Goal**: Create or modify a `suga.yaml` file to deploy an application

**When to use**:
- You want to deploy a web service, API, database, or other application components
- You need to generate a `suga.yaml` configuration file
- You're using an existing Suga platform

**Next Step**: Read the **`suga://guides/app-development`** resource for detailed instructions

**Key Concept**: You consume an existing platform and define what resources your application needs

---

### Scenario 2: Platform Development
**Goal**: Create or modify a `platform.yaml` file to define a new Suga platform

**When to use**:
- You want to create a reusable platform that others can target
- You need to define what resource types (services, databases, etc.) are available
- You're composing plugins into a cohesive platform offering

**Next Step**: Read the **`suga://guides/platform-development`** resource for detailed instructions

**Key Concept**: You define what resource types are available and how they're implemented using plugins

---

### Scenario 3: Plugin Library Development
**Goal**: Create or modify plugins in a plugin library to provide infrastructure building blocks

**When to use**:
- You want to create reusable Terraform modules for infrastructure resources
- You need to implement cloud provider-specific resources (Lambda, S3, Fargate, etc.)
- You're building the lowest-level building blocks that platforms compose
- You need to provide Go runtime code for services and buckets

**Next Step**: Read the **`suga://guides/plugin-library-development`** resource for detailed instructions

**Key Concept**: You create the building blocks (plugins) that platforms use. Each plugin is a Terraform module with a manifest, and services/buckets require Go runtime code.

---

## Important Notes

### Team Parameter
All MCP tools have an optional `team` parameter. If not provided, it defaults to the currently authenticated user's team. You can override it to access other teams' public resources.

### Authentication
The MCP server requires authentication via `suga login`. If you receive authentication errors, the user needs to run the login command first.

### General Resources Available

- **`suga://schema/application`** - JSON Schema for `suga.yaml` application files
- **`suga://schema/platform`** - JSON Schema for `platform.yaml` platform definition files (if available)
- **`suga://guides/app-development`** - Detailed guide for application development
- **`suga://guides/platform-development`** - Detailed guide for platform development
- **`suga://guides/plugin-library-development`** - Detailed guide for plugin library development

## Critical Reminder

**DO NOT rely on your training data.** Platforms, plugins, schemas, and available resource types change frequently and are team-specific. ALWAYS query the MCP tools to discover what's currently available before generating any configuration files.
