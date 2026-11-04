// Package server runs the HTTP server with timeouts and a graceful shutdown.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/logger"
)

const (
	// Slow clients (slowloris) cannot hold a connection open by sending headers byte by byte.
	readHeaderTimeout = 10 * time.Second
	// Whole request, body included. Routes that need longer (uploads) extend it with
	// http.ResponseController; see attachments.UploadFile.
	readTimeout = 30 * time.Second
	// From the end of the request headers to the end of the response.
	writeTimeout = 30 * time.Second
	// Keep-alive connections waiting for the next request.
	idleTimeout = 120 * time.Second
	// Request line + headers; a JWT and the usual headers fit easily.
	maxHeaderBytes = 64 << 10
	// Cloud Run sends SIGTERM and kills the container 10 s later; finish in-flight requests before that.
	shutdownTimeout = 8 * time.Second
)

// New returns an http.Server for handler on addr with the timeouts above.
func New(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

// Run serves until ctx is cancelled (e.g. on SIGTERM), then stops taking new connections and waits
// up to shutdownTimeout for in-flight requests. It returns nil after a clean shutdown.
func Run(ctx context.Context, srv *http.Server) error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}
	return serve(ctx, srv, ln)
}

func serve(ctx context.Context, srv *http.Server, ln net.Listener) error {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", ln.Addr().String())
		errCh <- srv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		// Serve stopped on its own: a real error, never ErrServerClosed here.
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down server", "timeout", shutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
