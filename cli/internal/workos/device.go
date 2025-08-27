package workos

import (
	"context"
	"errors"
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
	currentInterval := pollInterval
	
	// Track when we can make the next request
	nextPollTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("device authorization timed out")
		default:
			// Wait until it's time for the next poll
			waitTime := time.Until(nextPollTime)
			if waitTime > 0 {
				select {
				case <-ctx.Done():
					return fmt.Errorf("device authorization timed out")
				case <-time.After(waitTime):
					// Continue to poll
				}
			}

			// Set the next poll time before making the request
			// This ensures consistent intervals regardless of request duration
			nextPollTime = time.Now().Add(currentInterval)

			// Create a context with timeout for this specific request
			reqCtx, reqCancel := context.WithTimeout(ctx, 10*time.Second)
			tokenResp, err := a.httpClient.PollDeviceTokenWithContext(reqCtx, deviceResp.DeviceCode)
			reqCancel()

			if err != nil {
				// Check if it's a context error (timeout or cancellation)
				if errors.Is(err, context.DeadlineExceeded) {
					// Request timed out, continue with next poll
					fmt.Printf("%s Request timed out, retrying...\n", style.Yellow(icons.Warning))
					continue
				}
				if errors.Is(err, context.Canceled) {
					// Overall timeout reached
					return fmt.Errorf("device authorization timed out")
				}

				// Check for specific error messages from backend
				errMsg := err.Error()
				switch {
				case containsError(errMsg, "authorization_pending"):
					// Continue polling at normal interval
					continue
				case containsError(errMsg, "slow_down"):
					// Server asked us to slow down - add 5 seconds to interval
					currentInterval = pollInterval + (5 * time.Second)
					// Adjust next poll time to respect the new interval
					nextPollTime = time.Now().Add(currentInterval)
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