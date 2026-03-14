//go:build !linux

package server

import "github.com/tomatopunk/phantom/pkg/agent/runtime"

func readKernelStack(ev *runtime.Event) string {
	if ev == nil {
		return "bt: no event yet (hit a breakpoint first)"
	}
	return "bt: backtrace not supported on this platform"
}
