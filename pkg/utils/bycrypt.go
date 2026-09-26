package utils

import (
	"golang.org/x/crypto/bcrypt"
)

func GenerateHash(text string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(text), 8)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func CompareHash(hashStr string, text string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(text))
}
