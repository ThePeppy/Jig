// Package trace manages the telemetry and audit trail of MCP tool calls and worker operations.
package trace

import (
	"sync"
	"time"
)

// Entry records one operation or tool execution.
type Entry struct {
	ID        string        `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	Function  string        `json:"function"`
	Target    string        `json:"target"`
	Duration  time.Duration `json:"duration"`
	Status    string        `json:"status"` // "ok", "error", "running"
	Summary   string        `json:"summary"`
}

// FormatDuration returns a clean human-readable latency string.
func (e Entry) FormatDuration() string {
	if e.Duration < time.Millisecond {
		return e.Duration.Round(time.Microsecond).String()
	}
	if e.Duration < time.Second {
		return e.Duration.Round(time.Millisecond).String()
	}
	return e.Duration.Round(10 * time.Millisecond).String()
}

// Store holds the in-memory circular trace log buffer.
type Store struct {
	mu      sync.Mutex
	entries []Entry
	max     int
}

// NewStore creates a new trace store.
func NewStore(maxEntries int) *Store {
	if maxEntries <= 0 {
		maxEntries = 200
	}
	return &Store{
		entries: make([]Entry, 0, maxEntries),
		max:     maxEntries,
	}
}

// Append adds a new log entry to the front of the list.
func (s *Store) Append(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	s.entries = append([]Entry{e}, s.entries...)
	if len(s.entries) > s.max {
		s.entries = s.entries[:s.max]
	}
}

// Entries returns a copy of current recorded operations.
func (s *Store) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]Entry, len(s.entries))
	copy(res, s.entries)
	return res
}

// Clear drops all logs.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = s.entries[:0]
}
