package service

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/internal/exchange/dto"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/pkg/response"
)

const (
	// exchangeRateCacheKey holds the latest USD table as JSON (a snapshot); rates are shared by every user.
	exchangeRateCacheKey = "exchange_rates:usd"
	// exchangeRateCacheTTL keeps the last good table long after it is due, to answer while the source is down.
	exchangeRateCacheTTL = 7 * 24 * time.Hour
	// exchangeRateMaxAge refetches even if the source's next update time is far away or wrong.
	exchangeRateMaxAge = 24 * time.Hour
	// exchangeRateRetryAfter waits this long after a failed fetch before trying the source again.
	exchangeRateRetryAfter = 10 * time.Minute

	providerName = "ExchangeRate-API"
	providerURL  = "https://www.exchangerate-api.com"
)

type ExchangeRateServiceImpl interface {
	List(ctx context.Context, req *dto.ExchangeRateListReq) response.SvcRes
}

type exchangeRateCache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

// snapshot is one fetched USD table plus when to fetch the next one.
type snapshot struct {
	Rates        map[string]float64 `json:"rates"`
	UpdatedAt    time.Time          `json:"updated_at"`
	NextUpdateAt time.Time          `json:"next_update_at"`
	FetchedAt    time.Time          `json:"fetched_at"`
	// Set after a failed fetch, so a down source is not called on every request.
	RetryAt time.Time `json:"retry_at,omitzero"`
}

func (s *snapshot) isFresh(now time.Time) bool {
	due := now.After(s.NextUpdateAt) || now.Sub(s.FetchedAt) > exchangeRateMaxAge
	return !due || now.Before(s.RetryAt)
}

type exchangeRateSvc struct {
	cache exchangeRateCache
	fetch func(ctx context.Context) (*snapshot, error)
	now   func() time.Time
	// One refresh at a time per instance; the others wait and then read what it cached.
	mu sync.Mutex
}

func NewExchangeRateSvc() *exchangeRateSvc {
	client := &http.Client{Timeout: 10 * time.Second}
	return &exchangeRateSvc{
		cache: redis.Store{},
		fetch: func(ctx context.Context) (*snapshot, error) {
			return fetchUSDRates(ctx, client, config.ExchangeRateAPIURL)
		},
		now: time.Now,
	}
}

// List returns the rates against req.Base (USD by default), computed from the cached USD table. The table is
// fetched when it is due; if the source fails, the last good table is returned. A cache failure is logged and never
// fails the request.
func (s *exchangeRateSvc) List(ctx context.Context, req *dto.ExchangeRateListReq) response.SvcRes {
	base := req.Base
	if base == "" {
		base = "USD"
	}
	snap := s.cached(ctx)
	if snap == nil || !snap.isFresh(s.now()) {
		snap = s.refresh(ctx)
	}
	if snap == nil {
		return response.SvcRespErr(i18n.ExchangeRatesUnavailable, http.StatusServiceUnavailable)
	}
	rates, ok := crossRates(snap.Rates, base)
	if !ok {
		return response.SvcRespErr(i18n.Err(i18n.UnsupportedCurrency, i18n.Params{"currency": base}), http.StatusBadRequest)
	}
	return response.SvcRespSuccess(&dto.ExchangeRateRes{
		Base:         base,
		Rates:        rates,
		UpdatedAt:    snap.UpdatedAt,
		NextUpdateAt: snap.NextUpdateAt,
		Provider:     providerName,
		ProviderURL:  providerURL,
	}, http.StatusOK)
}

func (s *exchangeRateSvc) cached(ctx context.Context) *snapshot {
	raw, found, err := s.cache.Get(ctx, exchangeRateCacheKey)
	if err != nil {
		logger.Warn("read exchange rate cache failed", "err", err)
	}
	if !found {
		return nil
	}
	var snap snapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil || len(snap.Rates) == 0 {
		logger.Warn("decode exchange rate cache failed", "err", err)
		return nil
	}
	return &snap
}

// refresh fetches a new table and caches it. On failure it returns the last good table (or nil) and marks it to be
// retried later.
func (s *exchangeRateSvc) refresh(ctx context.Context) *snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	old := s.cached(ctx)
	if old != nil && old.isFresh(now) {
		return old // another request refreshed it while this one waited
	}

	snap, err := s.fetch(ctx)
	if err != nil {
		logger.Error("fetch exchange rates failed", "err", err)
		if old == nil {
			return nil
		}
		old.RetryAt = now.Add(exchangeRateRetryAfter)
		s.store(ctx, old)
		return old
	}
	snap.FetchedAt = now
	s.store(ctx, snap)
	return snap
}

func (s *exchangeRateSvc) store(ctx context.Context, snap *snapshot) {
	raw, err := json.Marshal(snap)
	if err == nil {
		err = s.cache.Set(ctx, exchangeRateCacheKey, string(raw), exchangeRateCacheTTL)
	}
	if err != nil {
		logger.Warn("write exchange rate cache failed", "err", err)
	}
}

// crossRates turns a USD table into rates against base: 1 base = usd[code] / usd[base] code.
func crossRates(usd map[string]float64, base string) (map[string]float64, bool) {
	baseRate, ok := usd[base]
	if !ok || baseRate <= 0 {
		return nil, false
	}
	rates := make(map[string]float64, len(usd))
	for code, rate := range usd {
		rates[code] = rate / baseRate
	}
	rates[base] = 1
	return rates, true
}
