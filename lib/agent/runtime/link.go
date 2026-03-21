package runtime

import (
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// attachKprobe attaches prog to the kernel symbol and returns a link that can be closed to detach.
func attachKprobe(prog *ebpf.Program, symbol string) (link.Link, error) {
	return link.Kprobe(symbol, prog, nil)
}

// attachUprobe attaches prog to the user binary at the given symbol (resolved by link.OpenExecutable).
func attachUprobe(prog *ebpf.Program, binaryPath, symbol string) (link.Link, error) {
	ex, err := link.OpenExecutable(binaryPath)
	if err != nil {
		return nil, err
	}
	return ex.Uprobe(symbol, prog, nil)
}

// attachUretprobe attaches prog as a uretprobe at the given symbol.
func attachUretprobe(prog *ebpf.Program, binaryPath, symbol string) (link.Link, error) {
	ex, err := link.OpenExecutable(binaryPath)
	if err != nil {
		return nil, err
	}
	return ex.Uretprobe(symbol, prog, nil)
}
