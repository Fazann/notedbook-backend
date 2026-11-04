package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		hops int
		xff  []string
		want string
	}{
		{name: "hops 0 ignores header", hops: 0, xff: []string{"9.9.9.9"}, want: "10.0.0.1"},
		{name: "cloud run: last entry", hops: 1, xff: []string{"9.9.9.9"}, want: "9.9.9.9"},
		{name: "faked entries on the left are ignored", hops: 1, xff: []string{"6.6.6.6, 9.9.9.9"}, want: "9.9.9.9"},
		{name: "load balancer: second from the right", hops: 2, xff: []string{"6.6.6.6, 9.9.9.9, 35.191.0.1"}, want: "9.9.9.9"},
		{name: "several header lines", hops: 1, xff: []string{"6.6.6.6", "9.9.9.9"}, want: "9.9.9.9"},
		{name: "ipv6", hops: 1, xff: []string{"2001:db8::1"}, want: "2001:db8::1"},
		{name: "no header keeps peer", hops: 1, want: "10.0.0.1"},
		{name: "too few entries keeps peer", hops: 2, xff: []string{"9.9.9.9"}, want: "10.0.0.1"},
		{name: "invalid entry keeps peer", hops: 1, xff: []string{"6.6.6.6, not-an-ip"}, want: "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			if err := r.SetTrustedProxies(nil); err != nil {
				t.Fatal(err)
			}
			var got string
			r.GET("/x", RealIP(tt.hops), func(c *gin.Context) { got = c.ClientIP() })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			for _, v := range tt.xff {
				req.Header.Add("X-Forwarded-For", v)
			}
			r.ServeHTTP(httptest.NewRecorder(), req)

			if got != tt.want {
				t.Fatalf("ClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
