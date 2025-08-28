package workos

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/nitrictech/suga/cli/internal/config"
	"github.com/nitrictech/suga/cli/internal/workos/http"
	"github.com/samber/do/v2"
)

var (
	ErrNotFound        = errors.New("no token found")
	ErrUnauthenticated = errors.New("unauthenticated")
)

type TokenStore interface {
	// GetTokens returns the tokens from the store, or nil if no tokens are found
	GetTokens() (*Tokens, error)
	// SaveTokens saves the tokens to the store
	SaveTokens(*Tokens) error
	// Clear clears the tokens from the store
	Clear() error
}

type Tokens struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         *http.User `json:"user"`
}

type WorkOSAuth struct {
	tokenStore TokenStore
	httpClient *http.HttpClient
}

func NewWorkOSAuth(inj do.Injector) (*WorkOSAuth, error) {
	config := do.MustInvoke[*config.Config](inj)
	sugaServerUrl := config.GetSugaServerUrl()

	if sugaServerUrl.Host == "" || sugaServerUrl.Scheme == "" {
		return nil, fmt.Errorf("invalid Suga server URL: missing scheme or host")
	}

	opts := []http.ClientOption{
		http.WithHostname(sugaServerUrl.Hostname()),
		http.WithScheme(sugaServerUrl.Scheme),
	}
	if p := sugaServerUrl.Port(); p != "" {
		if pn, err := strconv.Atoi(p); err == nil {
			opts = append(opts, http.WithPort(pn))
		}
	}
	httpClient := http.NewHttpClient("", opts...)

	tokenStore := do.MustInvokeAs[TokenStore](inj)

	return &WorkOSAuth{tokenStore: tokenStore, httpClient: httpClient}, nil
}

func (a *WorkOSAuth) Login() (*http.User, error) {
	tokens, _ := a.tokenStore.GetTokens()

	if tokens != nil {
		if err := a.RefreshToken(RefreshTokenOptions{}); err == nil {
			return tokens.User, nil
		}

		_ = a.tokenStore.Clear()
	}

	err := a.performDeviceAuth()
	if err != nil {
		return nil, err
	}

	tokens, err = a.tokenStore.GetTokens()
	if err != nil {
		return nil, err
	}

	return tokens.User, nil
}

func (a *WorkOSAuth) GetAccessToken(forceRefresh bool) (string, error) {
	tokens, err := a.tokenStore.GetTokens()
	if err != nil {
		return "", fmt.Errorf("no stored tokens found, please login: %w", err)
	}

	if forceRefresh {
		if err := a.RefreshToken(RefreshTokenOptions{}); err != nil {
			return "", fmt.Errorf("token refresh failed: %w", err)
		}
	}

	// Since we're proxying through the backend, we don't validate the JWT here
	// The backend will handle token validation
	// Just return the access token
	return tokens.AccessToken, nil
}

type RefreshTokenOptions struct {
	OrganizationID string
}

func (a *WorkOSAuth) RefreshToken(options RefreshTokenOptions) error {
	tokens, err := a.tokenStore.GetTokens()
	if err != nil {
		return fmt.Errorf("no stored tokens found, please login: %w", err)
	}

	if tokens.RefreshToken == "" {
		return fmt.Errorf("%w: no refresh token", ErrUnauthenticated)
	}

	workosToken, err := a.httpClient.AuthenticateWithRefreshToken(tokens.RefreshToken, http.AuthenticatedWithRefreshTokenOptions{
		OrganizationID: options.OrganizationID,
	})
	if err != nil {
		return err
	}

	tokens.AccessToken = workosToken.AccessToken
	tokens.RefreshToken = workosToken.RefreshToken

	err = a.tokenStore.SaveTokens(tokens)
	if err != nil {
		return err
	}

	return nil
}

func (a *WorkOSAuth) Logout() error {
	return a.tokenStore.Clear()
}
