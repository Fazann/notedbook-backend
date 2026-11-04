package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewSetsTimeouts(t *testing.T) {
	srv := New(":0", http.NotFoundHandler())
	for name, d := range map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	} {
		if d <= 0 {
			t.Errorf("%s = %v, want > 0", name, d)
		}
	}
	if srv.MaxHeaderBytes <= 0 {
		t.Errorf("MaxHeaderBytes = %d, want > 0", srv.MaxHeaderBytes)
	}
}

// A request in flight when shutdown starts still gets its response, and serve returns nil.
func TestGracefulShutdown(t *testing.T) {
	started := make(chan struct{})
	srv := New("", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond)
		io.WriteString(w, "done")
	}))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- serve(ctx, srv, ln) }()

	got := make(chan string, 1)
	go func() {
		res, err := http.Get("http://" + ln.Addr().String())
		if err != nil {
			got <- "error: " + err.Error()
			return
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		got <- string(b)
	}()

	<-started
	cancel()

	if body := <-got; body != "done" {
		t.Fatalf("in-flight response = %q, want %q", body, "done")
	}
	if err := <-served; err != nil {
		t.Fatalf("serve returned %v, want nil", err)
	}
	if _, err := http.Get("http://" + ln.Addr().String()); err == nil {
		t.Fatal("server still accepts requests after shutdown")
	}
}

func TestRunListenError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// The port is taken, so Run fails at once instead of blocking.
	if err := Run(context.Background(), New(ln.Addr().String(), http.NotFoundHandler())); err == nil {
		t.Fatal("Run on a used port returned nil, want an error")
	}
}
