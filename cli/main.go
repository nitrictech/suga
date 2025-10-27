package main

import (
	"os"

	"github.com/nitrictech/suga/cli/cmd"
	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/auth"
	"github.com/nitrictech/suga/cli/internal/build"
	"github.com/nitrictech/suga/cli/internal/config"
	"github.com/nitrictech/suga/cli/internal/workos"
	"github.com/nitrictech/suga/cli/pkg/app"
	"github.com/samber/do/v2"
	"github.com/spf13/afero"
)

func createTokenStore(inj do.Injector) (auth.TokenStore, error) {
	cfg := do.MustInvoke[*config.Config](inj)
	apiUrl := cfg.GetSugaServerUrl()

	homeConfigPath, err := config.HomeConfigPath()
	if err != nil {
		return nil, err
	}

	store, err := auth.NewTokenStore("suga.cli", apiUrl.String(), homeConfigPath)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func main() {
	injector := do.New()

	do.Provide(injector, createTokenStore)
	do.Provide(injector, api.NewSugaApiClient)
	do.Provide(injector, workos.NewWorkOSAuth)
	do.Provide(injector, app.NewSugaApp)
	do.Provide(injector, app.NewAuthApp)
	do.Provide(injector, build.NewBuilderService)
	do.ProvideValue(injector, afero.NewOsFs())

	rootCmd := cmd.NewRootCmd(injector)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
