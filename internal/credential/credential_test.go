package credential

import (
	"strings"
	"testing"
)

func TestManager_EncryptDecrypt(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"simple api key", "sk-1234567890abcdef"},
		{"long key", strings.Repeat("a", 1000)},
		{"unicode content", "api-key-日本語-🔑"},
		{"special chars", "key!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := manager.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			// Empty string should stay empty
			if tc.plaintext == "" {
				if encrypted != "" {
					t.Errorf("empty string should not be encrypted, got: %s", encrypted)
				}
				return
			}

			// Non-empty should be prefixed
			if !strings.HasPrefix(encrypted, EncryptedPrefix) {
				t.Errorf("encrypted value should have prefix, got: %s", encrypted)
			}

			// Encrypted should differ from plaintext
			if encrypted == tc.plaintext {
				t.Error("encrypted value should differ from plaintext")
			}

			decrypted, err := manager.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("decrypted value mismatch: got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestManager_DecryptPlaintext(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Unencrypted values should pass through for backward compatibility
	plaintext := "sk-not-encrypted"
	result, err := manager.Decrypt(plaintext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("plaintext should pass through unchanged: got %q, want %q", result, plaintext)
	}
}

func TestManager_DecryptInvalid(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	testCases := []struct {
		name  string
		input string
	}{
		{"invalid base64", EncryptedPrefix + "not-valid-base64!!!"},
		{"too short", EncryptedPrefix + "YWJj"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.Decrypt(tc.input)
			if err == nil {
				t.Error("expected error for invalid input")
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"sk-plaintext", false},
		{EncryptedPrefix + "data", true},
		{"enc:wrong:prefix", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := IsEncrypted(tc.input)
			if result != tc.expected {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestMaskSecret(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234...6789"},
		{"sk-1234567890abcdef", "sk-1...cdef"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := MaskSecret(tc.input)
			if result != tc.expected {
				t.Errorf("MaskSecret(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestManager_DifferentNonces(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	plaintext := "test-api-key"

	// Encrypt the same value twice
	enc1, _ := manager.Encrypt(plaintext)
	enc2, _ := manager.Encrypt(plaintext)

	// Each encryption should produce different ciphertext due to random nonce
	if enc1 == enc2 {
		t.Error("same plaintext should produce different ciphertext")
	}

	// Both should decrypt to the same value
	dec1, _ := manager.Decrypt(enc1)
	dec2, _ := manager.Decrypt(enc2)

	if dec1 != plaintext || dec2 != plaintext {
		t.Error("both should decrypt to original plaintext")
	}
}
