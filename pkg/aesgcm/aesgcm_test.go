package aesgcm

import (
	"errors"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	enc, err := Encrypt("secret", []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	other, _ := Encrypt("secret", []byte("hello"))
	if enc == other {
		t.Error("two encryptions of the same value are equal; nonce is not random")
	}

	tests := []struct {
		name    string
		secret  string
		value   string
		want    string
		wantErr error
	}{
		{name: "round trip", secret: "secret", value: enc, want: "hello"},
		{name: "wrong secret", secret: "other", value: enc, wantErr: ErrInvalid},
		{name: "tampered", secret: "secret", value: enc[:len(enc)-2] + "AA", wantErr: ErrInvalid},
		{name: "not base64", secret: "secret", value: "!!!", wantErr: ErrInvalid},
		{name: "too short", secret: "secret", value: "AAAA", wantErr: ErrInvalid},
		{name: "empty secret", secret: "", value: enc, wantErr: ErrEmptySecret},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decrypt(tt.secret, tt.value)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
