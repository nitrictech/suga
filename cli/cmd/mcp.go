package cmd

import (
	"github.com/nitrictech/suga/cli/pkg/app"
	"github.com/samber/do/v2"
	"github.com/spf13/cobra"
)

// NewMcpCmd creates the mcp command
func NewMcpCmd(injector do.Injector) *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the Suga MCP (Model Context Protocol) server",
		Long: `Start the Suga MCP server that provides access to Suga platform APIs
through the Model Context Protocol. This allows AI assistants to interact
with your Suga templates, platforms, and build manifests.

The server uses stdio transport and requires authentication via 'suga login'.`,
		Run: func(cmd *cobra.Command, args []string) {
			app, err := do.Invoke[*app.SugaApp](injector)
			if err != nil {
				cobra.CheckErr(err)
			}
			cobra.CheckErr(app.MCP())
		},
	}

	return mcpCmd
}
