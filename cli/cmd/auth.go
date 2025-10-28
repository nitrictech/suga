package cmd

import (
	"fmt"
	"os"

	"github.com/nitrictech/suga/cli/internal/style"
	"github.com/nitrictech/suga/cli/internal/style/icons"
	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/cli/pkg/app"
	"github.com/samber/do/v2"
	"github.com/spf13/cobra"
)

// NewLoginCmd creates the login command
func NewLoginCmd(injector do.Injector) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: fmt.Sprintf("Login to %s", version.ProductName),
		Long:  fmt.Sprintf("Login to the %s CLI.", version.ProductName),
		Run: func(cmd *cobra.Command, args []string) {
			if os.Getenv(AccessTokenEnvVar) != "" {
				fmt.Printf("\n%s Already authenticated using a Personal Access Token.\n", style.Yellow(icons.Info))
				fmt.Printf("  To use device authorization flow, unset the %s environment variable.\n", AccessTokenEnvVar)
				return
			}

			app, err := do.Invoke[*app.AuthApp](injector)
			if err != nil {
				cobra.CheckErr(err)
			}
			app.Login()
		},
	}
}

// NewLogoutCmd creates the logout command
func NewLogoutCmd(injector do.Injector) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: fmt.Sprintf("Logout from %s", version.ProductName),
		Long:  fmt.Sprintf("Logout from the %s CLI.", version.ProductName),
		Run: func(cmd *cobra.Command, args []string) {
			if os.Getenv(AccessTokenEnvVar) != "" {
				fmt.Printf("\n%s Cannot logout when authenticated using a Personal Access Token.\n", style.Yellow(icons.Info))
				fmt.Printf("  To stop using the token, unset the %s environment variable.\n", AccessTokenEnvVar)
				return
			}

			app, err := do.Invoke[*app.AuthApp](injector)
			if err != nil {
				cobra.CheckErr(err)
			}
			app.Logout()
		},
	}
}

// NewAccessTokenCmd creates the access token command
func NewAccessTokenCmd(injector do.Injector) *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "access-token",
		Short: "Get access token",
		Long:  `Get the current access token.`,
		Run: func(cmd *cobra.Command, args []string) {
			app, err := do.Invoke[*app.AuthApp](injector)
			if err != nil {
				cobra.CheckErr(err)
			}
			app.AccessToken(refresh)
		},
	}

	cmd.Flags().BoolVarP(&refresh, "refresh", "r", false, "Retrieve a new access token, ignoring any cached tokens")

	return cmd
}
