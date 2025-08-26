package workos

import (
	"context"
	"fmt"
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
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	User         http.User  `json:"user"`
}

// performDeviceAuth performs the WorkOS device authorization flow via backend proxy
func (a *WorkOSAuth) performDeviceAuth() error {
	// Step 1: Request device authorization from backend
	fmt.Printf("\n%s Requesting device authorization...\n", style.Purple(icons.Lightning))

	deviceResp, err := a.httpClient.RequestDeviceAuthorization()
	if err != nil {
		return fmt.Errorf("failed to request device authorization: %w", err)
	}

	// Step 2: Display verification URL and open browser
	fmt.Printf("\n%s Opening browser to complete authentication...\n", style.Gray(icons.Arrow))
	
	// Open browser to verification URL with pre-filled code
	err = browser.OpenURL(deviceResp.VerificationURIComplete)
	if err != nil {
		// If browser fails to open, show manual instructions
		fmt.Printf("%s Failed to open browser: %s\n", style.Yellow(icons.Warning), err)
		fmt.Printf("\n%s Please visit: %s\n", style.Green(icons.Globe), style.Cyan(deviceResp.VerificationURI))
		fmt.Printf("%s And enter this code: %s\n", style.Green(icons.Key), style.Bold(style.Yellow(deviceResp.UserCode)))
	} else {
		// Browser opened successfully - show code for verification
		fmt.Printf("%s Browser opened with authentication page\n", style.Green(icons.Check))
		fmt.Printf("%s Your verification code is: %s\n", style.Gray(icons.Info), style.Bold(style.Yellow(deviceResp.UserCode)))
	}

	// Step 3: Poll for token completion
	fmt.Printf("\n%s Waiting for authentication...\n", style.Purple(icons.Clock))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deviceResp.ExpiresIn)*time.Second)
	defer cancel()

	pollInterval := time.Duration(deviceResp.Interval) * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("device authorization timed out")
		case <-ticker.C:
			tokenResp, err := a.httpClient.PollDeviceToken(deviceResp.DeviceCode)
			if err != nil {
				// Check for specific error messages from backend
				errMsg := err.Error()
				switch {
				case containsError(errMsg, "authorization_pending"):
					// Continue polling
					continue
				case containsError(errMsg, "slow_down"):
					// Increase polling interval
					ticker.Reset(pollInterval * 2)
					continue
				case containsError(errMsg, "expired_token"):
					return fmt.Errorf("device code expired, please try again")
				case containsError(errMsg, "access_denied"):
					return fmt.Errorf("authentication was denied")
				default:
					return fmt.Errorf("failed to poll for token: %w", err)
				}
			}

			// Success! Save tokens
			a.tokens = &Tokens{
				AccessToken:  tokenResp.AccessToken,
				RefreshToken: tokenResp.RefreshToken,
				User:         &tokenResp.User,
			}

			err = a.tokenStore.SaveTokens(a.tokens)
			if err != nil {
				return fmt.Errorf("failed to save tokens: %w", err)
			}

			return nil
		}
	}
}

// containsError checks if the error message contains a specific error code
func containsError(errMsg, errorCode string) bool {
	return fmt.Sprintf("\"%s\"", errorCode) == errMsg || 
		   fmt.Sprintf("error: %s", errorCode) == errMsg ||
		   errorCode == errMsg
}