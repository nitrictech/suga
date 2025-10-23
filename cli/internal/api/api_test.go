package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type mockTokenProvider struct {
	token         string
	refreshCalled bool
	err           error
}

func (m *mockTokenProvider) GetAccessToken(forceRefresh bool) (string, error) {
	if forceRefresh {
		m.refreshCalled = true
	}
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func TestSugaApiClient_WithCallbackTokenProvider(t *testing.T) {
	tests := []struct {
		name            string
		token           string
		serverResponse  int
		expectAuthHeader string
		description     string
	}{
		{
			name:            "uses token from callback provider",
			token:           "callback-test-token",
			serverResponse:  200,
			expectAuthHeader: "Bearer callback-test-token",
			description:     "should send the callback token in Authorization header",
		},
		{
			name:            "sends bearer with empty token",
			token:           "",
			serverResponse:  200,
			expectAuthHeader: "Bearer",
			description:     "should send Bearer with empty token when token is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedAuthHeader string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedAuthHeader = r.Header.Get("Authorization")
				w.WriteHeader(tt.serverResponse)
			}))
			defer server.Close()

			serverURL, _ := url.Parse(server.URL)
			mockProvider := &mockTokenProvider{token: tt.token}

			client := &SugaApiClient{
				apiUrl:               serverURL,
				tokenProvider:        mockProvider,
				httpClient:           http.DefaultClient,
				publicTemplatesTeams: []string{},
			}

			_, err := client.get("/test", true)
			if err != nil {
				t.Fatalf("get() unexpected error = %v", err)
			}

			if receivedAuthHeader != tt.expectAuthHeader {
				t.Errorf("Authorization header = %v, want %v", receivedAuthHeader, tt.expectAuthHeader)
			}
		})
	}
}

func TestSugaApiClient_TokenRefreshCalledOn401(t *testing.T) {
	mockProvider := &mockTokenProvider{token: "static-token"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	client := &SugaApiClient{
		apiUrl:               serverURL,
		tokenProvider:        mockProvider,
		httpClient:           http.DefaultClient,
		publicTemplatesTeams: []string{},
	}

	_, _ = client.get("/test", true)

	if !mockProvider.refreshCalled {
		t.Error("GetAccessToken(forceRefresh=true) should be called to retry with refreshed token on 401")
	}
}

func TestSugaApiClient_WithoutTokenProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	client := &SugaApiClient{
		apiUrl:               serverURL,
		tokenProvider:        nil,
		httpClient:           http.DefaultClient,
		publicTemplatesTeams: []string{},
	}

	_, err := client.get("/test", true)
	if err == nil {
		t.Error("Expected error when tokenProvider is nil and requiresAuth is true")
	}

	if err != nil && !strings.Contains(fmt.Sprintf("%v", err), "no token provider") {
		t.Errorf("Expected 'no token provider' error, got: %v", err)
	}
}

func TestSugaApiClient_RequestWithoutAuth(t *testing.T) {
	var receivedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	client := &SugaApiClient{
		apiUrl:               serverURL,
		tokenProvider:        nil,
		httpClient:           http.DefaultClient,
		publicTemplatesTeams: []string{},
	}

	_, err := client.get("/test", false)
	if err != nil {
		t.Fatalf("get() unexpected error = %v", err)
	}

	if receivedAuthHeader != "" {
		t.Errorf("Authorization header should be empty for unauthenticated requests, got: %v", receivedAuthHeader)
	}
}
