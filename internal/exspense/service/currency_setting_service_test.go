package service

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/internal/exspense/dto"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// fakeCurrencySettingRepo keeps one row per user in memory and counts database reads.
type fakeCurrencySettingRepo struct {
	settings map[uint]model.CurrencySetting
	finds    int
	err      error
}

func (r *fakeCurrencySettingRepo) FindByUser(_ context.Context, userID uint) (*model.CurrencySetting, error) {
	r.finds++
	if r.err != nil {
		return nil, r.err
	}
	s, ok := r.settings[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &s, nil
}

func (r *fakeCurrencySettingRepo) Upsert(_ context.Context, s *model.CurrencySetting) error {
	r.settings[s.UserID] = *s
	return nil
}

// fakeCurrencySettingCache is an in-memory cache; err, when set, is returned by every call.
type fakeCurrencySettingCache struct {
	values map[string]string
	err    error
}

func (c *fakeCurrencySettingCache) Get(_ context.Context, key string) (string, bool, error) {
	if c.err != nil {
		return "", false, c.err
	}
	v, ok := c.values[key]
	return v, ok, nil
}

func (c *fakeCurrencySettingCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	if c.err != nil {
		return c.err
	}
	c.values[key] = value
	return nil
}

func (c *fakeCurrencySettingCache) Del(_ context.Context, keys ...string) error {
	if c.err != nil {
		return c.err
	}
	for _, k := range keys {
		delete(c.values, k)
	}
	return nil
}

func newCurrencySettingTestSvc() (*currencySettingSvc, *fakeCurrencySettingRepo, *fakeCurrencySettingCache) {
	r := &fakeCurrencySettingRepo{settings: map[uint]model.CurrencySetting{
		1: {UserID: 1, EnabledCurrencies: "KHR,USD", DefaultCurrency: "KHR"},
	}}
	c := &fakeCurrencySettingCache{values: map[string]string{}}
	return &currencySettingSvc{repo: r, cache: c}, r, c
}

func settingOf(t *testing.T, data any) *dto.CurrencySettingRes {
	t.Helper()
	res, ok := data.(*dto.CurrencySettingRes)
	if !ok {
		t.Fatalf("data is %T, want *dto.CurrencySettingRes", data)
	}
	return res
}

func TestCurrencySettingGetCachesAfterFirstRead(t *testing.T) {
	svc, r, c := newCurrencySettingTestSvc()
	ctx := context.Background()

	for range 3 {
		res := svc.Get(ctx, 1)
		if res.Error != nil {
			t.Fatalf("Get: %v", res.Error)
		}
		got := settingOf(t, res.Data)
		if !reflect.DeepEqual(got.EnabledCurrencies, []string{"USD", "KHR"}) || got.DefaultCurrency != "KHR" {
			t.Fatalf("Get = %+v, want USD,KHR default KHR", got)
		}
	}
	if r.finds != 1 {
		t.Errorf("database reads = %d, want 1 (then served from cache)", r.finds)
	}
	if _, ok := c.values[currencySettingCacheKey(1)]; !ok {
		t.Error("settings were not cached")
	}
}

func TestCurrencySettingGetDefaultsWhenNeverSaved(t *testing.T) {
	svc, _, _ := newCurrencySettingTestSvc()
	got := settingOf(t, svc.Get(context.Background(), 2).Data)
	if !reflect.DeepEqual(got.EnabledCurrencies, []string{"USD", "KHR", "MYR"}) || got.DefaultCurrency != "USD" {
		t.Errorf("Get = %+v, want every currency with USD", got)
	}
}

func TestCurrencySettingGetWorksWithoutCache(t *testing.T) {
	svc, r, c := newCurrencySettingTestSvc()
	c.err = errors.New("redis down")
	for range 2 {
		if res := svc.Get(context.Background(), 1); res.Error != nil {
			t.Fatalf("Get: %v", res.Error)
		}
	}
	if r.finds != 2 {
		t.Errorf("database reads = %d, want 2 (no cache)", r.finds)
	}
}

func TestCurrencySettingGetDatabaseError(t *testing.T) {
	svc, r, _ := newCurrencySettingTestSvc()
	r.err = errors.New("db down")
	if res := svc.Get(context.Background(), 1); res.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", res.StatusCode)
	}
}

func TestCurrencySettingUpdateClearsCache(t *testing.T) {
	svc, r, c := newCurrencySettingTestSvc()
	ctx := context.Background()
	svc.Get(ctx, 1) // caches USD,KHR

	res := svc.Update(ctx, 1, &dto.CurrencySettingReq{EnabledCurrencies: []string{"MYR", "USD", "MYR"}, DefaultCurrency: "MYR"})
	if res.Error != nil {
		t.Fatalf("Update: %v", res.Error)
	}
	if _, ok := c.values[currencySettingCacheKey(1)]; ok {
		t.Error("cache was not cleared after the update")
	}
	if saved := r.settings[1]; saved.EnabledCurrencies != "USD,MYR" || saved.DefaultCurrency != "MYR" {
		t.Errorf("saved = %+v, want USD,MYR default MYR", saved)
	}

	got := settingOf(t, svc.Get(ctx, 1).Data)
	if !reflect.DeepEqual(got.EnabledCurrencies, []string{"USD", "MYR"}) || got.DefaultCurrency != "MYR" {
		t.Errorf("Get after update = %+v, want the new settings", got)
	}
}

func TestCurrencySettingUpdateValidation(t *testing.T) {
	tests := []struct {
		name string
		req  dto.CurrencySettingReq
		want i18n.Key
	}{
		{"none enabled", dto.CurrencySettingReq{EnabledCurrencies: []string{}, DefaultCurrency: "USD"}, i18n.NoCurrencyEnabled},
		{"unknown currency", dto.CurrencySettingReq{EnabledCurrencies: []string{"EUR"}, DefaultCurrency: "EUR"}, i18n.InvalidCurrency},
		{"default not enabled", dto.CurrencySettingReq{EnabledCurrencies: []string{"USD"}, DefaultCurrency: "KHR"}, i18n.DefaultCurrencyNotEnabled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, r, c := newCurrencySettingTestSvc()
			c.values[currencySettingCacheKey(1)] = "cached"
			res := svc.Update(context.Background(), 1, &tt.req)
			key, _, _ := i18n.FromError(res.Error)
			if res.StatusCode != http.StatusBadRequest || key != tt.want {
				t.Fatalf("Update = %d %v, want 400 %v", res.StatusCode, res.Error, tt.want)
			}
			if r.settings[1].EnabledCurrencies != "KHR,USD" {
				t.Error("invalid input changed the saved settings")
			}
			if c.values[currencySettingCacheKey(1)] != "cached" {
				t.Error("invalid input cleared the cache")
			}
		})
	}
}

func TestToCurrencySettingResDropsUnsupportedCodes(t *testing.T) {
	got := toCurrencySettingRes(&model.CurrencySetting{EnabledCurrencies: "EUR,MYR", DefaultCurrency: "EUR"})
	if !reflect.DeepEqual(got.EnabledCurrencies, []string{"MYR"}) || got.DefaultCurrency != "MYR" {
		t.Errorf("got %+v, want MYR default MYR", got)
	}
	got = toCurrencySettingRes(&model.CurrencySetting{EnabledCurrencies: "", DefaultCurrency: ""})
	if len(got.EnabledCurrencies) != 3 || got.DefaultCurrency != "USD" {
		t.Errorf("got %+v, want every currency", got)
	}
}
