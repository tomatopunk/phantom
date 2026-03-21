package hook

import (
	"github.com/cilium/ebpf/ringbuf"
)

// HookTemplateProgramName is the BPF function name in embed/hook.c after template expansion.
const HookTemplateProgramName = "hook_handler"

// AttachKprobeFromObject loads the compiled .o, attaches the kprobe to symbol, and opens a ringbuf reader for the hook's events map.
// The caller must run a pump reading from the returned reader and broadcast events to the session; when done, call detach().
// detach does not close the reader (the pump should close it); detach closes the link, collection, and runs cleanup.
func AttachKprobeFromObject(objectPath, symbol string, cleanup func()) (detach func(), reader *ringbuf.Reader, err error) {
	pa := &ParsedAttach{Kind: AttachKprobe, KprobeSymbol: symbol}
	return AttachProbeFromObject(objectPath, pa, HookTemplateProgramName, cleanup)
}
