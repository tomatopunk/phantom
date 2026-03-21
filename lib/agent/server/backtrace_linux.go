//go:build linux

package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/tomatopunk/phantom/lib/agent/runtime"
)

// readKernelStack tries to read /proc/<tid>/stack for the thread from the last event.
// Prefers pid (thread id) then falls back to tgid. Returns stack text or an error message.
func readKernelStack(ev *runtime.Event) string {
	if ev == nil {
		return "bt: no event yet (hit a breakpoint first)"
	}
	tid := ev.PID
	if tid == 0 {
		tid = ev.Tgid
	}
	if tid == 0 {
		return "bt: no pid/tgid in event"
	}
	// Build /proc/<tid>/stack without literal path separator in Join (gocritic filepathJoin).
	path := string(filepath.Separator) + filepath.Join("proc", strconv.FormatUint(uint64(tid), 10), "stack")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("bt: thread %d not found (may have exited)", tid)
		}
		if os.IsPermission(err) {
			return fmt.Sprintf("bt: cannot read %s (permission denied; try root)", path)
		}
		return fmt.Sprintf("bt: %v", err)
	}
	return "bt:\n" + string(b)
}
