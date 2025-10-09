package workos

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/99designs/keyring"
)

type KeyringTokenStore struct {
	ring keyring.Keyring
	key  string
}

// NewKeyringTokenStore creates a token store using 99designs/keyring
// It automatically tries system keyring first, falls back to encrypted file
func NewKeyringTokenStore(serviceName, tokenKey string) (*KeyringTokenStore, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	if tokenKey == "" {
		return nil, fmt.Errorf("token key is required")
	}

	// Hash the token key for consistent length
	hash := sha256.Sum256([]byte(tokenKey))
	hashedKey := fmt.Sprintf("%x", hash)

	// Configure keyring with automatic backend selection
	// Priority: system keyring (Secret Service/Keychain/WinCred) -> encrypted file
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
		// File backend config (fallback)
		FileDir: "~/.suga",
		// Use a fixed passphrase derived from service name for transparent encryption
		// This avoids password prompts while still encrypting the file
		FilePasswordFunc: keyring.FixedStringPrompt(serviceName),
		// Allow all backends with system keyring prioritized
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KeychainBackend,
			keyring.WinCredBackend,
			keyring.FileBackend,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return &KeyringTokenStore{
		ring: ring,
		key:  hashedKey,
	}, nil
}

func (s *KeyringTokenStore) GetTokens() (*Tokens, error) {
	item, err := s.ring.Get(s.key)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to retrieve token from keyring: %w", err)
	}

	var tokens Tokens
	err = json.Unmarshal(item.Data, &tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &tokens, nil
}

func (s *KeyringTokenStore) SaveTokens(tokens *Tokens) error {
	data, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	err = s.ring.Set(keyring.Item{
		Key:  s.key,
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}

	return nil
}

func (s *KeyringTokenStore) Clear() error {
	err := s.ring.Remove(s.key)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}
	return nil
}
