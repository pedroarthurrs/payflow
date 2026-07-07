package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher handles bcrypt password hashing operations.
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a PasswordHasher with the specified bcrypt cost.
// Recommended cost is 12 for production (security vs performance tradeoff).
func NewPasswordHasher(cost int) *PasswordHasher {
	return &PasswordHasher{cost: cost}
}

// Hash generates a bcrypt hash of the password.
func (ph *PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), ph.cost)
	if err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}
	return string(hash), nil
}

// Compare verifies a password against its bcrypt hash.
// Returns nil if valid, bcrypt error otherwise.
func (ph *PasswordHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// KeyGenerator handles cryptographic key generation.
type KeyGenerator struct{}

// NewKeyGenerator creates a KeyGenerator.
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{}
}

// GenerateRandomBytes generates n random bytes and returns them as a hex string.
func (kg *KeyGenerator) GenerateRandomBytes(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateRandomBytes: failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateAPIKey generates a random API key with an environment-specific prefix.
// Format: "{ENV_PREFIX}_{hex(32 bytes)}"
// Prefixes: "I9TEST_" for test/dev, "I9LIVE_" for production.
func (kg *KeyGenerator) GenerateAPIKey(environment string) (string, error) {
	hexBytes, err := kg.GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}

	prefix := "I9TEST_"
	if environment == "production" {
		prefix = "I9LIVE_"
	}
	return prefix + hexBytes, nil
}

// Hasher handles SHA-256 hashing operations.
type Hasher struct{}

// NewHasher creates a Hasher.
func NewHasher() *Hasher {
	return &Hasher{}
}

// SHA256 computes the SHA-256 hash of data and returns it as a hex string.
func (h *Hasher) SHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA256WithPepper computes SHA-256(pepper + data) and returns it as a hex string.
// Used for API key hashing with a secret pepper.
func (h *Hasher) SHA256WithPepper(pepper, data string) string {
	hash := sha256.Sum256([]byte(pepper + data))
	return hex.EncodeToString(hash[:])
}

// HMACGenerator handles HMAC-SHA256 operations.
type HMACGenerator struct{}

// NewHMACGenerator creates an HMACGenerator.
func NewHMACGenerator() *HMACGenerator {
	return &HMACGenerator{}
}

// HMACSHA256 computes HMAC-SHA256 of data with the given secret and returns it as a hex string.
func (hg *HMACGenerator) HMACSHA256(secret string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
