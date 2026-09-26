package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	token "github.com/Fazann/notedbook-backend/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func TestAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret")
	now := time.Now()

	issue := func(typ token.Type, issuedAt time.Time) string {
		t.Helper()
		s, _, err := token.Issue(secret, typ, 42, "session-1", time.Hour, issuedAt)
		if err != nil {
			t.Fatalf("issue: %v", err)
		}
		return s
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantCode   i18n.Key
	}{
		{name: "valid", header: "Bearer " + issue(token.Access, now), wantStatus: http.StatusOK},
		{name: "missing header", header: "", wantStatus: http.StatusUnauthorized, wantCode: i18n.Unauthorized},
		{name: "no bearer prefix", header: issue(token.Access, now), wantStatus: http.StatusUnauthorized, wantCode: i18n.Unauthorized},
		{name: "expired", header: "Bearer " + issue(token.Access, now.Add(-2*time.Hour)), wantStatus: http.StatusUnauthorized, wantCode: i18n.TokenExpired},
		{name: "refresh token", header: "Bearer " + issue(token.Refresh, now), wantStatus: http.StatusUnauthorized, wantCode: i18n.Unauthorized},
		{name: "garbage", header: "Bearer abc", wantStatus: http.StatusUnauthorized, wantCode: i18n.Unauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser uint
			var gotSession string
			r := gin.New()
			r.GET("/me", Auth(secret), func(c *gin.Context) {
				gotUser, gotSession = CurrentUserID(c), CurrentSessionID(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				if gotUser != 42 || gotSession != "session-1" {
					t.Fatalf("user = %d, session = %q", gotUser, gotSession)
				}
				return
			}
			var body struct{ Code string }
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Code != string(tt.wantCode) {
				t.Fatalf("code = %q, want %q", body.Code, tt.wantCode)
			}
		})
	}
}
