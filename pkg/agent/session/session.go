package session

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/cilium/ebpf/ringbuf"

	"github.com/tomatopunk/phantom/pkg/agent/expression"
	"github.com/tomatopunk/phantom/pkg/agent/runtime"
)

func fmtID(prefix string, n uint64) string {
	return fmt.Sprintf("%s-%d", prefix, n)
}

// Session holds state for one debug session (breakpoints, traces, runtime, last event).
type Session struct {
	ID   string
	mu   sync.RWMutex
	stop context.CancelFunc

	// Lazy-init runtime; load from kprobePath on first use.
	kprobePath string
	runtime    *runtime.Runtime

	// Event pump: reads from runtime ringbuf and broadcasts to subscribers.
	pumpCancel context.CancelFunc

	// Breakpoints, traces, hooks, and watches.
	breakpoints map[string]*BreakpointState
	traces      map[string]*TraceState
	hooks       map[string]*HookState
	watches     map[string]*WatchState
	nextBPID    uint64
	nextTraceID uint64
	nextHookID  uint64
	nextWatchID uint64

	// Last event for "print" resolution; updated by event pump.
	lastEvent atomic.Value // *runtime.Event

	// Event subscribers get a copy of each event (e.g. StreamEvents RPC).
	subscribersMu sync.Mutex
	subscribers   []chan<- *runtime.Event
}

// NewSession creates a session with the given id and optional kprobe object path.
func NewSession(id, kprobePath string) *Session {
	ctx, stop := context.WithCancel(context.Background())
	s := &Session{
		ID:          id,
		stop:        stop,
		kprobePath:  kprobePath,
		breakpoints: make(map[string]*BreakpointState),
		traces:      make(map[string]*TraceState),
		hooks:       make(map[string]*HookState),
		watches:     make(map[string]*WatchState),
	}
	_ = ctx
	return s
}

// EnsureRuntime loads the kprobe .o if path is set and returns the runtime (may be nil if path empty).
func (s *Session) EnsureRuntime() (*runtime.Runtime, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtime != nil {
		return s.runtime, nil
	}
	if s.kprobePath == "" {
		return nil, nil
	}
	r := runtime.New()
	if err := r.LoadFromFile(s.kprobePath); err != nil {
		return nil, err
	}
	s.runtime = r
	return s.runtime, nil
}

// Runtime returns the session's runtime (nil if not yet loaded).
func (s *Session) Runtime() *runtime.Runtime {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtime
}

// AddBreakpoint stores a breakpoint and returns its id.
func (s *Session) AddBreakpoint(symbol string, detach func(), isTemp bool) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextBreakpointIDLocked()
	s.breakpoints[id] = &BreakpointState{ID: id, Symbol: symbol, Detach: detach, Enabled: true, IsTemp: isTemp}
	return id
}

// SetBreakpointCondition sets the condition expression for a breakpoint.
func (s *Session) SetBreakpointCondition(id, condition string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if bp, ok := s.breakpoints[id]; ok {
		bp.Condition = condition
		return true
	}
	return false
}

func (s *Session) nextBreakpointIDLocked() string {
	s.nextBPID++
	return fmtID("bp", s.nextBPID)
}

// RemoveBreakpoint detaches and removes the breakpoint.
func (s *Session) RemoveBreakpoint(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	bp, ok := s.breakpoints[id]
	if !ok {
		return false
	}
	if bp.Detach != nil {
		bp.Detach()
	}
	delete(s.breakpoints, id)
	return true
}

// RemoveTemporaryBreakpointsOnHit detaches and removes all breakpoints marked IsTemp (tbreak).
// Called from the event pump on BREAK_HIT so that temporary breakpoints disappear after first hit.
func (s *Session) RemoveTemporaryBreakpointsOnHit() {
	list := s.ListBreakpoints()
	var toRemove []string
	for _, bp := range list {
		if bp.IsTemp {
			toRemove = append(toRemove, bp.ID)
		}
	}
	for _, id := range toRemove {
		s.RemoveBreakpoint(id)
	}
}

// ShouldReportBreakHit returns true if this BREAK_HIT event should be reported (set last event, broadcast, remove tbreaks).
// We only suppress when at least one breakpoint has a condition and every such condition fails for this event.
func (s *Session) ShouldReportBreakHit(ev *runtime.Event) bool {
	list := s.ListBreakpoints()
	var withCondition []*BreakpointState
	for _, bp := range list {
		if bp.Enabled && bp.Condition != "" {
			withCondition = append(withCondition, bp)
		}
	}
	if len(withCondition) == 0 {
		return true
	}
	for _, bp := range withCondition {
		if expression.ConditionPasses(ev, bp.Condition) {
			return true
		}
	}
	return false
}

