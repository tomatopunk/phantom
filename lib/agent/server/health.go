package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ServeHealth listens on addr and responds 200 OK to GET /health until ctx is canceled.
func ServeHealth(ctx context.Context, addr string) error {
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("health listen %s: %w", addr, err)
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(ctx)
	}()
	return srv.Serve(lis)
}

// ServeMetrics listens on addr and serves Prometheus metrics at GET /metrics until ctx is canceled.
func ServeMetrics(ctx context.Context, addr string) error {
	if addr == "" {
		return nil
	}
	registerMetrics()
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("metrics listen %s: %w", addr, err)
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(ctx)
	}()
	return srv.Serve(lis)
}
