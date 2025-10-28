package mcp

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/build"
	"github.com/nitrictech/suga/cli/internal/config"
	"github.com/nitrictech/suga/cli/pkg/schema"
)

//go:embed instructions-main.md
var serverInstructions string

//go:embed instructions-app-development.md
var appDevelopmentInstructions string

//go:embed instructions-platform-development.md
var platformDevelopmentInstructions string

//go:embed instructions-plugin-library-development.md
var pluginLibraryDevelopmentInstructions string

// Server wraps the MCP server with Suga API client
type Server struct {
	mcpServer *mcp.Server
	apiClient *api.SugaApiClient
	config    *config.Config
	builder   *build.BuilderService
	docsProxy *DocsProxy
}

// NewServer creates a new MCP server with the given API client and config
func NewServer(apiClient *api.SugaApiClient, cfg *config.Config, builder *build.BuilderService) (*Server, error) {
	s := &Server{
		apiClient: apiClient,
		config:    cfg,
		builder:   builder,
	}

	// Create MCP server with instructions
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "suga-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Instructions: serverInstructions,
	})

	s.mcpServer = mcpServer

	// Initialize docs proxy (but don't fail if it can't connect)
	s.docsProxy = NewDocsProxy()

	// Register tools
	if err := s.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	// Register resources
	if err := s.registerResources(); err != nil {
		return nil, fmt.Errorf("failed to register resources: %w", err)
	}

	return s, nil
}

// getCurrentTeam retrieves the current team slug for the authenticated user.
// Returns the team slug and an error if the user is not authenticated or has no current team.
func (s *Server) getCurrentTeam() (string, error) {
	allTeams, err := s.apiClient.GetUserTeams()
	if err != nil {
		return "", fmt.Errorf("not authenticated: %w", err)
	}

	for _, t := range allTeams {
		if t.IsCurrent {
			return t.Slug, nil
		}
	}

	return "", fmt.Errorf("no current team set")
}

// getTeamOrDefault returns the provided team if non-empty, otherwise returns the current team.
func (s *Server) getTeamOrDefault(team string) (string, error) {
	if team != "" {
		return team, nil
	}
	return s.getCurrentTeam()
}

// Input types for tools

type ListTemplatesArgs struct {
	Team string `json:"team,omitempty" jsonschema:"Team slug to list templates for (defaults to current team if not specified)"`
}

type GetTemplateArgs struct {
	TeamSlug     string `json:"team_slug,omitempty" jsonschema:"Team slug that owns the template (defaults to current team if not specified)"`
	TemplateName string `json:"template_name" jsonschema:"Name of the template"`
	Version      string `json:"version,omitempty" jsonschema:"Version of the template (optional defaults to latest)"`
}

type GetPlatformArgs struct {
	Team     string `json:"team,omitempty" jsonschema:"Team slug that owns the platform (defaults to current team if not specified)"`
	Name     string `json:"name" jsonschema:"Name of the platform"`
	Revision int    `json:"revision" jsonschema:"Revision number of the platform"`
	Public   bool   `json:"public,omitempty" jsonschema:"Whether to fetch from public platforms (defaults to false)"`
}

type GetBuildManifestArgs struct {
	Team     string `json:"team,omitempty" jsonschema:"Team slug that owns the platform (defaults to current team if not specified)"`
	Platform string `json:"platform" jsonschema:"Name of the platform"`
	Revision int    `json:"revision" jsonschema:"Revision number of the platform"`
	Public   bool   `json:"public,omitempty" jsonschema:"Whether to fetch from public platforms (defaults to false)"`
}

type GetPluginManifestArgs struct {
	Team           string `json:"team,omitempty" jsonschema:"Team slug that owns the plugin library (defaults to current team if not specified)"`
	Library        string `json:"library" jsonschema:"Name of the plugin library"`
	LibraryVersion string `json:"library_version" jsonschema:"Version of the plugin library"`
	PluginName     string `json:"plugin_name" jsonschema:"Name of the plugin"`
	Public         bool   `json:"public,omitempty" jsonschema:"Whether to fetch from public plugin libraries (defaults to false)"`
}

type ListPlatformsArgs struct {
	Team   string `json:"team,omitempty" jsonschema:"Team slug to list platforms for (defaults to current team if not specified)"`
	Public bool   `json:"public,omitempty" jsonschema:"Whether to fetch from public platforms (defaults to false)"`
}

