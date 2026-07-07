package crypto

import (
	"strings"
	"testing"
)

func TestPasswordHasher_Hash(t *testing.T) {
	ph := NewPasswordHasher(10)
	password := "test_password_123"

	hash, err := ph.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash returned empty string")
	}

	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Fatalf("Hash does not look like bcrypt: %s", hash)
	}
}

func TestPasswordHasher_Compare_Valid(t *testing.T) {
	ph := NewPasswordHasher(10)
	password := "test_password_123"

	hash, err := ph.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	err = ph.Compare(hash, password)
	if err != nil {
		t.Fatalf("Compare failed for valid password: %v", err)
	}
}

func TestPasswordHasher_Compare_Invalid(t *testing.T) {
	ph := NewPasswordHasher(10)
	password := "test_password_123"
	wrongPassword := "wrong_password"

	hash, err := ph.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	err = ph.Compare(hash, wrongPassword)
	if err == nil {
		t.Fatal("Compare should fail for invalid password")
	}
}

func TestKeyGenerator_GenerateRandomBytes(t *testing.T) {
	kg := NewKeyGenerator()

	hex1, err := kg.GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("GenerateRandomBytes failed: %v", err)
	}

	hex2, err := kg.GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("GenerateRandomBytes failed: %v", err)
	}

	if hex1 == hex2 {
		t.Fatal("GenerateRandomBytes should produce different values")
	}

	if len(hex1) != 64 { // 32 bytes = 64 hex chars
		t.Fatalf("Expected 64 hex chars, got %d", len(hex1))
	}
}

func TestKeyGenerator_GenerateAPIKey_Test(t *testing.T) {
	kg := NewKeyGenerator()

	key, err := kg.GenerateAPIKey("test")
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if !strings.HasPrefix(key, "I9TEST_") {
		t.Fatalf("Expected I9TEST_ prefix, got: %s", key)
	}

	if len(key) != 7+64 { // prefix + 32 bytes as hex
		t.Fatalf("Expected length %d, got %d", 7+64, len(key))
	}
}

func TestKeyGenerator_GenerateAPIKey_Production(t *testing.T) {
	kg := NewKeyGenerator()

	key, err := kg.GenerateAPIKey("production")
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if !strings.HasPrefix(key, "I9LIVE_") {
		t.Fatalf("Expected I9LIVE_ prefix, got: %s", key)
	}

	if len(key) != 7+64 { // prefix + 32 bytes as hex
		t.Fatalf("Expected length %d, got %d", 7+64, len(key))
	}
}

func TestHasher_SHA256(t *testing.T) {
	h := NewHasher()

	hash1 := h.SHA256("test_data")
	hash2 := h.SHA256("test_data")

	if hash1 != hash2 {
		t.Fatal("SHA256 should produce consistent results")
	}

	if len(hash1) != 64 { // SHA256 = 32 bytes = 64 hex chars
		t.Fatalf("Expected 64 hex chars, got %d", len(hash1))
	}
}

func TestHasher_SHA256WithPepper(t *testing.T) {
	h := NewHasher()

	hash1 := h.SHA256WithPepper("pepper", "data")
	hash2 := h.SHA256WithPepper("pepper", "data")
	hash3 := h.SHA256WithPepper("different_pepper", "data")

	if hash1 != hash2 {
		t.Fatal("SHA256WithPepper should produce consistent results")
	}

	if hash1 == hash3 {
		t.Fatal("SHA256WithPepper should produce different results with different pepper")
	}
}

func TestHMACGenerator_HMACSHA256(t *testing.T) {
	hg := NewHMACGenerator()

	sig1 := hg.HMACSHA256("secret", []byte("payload"))
	sig2 := hg.HMACSHA256("secret", []byte("payload"))
	sig3 := hg.HMACSHA256("different_secret", []byte("payload"))

	if sig1 != sig2 {
		t.Fatal("HMACSHA256 should produce consistent results")
	}

	if sig1 == sig3 {
		t.Fatal("HMACSHA256 should produce different results with different secret")
	}

	if len(sig1) != 64 { // HMAC-SHA256 = 32 bytes = 64 hex chars
		t.Fatalf("Expected 64 hex chars, got %d", len(sig1))
	}
}

// Benchmark tests
func BenchmarkPasswordHasher_Hash(b *testing.B) {
	ph := NewPasswordHasher(10)
	password := "test_password_123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ph.Hash(password)
	}
}

func BenchmarkPasswordHasher_Compare(b *testing.B) {
	ph := NewPasswordHasher(10)
	password := "test_password_123"
	hash, _ := ph.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ph.Compare(hash, password)
	}
}

func BenchmarkHasher_SHA256(b *testing.B) {
	h := NewHasher()
	data := "test_data_for_hashing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.SHA256(data)
	}
}

func BenchmarkHMACGenerator_HMACSHA256(b *testing.B) {
	hg := NewHMACGenerator()
	secret := "secret_key"
	payload := []byte("test_payload_data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hg.HMACSHA256(secret, payload)
	}
}
