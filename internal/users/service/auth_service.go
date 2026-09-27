package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/internal/users/dto"
	"github.com/Fazann/notedbook-backend/internal/users/repo"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	token "github.com/Fazann/notedbook-backend/pkg/jwt"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/pkg/utils"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	registerMaxAttempts = 5
	registerBlockTime   = 5 * time.Minute

	loginMaxAttempts = 5
	loginBlockTime   = 5 * time.Minute
)

type AuthServiceImpl interface {
	Register(ctx context.Context, req *dto.RegisterReq, clientIP string) response.SvcRes
	Login(ctx context.Context, req *dto.LoginReq) response.SvcRes
	RefreshToken(ctx context.Context, req *dto.RefreshTokenReq) response.SvcRes
	RequestMailOTP(ctx context.Context, req *dto.MailOTPRequestReq) response.SvcRes
	VerifyMailOTP(ctx context.Context, req *dto.MailOTPVerifyReq) response.SvcRes
	ResetPassword(ctx context.Context, req *dto.ResetPasswordReq) response.SvcRes
}

type authSvc struct {
	userRepo         repo.UserRepoImpl
	refreshTokenRepo repo.RefreshTokenRepoImpl
	deviceRepo       repo.DeviceRepoImpl
}

func NewAuthSvc() *authSvc {
	return &authSvc{
		userRepo:         repo.NewUserRepo(),
		refreshTokenRepo: repo.NewRefreshTokenRepo(),
		deviceRepo:       repo.NewDeviceRepo(),
	}
}

func (s *authSvc) Register(ctx context.Context, req *dto.RegisterReq, clientIP string) response.SvcRes {
	username, err := cleanUsername(req.Username)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}

	email := ""
	if req.Email != "" {
		email, err = cleanEmail(req.Email)
		if err != nil {
			return response.SvcRespErr(err, http.StatusBadRequest)
		}
	}
	// 1. Limit attempts per device + IP
	limitKey := fmt.Sprintf("register:attempt:%s:%s", req.Device.UID, clientIP)
	allowed, _, err := redis.Hit(ctx, limitKey, registerMaxAttempts, registerBlockTime)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("register rate limit: %w", err), http.StatusInternalServerError)
	}
	if !allowed {
		return response.SvcRespErr(
			i18n.Err(i18n.TooManyAttempts, i18n.Params{"minutes": int(registerBlockTime.Minutes())}),
			http.StatusTooManyRequests,
		)
	}

	// 2. Check duplicate username and email
	existing, err := s.userRepo.FindByUsernameOrEmail(ctx, username, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if existing != nil {
		if existing.Username == username {
			return response.SvcRespErr(i18n.UsernameTaken, http.StatusConflict)
		}
		return response.SvcRespErr(i18n.EmailTaken, http.StatusConflict)
	}

	hashed, err := utils.GenerateHash(req.Password)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("hash password: %w", err), http.StatusInternalServerError)
	}

	// 3. Create user, store the device, issue tokens and store the refresh token in one transaction
	var res dto.TokenRes
	err = db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := s.userRepo.WithTx(tx).Create(ctx, &model.User{
			Fullname: strings.TrimSpace(req.Fullname),
			Username: username,
			Email:    email,
			Password: string(hashed),
		})
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		if err := s.storeDevice(ctx, tx, user.ID, &req.Device); err != nil {
			return err
		}

		tokens, err := s.issueTokens(user.ID)
		if err != nil {
			return err
		}
		_, err = s.refreshTokenRepo.WithTx(tx).IssueNewUserToken(ctx, model.RefreshToken{
			UserID:    user.ID,
			SessionID: tokens.SessionID,
			ExpiredAt: tokens.RefreshExpiresAt,
		})
		if err != nil {
			return fmt.Errorf("store refresh token: %w", err)
		}

		res.AccessToken = tokens.AccessToken
		res.RefreshToken = tokens.RefreshToken
		return nil
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(&res, http.StatusCreated)
}

