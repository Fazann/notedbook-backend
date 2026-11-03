package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/internal/exspense/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// currencySettingCacheTTL bounds how long a stale entry could live if invalidating it ever failed.
const currencySettingCacheTTL = 24 * time.Hour

// currencySettingCacheKey holds one user's settings as JSON (a dto.CurrencySettingRes).
func currencySettingCacheKey(userID uint) string {
	return fmt.Sprintf("currency_settings:user:%d", userID)
}

type CurrencySettingServiceImpl interface {
	Get(ctx context.Context, userID uint) response.SvcRes
	Update(ctx context.Context, userID uint, req *dto.CurrencySettingReq) response.SvcRes
}

type currencySettingCache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type currencySettingSvc struct {
	repo  repo.CurrencySettingRepoImpl
	cache currencySettingCache
}

func NewCurrencySettingSvc() *currencySettingSvc {
	return &currencySettingSvc{
		repo:  repo.NewCurrencySettingRepo(),
		cache: redis.Store{},
	}
}

// Get reads the settings from the cache, or else from the database (every currency when the user has none yet)
// and then caches them. A cache failure is logged and never fails the request.
func (s *currencySettingSvc) Get(ctx context.Context, userID uint) response.SvcRes {
	key := currencySettingCacheKey(userID)
	cached, found, err := s.cache.Get(ctx, key)
	if err != nil {
		logger.Warn("read currency setting cache failed", "err", err, "user_id", userID)
	}
	if found {
		var res dto.CurrencySettingRes
		if err := json.Unmarshal([]byte(cached), &res); err == nil {
			return response.SvcRespSuccess(&res, http.StatusOK)
		}
		logger.Warn("decode currency setting cache failed", "err", err, "user_id", userID)
	}

	res := defaultCurrencySetting()
	setting, err := s.repo.FindByUser(ctx, userID)
	switch {
	case err == nil:
		res = toCurrencySettingRes(setting)
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return response.SvcRespErr(fmt.Errorf("find currency setting: %w", err), http.StatusInternalServerError)
	}

	if data, err := json.Marshal(res); err != nil {
		logger.Warn("encode currency setting cache failed", "err", err, "user_id", userID)
	} else if err := s.cache.Set(ctx, key, string(data), currencySettingCacheTTL); err != nil {
		logger.Warn("write currency setting cache failed", "err", err, "user_id", userID)
	}
	return response.SvcRespSuccess(res, http.StatusOK)
}

// Update saves the settings and clears the user's cached copy, so the next Get reads the new ones.
func (s *currencySettingSvc) Update(ctx context.Context, userID uint, req *dto.CurrencySettingReq) response.SvcRes {
	enabled, err := cleanEnabledCurrencies(req.EnabledCurrencies)
	if err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	if err := checkCurrency(req.DefaultCurrency); err != nil {
		return response.SvcRespErr(err, http.StatusBadRequest)
	}
	if !slices.Contains(enabled, req.DefaultCurrency) {
		return response.SvcRespErr(i18n.DefaultCurrencyNotEnabled, http.StatusBadRequest)
	}

	setting := &model.CurrencySetting{
		UserID:            userID,
		EnabledCurrencies: strings.Join(enabled, ","),
		DefaultCurrency:   req.DefaultCurrency,
	}
	if err := s.repo.Upsert(ctx, setting); err != nil {
		return response.SvcRespErr(fmt.Errorf("save currency setting: %w", err), http.StatusInternalServerError)
	}
	s.invalidateCache(ctx, userID)

	return response.SvcRespSuccess(toCurrencySettingRes(setting), http.StatusOK)
}

// invalidateCache drops the user's cached settings; call it after every change to them.
// A failure is only logged: the entry still expires after currencySettingCacheTTL.
func (s *currencySettingSvc) invalidateCache(ctx context.Context, userID uint) {
	if err := s.cache.Del(ctx, currencySettingCacheKey(userID)); err != nil {
		logger.Warn("invalidate currency setting cache failed", "err", err, "user_id", userID)
	}
}

// cleanEnabledCurrencies checks every code and returns them without duplicates, in the order of enums.CurrencyValues.
func cleanEnabledCurrencies(codes []string) ([]string, error) {
	for _, code := range codes {
		if err := checkCurrency(code); err != nil {
			return nil, err
		}
	}
	enabled := supportedIn(codes)
	if len(enabled) == 0 {
		return nil, i18n.NoCurrencyEnabled
	}
	return enabled, nil
}

// supportedIn returns the supported currencies found in codes, in the order of enums.CurrencyValues.
func supportedIn(codes []string) []string {
	res := make([]string, 0, len(enums.CurrencyValues))
	for _, code := range enums.CurrencyValues {
		if slices.Contains(codes, code) {
			res = append(res, code)
		}
	}
	return res
}

// defaultCurrencySetting is what a user who never saved settings gets: every currency, USD first.
func defaultCurrencySetting() *dto.CurrencySettingRes {
	return &dto.CurrencySettingRes{
		EnabledCurrencies: slices.Clone(enums.CurrencyValues),
		DefaultCurrency:   string(enums.CurrencyEnumUSD),
	}
}

// toCurrencySettingRes drops stored codes that are no longer supported; with none left it falls back to every currency.
func toCurrencySettingRes(setting *model.CurrencySetting) *dto.CurrencySettingRes {
	enabled := supportedIn(strings.Split(setting.EnabledCurrencies, ","))
	if len(enabled) == 0 {
		return defaultCurrencySetting()
	}
	def := setting.DefaultCurrency
	if !slices.Contains(enabled, def) {
		def = enabled[0]
	}
	return &dto.CurrencySettingRes{EnabledCurrencies: enabled, DefaultCurrency: def}
}