type ListPluginLibrariesArgs struct {
	Team   string `json:"team,omitempty" jsonschema:"Team slug to list plugin libraries for (defaults to current team if not specified)"`
	Public bool   `json:"public,omitempty" jsonschema:"Whether to fetch from public plugin libraries (defaults to false)"`
}

type GetPluginLibraryVersionArgs struct {
	Team           string `json:"team,omitempty" jsonschema:"Team slug that owns the plugin library (defaults to current team if not specified)"`
	Library        string `json:"library" jsonschema:"Name of the plugin library"`
	LibraryVersion string `json:"library_version" jsonschema:"Version of the plugin library"`
	Public         bool   `json:"public,omitempty" jsonschema:"Whether to fetch from public plugin libraries (defaults to false)"`
}

type BuildArgs struct {
	Team        string `json:"team,omitempty" jsonschema:"Team slug for the build (defaults to current team if not specified)"`
	ProjectFile string `json:"project_file,omitempty" jsonschema:"Path to the suga.yaml project file (defaults to ./suga.yaml)"`
}

// registerTools registers all available tools with the MCP server
func (s *Server) registerTools() error {
	// Connect to docs proxy and register docs tools first
	ctx := context.Background()
	if err := s.docsProxy.Connect(ctx); err != nil {
		log.Printf("Warning: failed to connect to docs server, documentation features will be unavailable: %v", err)
	} else {
		// Register docs tools if connection succeeded
		if err := s.registerDocsProxyTools(); err != nil {
			log.Printf("Warning: failed to register docs tools: %v", err)
		}
	}

	// Register list_templates tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_templates",
		Description: "List all available templates for a team, including both team-specific and public templates",
	}, s.handleListTemplates)

	// Register get_template tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_template",
		Description: "Get details for a specific template by team slug, template name, and optional version",
	}, s.handleGetTemplate)

	// Register get_platform tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_platform",
		Description: "Get platform specification by team slug, platform name, and revision number",
	}, s.handleGetPlatform)

	// Register get_build_manifest tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_build_manifest",
		Description: "Get complete build manifest including platform spec and all plugin manifests for a platform revision",
	}, s.handleGetBuildManifest)

	// Register get_plugin_manifest tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_plugin_manifest",
		Description: "Get plugin manifest by team slug, library name, library version, and plugin name",
	}, s.handleGetPluginManifest)

	// Register list_platforms tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_platforms",
		Description: "List all platforms for a team with their available revisions, including both team-specific and public platforms",
	}, s.handleListPlatforms)

	// Register list_plugin_libraries tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_plugin_libraries",
		Description: "List all plugin libraries for a team with their available versions, including both team-specific and public libraries",
	}, s.handleListPluginLibraries)

	// Register get_plugin_library_version tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_plugin_library_version",
		Description: "Get details about a specific plugin library version, including all plugins in that version with their metadata",
	}, s.handleGetPluginLibraryVersion)

	// Register build tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "build",
		Description: "Build a Suga application, generating Terraform infrastructure code from the application specification",
	}, s.handleBuild)

	return nil
}

// registerResources registers all available resources with the MCP server
func (s *Server) registerResources() error {
	// Register application schema resource
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "suga://schema/application",
		Name:        "Application Schema",
		Description: "JSON Schema for suga.yaml application configuration files",
		MIMEType:    "application/schema+json",
	}, s.handleApplicationSchema)

	// Register application development guide
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "suga://guides/app-development",
		Name:        "Application Development Guide",
		Description: "Complete guide for creating suga.yaml application configuration files",
		MIMEType:    "text/markdown",
	}, s.handleAppDevelopmentGuide)

	// Register platform development guide
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "suga://guides/platform-development",
		Name:        "Platform Development Guide",
		Description: "Complete guide for creating platform.yaml platform definition files",
		MIMEType:    "text/markdown",
	}, s.handlePlatformDevelopmentGuide)

	// Register plugin library development guide
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "suga://guides/plugin-library-development",
		Name:        "Plugin Library Development Guide",
		Description: "Complete guide for creating Suga plugin libraries with Terraform modules and Go runtime code",
		MIMEType:    "text/markdown",
	}, s.handlePluginLibraryDevelopmentGuide)

	return nil
}

// Run starts the MCP server with stdio transport
func (s *Server) Run(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// Tool handlers

func (s *Server) handleWithTeamAndJSON(
	teamExtractor func() string,
	apiCall func(team string) (interface{}, error),
	resultTransform func(interface{}) (interface{}, error),
) (*mcp.CallToolResult, any, error) {
	team, err := s.getTeamOrDefault(teamExtractor())
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to get team: %v", err)},
			},
		}, nil, nil
	}

	result, err := apiCall(team)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("%v", err)},
			},
		}, nil, nil
	}

	if resultTransform != nil {
		result, err = resultTransform(result)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("%v", err)},
				},
			}, nil, nil
		}
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to marshal result: %v", err)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil, nil
}

