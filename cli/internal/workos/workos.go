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
	tokens     *Tokens
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
	tokens, _ := tokenStore.GetTokens()

	return &WorkOSAuth{tokenStore: tokenStore, httpClient: httpClient, tokens: tokens}, nil
}

func (a *WorkOSAuth) Login() (*http.User, error) {
	// If we have existing tokens, try to validate/refresh them first
	if a.tokens != nil {
		// Attempt to refresh the token to verify it's still valid
		err := a.refreshToken()
		if err == nil {
			// Token is valid and refreshed successfully
			return a.tokens.User, nil
		}
		
		// Token refresh failed - clear the invalid tokens and proceed with new login
		a.tokens = nil
		_ = a.tokenStore.Clear() // Clear stored tokens since they're invalid
	}

	// Perform new device authentication
	err := a.performDeviceAuth()
	if err != nil {
		return nil, err
	}

	return a.tokens.User, nil
}

func (a *WorkOSAuth) GetAccessToken(forceRefresh bool) (string, error) {
	if a.tokens == nil {
		tokens, err := a.tokenStore.GetTokens()
		if err != nil {
			return "", fmt.Errorf("no stored tokens found, please login: %w", err)
		}
		a.tokens = tokens
	}

	if forceRefresh {
		if err := a.refreshToken(); err != nil {
			return "", fmt.Errorf("token refresh failed: %w", err)
		}
	}

	// Since we're proxying through the backend, we don't validate the JWT here
	// The backend will handle token validation
	// Just return the access token
	return a.tokens.AccessToken, nil
}

func (a *WorkOSAuth) refreshToken() error {
	if a.tokens.RefreshToken == "" {
		return fmt.Errorf("%w: no refresh token", ErrUnauthenticated)
	}

	// Store existing user data before refresh
	existingUser := a.tokens.User

	workosToken, err := a.httpClient.AuthenticateWithRefreshToken(a.tokens.RefreshToken, nil)
	if err != nil {
		return err
	}

	a.tokens = &Tokens{
		AccessToken:  workosToken.AccessToken,
		RefreshToken: workosToken.RefreshToken,
		User:         existingUser, // Preserve existing user data since refresh doesn't return it
	}

	err = a.tokenStore.SaveTokens(a.tokens)
	if err != nil {
		return err
	}

	return nil
}

func (a *WorkOSAuth) RefreshTokenForOrganization(organizationId string) error {
	if a.tokens == nil {
		tokens, err := a.tokenStore.GetTokens()
		if err != nil {
			return fmt.Errorf("no stored tokens found, please login: %w", err)
		}
		a.tokens = tokens
	}

	if a.tokens.RefreshToken == "" {
		return fmt.Errorf("%w: no refresh token available", ErrUnauthenticated)
	}

	// Store existing user data before refresh
	existingUser := a.tokens.User

	// Use organization-scoped refresh token
	workosToken, err := a.httpClient.AuthenticateWithRefreshToken(a.tokens.RefreshToken, &organizationId)
	if err != nil {
		return err
	}

	a.tokens = &Tokens{
		AccessToken:  workosToken.AccessToken,
		RefreshToken: workosToken.RefreshToken,
		User:         existingUser, // Preserve existing user data since refresh doesn't return it
	}

	err = a.tokenStore.SaveTokens(a.tokens)
	if err != nil {
		return err
	}

	return nil
}

func (a *WorkOSAuth) Logout() error {
	a.tokens = nil
	return a.tokenStore.Clear()
}
