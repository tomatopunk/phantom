// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("[phantom] health HTTP listening on %s (/health)", lis.Addr().String())
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
	log.Printf("[phantom] metrics HTTP listening on %s (/metrics)", lis.Addr().String())
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(ctx)
	}()
	return srv.Serve(lis)
}
