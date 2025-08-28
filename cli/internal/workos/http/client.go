package http

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"path"
)

const DEFAULT_HOSTNAME = "api.workos.com"

// Errors
type CodeExchangeError struct {
	Message string
}

func (e *CodeExchangeError) Error() string {
	return e.Message
}

type RefreshError struct {
	Message string
}

func (e *RefreshError) Error() string {
	return e.Message
}

// Authentication response types
type AuthenticationResponseRaw struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type User struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	ProfilePictureURL string `json:"profile_picture_url"`
	LastSignInAt      string `json:"last_sign_in_at"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	ExternalID        string `json:"external_id"`
}

type AuthenticationResponse struct {
	AccessToken  string
	RefreshToken string
	User         User
}

type JWKsResponse struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty     string   `json:"kty"`
	Kid     string   `json:"kid"`
	Use     string   `json:"use"`
	Alg     string   `json:"alg"`
	N       string   `json:"n"`
	E       string   `json:"e"`
	X5c     []string `json:"x5c"`
	X5tS256 string   `json:"x5t#S256"`
}

// Authorization URL options
type GetAuthorizationUrlOptions struct {
	ConnectionID        string
	Context             string
	DomainHint          string
	LoginHint           string
	OrganizationID      string
	Provider            string
	RedirectURI         string
	State               string
	ScreenHint          string
	PasswordResetToken  string
	InvitationToken     string
	CodeChallenge       string
	CodeChallengeMethod string
}

// HttpClient represents the WorkOS HTTP client
type HttpClient struct {
	baseURL  string
	clientID string
	client   *http.Client
}

// NewHttpClient creates a new WorkOS HTTP client
func NewHttpClient(clientID string, options ...ClientOption) *HttpClient {
	config := &clientConfig{
		hostname: DEFAULT_HOSTNAME,
		scheme:   "https",
	}

	for _, option := range options {
		option(config)
	}

	baseURL := &url.URL{
		Scheme: config.scheme,
		Host:   config.hostname,
	}

	if config.port != 0 {
		baseURL.Host = fmt.Sprintf("%s:%d", config.hostname, config.port)
	}

	return &HttpClient{
		baseURL:  baseURL.String(),
		clientID: clientID,
		client:   &http.Client{},
	}
}

type clientConfig struct {
	hostname string
	port     int
	scheme   string
}

type ClientOption func(*clientConfig)

func WithHostname(hostname string) ClientOption {
	return func(c *clientConfig) {
		c.hostname = hostname
	}
}

func WithPort(port int) ClientOption {
	return func(c *clientConfig) {
		c.port = port
	}
}

func WithScheme(scheme string) ClientOption {
	return func(c *clientConfig) {
		c.scheme = scheme
	}
}

func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode the modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}

	// Decode the exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}

	// Create the RSA public key
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

// GetRSAPublicKey gets the RSA public key for a given JWT kid
func (h *HttpClient) GetRSAPublicKey(kid string) (*rsa.PublicKey, error) {
	jwk, err := h.GetJWK(kid)
	if err != nil {
		return nil, err
	}
	return jwkToRSAPublicKey(jwk)
}

func (h *HttpClient) GetJWK(kid string) (JWK, error) {
	jwks, err := h.GetJWKs()
	if err != nil {
		return JWK{}, err
	}

	for _, jwk := range jwks {
		if jwk.Kid == kid {
			return jwk, nil
		}
	}

	return JWK{}, fmt.Errorf("JWK not found")
}

func (h *HttpClient) GetJWKs() ([]JWK, error) {
	jwkPath := path.Join("sso/jwks", h.clientID)

	response, err := h.get(jwkPath)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var jwksResponse JWKsResponse
	if err := json.Unmarshal(body, &jwksResponse); err != nil {
		return nil, err
	}

	return jwksResponse.Keys, nil
}

// AuthenticateWithCode authenticates using an authorization code
func (h *HttpClient) AuthenticateWithCode(code, codeVerifier string) (*AuthenticationResponse, error) {
	body := map[string]interface{}{
		"code":          code,
		"client_id":     h.clientID,
		"grant_type":    "authorization_code",
		"code_verifier": codeVerifier,
	}

	response, err := h.post("/user_management/authenticate", body)
	if err != nil {
		return nil, err
	}

	// read the body into a string
	resBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusOK {
		var data AuthenticationResponseRaw
		if err := json.Unmarshal(resBody, &data); err != nil {
			return nil, err
		}
		return deserializeAuthenticationResponse(data), nil
	}

	return nil, &CodeExchangeError{Message: fmt.Sprintf("Error authenticating with API, status: %d, body: %s", response.StatusCode, string(resBody))}
}

// post performs a POST request to the specified path
func (h *HttpClient) post(path string, body map[string]interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(h.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Join the base URL with the path safely
	fullURL := baseURL.JoinPath(path)

	req, err := http.NewRequest("POST", fullURL.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")

	return h.client.Do(req)
}

func (h *HttpClient) get(path string) (*http.Response, error) {
	baseURL, err := url.Parse(h.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	fullURL := baseURL.JoinPath(path)

	req, err := http.NewRequest("GET", fullURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")

	return h.client.Do(req)
}

type AuthenticatedWithRefreshTokenOptions struct {
	OrganizationID string `json:"organization_id,omitempty"`
}

// AuthenticateWithRefreshToken authenticates using a refresh token via backend proxy
func (h *HttpClient) AuthenticateWithRefreshToken(refreshToken string, options AuthenticatedWithRefreshTokenOptions) (*AuthenticationResponse, error) {
	body := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	if options.OrganizationID != "" {
		body["organization_id"] = options.OrganizationID
	}

	response, err := h.post("/auth/refresh", body)
	if err != nil {
		return nil, err
	}

	resBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusOK {
		var data AuthenticationResponseRaw
		if err := json.Unmarshal(resBody, &data); err != nil {
			return nil, err
		}
		return deserializeAuthenticationResponse(data), nil
	}

	return nil, &RefreshError{Message: fmt.Sprintf("Error refreshing token: status %d, body: %s", response.StatusCode, string(resBody))}
}

// GetAuthorizationUrl generates an authorization URL
func (h *HttpClient) GetAuthorizationUrl(options GetAuthorizationUrlOptions) (string, error) {
	if options.Provider == "" && options.ConnectionID == "" && options.OrganizationID == "" {
		return "", fmt.Errorf("incomplete arguments. need to specify either a 'connectionId', 'organizationId', or 'provider'")
	}

	if options.Provider != "" && options.Provider != "authkit" && options.ScreenHint != "" {
		return "", fmt.Errorf("'screenHint' is only supported for 'authkit' provider")
	}

	baseURL, err := url.Parse(h.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Join the base URL with the authorize path
	authURL := baseURL.JoinPath("user_management", "authorize")

	// Build query parameters
	params := url.Values{}

	if options.ConnectionID != "" {
		params.Set("connection_id", options.ConnectionID)
	}
	if options.Context != "" {
		params.Set("context", options.Context)
	}
	if options.OrganizationID != "" {
		params.Set("organization_id", options.OrganizationID)
	}
	if options.DomainHint != "" {
		params.Set("domain_hint", options.DomainHint)
	}
	if options.LoginHint != "" {
		params.Set("login_hint", options.LoginHint)
	}
	if options.Provider != "" {
		params.Set("provider", options.Provider)
	}
	params.Set("client_id", h.clientID)
	if options.RedirectURI != "" {
		params.Set("redirect_uri", options.RedirectURI)
	}
	params.Set("response_type", "code")
	if options.State != "" {
		params.Set("state", options.State)
	}
	if options.ScreenHint != "" {
		params.Set("screen_hint", options.ScreenHint)
	}
	if options.InvitationToken != "" {
		params.Set("invitation_token", options.InvitationToken)
	}
	if options.PasswordResetToken != "" {
		params.Set("password_reset_token", options.PasswordResetToken)
	}
	if options.CodeChallenge != "" {
		params.Set("code_challenge", options.CodeChallenge)
	}
	if options.CodeChallengeMethod != "" {
		params.Set("code_challenge_method", options.CodeChallengeMethod)
	}

	authURL.RawQuery = params.Encode()
	return authURL.String(), nil
}

// GetLogoutUrl generates a logout URL
func (h *HttpClient) GetLogoutUrl(sessionID, returnTo string) string {
	baseURL, err := url.Parse(h.baseURL)
	if err != nil {
		// If base URL is invalid, return a basic string (this shouldn't happen with proper initialization)
		return ""
	}

	// Join the base URL with the logout path
	logoutURL := baseURL.JoinPath("user_management", "sessions", "logout")

	// Build query parameters
	params := url.Values{}
	params.Set("session_id", sessionID)
	if returnTo != "" {
		params.Set("return_to", returnTo)
	}

	logoutURL.RawQuery = params.Encode()
	return logoutURL.String()
}

// deserializeAuthenticationResponse converts the raw response to the structured response
func deserializeAuthenticationResponse(raw AuthenticationResponseRaw) *AuthenticationResponse {
	return &AuthenticationResponse{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		User:         raw.User,
	}
}

// RequestDeviceAuthorization requests device authorization from backend
func (c *HttpClient) RequestDeviceAuthorization() (*DeviceAuthorizationResponse, error) {
	response, err := c.post("/auth/device", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to make device authorization request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read device authorization response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorization request failed with status %d: %s", response.StatusCode, string(body))
	}

	var deviceResp DeviceAuthorizationResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device authorization response: %w", err)
	}

	return &deviceResp, nil
}

// PollDeviceTokenWithContext polls for device token from backend with a context for cancellation/timeout
func (c *HttpClient) PollDeviceTokenWithContext(ctx context.Context, deviceCode string) (*DeviceTokenResponse, error) {
	reqBody := map[string]interface{}{"device_code": deviceCode}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Join the base URL with the path safely
	fullURL := baseURL.JoinPath("/auth/device/token")

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make device token request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read device token response: %w", err)
	}

	if response.StatusCode == http.StatusOK {
		var tokenResp DeviceTokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return nil, fmt.Errorf("failed to parse device token response: %w", err)
		}
		return &tokenResp, nil
	}

	// Handle error response from backend
	var errorResp map[string]string
	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errorCode, ok := errorResp["error"]; ok {
			return nil, errors.New(errorCode)
		}
	}

	return nil, fmt.Errorf("device token request failed with status %d: %s", response.StatusCode, string(body))
}

// DeviceAuthorizationResponse represents device authorization response
type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// DeviceTokenResponse represents device token response
type DeviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	User         User   `json:"user"`
}
