// Package crypto provides AES-256-GCM encryption/decryption for device secrets.
// The server stores encrypted device secrets (ciphertext + nonce) in the DB.
// On each HMAC verification request, the secret is decrypted in-memory.
//
// Design note (REQUIREMENT.md §12.1, plan §8):
//   - Storing SHA-256(secret) and using it as the HMAC key is broken:
//     a DB leak would let an attacker forge signatures directly.
//   - Instead we store AES-GCM ciphertext. The DEVICE_SECRET_ENCRYPTION_KEY
//     env var is the actual trust boundary — protect it like a root secret.
//   - Key rotation requires all devices to re-pair.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// key must be exactly 32 bytes. Returns (ciphertext, nonce, error).
func Encrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	if len(key) != 32 {
		return nil, nil, fmt.Errorf("crypto.Encrypt: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto.Encrypt: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto.Encrypt: new GCM: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("crypto.Encrypt: rand nonce: %w", err)
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext produced by Encrypt.
// key must be exactly 32 bytes.
func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto.Decrypt: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto.Decrypt: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto.Decrypt: new GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("crypto.Decrypt: authentication failed")
	}
	return plaintext, nil
}
