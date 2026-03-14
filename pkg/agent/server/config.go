package server

// Config holds agent server options (address, auth, MCP, limits, health).
type Config struct {
	ListenAddr       string
	Token            string
	TokenPath        string // optional: re-read token from file for rotation
	EnableMCP        bool
	MCPAddr          string
	HealthAddr       string    // optional: HTTP /health on this address
	MetricsAddr      string    // optional: HTTP /metrics for Prometheus
	KprobeObjectPath string    // path to kprobe .o for real break/trace (Linux)
	BpfIncludeDir    string    // path to bpf/include for C hook compile (e.g. ./bpf/include)
	RateLimit        float64   // requests per second per session (0 = no limit)
	RateBurst        int       // burst size for rate limiter
	MaxBreak         int       // max breakpoints per session (0 = no limit)
	MaxTrace         int       // max traces per session (0 = no limit)
	MaxHooks         int       // max hooks per session (0 = no limit)
	Audit            *AuditLog // optional audit logger
}

// DefaultConfig returns a config suitable for local dev.
func DefaultConfig() Config {
	return Config{
		ListenAddr: ":9090",
		Token:      "",
		HealthAddr: "",
		RateLimit:  100,
		RateBurst:  20,
		MaxBreak:   64,
		MaxTrace:   32,
		MaxHooks:   8,
		Audit:      nil,
	}
}