func (s *Server) handleListTemplates(ctx context.Context, req *mcp.CallToolRequest, args ListTemplatesArgs) (*mcp.CallToolResult, any, error) {
	return s.handleWithTeamAndJSON(
		func() string { return args.Team },
		func(team string) (interface{}, error) {
			templates, err := s.apiClient.GetTemplates(team)
			if err != nil {
				return nil, fmt.Errorf("Failed to list templates: %w", err)
			}
			return templates, nil
		},
		nil,
	)
}

func (s *Server) handleGetTemplate(ctx context.Context, req *mcp.CallToolRequest, args GetTemplateArgs) (*mcp.CallToolResult, any, error) {
	return s.handleWithTeamAndJSON(
		func() string { return args.TeamSlug },
		func(team string) (interface{}, error) {
			template, err := s.apiClient.GetTemplate(team, args.TemplateName, args.Version)
			if err != nil {
				return nil, fmt.Errorf("Failed to get template: %w", err)
			}
			return template, nil
		},
		nil,
	)
}

func (s *Server) handleGetPlatform(ctx context.Context, req *mcp.CallToolRequest, args GetPlatformArgs) (*mcp.CallToolResult, any, error) {
	return s.handleWithTeamAndJSON(
		func() string { return args.Team },
		func(team string) (interface{}, error) {
			var platform interface{}
			var err error
			if args.Public {
				platform, err = s.apiClient.GetPublicPlatform(team, args.Name, args.Revision)
			} else {
				platform, err = s.apiClient.GetPlatform(team, args.Name, args.Revision)
			}
			if err != nil {
				return nil, fmt.Errorf("Failed to get platform: %w", err)
			}
			return platform, nil
		},
		nil,
	)
}

func (s *Server) handleGetBuildManifest(ctx context.Context, req *mcp.CallToolRequest, args GetBuildManifestArgs) (*mcp.CallToolResult, any, error) {
	return s.handleWithTeamAndJSON(
		func() string { return args.Team },
		func(team string) (interface{}, error) {
			var platformSpec interface{}
			var plugins map[string]map[string]any
			var err error
			if args.Public {
				platformSpec, plugins, err = s.apiClient.GetPublicBuildManifest(team, args.Platform, args.Revision)
			} else {
				platformSpec, plugins, err = s.apiClient.GetBuildManifest(team, args.Platform, args.Revision)
			}
			if err != nil {
				return nil, fmt.Errorf("Failed to get build manifest: %w", err)
			}
			return map[string]interface{}{
				"platform": platformSpec,
				"plugins":  plugins,
			}, nil
		},
		nil,
	)
}

func (s *Server) handleGetPluginManifest(ctx context.Context, req *mcp.CallToolRequest, args GetPluginManifestArgs) (*mcp.CallToolResult, any, error) {
	return s.handleWithTeamAndJSON(
		func() string { return args.Team },
		func(team string) (interface{}, error) {
			var manifest interface{}
			var err error
			if args.Public {
				manifest, err = s.apiClient.GetPublicPluginManifest(team, args.Library, args.LibraryVersion, args.PluginName)
			} else {
				manifest, err = s.apiClient.GetPluginManifest(team, args.Library, args.LibraryVersion, args.PluginName)
			}
			if err != nil {
				return nil, fmt.Errorf("Failed to get plugin manifest: %w", err)
			}
			return manifest, nil
		},
		nil,
	)
}

func (s *Server) handleListPlatforms(ctx context.Context, req *mcp.CallToolRequest, args ListPlatformsArgs) (*mcp.CallToolResult, any, error) {
	if args.Public {
		// Public platforms don't require team parameter
		platforms, err := s.apiClient.ListPublicPlatforms()
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to list platforms: %v", err)},
				},
			}, nil, nil
		}

		resultJSON, err := json.MarshalIndent(platforms, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to marshal result: %v", err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(resultJSON)},
			},
		}, nil, nil
	}

	return s.handleWithTeamAndJSON(
		func() string { return args.Team },
		func(team string) (interface{}, error) {
			platforms, err := s.apiClient.ListPlatforms(team)
			if err != nil {
				return nil, fmt.Errorf("Failed to list platforms: %w", err)
			}
			return platforms, nil
		},
		nil,
	)
}

