// Package token signs and parses the API's JWTs. It does not read config, so callers pass the secret.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Type string

const (
	Access  Type = "access"
	Refresh Type = "refresh"
)

var (
	ErrEmptySecret = errors.New("jwt secret is empty")
	ErrWrongType   = errors.New("wrong token type")
	// ErrExpired is returned (wrapped) by Parse when the token has expired.
	ErrExpired = jwt.ErrTokenExpired
)

type Claims struct {
	UserID    uint   `json:"user_id"`
	SessionID string `json:"session_id"`
	Type      Type   `json:"type"`
	jwt.RegisteredClaims
}

// Issue returns a signed HS256 token for the user's session and its expiry time.
func Issue(secret []byte, typ Type, userID uint, sessionID string, ttl time.Duration, now time.Time) (string, time.Time, error) {
	if len(secret) == 0 {
		return "", time.Time{}, ErrEmptySecret
	}

	expiresAt := now.Add(ttl)
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		Type:      typ,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign %s token: %w", typ, err)
	}

	return signed, expiresAt, nil
}

// Parse verifies the signature, expiry and type of a token and returns its claims.
func Parse(secret []byte, typ Type, tokenStr string) (*Claims, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	var claims Claims
	_, err := jwt.ParseWithClaims(tokenStr, &claims, func(*jwt.Token) (any, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if claims.Type != typ {
		return nil, ErrWrongType
	}

	return &claims, nil
}
