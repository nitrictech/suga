package api

import (
	"errors"
	"testing"
)

func TestCallbackTokenProvider_GetAccessToken(t *testing.T) {
	tests := []struct {
		name         string
		getTokenFn   func() (string, error)
		forceRefresh bool
		wantToken    string
		wantErr      bool
		description  string
	}{
		{
			name: "returns token from callback",
			getTokenFn: func() (string, error) {
				return "test-token-123", nil
			},
			forceRefresh: false,
			wantToken:    "test-token-123",
			wantErr:      false,
			description:  "should return the token from callback function",
		},
		{
			name: "returns token with force refresh",
			getTokenFn: func() (string, error) {
				return "test-token-456", nil
			},
			forceRefresh: true,
			wantToken:    "test-token-456",
			wantErr:      false,
			description:  "should return the token even when forceRefresh is true",
		},
		{
			name: "returns empty string from callback",
			getTokenFn: func() (string, error) {
				return "", nil
			},
			forceRefresh: false,
			wantToken:    "",
			wantErr:      false,
			description:  "should return empty string when callback returns empty",
		},
		{
			name: "returns error from callback",
			getTokenFn: func() (string, error) {
				return "", errors.New("token retrieval failed")
			},
			forceRefresh: false,
			wantToken:    "",
			wantErr:      true,
			description:  "should return error when callback returns error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewCallbackTokenProvider(tt.getTokenFn)
			got, err := provider.GetAccessToken(tt.forceRefresh)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.wantToken {
				t.Errorf("GetAccessToken() = %v, want %v", got, tt.wantToken)
			}
		})
	}
}

func TestCallbackTokenProvider_CallbackIsInvokedEachTime(t *testing.T) {
	callCount := 0
	provider := NewCallbackTokenProvider(func() (string, error) {
		callCount++
		return "token", nil
	})

	_, err := provider.GetAccessToken(false)
	if err != nil {
		t.Fatalf("GetAccessToken() first call unexpected error = %v", err)
	}

	_, err = provider.GetAccessToken(false)
	if err != nil {
		t.Fatalf("GetAccessToken() second call unexpected error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected callback to be invoked 2 times, got %d", callCount)
	}
}
