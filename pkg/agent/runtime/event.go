package runtime

// Event type constants; match bpf/include/common.h and proto.
const (
	EventTypeBreakHit = 1
)

// Event is a single debug event from the eBPF ring buffer (matches event_header; optional Args/Ret/Comm from payload).
type Event struct {
	TimestampNs uint64
	SessionID   uint32
	EventType   uint32
	PID         uint32
	Tgid        uint32
	CPU         uint32
	ProbeID     uint32
	Payload     []byte
	// Optional: filled from payload when BPF sends pt_regs / ABI (arg0-arg5, ret, comm).
	Args [6]uint64
	Ret  uint64
	Comm string
}
