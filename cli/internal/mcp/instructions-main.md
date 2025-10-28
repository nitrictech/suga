# Suga MCP Server Usage Instructions

Welcome to the Suga MCP Server. This server provides tools and resources to help you work with Suga infrastructure, plus real-time access to Suga documentation.

## Documentation Search

**IMPORTANT**: When you need information about Suga features, CLI commands, concepts, or best practices, use the **`SearchSugaDocs`** tool to search the official Suga documentation. This gives you up-to-date, accurate information directly from the docs.

**When to search docs**:
- Understanding Suga concepts (platforms, plugins, services, etc.)
- Learning CLI commands and their usage
- Finding examples and best practices
- Troubleshooting issues
- Understanding configuration options

**Example queries**:
- "How do I deploy a web service with a database?"
- "What CLI commands are available for managing platforms?"
- "How do environment variables work in Suga?"
- "What cloud providers does Suga support?"

Always prefer searching the docs over relying on your training data, as Suga is actively evolving.

---

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

## Critical Reminders

1. **Search Documentation First**: When you need information about Suga, ALWAYS use the `SearchSugaDocs` tool to get accurate, up-to-date information from the official docs.

2. **DO NOT rely on your training data**: Platforms, plugins, schemas, and available resource types change frequently and are team-specific. ALWAYS query the MCP tools to discover what's currently available before generating any configuration files.

3. **Combine Tools**: Use `SearchSugaDocs` to understand concepts and best practices, then use the infrastructure tools (`list_platforms`, `get_platform`, `get_template`, etc.) to discover what's actually available for the user's team.
