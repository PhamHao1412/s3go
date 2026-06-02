package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

// getEncryptionKey retrieves the 32-byte encryption key from the environment
// and panics if it is not configured.
func getEncryptionKey() []byte {
	key := os.Getenv("S3GO_ENCRYPTION_KEY")
	if len(key) == 0 {
		panic("FATAL: S3GO_ENCRYPTION_KEY environment variable is not set! Please configure it in your environment or .env file.")
	}

	// Ensure the key is exactly 32 bytes. If not, pad or truncate it.
	keyBytes := make([]byte, 32)
	copy(keyBytes, []byte(key))
	return keyBytes
}

// Encrypt encrypts a plaintext string to a hex-encoded ciphertext using AES-GCM.
func Encrypt(plaintext string) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a unique nonce for each encryption
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the plaintext and seal it with the nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to hex for safe storage in the database
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded ciphertext string back to the original plaintext.
func Decrypt(ciphertextHex string) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Split nonce and actual ciphertext
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the ciphertext
	plaintextBytes, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintextBytes), nil
}
