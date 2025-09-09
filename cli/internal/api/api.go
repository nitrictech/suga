package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/nitrictech/suga/cli/internal/config"
	"github.com/nitrictech/suga/cli/internal/utils"
	"github.com/pkg/errors"
	"github.com/samber/do/v2"
)

type TokenProvider interface {
	// GetAccessToken returns the access token for the user
	GetAccessToken(forceRefresh bool) (string, error)
}

type SugaApiClient struct {
	tokenProvider TokenProvider
	apiUrl        *url.URL
}

func NewSugaApiClient(injector do.Injector) (*SugaApiClient, error) {
	config, err := do.Invoke[*config.Config](injector)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	apiUrl := config.GetSugaServerUrl()

	tokenProvider, err := do.InvokeAs[TokenProvider](injector)
	if err != nil {
		return nil, fmt.Errorf("failed to get token provider: %w", err)
	}

	return &SugaApiClient{
		apiUrl:        apiUrl,
		tokenProvider: tokenProvider,
	}, nil
}

// doRequestWithRetry executes an HTTP request and retries once with a refreshed token on 401/403.
// Reuses req.Context() and req.GetBody (when available) to rebuild the body.
func (c *SugaApiClient) doRequestWithRetry(req *http.Request, requiresAuth bool) (*http.Response, error) {
	if requiresAuth {
		if c.tokenProvider == nil {
			return nil, errors.Wrap(ErrPreconditionFailed, "no token provider provided")
		}

		// First attempt with existing token
		token, err := c.tokenProvider.GetAccessToken(false)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnauthenticated, err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If we got a 401 or 403 and auth is required, try refreshing the token
	if requiresAuth && (resp.StatusCode == 401 || resp.StatusCode == 403) {
		resp.Body.Close() // Close the first response body

		// Force token refresh
		token, err := c.tokenProvider.GetAccessToken(true)
		if err != nil {
			return nil, fmt.Errorf("%w: token refresh failed: %v", ErrUnauthenticated, err)
		}

		// Clone the request for retry - use GetBody to recreate the body if available
		var bodyReader io.Reader
		var retryBodyRC io.ReadCloser
		// Cannot safely retry request with a consumed, non-rewindable body
		if req.Body != nil && req.GetBody == nil && req.ContentLength != 0 {
			return nil, fmt.Errorf("%w: cannot retry request with non-rewindable body", ErrUnauthenticated)
		}
		if req.GetBody != nil {
			var err2 error
			retryBodyRC, err2 = req.GetBody()
			if err2 != nil {
				return nil, err2
			}
			bodyReader = retryBodyRC
		}

		retryReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), bodyReader)
		if err != nil {
			if retryBodyRC != nil {
				_ = retryBodyRC.Close()
			}
			return nil, err
		}

		// Copy headers
		retryReq.Header = req.Header.Clone()
		
		// Update authorization header with new token
		retryReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		// Retry the request
		return http.DefaultClient.Do(retryReq)
	}

	return resp, nil
}

func (c *SugaApiClient) get(path string, requiresAuth bool) (*http.Response, error) {
	apiUrl, err := url.JoinPath(c.apiUrl.String(), path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	return c.doRequestWithRetry(req, requiresAuth)
}

func (c *SugaApiClient) post(path string, requiresAuth bool, body []byte) (*http.Response, error) {
	apiUrl, err := url.JoinPath(c.apiUrl.String(), path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiUrl, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-amz-content-sha256", utils.CalculateSHA256(body))

	return c.doRequestWithRetry(req, requiresAuth)
}