func (s *authSvc) Login(ctx context.Context, req *dto.LoginReq) response.SvcRes {
	username := strings.TrimSpace(req.Username)

	// Limit attempts per device + IP; cleared after a successful login
	limitKey := fmt.Sprintf("login:attempt:%s:%s", req.Device.UID, req.IP)
	allowed, _, err := redis.Hit(ctx, limitKey, loginMaxAttempts, loginBlockTime)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("login rate limit: %w", err), http.StatusInternalServerError)
	}
	if !allowed {
		return response.SvcRespErr(
			i18n.Err(i18n.TooManyAttempts, i18n.Params{"minutes": int(loginBlockTime.Minutes())}),
			http.StatusTooManyRequests,
		)
	}

	// Find user
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.InvalidCredentials, http.StatusUnauthorized)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	// Validate password
	err = utils.CompareHash(user.Password, req.Password)
	if err != nil {
		return response.SvcRespErr(i18n.InvalidCredentials, http.StatusUnauthorized)
	}
	if user.IsDisabled {
		return response.SvcRespErr(i18n.AccountDisabled, http.StatusForbidden)
	}

	// Store the device, issue new tokens for a new session and store its refresh token in one transaction
	var res dto.TokenRes
	err = db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.storeDevice(ctx, tx, user.ID, &req.Device); err != nil {
			return err
		}

		tokens, err := s.issueTokens(user.ID)
		if err != nil {
			return err
		}
		_, err = s.refreshTokenRepo.WithTx(tx).IssueNewUserToken(ctx, model.RefreshToken{
			UserID:    user.ID,
			SessionID: tokens.SessionID,
			ExpiredAt: tokens.RefreshExpiresAt,
		})
		if err != nil {
			return fmt.Errorf("store refresh token: %w", err)
		}

		res.AccessToken = tokens.AccessToken
		res.RefreshToken = tokens.RefreshToken
		return nil
	})
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	if err := redis.Reset(ctx, limitKey); err != nil {
		logger.Warn("reset login attempts failed", "err", err)
	}

	return response.SvcRespSuccess(&res, http.StatusOK)
}

func (s *authSvc) RefreshToken(ctx context.Context, req *dto.RefreshTokenReq) response.SvcRes {
	// 1. Verify jwt
	claims, err := token.Parse([]byte(config.JwtSecret), token.Refresh, req.RefreshToken)
	if err != nil {
		if errors.Is(err, token.ErrExpired) {
			return response.SvcRespErr(i18n.TokenExpired, http.StatusUnauthorized)
		}
		return response.SvcRespErr(i18n.Unauthorized, http.StatusUnauthorized)
	}

	// 2. Check the session exists in db
	stored, err := s.refreshTokenRepo.FindUserTokenWithSessionID(ctx, claims.UserID, claims.SessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.Unauthorized, http.StatusUnauthorized)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if stored.IsRevoked {
		// A revoked token used again was likely stolen: end every session of this user.
		if err := s.refreshTokenRepo.RevokeUserToken(ctx, claims.UserID); err != nil {
			return response.SvcRespErr(fmt.Errorf("revoke user tokens: %w", err), http.StatusInternalServerError)
		}
		logger.Warn("revoked refresh token reused", "user_id", claims.UserID)
		return response.SvcRespErr(i18n.Unauthorized, http.StatusUnauthorized)
	}
	if stored.ExpiredAt < time.Now().Unix() {
		return response.SvcRespErr(i18n.TokenExpired, http.StatusUnauthorized)
	}

	user, err := s.userRepo.FindOne(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.Unauthorized, http.StatusUnauthorized)
		}
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}
	if user.IsDisabled {
		return response.SvcRespErr(i18n.AccountDisabled, http.StatusForbidden)
	}

	// 3. Revoke the old token and 4. issue new tokens, in one transaction
	var res dto.TokenRes
	err = db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := s.refreshTokenRepo.WithTx(tx)
		revoked, err := repoTx.RevokeByID(ctx, stored.ID)
		if err != nil {
			return fmt.Errorf("revoke refresh token: %w", err)
		}
		if !revoked {
			// Another request used this token first.
			return i18n.Unauthorized
		}

		tokens, err := s.issueTokens(user.ID)
		if err != nil {
			return err
		}
		_, err = repoTx.IssueNewUserToken(ctx, model.RefreshToken{
			UserID:    user.ID,
			SessionID: tokens.SessionID,
			ExpiredAt: tokens.RefreshExpiresAt,
		})
		if err != nil {
			return fmt.Errorf("store refresh token: %w", err)
		}

		res.AccessToken = tokens.AccessToken
		res.RefreshToken = tokens.RefreshToken
		return nil
	})
	if errors.Is(err, i18n.Unauthorized) {
		return response.SvcRespErr(i18n.Unauthorized, http.StatusUnauthorized)
	}
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(&res, http.StatusOK)
}

