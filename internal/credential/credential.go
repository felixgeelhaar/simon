// Package credential provides secure storage for sensitive data like API keys.
// It uses AES-256-GCM encryption with a machine-derived key to encrypt credentials
// before storing them in the database.
package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

const (
	// EncryptedPrefix marks values as encrypted in storage
	EncryptedPrefix = "enc:v1:"
)

var (
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrInvalidFormat    = errors.New("invalid encrypted format")
)

// Manager handles secure storage and retrieval of credentials.
type Manager struct {
	key []byte
}

// NewManager creates a new credential manager with a machine-derived encryption key.
// The key is derived from machine-specific identifiers to ensure credentials
// can only be decrypted on the same machine.
func NewManager() (*Manager, error) {
	key, err := deriveKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}
	return &Manager{key: key}, nil
}

// Encrypt encrypts a plaintext value and returns a storable string.
func (m *Manager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts a stored encrypted value back to plaintext.
func (m *Manager) Decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}

	// If not encrypted, return as-is (for backward compatibility)
	if !strings.HasPrefix(stored, EncryptedPrefix) {
		return stored, nil
	}

	encoded := strings.TrimPrefix(stored, EncryptedPrefix)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64: %v", ErrInvalidFormat, err)
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrInvalidFormat
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a value is already encrypted.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncryptedPrefix)
}

// deriveKey creates a machine-specific 32-byte key for AES-256.
// The key is derived from multiple machine identifiers to ensure
// it's unique to this machine but consistent across restarts.
func deriveKey() ([]byte, error) {
	// Collect machine-specific entropy
	var entropy strings.Builder

	// 1. Hostname
	hostname, _ := os.Hostname()
	entropy.WriteString(hostname)

	// 2. Home directory
	home, _ := os.UserHomeDir()
	entropy.WriteString(home)

	// 3. OS and architecture
	entropy.WriteString(runtime.GOOS)
	entropy.WriteString(runtime.GOARCH)

	// 4. Simon-specific salt
	entropy.WriteString("simon-credential-manager-v1")

	// 5. User-specific identifier (UID on Unix, username on Windows)
	if uid := os.Getuid(); uid != -1 {
		entropy.WriteString(fmt.Sprintf("uid:%d", uid))
	}
	if username := os.Getenv("USER"); username != "" {
		entropy.WriteString(username)
	}

	// Hash to create a consistent 32-byte key
	hash := sha256.Sum256([]byte(entropy.String()))
	return hash[:], nil
}

// MaskSecret returns a masked version of a secret for display purposes.
// Shows only the first and last 4 characters if the secret is long enough.
func MaskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
