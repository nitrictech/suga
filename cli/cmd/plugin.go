package cmd

import (
	"fmt"
	"os"

	"github.com/nitrictech/suga/cli/internal/pluginserver"
	"github.com/nitrictech/suga/cli/internal/style"
	"github.com/samber/do/v2"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewPluginCmd creates the plugin command with subcommands
func NewPluginCmd(injector do.Injector) *cobra.Command {
	pluginCmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin development tools",
		Long:  "Tools for developing and testing Suga plugins locally",
	}

	pluginCmd.AddCommand(NewPluginServeCmd(injector))
	return pluginCmd
}

// NewPluginServeCmd creates the plugin serve command
func NewPluginServeCmd(injector do.Injector) *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve plugins from local filesystem for development",
		Long: `Start a local HTTP server to serve plugin manifests and Go modules for development.

This allows plugin developers to test their plugins locally without publishing them.
The server implements:
  - Plugin manifest API (compatible with Suga API format)
  - Go module proxy (GOPROXY protocol)

Example usage:
  # Serve plugins from current directory
  suga plugin serve

Plugin directory structure:
  {plugin-name}/manifest.yaml

Example platform.yaml configuration:
  libraries:
    myteam/myplugins: http://localhost:9000

Then use plugins from that library in your resource blueprints.
The team, library, and version are specified in the platform.yaml, not in the directory structure.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := afero.NewOsFs()

			// Use current directory
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// Create and start server (automatically discovers plugins and modules)
			server, err := pluginserver.NewPluginServer(fs, cwd)
			if err != nil {
				return fmt.Errorf("failed to create plugin server: %w", err)
			}

			// Get discovered plugins
			plugins := server.GetPlugins()

			// Output server info
			fmt.Printf("\n%s\n", style.Bold("Suga Plugin Development Server"))
			fmt.Printf("Listening on: %s\n\n", style.Teal(fmt.Sprintf("http://localhost:%d", port)))

			// Show discovered plugins
			if len(plugins) > 0 {
				fmt.Printf("%s\n", style.Bold("Discovered Plugins:"))
				for _, plugin := range plugins {
					fmt.Printf("  %s %s\n", style.Green("✓"), plugin.String())
				}
			} else {
				fmt.Printf("%s\n", style.Yellow("No plugins discovered"))
				fmt.Printf("\n%s\n", style.Gray("Expected directory structure:"))
				fmt.Printf("  %s\n", style.Gray("{plugin-name}/manifest.yaml"))
			}

			fmt.Printf("\n%s\n", style.Bold("Configuration:"))
			fmt.Printf("Add to your %s:\n", style.Cyan("platform.yaml"))
			fmt.Printf("  %s\n", style.Gray("libraries:"))
			fmt.Printf("    %s\n", style.Gray(fmt.Sprintf("myteam/myplugins: http://localhost:%d", port)))
			fmt.Printf("\n%s\n\n", style.Gray("Press Ctrl+C to stop"))

			addr := fmt.Sprintf(":%d", port)
			return server.Start(addr)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 9000, "Port to listen on")

	return cmd
}
