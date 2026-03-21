// Copyright 2026 The Phantom Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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
