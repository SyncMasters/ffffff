package httpapi

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/johan-larp/agentsearch/internal/config"
)

// transportLog deliberately discards net/http diagnostic text, which may include
// untrusted headers, URLs or panic values. Handler logs contain safe fields only.
type transportLog struct{ log *slog.Logger }

func (w transportLog) Write(b []byte) (int, error) {
	w.log.Warn("HTTP transport error")
	return len(b), nil
}

// Serve owns the listener and drains/cancels work on shutdown. Search contexts
// derive from both the incoming request and this server lifetime.
func Serve(ctx context.Context, listener net.Listener, handler http.Handler, cfg config.ServerConfig, logger *slog.Logger) error {
	if e := cfg.Validate(); e != nil {
		listener.Close()
		return e
	}
	if logger == nil {
		logger = slog.Default()
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: min(5*time.Second, cfg.ReadTimeout), ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout, MaxHeaderBytes: 16 << 10, ErrorLog: log.New(transportLog{logger}, "", 0), BaseContext: func(net.Listener) context.Context { return ctx }}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errors.New("HTTP server stopped unexpectedly")
	case <-ctx.Done():
		// Background context is only for cleanup, never for executing a search.
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := server.Shutdown(shutdown)
		if err != nil {
			_ = server.Close()
		}
		<-done
		if err != nil {
			return errors.New("HTTP shutdown exceeded grace period")
		}
		return nil
	}
}
