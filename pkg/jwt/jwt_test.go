package token

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndParse(t *testing.T) {
	secret := []byte("test-secret")
	now := time.Now()

	valid, _, err := Issue(secret, Access, 7, "session-1", time.Hour, now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	expired, _, err := Issue(secret, Access, 7, "session-1", time.Hour, now.Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("issue expired: %v", err)
	}
	refresh, _, err := Issue(secret, Refresh, 7, "session-1", time.Hour, now)
	if err != nil {
		t.Fatalf("issue refresh: %v", err)
	}

	tests := []struct {
		name    string
		secret  []byte
		typ     Type
		token   string
		wantErr error
	}{
		{name: "valid", secret: secret, typ: Access, token: valid},
		{name: "wrong secret", secret: []byte("other"), typ: Access, token: valid, wantErr: jwt.ErrTokenSignatureInvalid},
		{name: "expired", secret: secret, typ: Access, token: expired, wantErr: jwt.ErrTokenExpired},
		{name: "refresh used as access", secret: secret, typ: Access, token: refresh, wantErr: ErrWrongType},
		{name: "malformed", secret: secret, typ: Access, token: "not-a-token", wantErr: jwt.ErrTokenMalformed},
		{name: "empty secret", secret: nil, typ: Access, token: valid, wantErr: ErrEmptySecret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := Parse(tt.secret, tt.typ, tt.token)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if claims.UserID != 7 || claims.SessionID != "session-1" || claims.Type != Access {
				t.Fatalf("claims = %+v", claims)
			}
		})
	}
}

func TestIssueEmptySecret(t *testing.T) {
	if _, _, err := Issue(nil, Access, 1, "s", time.Hour, time.Now()); !errors.Is(err, ErrEmptySecret) {
		t.Fatalf("err = %v, want %v", err, ErrEmptySecret)
	}
}
