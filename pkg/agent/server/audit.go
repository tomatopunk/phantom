package server

import (
	"io"
	"log"
	"sync"
)

// AuditLog writes one line per command (session_id, command, ok, err) for audit trail.
type AuditLog struct {
	mu     sync.Mutex
	logger *log.Logger
}

// NewAuditLog creates an audit logger writing to w (e.g. os.Stderr or a file).
func NewAuditLog(w io.Writer) *AuditLog {
	return &AuditLog{
		logger: log.New(w, "[audit] ", log.LstdFlags),
	}
}

// LogCommand records a command execution (call from Execute handler).
func (a *AuditLog) LogCommand(sessionID, commandLine string, ok bool, errMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	status := "ok"
	if !ok {
		status = "err"
	}
	if errMsg != "" {
		status += " " + errMsg
	}
	a.logger.Printf("%s %s %s", sessionID, commandLine, status)
}

// NopAuditLog is a no-op audit log for when auditing is disabled.
type NopAuditLog struct{}

func (NopAuditLog) LogCommand(_, _ string, _ bool, _ string) {}