func (s *Server) handleListPluginLibraries(ctx context.Context, req *mcp.CallToolRequest, args ListPluginLibrariesArgs) (*mcp.CallToolResult, any, error) {
	if args.Public {
		// Public plugin libraries don't require team parameter
		libraries, err := s.apiClient.ListPublicPluginLibraries()
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to list plugin libraries: %v", err)},
				},
			}, nil, nil
		}

		resultJSON, err := json.MarshalIndent(libraries, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to marshal result: %v", err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(resultJSON)},
			},
		}, nil, nil
	}

	return s.handleWithTeamAndJSON(
		func() string { return args.Team },
		func(team string) (interface{}, error) {
			libraries, err := s.apiClient.ListPluginLibraries(team)
			if err != nil {
				return nil, fmt.Errorf("Failed to list plugin libraries: %w", err)
			}
			return libraries, nil
		},
		nil,
	)
}

func (s *Server) handleGetPluginLibraryVersion(ctx context.Context, req *mcp.CallToolRequest, args GetPluginLibraryVersionArgs) (*mcp.CallToolResult, any, error) {
	return s.handleWithTeamAndJSON(
		func() string { return args.Team },
		func(team string) (interface{}, error) {
			var version *api.PluginLibraryVersion
			var err error
			if args.Public {
				version, err = s.apiClient.GetPublicPluginLibraryVersion(team, args.Library, args.LibraryVersion)
			} else {
				version, err = s.apiClient.GetPluginLibraryVersion(team, args.Library, args.LibraryVersion)
			}
			if err != nil {
				return nil, fmt.Errorf("Failed to get plugin library version: %w", err)
			}
			return version, nil
		},
		nil,
	)
}

func (s *Server) handleBuild(ctx context.Context, req *mcp.CallToolRequest, args BuildArgs) (*mcp.CallToolResult, any, error) {
	team, err := s.getTeamOrDefault(args.Team)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to get team: %v", err)},
			},
		}, nil, nil
	}

	projectFile := args.ProjectFile
	if projectFile == "" {
		projectFile = "./suga.yaml"
	}

	// Prevent path traversal
	clean := filepath.Clean(projectFile)
	absProject, err := filepath.Abs(clean)
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Invalid project file path: %v", err)}}}, nil, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Cannot determine working directory: %v", err)}}}, nil, nil
	}
	if !strings.HasPrefix(absProject, wd+string(os.PathSeparator)) && absProject != filepath.Join(wd, "suga.yaml") {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "project_file must be within the current workspace"}}}, nil, nil
	}

	stackPath, err := s.builder.BuildProjectFromFile(clean, team, build.BuildOptions{})
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Build failed: %v", err)},
			},
		}, nil, nil
	}

	result := map[string]interface{}{
		"status":     "success",
		"stack_path": stackPath,
		"message":    fmt.Sprintf("Terraform generated successfully at %s", stackPath),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to marshal result: %v", err)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil, nil
}

// Resource handlers

func (s *Server) handleApplicationSchema(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	schemaString := schema.ApplicationJsonSchemaString()

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "suga://schema/application",
				MIMEType: "application/schema+json",
				Text:     schemaString,
			},
		},
	}, nil
}

func (s *Server) handleAppDevelopmentGuide(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "suga://guides/app-development",
				MIMEType: "text/markdown",
				Text:     appDevelopmentInstructions,
			},
		},
	}, nil
}

func (s *Server) handlePlatformDevelopmentGuide(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "suga://guides/platform-development",
				MIMEType: "text/markdown",
				Text:     platformDevelopmentInstructions,
			},
		},
	}, nil
}

func (s *Server) handlePluginLibraryDevelopmentGuide(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "suga://guides/plugin-library-development",
				MIMEType: "text/markdown",
				Text:     pluginLibraryDevelopmentInstructions,
			},
		},
	}, nil
}

// registerDocsProxyTools registers all tools from the docs proxy server
func (s *Server) registerDocsProxyTools() error {
	tools := s.docsProxy.GetTools()
	for _, tool := range tools {
		// Create a closure to capture the tool name for this iteration
		toolName := tool.Name

		// Register a proxy handler for this tool
		mcp.AddTool(s.mcpServer, tool, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]interface{}) (*mcp.CallToolResult, any, error) {
			result, err := s.docsProxy.CallTool(ctx, toolName, args)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Failed to call docs tool: %v", err)},
					},
				}, nil, nil
			}
			return result, nil, nil
		})
	}
	return nil
}
