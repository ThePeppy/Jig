// Package trace keeps the in-memory record of MCP tool calls and worker
// operations. It does not write to disk and does not format for a window
// beyond the duration label.
package trace

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

// DetailLimit bounds the arguments and result one entry keeps.
const DetailLimit = 64 << 10

// Entry records one operation or tool execution.
type Entry struct {
	ID        string        `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	Function  string        `json:"function"`
	Target    string        `json:"target"`
	Duration  time.Duration `json:"duration"`
	Status    string        `json:"status"` // "ok", "error", "running"
	Summary   string        `json:"summary"`
	// Args is the call's input as indented JSON, empty when it had none.
	Args string `json:"args,omitempty"`
	// Result is what the call returned, or the error text.
	Result string `json:"result,omitempty"`
}

// Failed reports an entry that ended in an error.
func (e Entry) Failed() bool { return e.Status == "error" }

// FormatDuration is the latency as a list shows it.
func (e Entry) FormatDuration() string {
	d := e.Duration
	switch {
	case d < 100*time.Millisecond:
		return fmt.Sprintf("%d 毫秒", d.Milliseconds())
	case d < 10*time.Second:
		return fmt.Sprintf("%.1f 秒", d.Seconds())
	default:
		return fmt.Sprintf("%d 秒", int(d.Round(time.Second).Seconds()))
	}
}

// Clip cuts s to DetailLimit bytes on a rune boundary.
func Clip(s string) string {
	if len(s) <= DetailLimit {
		return s
	}
	cut := DetailLimit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "\n…"
}

// Store holds the in-memory ring of entries, newest first.
type Store struct {
	mu      sync.Mutex
	entries []Entry
	max     int
	seq     uint64
}

// NewStore creates a store keeping at most maxEntries.
func NewStore(maxEntries int) *Store {
	if maxEntries <= 0 {
		maxEntries = 200
	}
	return &Store{
		entries: make([]Entry, 0, maxEntries+1),
		max:     maxEntries,
	}
}

// Max is how many entries the store keeps.
func (s *Store) Max() int { return s.max }

// Append adds e to the front and gives it an ID when it has none.
func (s *Store) Append(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if e.ID == "" {
		s.seq++
		e.ID = strconv.FormatUint(s.seq, 10)
	}
	s.entries = append(s.entries, Entry{})
	copy(s.entries[1:], s.entries)
	s.entries[0] = e
	if len(s.entries) > s.max {
		s.entries = s.entries[:s.max]
	}
}

// ReplaceSummary rewrites matching summaries in place. Close uses it so a
// launch row does not keep saying the worker is ready after it has stopped.
func (s *Store) ReplaceSummary(from, to string) {
	if from == "" || from == to {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].Summary == from {
			s.entries[i].Summary = to
		}
	}
}

// Update rewrites every entry match accepts. Close uses it when a worker
// that was still packing has actually exited.
func (s *Store) Update(match func(Entry) bool, update func(*Entry)) {
	if match == nil || update == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if match(s.entries[i]) {
			update(&s.entries[i])
		}
	}
}

// Entries returns a copy of the current entries, newest first.
func (s *Store) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]Entry, len(s.entries))
	copy(res, s.entries)
	return res
}

// Clear drops all entries.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.entries)
	s.entries = s.entries[:0]
}
