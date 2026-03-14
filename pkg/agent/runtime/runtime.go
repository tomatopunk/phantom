package runtime

import "github.com/cilium/ebpf"

// Runtime loads eBPF programs, manages maps, and consumes ring buffer events.
type Runtime struct {
	collection      *ebpf.Collection      // kprobe object
	uprobeCollection *ebpf.Collection    // optional uprobe object
}

// New returns a new eBPF runtime (call LoadFromFile then AttachKprobe / OpenEventReader).
func New() *Runtime {
	return &Runtime{}
}
