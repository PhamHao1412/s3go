package crypto

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Set mock environment key for test isolation
	t.Setenv("S3GO_ENCRYPTION_KEY", "test-encryption-key-32-chars-!!!")

	originalText := "my-super-secret-aws-key-12345!"

	// Encrypt the text
	encrypted, err := Encrypt(originalText)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if encrypted == originalText {
		t.Fatalf("Encrypted text should not match original text")
	}

	// Decrypt back
	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != originalText {
		t.Fatalf("Decrypted text '%s' does not match original '%s'", decrypted, originalText)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	// Set mock environment key for test isolation
	t.Setenv("S3GO_ENCRYPTION_KEY", "test-encryption-key-32-chars-!!!")

	_, err := Decrypt("invalid-hex-string")
	if err == nil {
		t.Fatalf("Expected error when decrypting invalid hex string, got nil")
	}
}

func TestGetEncryptionKeyPanic(t *testing.T) {
	// Temporarily clear the environment variable
	t.Setenv("S3GO_ENCRYPTION_KEY", "")

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected getEncryptionKey to panic when S3GO_ENCRYPTION_KEY is not set, but it did not panic")
		}
	}()

	// This should panic
	getEncryptionKey()
}
