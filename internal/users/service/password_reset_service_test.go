package service

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/aesgcm"
)

func TestGenerateOTP(t *testing.T) {
	digits := regexp.MustCompile(`^[0-9]{6}$`)
	for range 100 {
		otp, err := generateOTP()
		if err != nil {
			t.Fatal(err)
		}
		if !digits.MatchString(otp) {
			t.Fatalf("otp %q is not 6 digits", otp)
		}
	}
}

func TestResetKey(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	key, err := sealResetKey("secret", resetPayload{Email: "me@example.com", OTP: "123456", ExpiresAt: now.Add(resetKeyTTL).Unix()})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		secret  string
		key     string
		now     time.Time
		wantErr error
	}{
		{name: "valid", secret: "secret", key: key, now: now},
		{name: "last second", secret: "secret", key: key, now: now.Add(resetKeyTTL)},
		{name: "expired", secret: "secret", key: key, now: now.Add(resetKeyTTL + time.Second), wantErr: errResetKeyExpired},
		{name: "wrong secret", secret: "other", key: key, now: now, wantErr: aesgcm.ErrInvalid},
		{name: "garbage", secret: "secret", key: "abc", now: now, wantErr: aesgcm.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := openResetKey(tt.secret, tt.key, tt.now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (p.Email != "me@example.com" || p.OTP != "123456") {
				t.Errorf("payload = %+v", p)
			}
		})
	}
}
