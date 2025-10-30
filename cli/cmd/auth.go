package cmd

import (
	"fmt"
	"os"

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
				cobra.CheckErr(fmt.Errorf("currently using a personal access token via %s environment variable\nUnset %s to login manually", AccessTokenEnvVar, AccessTokenEnvVar))
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
				cobra.CheckErr(fmt.Errorf("currently using a personal access token via %s environment variable\nUnset %s to logout", AccessTokenEnvVar, AccessTokenEnvVar))
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
			if token := os.Getenv(AccessTokenEnvVar); token != "" {
				if refresh {
					cobra.CheckErr(fmt.Errorf("cannot refresh a personal access token provided via %s\nThe token is static and controlled externally", AccessTokenEnvVar))
					return
				}
				fmt.Println(token)
				return
			}
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
