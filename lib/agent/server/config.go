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

// Config holds agent server options (address, auth, MCP, limits, health).
type Config struct {
	ListenAddr       string
	Token            string
	TokenPath        string // optional: re-read token from file for rotation
	EnableMCP        bool
	MCPAddr          string
	HealthAddr       string    // optional: HTTP /health on this address
	MetricsAddr      string    // optional: HTTP /metrics for Prometheus
	KprobeObjectPath string    // path to kprobe .o for legacy ringbuf pump (Linux)
	VmlinuxPath      string    // optional: vmlinux ELF for list disasm and BTF fallback when sysfs BTF is missing (Linux); see docs/vmlinux.md
	BpfIncludeDir    string    // path to bpf/include for C hook compile (e.g. ./src/agent/bpf/include)
	RateLimit        float64   // requests per second per session (0 = no limit)
	RateBurst        int       // burst size for rate limiter
	MaxBreak         int       // max breakpoints per session (0 = no limit)
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
		MaxHooks:   8,
		Audit:      nil,
	}
}
