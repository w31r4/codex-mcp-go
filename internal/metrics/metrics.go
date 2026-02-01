package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	totalRequests   atomic.Int64
	successRequests atomic.Int64
	failedRequests  atomic.Int64

	totalLatencyMs atomic.Int64
	maxLatencyMs   atomic.Int64
	minLatencyMs   atomic.Int64 // 0 means "unset"

	mu          sync.RWMutex
	toolCalls   map[string]*atomic.Int64
	errorCounts map[string]*atomic.Int64
}

type Snapshot struct {
	TotalRequests   int64            `json:"total_requests"`
	SuccessRequests int64            `json:"success_requests"`
	FailedRequests  int64            `json:"failed_requests"`
	AvgLatencyMs    int64            `json:"avg_latency_ms"`
	MaxLatencyMs    int64            `json:"max_latency_ms"`
	MinLatencyMs    int64            `json:"min_latency_ms"`
	ToolCalls       map[string]int64 `json:"tool_calls"`
	ErrorCounts     map[string]int64 `json:"error_counts"`
}

func New() *Metrics {
	return &Metrics{
		toolCalls:   make(map[string]*atomic.Int64),
		errorCounts: make(map[string]*atomic.Int64),
	}
}

func (m *Metrics) RecordRequest(toolName string, success bool, latency time.Duration) {
	m.totalRequests.Add(1)
	if success {
		m.successRequests.Add(1)
	} else {
		m.failedRequests.Add(1)
	}

	latencyMs := latency.Milliseconds()
	m.totalLatencyMs.Add(latencyMs)
	m.updateMaxLatency(latencyMs)
	m.updateMinLatency(latencyMs)

	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.toolCalls[toolName]
	if !ok {
		c = &atomic.Int64{}
		m.toolCalls[toolName] = c
	}
	c.Add(1)
}

func (m *Metrics) RecordError(codeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.errorCounts[codeName]
	if !ok {
		c = &atomic.Int64{}
		m.errorCounts[codeName] = c
	}
	c.Add(1)
}

func (m *Metrics) Snapshot() Snapshot {
	total := m.totalRequests.Load()
	avg := int64(0)
	if total > 0 {
		avg = m.totalLatencyMs.Load() / total
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	toolCalls := make(map[string]int64, len(m.toolCalls))
	for k, v := range m.toolCalls {
		toolCalls[k] = v.Load()
	}

	errorCounts := make(map[string]int64, len(m.errorCounts))
	for k, v := range m.errorCounts {
		errorCounts[k] = v.Load()
	}

	return Snapshot{
		TotalRequests:   total,
		SuccessRequests: m.successRequests.Load(),
		FailedRequests:  m.failedRequests.Load(),
		AvgLatencyMs:    avg,
		MaxLatencyMs:    m.maxLatencyMs.Load(),
		MinLatencyMs:    m.minLatencyMs.Load(),
		ToolCalls:       toolCalls,
		ErrorCounts:     errorCounts,
	}
}

func (m *Metrics) updateMaxLatency(latencyMs int64) {
	for {
		max := m.maxLatencyMs.Load()
		if latencyMs <= max {
			return
		}
		if m.maxLatencyMs.CompareAndSwap(max, latencyMs) {
			return
		}
	}
}

func (m *Metrics) updateMinLatency(latencyMs int64) {
	for {
		min := m.minLatencyMs.Load()
		if min == 0 || latencyMs < min {
			if m.minLatencyMs.CompareAndSwap(min, latencyMs) {
				return
			}
			continue
		}
		return
	}
}
