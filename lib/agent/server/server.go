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
	"net"

	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
	"google.golang.org/grpc"
)

// SessionManagerForAgent returns a session manager with quota sink wired when PrepareServerConfig sets quota.
func SessionManagerForAgent(cfg *Config) *session.Manager {
	if cfg == nil {
		return session.NewManager("", nil)
	}
	sc := PrepareServerConfig(cfg)
	return session.NewManager(cfg.KprobeObjectPath, QuotaSessionSink(sc.quota))
}

// Run starts the gRPC debugger server and blocks until ctx is canceled.
func Run(ctx context.Context, cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.ListenAddr, err)
	}
	defer listener.Close()

	sc := PrepareServerConfig(cfg)
	mgr := session.NewManager(cfg.KprobeObjectPath, QuotaSessionSink(sc.quota))
	dbg := NewDebuggerServerWithConfig(mgr, sc)

	if cfg.HealthAddr != "" {
		go func() { _ = ServeHealth(ctx, cfg.HealthAddr) }()
	}
	if cfg.MetricsAddr != "" {
		go func() { _ = ServeMetrics(ctx, cfg.MetricsAddr) }()
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(authUnaryInterceptor(cfg.Token)),
		grpc.ChainStreamInterceptor(authStreamInterceptor(cfg.Token)),
	)
	proto.RegisterDebuggerServiceServer(srv, dbg)

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	if err := srv.Serve(listener); err != nil && ctx.Err() == nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// PrepareServerConfig merges Config into serverConfig, loads kernel BTF when possible, and is used by Run and MCP.
func PrepareServerConfig(cfg *Config) *serverConfig {
	if cfg == nil {
		sc := &serverConfig{}
		spec, autoELF := loadExecutorBTF("")
		sc.btfSpec = spec
		if sc.vmlinuxPath == "" && autoELF != "" {
			sc.vmlinuxPath = autoELF
		}
		return sc
	}
	sc := BuildServerConfig(cfg)
	if sc == nil {
		sc = &serverConfig{}
	}
	if sc.bpfIncludeDir == "" {
		sc.bpfIncludeDir = cfg.BpfIncludeDir
	}
	if sc.vmlinuxPath == "" {
		sc.vmlinuxPath = cfg.VmlinuxPath
	}
	spec, autoELF := loadExecutorBTF(sc.vmlinuxPath)
	sc.btfSpec = spec
	if sc.vmlinuxPath == "" && autoELF != "" {
		sc.vmlinuxPath = autoELF
	}
	return sc
}

// BuildServerConfig builds serverConfig from Config (exported for MCP-only mode).
func BuildServerConfig(cfg *Config) *serverConfig {
	if cfg == nil {
		return nil
	}
	if cfg.RateLimit <= 0 && cfg.MaxBreak == 0 && cfg.MaxTrace == 0 && cfg.MaxHooks == 0 &&
		cfg.Audit == nil && cfg.BpfIncludeDir == "" && cfg.VmlinuxPath == "" {
		return nil
	}
	// Build config so that BpfIncludeDir or other options are applied when only one is set.
	sc := &serverConfig{}
	if cfg.RateLimit > 0 {
		burst := cfg.RateBurst
		if burst <= 0 {
			burst = 20
		}
		sc.rateLimiter = NewRateLimiter(cfg.RateLimit, burst)
	}
	if cfg.MaxBreak > 0 || cfg.MaxTrace > 0 || cfg.MaxHooks > 0 {
		sc.quota = NewSessionQuota(cfg.MaxBreak, cfg.MaxTrace, cfg.MaxHooks)
	}
	if cfg.Audit != nil {
		sc.audit = cfg.Audit
	}
	sc.bpfIncludeDir = cfg.BpfIncludeDir
	sc.vmlinuxPath = cfg.VmlinuxPath
	return sc
}
