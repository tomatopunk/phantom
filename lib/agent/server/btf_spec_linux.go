//go:build linux

package server

import (
	"log"

	"github.com/cilium/ebpf/btf"
)

// loadExecutorBTF loads kernel BTF for CO-RE and optional type queries.
// Tries /sys/kernel/btf/vmlinux first, then optional vmlinux ELF path.
func loadExecutorBTF(vmlinuxPath string) *btf.Spec {
	spec, err := btf.LoadKernelSpec()
	if err == nil {
		return spec
	}
	log.Printf("phantom: kernel BTF unavailable (%v); CO-RE compile may fail on this host", err)
	if vmlinuxPath == "" {
		return nil
	}
	fallback, err := btf.LoadSpec(vmlinuxPath)
	if err != nil {
		log.Printf("phantom: load BTF from vmlinux %q: %v", vmlinuxPath, err)
		return nil
	}
	return fallback
}
