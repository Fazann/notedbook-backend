package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/internal/users/dto"
	"github.com/Fazann/notedbook-backend/pkg/aesgcm"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/mail"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/pkg/utils"
	"gorm.io/gorm"
)

const (
	otpTTL            = 5 * time.Minute
	otpResendInterval = time.Minute
	otpMaxAttempts    = 5
	resetKeyTTL       = 10 * time.Minute
)

func otpKey(email string) string        { return "reset:otp:" + email }
func otpAttemptKey(email string) string { return "reset:otp:attempt:" + email }
func otpResendKey(email string) string  { return "reset:otp:resend:" + email }
func resetKeyKey(key string) string     { return "reset:key:" + sha256Hex(key) }

// resetPayload is encrypted into the key returned by VerifyMailOTP.
type resetPayload struct {
	Email     string `json:"email"`
	OTP       string `json:"otp"`
	ExpiresAt int64  `json:"exp"`
}

// RequestMailOTP emails a one-time code to a registered address. It answers the same way whether the
// email is registered or not, so it cannot be used to find accounts.
func (s *authSvc) RequestMailOTP(ctx context.Context, req *dto.MailOTPRequestReq) response.SvcRes {
	email, err := cleanEmail(req.Email)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	allowed, retryAfter, err := redis.Hit(ctx, otpResendKey(email), 1, otpResendInterval)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("otp resend limit: %w", err), http.StatusInternalServerError)
	}
	if !allowed {
		return response.SvcRespErr(
			i18n.Err(i18n.TooManyRequests, i18n.Params{"seconds": int(retryAfter.Seconds()) + 1}),
			http.StatusTooManyRequests,
		)
	}

	res := &dto.MailOTPRequestRes{ExpiresIn: int(otpTTL.Seconds())}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespSuccess(res, http.StatusOK)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if user.IsDisabled {
		return response.SvcRespSuccess(res, http.StatusOK)
	}

	otp, err := generateOTP()
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("generate otp: %w", err), http.StatusInternalServerError)
	}
	html, err := mail.RenderOTP(mail.OTPData{AppName: config.SmtpFromName, Code: otp, ExpiresMinutes: int(otpTTL.Minutes())})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	// A new code replaces the old one and gets a fresh attempt count.
	if err := redis.Set(ctx, otpKey(email), sha256Hex(otp), otpTTL); err != nil {
		return response.SvcRespErr(fmt.Errorf("store otp: %w", err), http.StatusInternalServerError)
	}
	if err := redis.Del(ctx, otpAttemptKey(email)); err != nil {
		logger.Warn("reset otp attempts failed", "err", err)
	}

	// Sent in the background so the response time does not show whether the email is registered.
	go func() {
		if err := mail.Send(email, "Your password reset code", html); err != nil {
			logger.Error("send otp email failed", "err", err, "user_id", user.ID)
		}
	}()

	return response.SvcRespSuccess(res, http.StatusOK)
}

// VerifyMailOTP checks the code and returns a single-use key for ResetPassword.
func (s *authSvc) VerifyMailOTP(ctx context.Context, req *dto.MailOTPVerifyReq) response.SvcRes {
	email, err := cleanEmail(req.Email)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	stored, found, err := redis.Get(ctx, otpKey(email))
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("get otp: %w", err), http.StatusInternalServerError)
	}
	if !found {
		return response.SvcRespErr(i18n.InvalidOTP, http.StatusBadRequest)
	}

	allowed, _, err := redis.Hit(ctx, otpAttemptKey(email), otpMaxAttempts, otpTTL)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("otp attempt limit: %w", err), http.StatusInternalServerError)
	}
	if !allowed {
		if err := redis.Del(ctx, otpKey(email)); err != nil {
			logger.Warn("delete otp failed", "err", err)
		}
		return response.SvcRespErr(i18n.OTPTooManyAttempts, http.StatusTooManyRequests)
	}

	if subtle.ConstantTimeCompare([]byte(stored), []byte(sha256Hex(req.OTP))) != 1 {
		return response.SvcRespErr(i18n.InvalidOTP, http.StatusBadRequest)
	}

	// The code is used up; only the key can continue the reset.
	if err := redis.Del(ctx, otpKey(email), otpAttemptKey(email)); err != nil {
		return response.SvcRespErr(fmt.Errorf("delete otp: %w", err), http.StatusInternalServerError)
	}

	key, err := sealResetKey(config.OtpSecret, resetPayload{
		Email:     email,
		OTP:       req.OTP,
		ExpiresAt: time.Now().Add(resetKeyTTL).Unix(),
	})
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("seal reset key: %w", err), http.StatusInternalServerError)
	}
	if err := redis.Set(ctx, resetKeyKey(key), email, resetKeyTTL); err != nil {
		return response.SvcRespErr(fmt.Errorf("store reset key: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(&dto.MailOTPVerifyRes{Key: key, ExpiresIn: int(resetKeyTTL.Seconds())}, http.StatusOK)
}

// ResetPassword sets a new password with a key from VerifyMailOTP and signs the user out everywhere.
func (s *authSvc) ResetPassword(ctx context.Context, req *dto.ResetPasswordReq) response.SvcRes {
	payload, err := openResetKey(config.OtpSecret, req.Key, time.Now())
	if err != nil {
		if errors.Is(err, aesgcm.ErrEmptySecret) {
			return response.SvcRespErr(err, http.StatusInternalServerError)
		}
		return response.SvcRespErr(i18n.InvalidResetKey, http.StatusBadRequest)
	}

	// GetDel makes the key single-use, even when two requests race.
	email, found, err := redis.GetDel(ctx, resetKeyKey(req.Key))
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("get reset key: %w", err), http.StatusInternalServerError)
	}
	if !found || email != payload.Email {
		return response.SvcRespErr(i18n.InvalidResetKey, http.StatusBadRequest)
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.InvalidResetKey, http.StatusBadRequest)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if user.IsDisabled {
		return response.SvcRespErr(i18n.AccountDisabled, http.StatusForbidden)
	}

	hashed, err := utils.GenerateHash(req.NewPassword)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("hash password: %w", err), http.StatusInternalServerError)
	}

	err = db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"password": hashed, "updated_by": user.ID}
		if err := s.userRepo.WithTx(tx).Update(ctx, map[string]any{"id": user.ID}, updates); err != nil {
			return fmt.Errorf("update password: %w", err)
		}
		if err := s.refreshTokenRepo.WithTx(tx).RevokeUserToken(ctx, user.ID); err != nil {
			return fmt.Errorf("revoke user tokens: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(nil, http.StatusOK)
}

// generateOTP returns a random 6-digit code.
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func sealResetKey(secret string, p resetPayload) (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return aesgcm.Encrypt(secret, data)
}

var errResetKeyExpired = errors.New("reset key expired")

// openResetKey decrypts key and checks that it has not expired at now.
func openResetKey(secret, key string, now time.Time) (*resetPayload, error) {
	data, err := aesgcm.Decrypt(secret, key)
	if err != nil {
		return nil, err
	}
	var p resetPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, aesgcm.ErrInvalid
	}
	if now.Unix() > p.ExpiresAt {
		return nil, errResetKeyExpired
	}
	return &p, nil
}
