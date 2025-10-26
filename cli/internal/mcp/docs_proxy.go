package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const docsServerURL = "https://docs.addsuga.com/mcp"

// DocsProxy manages a connection to the Suga docs MCP server and proxies requests
type DocsProxy struct {
	client  *mcp.Client
	session *mcp.ClientSession
	mu      sync.RWMutex

	// Cache of docs server capabilities
	tools []*mcp.Tool
}

// NewDocsProxy creates a new proxy to the Suga docs MCP server
func NewDocsProxy() *DocsProxy {
	return &DocsProxy{}
}

// Connect establishes a connection to the docs MCP server
func (d *DocsProxy) Connect(ctx context.Context) error {
	// Create MCP client
	impl := &mcp.Implementation{
		Name:    "suga-cli-docs-client",
		Version: "1.0.0",
	}

	client := mcp.NewClient(impl, nil)

	// Create streamable HTTP transport to docs server
	transport := &mcp.StreamableClientTransport{
		Endpoint:   docsServerURL,
		HTTPClient: &http.Client{},
	}

	// Connect to docs server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to docs server: %w", err)
	}

	d.mu.Lock()
	d.client = client
	d.session = session
	d.mu.Unlock()

	// Fetch available tools from docs server
	if err := d.fetchCapabilities(ctx); err != nil {
		log.Printf("Warning: failed to fetch docs server capabilities: %v", err)
	}

	return nil
}

// fetchCapabilities queries the docs server for available tools
func (d *DocsProxy) fetchCapabilities(ctx context.Context) error {
	d.mu.RLock()
	session := d.session
	d.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("not connected to docs server")
	}

	// List tools from docs server
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list docs server tools: %w", err)
	}

	// Fix JSON Schema versions in tools - MCP SDK only supports draft 2020-12
	for _, tool := range toolsResult.Tools {
		if tool.InputSchema != nil {
			if schema, ok := tool.InputSchema.(map[string]interface{}); ok {
				d.fixSchemaVersion(schema)
			}
		}
	}

	d.mu.Lock()
	d.tools = toolsResult.Tools
	d.mu.Unlock()

	return nil
}

// fixSchemaVersion updates JSON Schema $schema to the version supported by MCP SDK
func (d *DocsProxy) fixSchemaVersion(schema map[string]interface{}) {
	if schema == nil {
		return
	}

	// Check if there's a $schema field with draft-07 and update it to 2020-12
	if schemaVersion, ok := schema["$schema"].(string); ok {
		if strings.Contains(schemaVersion, "draft-07") || strings.Contains(schemaVersion, "draft/07") {
			schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		}
	}
}

// GetTools returns the list of tools available from the docs server
func (d *DocsProxy) GetTools() []*mcp.Tool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.tools
}

// CallTool proxies a tool call to the docs server
func (d *DocsProxy) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	d.mu.RLock()
	session := d.session
	d.mu.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("not connected to docs server")
	}

	params := &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("docs server tool call failed: %w", err)
	}

	return result, nil
}

// Close closes the connection to the docs server
func (d *DocsProxy) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.session != nil {
		return d.session.Close()
	}

	return nil
}
