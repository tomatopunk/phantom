//go:build !linux

package server

import "github.com/cilium/ebpf/btf"

func loadExecutorBTF(_ string) *btf.Spec {
	return nil
}
