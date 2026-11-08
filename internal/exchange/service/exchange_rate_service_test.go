package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/exchange/dto"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
)

type fakeCache struct {
	values map[string]string
	err    error
}

func (c *fakeCache) Get(_ context.Context, key string) (string, bool, error) {
	if c.err != nil {
		return "", false, c.err
	}
	v, ok := c.values[key]
	return v, ok, nil
}

func (c *fakeCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	if c.err != nil {
		return c.err
	}
	c.values[key] = value
	return nil
}

var now = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

func usdTable(fetchedAt, nextUpdate time.Time) *snapshot {
	return &snapshot{
		Rates:        map[string]float64{"USD": 1, "KHR": 4000, "MYR": 4},
		UpdatedAt:    fetchedAt,
		NextUpdateAt: nextUpdate,
		FetchedAt:    fetchedAt,
	}
}

func cacheWith(t *testing.T, snap *snapshot) *fakeCache {
	t.Helper()
	c := &fakeCache{values: map[string]string{}}
	if snap != nil {
		raw, err := json.Marshal(snap)
		if err != nil {
			t.Fatal(err)
		}
		c.values[exchangeRateCacheKey] = string(raw)
	}
	return c
}

func TestExchangeRateList(t *testing.T) {
	fresh := usdTable(now.Add(-time.Hour), now.Add(time.Hour))
	due := usdTable(now.Add(-25*time.Hour), now.Add(-time.Hour))
	fetched := &snapshot{
		Rates:        map[string]float64{"USD": 1, "KHR": 4100, "MYR": 4.1},
		UpdatedAt:    now,
		NextUpdateAt: now.Add(24 * time.Hour),
	}
	sourceDown := errors.New("source down")

	tests := []struct {
		name       string
		cached     *snapshot
		cacheErr   error
		fetched    *snapshot
		fetchErr   error
		base       string
		wantStatus int
		wantErr    i18n.Key
		wantKHR    float64
		wantFetch  int
	}{
		{name: "fresh cache, no fetch", cached: fresh, wantStatus: 200, wantKHR: 4000},
		{name: "empty cache fetches", fetched: fetched, wantStatus: 200, wantKHR: 4100, wantFetch: 1},
		{name: "due cache fetches", cached: due, fetched: fetched, wantStatus: 200, wantKHR: 4100, wantFetch: 1},
		{name: "source down serves last good table", cached: due, fetchErr: sourceDown, wantStatus: 200, wantKHR: 4000, wantFetch: 1},
		{name: "source down, nothing cached", fetchErr: sourceDown, wantStatus: 503, wantErr: i18n.ExchangeRatesUnavailable, wantFetch: 1},
		{name: "cache error still fetches", cacheErr: errors.New("redis down"), fetched: fetched, wantStatus: 200, wantKHR: 4100, wantFetch: 1},
		{name: "cross rate against another base", cached: fresh, base: "MYR", wantStatus: 200, wantKHR: 1000},
		{name: "unsupported base", cached: fresh, base: "XYZ", wantStatus: 400, wantErr: i18n.UnsupportedCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := cacheWith(t, tt.cached)
			cache.err = tt.cacheErr
			fetches := 0
			svc := &exchangeRateSvc{
				cache: cache,
				now:   func() time.Time { return now },
				fetch: func(context.Context) (*snapshot, error) {
					fetches++
					if tt.fetchErr != nil {
						return nil, tt.fetchErr
					}
					copied := *tt.fetched
					return &copied, nil
				},
			}

			res := svc.List(context.Background(), &dto.ExchangeRateListReq{Base: tt.base})

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err %v)", res.StatusCode, tt.wantStatus, res.Error)
			}
			if fetches != tt.wantFetch {
				t.Errorf("fetches = %d, want %d", fetches, tt.wantFetch)
			}
			if tt.wantErr != "" {
				if key, _, _ := i18n.FromError(res.Error); key != tt.wantErr {
					t.Errorf("err = %v, want %v", res.Error, tt.wantErr)
				}
				return
			}
			data := res.Data.(*dto.ExchangeRateRes)
			wantBase := tt.base
			if wantBase == "" {
				wantBase = "USD"
			}
			if data.Base != wantBase || data.Rates[wantBase] != 1 {
				t.Errorf("base = %s (rate %v), want %s (rate 1)", data.Base, data.Rates[wantBase], wantBase)
			}
			if data.Rates["KHR"] != tt.wantKHR {
				t.Errorf("KHR = %v, want %v", data.Rates["KHR"], tt.wantKHR)
			}
			if data.Provider == "" || data.ProviderURL == "" {
				t.Error("provider attribution missing")
			}
		})
	}
}

func TestExchangeRateRetryAfterFailure(t *testing.T) {
	due := usdTable(now.Add(-25*time.Hour), now.Add(-time.Hour))
	cache := cacheWith(t, due)
	fetches := 0
	svc := &exchangeRateSvc{
		cache: cache,
		now:   func() time.Time { return now },
		fetch: func(context.Context) (*snapshot, error) {
			fetches++
			return nil, errors.New("source down")
		},
	}
	for range 3 {
		if res := svc.List(context.Background(), &dto.ExchangeRateListReq{}); res.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", res.StatusCode)
		}
	}
	if fetches != 1 {
		t.Errorf("fetches = %d, want 1: a failed source is retried only after %v", fetches, exchangeRateRetryAfter)
	}
}

func TestFetchUSDRates(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "valid table",
			status: 200,
			body: `{"result":"success","base_code":"USD","time_last_update_unix":1794096151,` +
				`"time_next_update_unix":1794183261,"rates":{"USD":1,"KHR":4093.99,"MYR":4.0559}}`,
		},
		{name: "http error", status: 500, body: `{}`, wantErr: true},
		{name: "source error", status: 200, body: `{"result":"error","error-type":"quota-reached"}`, wantErr: true},
		{name: "wrong base", status: 200, body: `{"result":"success","base_code":"EUR","rates":{"EUR":1}}`, wantErr: true},
		{name: "no rates", status: 200, body: `{"result":"success","base_code":"USD","rates":{}}`, wantErr: true},
		{name: "bad rate", status: 200, body: `{"result":"success","base_code":"USD","rates":{"USD":1,"KHR":0}}`, wantErr: true},
		{name: "bad code", status: 200, body: `{"result":"success","base_code":"USD","rates":{"USD":1,"<b>":2}}`, wantErr: true},
		{name: "not json", status: 200, body: `<html>`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			snap, err := fetchUSDRates(context.Background(), srv.Client(), srv.URL)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if snap.Rates["KHR"] != 4093.99 || !snap.NextUpdateAt.Equal(time.Unix(1794183261, 0)) {
				t.Errorf("snapshot = %+v", snap)
			}
		})
	}
}
