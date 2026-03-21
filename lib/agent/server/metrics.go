package server

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	commandsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "phantom_commands_total", Help: "Total commands executed"},
		[]string{"session_id", "ok"},
	)
	sessionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "phantom_sessions_active", Help: "Number of active sessions"},
	)
	eventsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "phantom_events_total", Help: "Total events streamed"},
	)
	metricsOnce sync.Once
)

func registerMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(commandsTotal, sessionsActive, eventsTotal)
	})
}

// IncCommandsTotal increments the commands counter (call from Execute).
func IncCommandsTotal(sessionID string, ok bool) {
	registerMetrics()
	okStr := "false"
	if ok {
		okStr = "true"
	}
	commandsTotal.WithLabelValues(sessionID, okStr).Inc()
}

// SetSessionsActive sets the active sessions gauge (call when sessions change).
func SetSessionsActive(n int) {
	registerMetrics()
	sessionsActive.Set(float64(n))
}

// IncEventsTotal increments the events counter (call when sending an event).
func IncEventsTotal() {
	registerMetrics()
	eventsTotal.Inc()
}