// GetBreakpoint returns breakpoint state by id.
func (s *Session) GetBreakpoint(id string) *BreakpointState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.breakpoints[id]
}

// ListBreakpoints returns all breakpoint states.
func (s *Session) ListBreakpoints() []*BreakpointState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*BreakpointState, 0, len(s.breakpoints))
	for _, bp := range s.breakpoints {
		out = append(out, bp)
	}
	return out
}

// EnableBreakpoint enables a breakpoint (no-op if already enabled; re-attach if was disabled).
func (s *Session) EnableBreakpoint(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	bp, ok := s.breakpoints[id]
	if !ok {
		return false
	}
	if bp.Enabled && bp.Detach != nil {
		return true // already enabled and attached
	}
	if bp.Detach == nil {
		// Was disabled; re-attach so the breakpoint can fire again.
		if s.runtime == nil || bp.Symbol == "" {
			return false
		}
		detach, err := s.runtime.AttachKprobe(bp.Symbol)
		if err != nil {
			return false
		}
		bp.Detach = detach
	}
	bp.Enabled = true
	return true
}

// DisableBreakpoint marks breakpoint disabled (detach and keep entry; enable will re-attach).
func (s *Session) DisableBreakpoint(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	bp, ok := s.breakpoints[id]
	if !ok {
		return false
	}
	if bp.Detach != nil {
		bp.Detach()
		bp.Detach = nil
	}
	bp.Enabled = false
	return true
}

// AddTrace stores a trace and returns its id.
func (s *Session) AddTrace(expressions []string, detach func()) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextTraceIDLocked()
	s.traces[id] = &TraceState{ID: id, Expressions: expressions, Detach: detach}
	return id
}

func (s *Session) nextTraceIDLocked() string {
	s.nextTraceID++
	return fmtID("trace", s.nextTraceID)
}

// RemoveTrace removes and detaches a trace.
func (s *Session) RemoveTrace(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tr, ok := s.traces[id]
	if !ok {
		return false
	}
	if tr.Detach != nil {
		tr.Detach()
	}
	delete(s.traces, id)
	return true
}

// ListTraces returns all trace states.
func (s *Session) ListTraces() []*TraceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*TraceState, 0, len(s.traces))
	for _, tr := range s.traces {
		out = append(out, tr)
	}
	return out
}

// AddHook stores a hook, starts an event pump reading from the hook's ringbuf, and returns its id.
// When the hook is removed or the session stops, the pump is cancelled and detach is called.
// limit is 0 for no limit; when > 0 the hook auto-detaches after that many events.
func (s *Session) AddHook(attachPoint string, detach func(), reader *ringbuf.Reader, limit int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextHookIDLocked()
	ctx, cancel := context.WithCancel(context.Background())
	s.hooks[id] = &HookState{ID: id, AttachPoint: attachPoint, Detach: detach, Cancel: cancel, Limit: limit}
	go runHookEventPump(ctx, s, reader, id)
	return id
}

// IncrementHookHitCount increments the hook's hit count and returns the new count and its limit.
// Returns ok=false if the hook was not found (e.g. already removed).
func (s *Session) IncrementHookHitCount(hookID string) (newCount, limit int, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.hooks[hookID]
	if !ok {
		return 0, 0, false
	}
	h.HitCount++
	return h.HitCount, h.Limit, true
}

func (s *Session) nextHookIDLocked() string {
	s.nextHookID++
	return fmtID("hook", s.nextHookID)
}

// RemoveHook cancels the hook's event pump (so the reader is closed) and detaches the hook.
func (s *Session) RemoveHook(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.hooks[id]
	if !ok {
		return false
	}
	if h.Cancel != nil {
		h.Cancel()
	}
	if h.Detach != nil {
		h.Detach()
	}
	delete(s.hooks, id)
	return true
}

// ListHooks returns all hook states.
func (s *Session) ListHooks() []*HookState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*HookState, 0, len(s.hooks))
	for _, h := range s.hooks {
		out = append(out, h)
	}
	return out
}

// AddWatch stores a watch expression and returns its id.
func (s *Session) AddWatch(expr string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextWatchIDLocked()
	s.watches[id] = &WatchState{ID: id, Expression: expr}
	return id
}

func (s *Session) nextWatchIDLocked() string {
	s.nextWatchID++
	return fmtID("watch", s.nextWatchID)
}

