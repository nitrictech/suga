package auth

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nitrictech/suga/cli/internal/keyring"
)

var (
	ErrNotFound        = errors.New("no token found")
	ErrUnauthenticated = errors.New("unauthenticated")
)

type User struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
}

type Auth interface {
	Login() (*User, error)
	GetAccessToken(forceRefresh bool) (string, error)
	Logout() error
}

type TokenStore interface {
	GetTokens() (*Tokens, error)
	SaveTokens(*Tokens) error
	Clear() error
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user,omitempty"`
}

type Store struct {
	service  string
	tokenKey string
	filePath string
}

func hashTokenKey(tokenKey string) string {
	hash := sha256.Sum256([]byte(tokenKey))
	return fmt.Sprintf("%x", hash)
}

func NewTokenStore(serviceName, tokenKey, configDir string) (*Store, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	if tokenKey == "" {
		return nil, fmt.Errorf("token key is required")
	}

	hashedTokenKey := hashTokenKey(tokenKey)

	filePath := filepath.Join(configDir, "tokens.json")

	return &Store{
		service:  serviceName,
		tokenKey: hashedTokenKey,
		filePath: filePath,
	}, nil
}

func (s *Store) GetTokens() (*Tokens, error) {
	token, keyringErr := keyring.Get(s.service, s.tokenKey)
	if keyringErr == nil {
		var tokens Tokens
		if err := json.Unmarshal([]byte(token), &tokens); err != nil {
			return nil, fmt.Errorf("failed to unmarshal token from keyring: %w", err)
		}
		return &tokens, nil
	}

	tokens, fileErr := s.getTokensFromFile()
	if fileErr != nil {
		return nil, fmt.Errorf("failed to get tokens from keyring or local file: keyring error: %v, file error: %v", keyringErr, fileErr)
	}

	return tokens, nil
}

func (s *Store) getTokensFromFile() (*Tokens, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var fileTokens map[string]*Tokens
	if err := json.Unmarshal(data, &fileTokens); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token file: %w", err)
	}

	tokens, ok := fileTokens[s.tokenKey]
	if !ok {
		return nil, ErrNotFound
	}

	return tokens, nil
}

func (s *Store) SaveTokens(tokens *Tokens) error {
	payload, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	setErr := keyring.Set(s.service, s.tokenKey, string(payload))
	if setErr == nil {
		_ = s.removeTokenFromFile()
		return nil
	}

	// If keyring set fails, fall back to file storage
	return s.saveTokensToFile(tokens)
}

func (s *Store) saveTokensToFile(tokens *Tokens) error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var fileTokens map[string]*Tokens
	data, err := os.ReadFile(s.filePath)
	if err == nil {
		if err := json.Unmarshal(data, &fileTokens); err != nil {
			return fmt.Errorf("failed to unmarshal existing token file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	if fileTokens == nil {
		fileTokens = make(map[string]*Tokens)
	}

	fileTokens[s.tokenKey] = tokens

	data, err = json.MarshalIndent(fileTokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

func (s *Store) Clear() error {
	keyringErr := keyring.Delete(s.service, s.tokenKey)

	fileErr := s.removeTokenFromFile()

	// If both keyring and file report not found, return auth.ErrNotFound
	if errors.Is(keyringErr, keyring.ErrNotFound) && errors.Is(fileErr, ErrNotFound) {
		return ErrNotFound
	}

	// Return file error if it's not ErrNotFound
	if fileErr != nil && !errors.Is(fileErr, ErrNotFound) {
		return fileErr
	}

	return nil
}

func (s *Store) removeTokenFromFile() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var fileTokens map[string]*Tokens
	if err := json.Unmarshal(data, &fileTokens); err != nil {
		return nil
	}

	_, exists := fileTokens[s.tokenKey]
	delete(fileTokens, s.tokenKey)

	if !exists {
		return ErrNotFound
	}

	if len(fileTokens) == 0 {
		if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove token file: %w", err)
		}
		return nil
	}

	data, err = json.MarshalIndent(fileTokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}
