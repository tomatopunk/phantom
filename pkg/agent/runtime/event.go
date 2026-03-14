package runtime

// Event is a single debug event from the eBPF ring buffer (matches event_header).
type Event struct {
	TimestampNs uint64
	SessionID   uint32
	EventType   uint32
	PID         uint32
	Tgid        uint32
	CPU         uint32
	ProbeID     uint32
	Payload     []byte
}