// RemoveWatch removes a watch by id.
func (s *Session) RemoveWatch(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.watches[id]; !ok {
		return false
	}
	delete(s.watches, id)
	return true
}

// ListWatches returns all watch states.
func (s *Session) ListWatches() []*WatchState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*WatchState, 0, len(s.watches))
	for _, w := range s.watches {
		out = append(out, w)
	}
	return out
}

// EvaluateWatchChanges evaluates all watch expressions against ev, updates last values,
// and returns triggers for watches whose value changed.
func (s *Session) EvaluateWatchChanges(ev *runtime.Event) []WatchTrigger {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []WatchTrigger
	for _, w := range s.watches {
		newVal := expression.Evaluate(ev, w.Expression)
		if w.HasValue && w.LastValue != newVal {
			out = append(out, WatchTrigger{ID: w.ID, Expression: w.Expression, OldValue: w.LastValue, NewValue: newVal})
		}
		w.LastValue = newVal
		w.HasValue = true
	}
	return out
}

// EvaluateTraceSamples evaluates all trace expressions against ev and returns one TraceSampleResult per trace.
func (s *Session) EvaluateTraceSamples(ev *runtime.Event) []TraceSampleResult {
	s.mu.RLock()
	traces := make([]*TraceState, 0, len(s.traces))
	for _, tr := range s.traces {
		traces = append(traces, tr)
	}
	s.mu.RUnlock()
	if len(traces) == 0 {
		return nil
	}
	out := make([]TraceSampleResult, 0, len(traces))
	for _, tr := range traces {
		values := make(map[string]string, len(tr.Expressions))
		for _, expr := range tr.Expressions {
			values[expr] = expression.Evaluate(ev, expr)
		}
		out = append(out, TraceSampleResult{TraceID: tr.ID, Expressions: tr.Expressions, Values: values})
	}
	return out
}

// SetLastEvent updates the last event for "print" resolution.
func (s *Session) SetLastEvent(ev *runtime.Event) {
	if ev == nil {
		return
	}
	s.lastEvent.Store(ev)
}

// GetLastEvent returns the most recent event (may be nil).
func (s *Session) GetLastEvent() *runtime.Event {
	v := s.lastEvent.Load()
	if v == nil {
		return nil
	}
	return v.(*runtime.Event)
}

// SubscribeEvents adds a channel that will receive a copy of each event; call Unsubscribe to remove.
func (s *Session) SubscribeEvents(ch chan<- *runtime.Event) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	s.subscribers = append(s.subscribers, ch)
}

// UnsubscribeEvents removes the channel from subscribers.
func (s *Session) UnsubscribeEvents(ch chan<- *runtime.Event) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	for i, c := range s.subscribers {
		if c == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			return
		}
	}
}

// BroadcastEvent sends a copy of the event to all subscribers (non-blocking).
func (s *Session) BroadcastEvent(ev *runtime.Event) {
	s.subscribersMu.Lock()
	list := make([]chan<- *runtime.Event, len(s.subscribers))
	copy(list, s.subscribers)
	s.subscribersMu.Unlock()
	for _, ch := range list {
		select {
		case ch <- ev:
		default:
			// drop if full to avoid blocking reader
		}
	}
}

// EnsureEventPump starts the ringbuf reader goroutine if runtime is loaded and pump not yet started.
func (s *Session) EnsureEventPump() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pumpCancel != nil || s.runtime == nil {
		return s.pumpCancel != nil
	}
	reader, err := s.runtime.OpenEventReader()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.pumpCancel = cancel
	go runEventPump(ctx, s, reader)
	return true
}

// Stop cancels the session context and releases all resources (detach, close runtime).
func (s *Session) Stop() {
	s.mu.Lock()
	if s.pumpCancel != nil {
		s.pumpCancel()
		s.pumpCancel = nil
	}
	if s.stop != nil {
		s.stop()
		s.stop = nil
	}
	for _, bp := range s.breakpoints {
		if bp.Detach != nil {
			bp.Detach()
		}
	}
	s.breakpoints = make(map[string]*BreakpointState)
	for _, tr := range s.traces {
		if tr.Detach != nil {
			tr.Detach()
		}
	}
	s.traces = make(map[string]*TraceState)
	for _, h := range s.hooks {
		if h.Cancel != nil {
			h.Cancel()
		}
		if h.Detach != nil {
			h.Detach()
		}
	}
	s.hooks = make(map[string]*HookState)
	s.watches = make(map[string]*WatchState)
	if s.runtime != nil {
		s.runtime.Close()
		s.runtime = nil
	}
	s.mu.Unlock()
}
