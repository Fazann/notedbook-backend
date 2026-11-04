package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func TestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bind := func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.JsonErrRes(c, err, http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	}
	raise := func(c *gin.Context) { SetBodyLimit(c, 1<<10) }

	tests := []struct {
		name     string
		size     int
		handlers []gin.HandlerFunc
		want     int
	}{
		{name: "under the limit", size: 50, handlers: []gin.HandlerFunc{bind}, want: http.StatusOK},
		{name: "over the limit", size: 200, handlers: []gin.HandlerFunc{bind}, want: http.StatusRequestEntityTooLarge},
		{name: "route raises the limit", size: 200, handlers: []gin.HandlerFunc{raise, bind}, want: http.StatusOK},
		{name: "raised limit still applies", size: 2000, handlers: []gin.HandlerFunc{raise, bind}, want: http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(BodyLimit(100))
			r.POST("/x", tt.handlers...)

			body := `{"name":"` + strings.Repeat("a", tt.size) + `"}`
			req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
			if tt.want != http.StatusRequestEntityTooLarge {
				return
			}
			var res struct{ Code string }
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatal(err)
			}
			if res.Code != string(i18n.RequestTooLarge) {
				t.Fatalf("code = %q, want %q", res.Code, i18n.RequestTooLarge)
			}
		})
	}
}
