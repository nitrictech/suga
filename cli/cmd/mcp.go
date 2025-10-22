package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/build"
	"github.com/nitrictech/suga/cli/internal/config"
	"github.com/nitrictech/suga/cli/internal/mcp"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get dependencies from injector
			apiClient := do.MustInvoke[*api.SugaApiClient](injector)
			cfg := do.MustInvoke[*config.Config](injector)
			builder := do.MustInvoke[*build.BuilderService](injector)

			// Create MCP server
			server, err := mcp.NewServer(apiClient, cfg, builder)
			if err != nil {
				return fmt.Errorf("failed to create MCP server: %w", err)
			}

			// Setup context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle shutdown signals
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Run server in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Run(ctx)
			}()

			// Wait for either error or shutdown signal
			select {
			case err := <-errChan:
				if err != nil {
					return fmt.Errorf("MCP server error: %w", err)
				}
			case <-sigChan:
				cancel()
				<-errChan // Wait for server to shutdown
			}

			return nil
		},
	}

	return mcpCmd
}
