package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// maxSourceBody bounds what is read from the source (the full table is about 4 KB).
const maxSourceBody = 1 << 20

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

// sourceRes is the part of the open.er-api.com /v6/latest/USD answer that is used.
type sourceRes struct {
	Result             string             `json:"result"`
	ErrorType          string             `json:"error-type"`
	BaseCode           string             `json:"base_code"`
	TimeLastUpdateUnix int64              `json:"time_last_update_unix"`
	TimeNextUpdateUnix int64              `json:"time_next_update_unix"`
	Rates              map[string]float64 `json:"rates"`
}

// fetchUSDRates downloads the latest table and checks it before it is cached: base USD, at least one rate, and only
// well-formed codes with positive rates.
func fetchUSDRates(ctx context.Context, client *http.Client, url string) (*snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build exchange rate request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call exchange rate source: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange rate source answered %d", resp.StatusCode)
	}

	var body sourceRes
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxSourceBody)).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode exchange rates: %w", err)
	}
	if body.Result != "success" {
		return nil, fmt.Errorf("exchange rate source error: %q", body.ErrorType)
	}
	if body.BaseCode != "USD" {
		return nil, fmt.Errorf("exchange rate source base is %q, want USD", body.BaseCode)
	}
	if len(body.Rates) == 0 {
		return nil, errors.New("exchange rate source sent no rates")
	}
	for code, rate := range body.Rates {
		if !currencyCode.MatchString(code) || rate <= 0 {
			return nil, fmt.Errorf("exchange rate source sent an invalid rate %q: %v", code, rate)
		}
	}
	return &snapshot{
		Rates:        body.Rates,
		UpdatedAt:    time.Unix(body.TimeLastUpdateUnix, 0).UTC(),
		NextUpdateAt: time.Unix(body.TimeNextUpdateUnix, 0).UTC(),
	}, nil
}
