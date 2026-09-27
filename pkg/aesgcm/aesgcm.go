// Package aesgcm encrypts small payloads with AES-256-GCM into URL-safe strings.
package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

var (
	ErrEmptySecret = errors.New("aesgcm: empty secret")
	ErrInvalid     = errors.New("aesgcm: invalid ciphertext")
)

// newGCM derives a 256-bit key from secret, so the secret can be any non-empty string.
func newGCM(secret string) (cipher.AEAD, error) {
	if secret == "" {
		return nil, ErrEmptySecret
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Encrypt returns base64url(nonce || ciphertext || tag).
func Encrypt(secret string, plaintext []byte) (string, error) {
	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. It returns ErrInvalid when the value was not made with secret or was changed.
func Decrypt(secret, value string) ([]byte, error) {
	gcm, err := newGCM(secret)
	if err != nil {
		return nil, err
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(data) < gcm.NonceSize() {
		return nil, ErrInvalid
	}
	nonce, sealed := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, ErrInvalid
	}
	return plaintext, nil
}
