package server

import (
	"context"
	"fmt"
	"net"

	"github.com/tomatopunk/phantom/lib/agent/session"
	"github.com/tomatopunk/phantom/lib/proto"
	"google.golang.org/grpc"
)

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

	mgr := session.NewManager(cfg.KprobeObjectPath)
	dbg := NewDebuggerServerWithConfig(mgr, PrepareServerConfig(cfg))

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
		sc.btfSpec = loadExecutorBTF("")
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
	sc.btfSpec = loadExecutorBTF(sc.vmlinuxPath)
	return sc
}

// BuildServerConfig builds serverConfig from Config (exported for MCP-only mode).
func BuildServerConfig(cfg *Config) *serverConfig {
	if cfg == nil {
		return nil
	}
	if cfg.RateLimit <= 0 && cfg.MaxBreak == 0 && cfg.MaxTrace == 0 && cfg.MaxHooks == 0 && cfg.Audit == nil && cfg.BpfIncludeDir == "" && cfg.VmlinuxPath == "" {
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