// ============================================================
// Local functions
// ============================================================

func cleanUsername(username string) (string, error) {
	// English letters and numbers, with single spaces between words.
	var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9]+( [A-Za-z0-9]+)*$`)
	username = strings.TrimSpace(username)
	if len(username) < 2 || len(username) > 50 || !usernameRegex.MatchString(username) {
		return "", i18n.InvalidUsername
	}

	return username, nil
}

func cleanEmail(email string) (string, error) {
	// Local part, "@", then dot-separated domain labels ending in a TLD of 2+ letters. Applied after lowercasing.
	var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9](?:[a-z0-9\-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9\-]*[a-z0-9])?)*\.[a-z]{2,}$`)
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 100 || !emailRegex.MatchString(email) {
		return "", i18n.InvalidEmail
	}

	local, _, _ := strings.Cut(email, "@")
	if strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return "", i18n.InvalidEmail
	}

	return email, nil
}

// storeDevice signs other users out of this device, then marks it as logged in for userID:
// an existing device of this user is updated, a new one is created.
func (s *authSvc) storeDevice(ctx context.Context, tx *gorm.DB, userID uint, req *dto.DeviceReq) error {
	deviceRepo := s.deviceRepo.WithTx(tx)
	if err := deviceRepo.LogoutOtherUsers(ctx, req.UID, userID); err != nil {
		return fmt.Errorf("logout other users from device: %w", err)
	}

	device, err := deviceRepo.FindByUserWithUID(ctx, userID, req.UID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find device: %w", err)
	}

	if device != nil {
		device.Name = req.Name
		device.OS = enums.DeviceOSEnum(req.OS)
		device.Version = req.Version
		device.UserAgent = truncate(req.UserAgent, 300)
		device.LoginAt = time.Now()
		device.IsLogin = true
		if err := deviceRepo.Save(ctx, device); err != nil {
			return fmt.Errorf("update device: %w", err)
		}
		return nil
	}

	_, err = deviceRepo.Create(ctx, &model.Device{
		UserID:    userID,
		Name:      req.Name,
		OS:        enums.DeviceOSEnum(req.OS),
		Version:   req.Version,
		UID:       req.UID,
		UserAgent: truncate(req.UserAgent, 300),
		LoginAt:   time.Now(),
		IsLogin:   true,
	})
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	}

	return nil
}

// truncate cuts s to at most max bytes without splitting a UTF-8 character.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	s = s[:max]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

func (s *authSvc) issueTokens(userID uint) (*dto.TokenRes, error) {
	sessionID := uuid.NewString()
	now := time.Now()
	secret := []byte(config.JwtSecret)

	accToken, _, err := token.Issue(secret, token.Access, userID, sessionID, accessTokenTTL(), now)
	if err != nil {
		return nil, err
	}

	var refreshExpiresAt time.Time
	refToken, refreshExpiresAt, err := token.Issue(secret, token.Refresh, userID, sessionID, refreshTokenTTL(), now)
	if err != nil {
		return nil, err
	}

	return &dto.TokenRes{
		AccessToken:      accToken,
		RefreshToken:     refToken,
		RefreshExpiresAt: refreshExpiresAt.Unix(),
		SessionID:        sessionID,
	}, nil
}

// ACC_TK_EXPIRED is in hours.
func accessTokenTTL() time.Duration {
	return time.Duration(config.AccessTokenExp) * time.Hour
}

// REF_TK_EXPIRED is in days.
func refreshTokenTTL() time.Duration {
	return time.Duration(config.RefreshTokenExp) * 24 * time.Hour
}
