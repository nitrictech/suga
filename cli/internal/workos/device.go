package workos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nitrictech/suga/cli/internal/style"
	"github.com/nitrictech/suga/cli/internal/style/icons"
	"github.com/nitrictech/suga/cli/internal/workos/http"
	"github.com/pkg/browser"
)

// DeviceAuthResponse represents the device authorization response from backend
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// DeviceTokenResponse represents the token response from backend
type DeviceTokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	User         http.User `json:"user"`
}

// performDeviceAuth performs the WorkOS device authorization flow via backend proxy
func (a *WorkOSAuth) performDeviceAuth() error {
	deviceResp, err := a.httpClient.RequestDeviceAuthorization()
	if err != nil {
		return fmt.Errorf("failed to request device authorization: %w", err)
	}

	fmt.Printf("\nYour code is: %s\n", style.Bold(style.Yellow(deviceResp.UserCode)))

	err = browser.OpenURL(deviceResp.VerificationURIComplete)
	if err != nil {
		fmt.Printf("\nPlease visit: %s\n", style.Cyan(deviceResp.VerificationURI))
		fmt.Println("and enter the code above to login")
	} else {
		fmt.Printf("\nOpening your browser to the login page, please use the code shown above to confirm the login.\n")
	}

	fmt.Println("\nWaiting for authentication...")

	timeout := time.Duration(deviceResp.ExpiresIn) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Enforce a minimum polling interval to prevent aggressive polling
	const minInterval = 5 * time.Second
	pollInterval := time.Duration(deviceResp.Interval) * time.Second
	if pollInterval <= 0 || pollInterval < minInterval {
		pollInterval = minInterval
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("device authorization timed out")
		default:
			select {
			case <-ctx.Done():
				return fmt.Errorf("device authorization timed out")
			case <-time.After(time.Until(time.Now().Add(pollInterval))):
				// Continue to poll
			}

			// Create a context with timeout for this specific request
			reqCtx, reqCancel := context.WithTimeout(ctx, 10*time.Second)
			tokenResp, err := a.httpClient.PollDeviceTokenWithContext(reqCtx, deviceResp.DeviceCode)
			reqCancel()

			if err == nil {
				err := a.tokenStore.SaveTokens(&Tokens{
					AccessToken:  tokenResp.AccessToken,
					RefreshToken: tokenResp.RefreshToken,
					User:         &tokenResp.User,
				})
				if err != nil {
					return fmt.Errorf("failed to save tokens: %w", err)
				}
				return nil
			}

			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("%s Request timed out, retrying...\n", style.Yellow(icons.Warning))
				continue
			}

			if errors.Is(err, context.Canceled) {
				return fmt.Errorf("device authorization timed out")
			}

			errMsg := err.Error()
			switch {
			case containsError(errMsg, "authorization_pending"):
				// Auth still pending, continue polling at the same interval
			case containsError(errMsg, "slow_down"):
				pollInterval += (1 * time.Second)
			case containsError(errMsg, "expired_token"):
				return fmt.Errorf("device code expired, please try again")
			case containsError(errMsg, "access_denied"):
				return fmt.Errorf("authentication was denied")
			default:
				return fmt.Errorf("failed to poll for token: %w", err)
			}
		}
	}
}

// containsError checks if the error message contains a specific error code
func containsError(errMsg, errorCode string) bool {
	// Match common wrappers but default to substring containment
	if strings.Contains(errMsg, fmt.Sprintf("\"%s\"", errorCode)) {
		return true
	}
	if strings.Contains(errMsg, fmt.Sprintf("error: %s", errorCode)) {
		return true
	}
	return strings.Contains(errMsg, errorCode)
}
