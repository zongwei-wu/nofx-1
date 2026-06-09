package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	// APIKeyPrefix is the prefix for all user-facing API keys
	APIKeyPrefix = "nfx_sk_"
	// APIKeyRandomBytes is the number of random bytes used for key generation
	APIKeyRandomBytes = 32
)

// GenerateAPIKey creates a new API key.
// Returns the plaintext key (to show the user once) and the SHA-256 hash (to store).
func GenerateAPIKey() (plaintext, hashed string, err error) {
	random := make([]byte, APIKeyRandomBytes)
	if _, err := rand.Read(random); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	plaintext = APIKeyPrefix + base64.RawURLEncoding.EncodeToString(random)
	hashed = HashAPIKey(plaintext)
	return plaintext, hashed, nil
}

// HashAPIKey returns the SHA-256 hex digest of a plaintext API key.
func HashAPIKey(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return fmt.Sprintf("%x", h)
}

// ValidateAPIKey checks whether a plaintext key matches a stored hash.
func ValidateAPIKey(hashed, plaintext string) bool {
	return HashAPIKey(plaintext) == hashed
}

// MaskAPIKey returns a masked version for display, e.g. "nfx_sk_a1b2****".
// Shows the first 12 characters then masks the rest.
func MaskAPIKey(plaintext string) string {
	if len(plaintext) <= 12 {
		return plaintext
	}
	return plaintext[:12] + "****"
}
