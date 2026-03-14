// Command agent runs the Phantom agent (gRPC or MCP stdio).
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tomatopunk/phantom/pkg/agent/mcp"
	"github.com/tomatopunk/phantom/pkg/agent/server"
	"github.com/tomatopunk/phantom/pkg/agent/session"
)

func main() {
	listen := flag.String("listen", ":9090", "gRPC listen address")
	token := flag.String("token", os.Getenv("PHANTOM_TOKEN"), "optional bearer token")
	health := flag.String("health", os.Getenv("PHANTOM_HEALTH"), "optional health HTTP address (e.g. :8080)")
	metrics := flag.String("metrics", os.Getenv("PHANTOM_METRICS"), "optional Prometheus metrics HTTP address (e.g. :9091)")
	kprobe := flag.String("kprobe", os.Getenv("PHANTOM_KPROBE"), "path to kprobe .o for real break/trace (Linux)")
	vmlinux := flag.String("vmlinux", os.Getenv("PHANTOM_VMLINUX"), "optional path to vmlinux for list disasm (Linux)")
	bpfInclude := flag.String("bpf-include", os.Getenv("PHANTOM_BPF_INCLUDE"), "path to bpf/include for C hook compile")
	enableMCP := flag.Bool("mcp", false, "run MCP server on stdio instead of gRPC")
	flag.Parse()

	cfg := server.DefaultConfig()
	cfg.ListenAddr = *listen
	cfg.Token = *token
	cfg.HealthAddr = *health
	cfg.MetricsAddr = *metrics
	cfg.KprobeObjectPath = *kprobe
	cfg.VmlinuxPath = *vmlinux
	cfg.BpfIncludeDir = *bpfInclude

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *enableMCP {
		mgr := session.NewManager(cfg.KprobeObjectPath)
		serverCfg := server.BuildServerConfig(&cfg)
		dbg := server.NewDebuggerServerWithConfig(mgr, serverCfg)
		backend := server.NewMCPServerBackend(dbg)
		mcpSrv := mcp.NewServer(backend)
		if err := mcpSrv.Run(ctx); err != nil && ctx.Err() == nil {
			stop()
			log.Fatalf("mcp: %v", err) //nolint:gocritic // exitAfterDefer: stop() called explicitly before exit
		}
		return
	}

	if err := server.Run(ctx, &cfg); err != nil && ctx.Err() == nil {
		log.Fatalf("agent: %v", err)
	}
}
