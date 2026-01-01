package oauth

import (
	"testing"
)

func TestTokenEncryptor_EncryptDecrypt(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes for AES-256

	enc, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor() error = %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "long text",
			plaintext: "This is a much longer string that contains various characters and symbols! @#$%^&*()_+",
		},
		{
			name:      "unicode",
			plaintext: "こんにちは世界 🌍🔐",
		},
		{
			name:      "oauth token format",
			plaintext: "ya29.a0AfH6SMB2xS5lZC0M7qR_Xy_example_access_token_1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Ciphertext should be different from plaintext (unless empty)
			if len(tt.plaintext) > 0 && ciphertext == tt.plaintext {
				t.Error("Encrypt() returned plaintext as ciphertext")
			}

			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestTokenEncryptor_DifferentCiphertexts(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	plaintext := "test message"

	enc, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor() error = %v", err)
	}

	ciphertext1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	ciphertext2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Due to random nonce, same plaintext should produce different ciphertexts
	if ciphertext1 == ciphertext2 {
		t.Error("Encrypt() produced same ciphertext for same plaintext (nonce not random)")
	}

	// Both should decrypt to the same plaintext
	dec1, _ := enc.Decrypt(ciphertext1)
	dec2, _ := enc.Decrypt(ciphertext2)

	if dec1 != dec2 {
		t.Error("Different ciphertexts didn't decrypt to same plaintext")
	}
}

func TestNewTokenEncryptor_InvalidKeySize(t *testing.T) {
	invalidKeys := [][]byte{
		{},                                                            // empty
		[]byte("short"),                                               // too short
		[]byte("1234567890123456"),                                    // 16 bytes (AES-128)
		[]byte("123456789012345678901234"),                            // 24 bytes (AES-192)
		[]byte("123456789012345678901234567890123456789012345678901"), // too long
	}

	for _, key := range invalidKeys {
		_, err := NewTokenEncryptor(key)
		if err == nil {
			t.Errorf("NewTokenEncryptor() with key size %d should return error", len(key))
		}
	}
}

func TestTokenEncryptor_DecryptInvalidCiphertext(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	enc, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor() error = %v", err)
	}

	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!!!",
		},
		{
			name:       "too short",
			ciphertext: "dG9vLXNob3J0", // "too-short" base64 encoded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("Decrypt() should return error for invalid ciphertext")
			}
		})
	}
}

func TestTokenEncryptor_DecryptWrongKey(t *testing.T) {
	key1 := []byte("12345678901234567890123456789012")
	key2 := []byte("abcdefghijklmnopqrstuvwxyz123456")

	plaintext := "secret message"

	enc1, err := NewTokenEncryptor(key1)
	if err != nil {
		t.Fatalf("NewTokenEncryptor() error = %v", err)
	}

	enc2, err := NewTokenEncryptor(key2)
	if err != nil {
		t.Fatalf("NewTokenEncryptor() error = %v", err)
	}

	ciphertext, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Decrypting with wrong key should fail
	_, err = enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() with wrong key should return error")
	}
}

func TestTokenEncryptor_EmptyString(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	enc, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor() error = %v", err)
	}

	ciphertext, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if ciphertext != "" {
		t.Errorf("Encrypt() of empty string = %q, want empty string", ciphertext)
	}

	decrypted, err := enc.Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != "" {
		t.Errorf("Decrypt() of empty string = %q, want empty string", decrypted)
	}
}

func TestNoOpEncryptor_EncryptDecrypt(t *testing.T) {
	enc := NewNoOpEncryptor()

	tests := []struct {
		name      string
		plaintext string
	}{
		{name: "simple", plaintext: "hello"},
		{name: "empty", plaintext: ""},
		{name: "token", plaintext: "access-token-12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			decrypted, err := enc.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestNoOpEncryptor_InvalidBase64(t *testing.T) {
	enc := NewNoOpEncryptor()

	_, err := enc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Error("Decrypt() should return error for invalid base64")
	}
}
