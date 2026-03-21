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
