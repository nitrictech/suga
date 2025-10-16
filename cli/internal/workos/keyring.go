package workos

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
)

type KeyringTokenStore struct {
	ring keyring.Keyring
	key  string
}

// NewKeyringTokenStore tries system keyring first, falls back to encrypted file
func NewKeyringTokenStore(serviceName, tokenKey string) (*KeyringTokenStore, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	if tokenKey == "" {
		return nil, fmt.Errorf("token key is required")
	}

	hash := sha256.Sum256([]byte(tokenKey))
	hashedKey := fmt.Sprintf("%x", hash)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve home directory: %w", err)
	}

	fileDir := filepath.Join(homeDir, ".suga")

	passphrase, err := getOrCreateFilePassphrase(fileDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get file passphrase: %w", err)
	}

	ring, err := keyring.Open(keyring.Config{
		ServiceName:              serviceName,
		FileDir:                  fileDir,
		FilePasswordFunc:         keyring.FixedStringPrompt(passphrase),
		LibSecretCollectionName:  "login",
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

// getOrCreateFilePassphrase uses a per-user random passphrase instead of hardcoded constant
// to prevent mass decryption and follow security best practices (defense in depth)
func getOrCreateFilePassphrase(fileDir string) (string, error) {
	passphrasePath := filepath.Join(fileDir, ".keyring-passphrase")

	data, err := os.ReadFile(passphrasePath)
	if err == nil {
		return string(data), nil
	}

	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read passphrase file: %w", err)
	}

	if err := os.MkdirAll(fileDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	passphraseBytes := make([]byte, 32)
	if _, err := rand.Read(passphraseBytes); err != nil {
		return "", fmt.Errorf("failed to generate random passphrase: %w", err)
	}

	passphrase := base64.StdEncoding.EncodeToString(passphraseBytes)

	if err := os.WriteFile(passphrasePath, []byte(passphrase), 0600); err != nil {
		return "", fmt.Errorf("failed to write passphrase file: %w", err)
	}

	return passphrase, nil
}
