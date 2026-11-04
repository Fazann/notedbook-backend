package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vercel := []string{"https://notebook.vercel.app", "https://notebook-*-myteam.vercel.app"}

	tests := []struct {
		name    string
		allowed []string
		origin  string
		method  string
		want    string // Access-Control-Allow-Origin
		status  int
	}{
		{name: "no list allows all", origin: "https://any.example", method: http.MethodGet, want: "*", status: http.StatusOK},
		{name: "exact origin", allowed: vercel, origin: "https://notebook.vercel.app", method: http.MethodGet, want: "https://notebook.vercel.app", status: http.StatusOK},
		{name: "preview pattern", allowed: vercel, origin: "https://notebook-git-main-myteam.vercel.app", method: http.MethodGet, want: "https://notebook-git-main-myteam.vercel.app", status: http.StatusOK},
		{name: "other team's preview", allowed: vercel, origin: "https://notebook-abc-otherteam.vercel.app", method: http.MethodGet, want: "", status: http.StatusOK},
		{name: "other site", allowed: vercel, origin: "https://evil.example", method: http.MethodGet, want: "", status: http.StatusOK},
		{name: "http instead of https", allowed: vercel, origin: "http://notebook.vercel.app", method: http.MethodGet, want: "", status: http.StatusOK},
		{name: "preflight", allowed: vercel, origin: "https://notebook.vercel.app", method: http.MethodOptions, want: "https://notebook.vercel.app", status: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(CORS(tt.allowed))
			r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(tt.method, "/x", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d", w.Code, tt.status)
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Fatalf("Allow-Origin = %q, want %q", got, tt.want)
			}
			if got := w.Header().Get("Access-Control-Expose-Headers"); got != "Retry-After" {
				t.Fatalf("Expose-Headers = %q, want Retry-After", got)
			}
		})
	}
}
