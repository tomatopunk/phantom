package hook

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

const (
	hookProgramName = "hook_handler"
	hookMapEvents   = "events"
)

// AttachKprobeFromObject loads the compiled .o, attaches the kprobe to symbol, and opens a ringbuf reader for the hook's events map.
// The caller must run a pump reading from the returned reader and broadcast events to the session; when done, call detach().
// detach does not close the reader (the pump should close it); detach closes the link, collection, and runs cleanup.
func AttachKprobeFromObject(objectPath, symbol string, cleanup func()) (detach func(), reader *ringbuf.Reader, err error) {
	spec, err := ebpf.LoadCollectionSpec(objectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load hook spec: %w", err)
	}
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("new collection: %w", err)
	}
	prog, ok := coll.Programs[hookProgramName]
	if !ok {
		coll.Close()
		return nil, nil, fmt.Errorf("program %q not found", hookProgramName)
	}
	lk, err := link.Kprobe(symbol, prog, nil)
	if err != nil {
		coll.Close()
		return nil, nil, fmt.Errorf("attach kprobe: %w", err)
	}
	m, ok := coll.Maps[hookMapEvents]
	if !ok {
		lk.Close()
		coll.Close()
		return nil, nil, fmt.Errorf("hook map %q not found", hookMapEvents)
	}
	rd, err := ringbuf.NewReader(m)
	if err != nil {
		lk.Close()
		coll.Close()
		return nil, nil, fmt.Errorf("hook ringbuf reader: %w", err)
	}
	detachFn := func() {
		lk.Close()
		coll.Close()
		if cleanup != nil {
			cleanup()
		}
	}
	return detachFn, rd, nil
}
