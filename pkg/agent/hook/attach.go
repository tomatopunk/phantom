package hook

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

const hookProgramName = "hook_handler"

// AttachKprobeFromObject loads the compiled .o and attaches the kprobe to symbol.
// cleanup is called when detach runs (e.g. remove temp dir).
func AttachKprobeFromObject(objectPath, symbol string, cleanup func()) (detach func(), err error) {
	spec, err := ebpf.LoadCollectionSpec(objectPath)
	if err != nil {
		return nil, fmt.Errorf("load hook spec: %w", err)
	}
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("new collection: %w", err)
	}
	prog, ok := coll.Programs[hookProgramName]
	if !ok {
		coll.Close()
		return nil, fmt.Errorf("program %q not found", hookProgramName)
	}
	lk, err := link.Kprobe(symbol, prog, nil)
	if err != nil {
		coll.Close()
		return nil, fmt.Errorf("attach kprobe: %w", err)
	}
	return func() {
		lk.Close()
		coll.Close()
		if cleanup != nil {
			cleanup()
		}
	}, nil
}
