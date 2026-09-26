package i18n

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestEveryKeyIsTranslated(t *testing.T) {
	for _, lang := range Langs {
		for _, key := range AllKeys {
			if _, ok := catalog[lang][key]; !ok {
				t.Errorf("%s.yml is missing %s; run `make locale` to see all problems", lang, key)
			}
		}
		if got, want := len(catalog[lang]), len(AllKeys); got != want {
			t.Errorf("%s.yml has %d keys, keys_gen.go has %d; run `make locale`", lang, got, want)
		}
	}
}

func TestT(t *testing.T) {
	if got, want := T(KM, RecordNotFound, nil), "រកមិនឃើញទិន្នន័យទេ។"; got != want {
		t.Errorf("T(km) = %q, want %q", got, want)
	}
	if got, want := T(MS, FileTooLarge, Params{"max": "100 MB"}), "Fail terlalu besar. Saiz maksimum ialah 100 MB."; got != want {
		t.Errorf("T(ms, params) = %q, want %q", got, want)
	}
	if got, want := T("fr", RecordNotFound, nil), "The record was not found."; got != want {
		t.Errorf("unknown lang = %q, want English fallback %q", got, want)
	}
	if got, want := T(EN, "NO_SUCH_KEY", nil), "NO_SUCH_KEY"; got != want {
		t.Errorf("unknown key = %q, want the key %q", got, want)
	}
}

func TestFromError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		key    Key
		params Params
		ok     bool
	}{
		{"key", RecordNotFound, RecordNotFound, nil, true},
		{"wrapped key", fmt.Errorf("get attachment: %w", RecordNotFound), RecordNotFound, nil, true},
		{"with params", Err(FileTooLarge, Params{"max": "1 MB"}), FileTooLarge, Params{"max": "1 MB"}, true},
		{"unknown key", Key("NO_SUCH_KEY"), "", nil, false},
		{"plain error", fmt.Errorf("boom"), "", nil, false},
		{"nil", nil, "", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, params, ok := FromError(tt.err)
			if key != tt.key || ok != tt.ok || fmt.Sprint(params) != fmt.Sprint(tt.params) {
				t.Errorf("FromError = (%q, %v, %v), want (%q, %v, %v)", key, params, ok, tt.key, tt.params, tt.ok)
			}
		})
	}
}

func TestFromAcceptLanguage(t *testing.T) {
	tests := map[string]Lang{
		"":                        EN,
		"fr-FR,fr;q=0.9":          EN,
		"km-KH,km;q=0.9,en;q=0.8": KM,
		"en;q=0.5,ms-MY;q=0.9":    MS,
		"kh":                      KM,
		"fr,ms;q=0.7,en;q=0.3":    MS,
		"km;q=0,en":               EN,
	}
	for header, want := range tests {
		if got := FromAcceptLanguage(header); got != want {
			t.Errorf("FromAcceptLanguage(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestFromRequestQueryWins(t *testing.T) {
	r := httptest.NewRequest("GET", "/attachments/1?lang=ms", nil)
	r.Header.Set("Accept-Language", "km")
	if got := FromRequest(r); got != MS {
		t.Errorf("FromRequest = %q, want ms", got)
	}
}
