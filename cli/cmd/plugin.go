package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nitrictech/suga/cli/internal/pluginserver"
	"github.com/nitrictech/suga/cli/internal/version"
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
	var (
		port        int
		pluginPaths []string
		modulePaths []string
	)

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

  # Serve plugins from specific directory
  suga plugin serve --plugin-path ./my-plugins

  # Serve multiple plugin directories
  suga plugin serve --plugin-path ./plugins1 --plugin-path ./plugins2

  # Also serve Go modules for runtime dependencies
  suga plugin serve --plugin-path ./plugins --module-path ./go-modules

Plugin directory structure:
  {plugin-path}/{plugin-name}/manifest.yaml

Example platform.yaml configuration:
  libraries:
    myteam/myplugins: http://localhost:8080

Then use plugins from that library in your resource blueprints.
The team, library, and version are specified in the platform.yaml, not in the directory structure.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fs := afero.NewOsFs()

			// Default to current directory if no paths specified
			if len(pluginPaths) == 0 {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				pluginPaths = []string{cwd}
			}

			// Convert relative paths to absolute
			for i, path := range pluginPaths {
				if !filepath.IsAbs(path) {
					absPath, err := filepath.Abs(path)
					if err != nil {
						return fmt.Errorf("failed to resolve path %s: %w", path, err)
					}
					pluginPaths[i] = absPath
				}
			}

			for i, path := range modulePaths {
				if !filepath.IsAbs(path) {
					absPath, err := filepath.Abs(path)
					if err != nil {
						return fmt.Errorf("failed to resolve path %s: %w", path, err)
					}
					modulePaths[i] = absPath
				}
			}

			// Create and start server
			server := pluginserver.NewPluginServer(fs, pluginserver.ServerConfig{
				PluginPaths: pluginPaths,
				ModulePaths: modulePaths,
			})

			// Discover and list available plugins
			plugins, err := server.DiscoverPlugins()
			if err != nil {
				return fmt.Errorf("failed to discover plugins: %w", err)
			}

			fmt.Printf("\n%s Plugin Server\n", version.ProductName)
			fmt.Println("==================")
			fmt.Printf("Listening on: http://localhost:%d\n", port)
			fmt.Println()

			if len(plugins) > 0 {
				fmt.Printf("Discovered %d plugin(s):\n", len(plugins))
				for _, plugin := range plugins {
					fmt.Printf("  - %s\n", plugin.String())
				}
			} else {
				fmt.Println("No plugins discovered")
				fmt.Println()
				fmt.Println("Expected directory structure:")
				fmt.Println("  {plugin-path}/{plugin-name}/manifest.yaml")
				fmt.Println()
				fmt.Println("Example:")
				fmt.Println("  ./plugins/my-service/manifest.yaml")
			}

			fmt.Println()
			fmt.Println("Configuration:")
			fmt.Printf("  In your platform.yaml, set:\n")
			fmt.Printf("    libraries:\n")
			fmt.Printf("      myteam/myplugins: http://localhost:%d\n", port)
			fmt.Println()
			fmt.Println("Press Ctrl+C to stop")
			fmt.Println()

			addr := fmt.Sprintf(":%d", port)
			return server.Start(addr)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	cmd.Flags().StringSliceVar(&pluginPaths, "plugin-path", nil, "Path(s) to search for plugin manifests (can be specified multiple times)")
	cmd.Flags().StringSliceVar(&modulePaths, "module-path", nil, "Path(s) to search for Go modules (can be specified multiple times)")

	return cmd
}
