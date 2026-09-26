// Package i18n translates message keys into English, Khmer and Malay.
//
// Translations live in locales/<lang>.yml. Keys are generated as constants in keys_gen.go:
//
//	make locale   (or: go generate ./pkg/i18n)
//
// Services return keys as errors, and the JSON response translates them:
//
//	return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
//	return response.SvcRespErr(i18n.Err(i18n.FileTooLarge, i18n.Params{"max": "100 MB"}), http.StatusRequestEntityTooLarge)
package i18n

//go:generate go run ../../cmd/locale

import (
	"embed"
	"errors"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

// Lang is a supported language code.
type Lang string

const (
	EN Lang = "en"
	KM Lang = "km"
	MS Lang = "ms"

	// Default is used when the request asks for no supported language, and as the
	// fallback when a key is missing in the requested language.
	Default = EN
)

// Langs lists the supported languages.
var Langs = []Lang{EN, KM, MS}

// Key is a translation key from locales/*.yml. It implements error, so a service can return it directly.
type Key string

func (k Key) Error() string { return string(k) }

// Params fills {name} placeholders in a message.
type Params map[string]any

// Error is a translation key with placeholder values.
type Error struct {
	Key    Key
	Params Params
}

func (e *Error) Error() string { return string(e.Key) }

// Err returns an error that translates to key with params filled in.
func Err(key Key, params Params) error {
	return &Error{Key: key, Params: params}
}

//go:embed locales/*.yml
var localeFS embed.FS

var catalog = mustLoad()

func mustLoad() map[Lang]map[Key]string {
	c := make(map[Lang]map[Key]string, len(Langs))
	for _, lang := range Langs {
		data, err := localeFS.ReadFile(path.Join("locales", string(lang)+".yml"))
		if err != nil {
			panic(fmt.Sprintf("i18n: read %s.yml: %v", lang, err))
		}
		var raw map[string]string
		if err := yaml.Unmarshal(data, &raw); err != nil {
			panic(fmt.Sprintf("i18n: parse %s.yml: %v", lang, err))
		}
		msgs := make(map[Key]string, len(raw))
		for k, v := range raw {
			msgs[Key(k)] = v
		}
		c[lang] = msgs
	}
	return c
}

// T translates key into lang, falling back to the default language, then to the key itself.
func T(lang Lang, key Key, params Params) string {
	msg, ok := catalog[lang][key]
	if !ok {
		msg, ok = catalog[Default][key]
	}
	if !ok {
		return string(key)
	}
	for name, value := range params {
		msg = strings.ReplaceAll(msg, "{"+name+"}", fmt.Sprint(value))
	}
	return msg
}

// Has reports whether key exists in the default language.
func Has(key Key) bool {
	_, ok := catalog[Default][key]
	return ok
}

// FromError returns the key and params inside err (also when wrapped with %w).
// ok is false when err carries no known key.
func FromError(err error) (key Key, params Params, ok bool) {
	var e *Error
	if errors.As(err, &e) && Has(e.Key) {
		return e.Key, e.Params, true
	}
	var k Key
	if errors.As(err, &k) && Has(k) {
		return k, nil, true
	}
	return "", nil, false
}

// KeyForStatus is the generic message key for an HTTP status, used for errors without their own key.
func KeyForStatus(status int) Key {
	switch status {
	case http.StatusBadRequest:
		return BadRequest
	case http.StatusUnauthorized:
		return Unauthorized
	case http.StatusForbidden:
		return Forbidden
	case http.StatusNotFound:
		return NotFound
	case http.StatusConflict:
		return Conflict
	case http.StatusUnprocessableEntity:
		return UnprocessableEntity
	default:
		return InternalServerError
	}
}

// Parse matches a language tag such as "km", "km-KH", "kh" or "ms-MY" to a supported language.
func Parse(tag string) (Lang, bool) {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	base, _, _ = strings.Cut(base, "_")
	switch base {
	case "en":
		return EN, true
	case "km", "kh":
		return KM, true
	case "ms":
		return MS, true
	}
	return "", false
}

// FromRequest picks the language for a request: the `lang` query parameter first,
// then the Accept-Language header (by q weight), then Default.
func FromRequest(r *http.Request) Lang {
	if lang, ok := Parse(r.URL.Query().Get("lang")); ok {
		return lang
	}
	return FromAcceptLanguage(r.Header.Get("Accept-Language"))
}

// FromAcceptLanguage picks the best supported language from an Accept-Language header,
// e.g. "km-KH,km;q=0.9,en;q=0.8".
func FromAcceptLanguage(header string) Lang {
	type candidate struct {
		lang Lang
		q    float64
	}
	var candidates []candidate
	for _, part := range strings.Split(header, ",") {
		tag, weight, _ := strings.Cut(part, ";")
		lang, ok := Parse(tag)
		if !ok {
			continue
		}
		q := 1.0
		if v, found := strings.CutPrefix(strings.TrimSpace(weight), "q="); found {
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				q = parsed
			}
		}
		if q > 0 {
			candidates = append(candidates, candidate{lang, q})
		}
	}
	if len(candidates) == 0 {
		return Default
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].q > candidates[j].q })
	return candidates[0].lang
}
