// Package catalog is the in-memory library. Disk lives in desk.
package catalog

import (
	"os"
	"path/filepath"
	"time"

	"jig/internal/intake"
)

// Entry is one file the user kept. Path is what idalib will open. Source
// is what the user picked; they differ when a companion database exists.
type Entry struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Companion bool      `json:"companion"`
	Added     time.Time `json:"added"`
}

// Missing reports that the open path is gone.
func (e Entry) Missing() bool {
	_, err := os.Stat(e.Path)
	return err != nil
}

// Catalog is the list the library page shows.
type Catalog struct {
	entries  []Entry
	selected string
	seq      int
	now      func() time.Time
	nextID   func() string
}

// New returns an empty catalog.
func New() *Catalog {
	c := &Catalog{
		now: time.Now,
	}
	c.nextID = func() string {
		c.seq++
		return itoa(c.seq)
	}
	return c
}

// Restore replaces the entries, repairing any empty or duplicate IDs and
// advancing the ID sequence past all existing entries. The first entry is
// selected when present.
func (c *Catalog) Restore(entries []Entry) {
	c.entries = make([]Entry, 0, len(entries))
	for _, e := range entries {
		if v := atoi(e.ID); v > c.seq {
			c.seq = v
		}
	}
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.ID == "" || seen[e.ID] {
			e.ID = c.nextID()
		}
		seen[e.ID] = true
		c.entries = append(c.entries, e)
	}
	c.selected = ""
	if len(c.entries) > 0 {
		c.selected = c.entries[0].ID
	}
}

// Entries returns the library in add order.
func (c *Catalog) Entries() []Entry {
	return append([]Entry(nil), c.entries...)
}

// Selected returns the entry the window is showing.
func (c *Catalog) Selected() (Entry, bool) {
	for _, e := range c.entries {
		if e.ID == c.selected {
			return e, true
		}
	}
	if len(c.entries) > 0 {
		return c.entries[0], true
	}
	return Entry{}, false
}

// Select chooses id when it is in the library.
func (c *Catalog) Select(id string) {
	for _, e := range c.entries {
		if e.ID == id {
			c.selected = id
			return
		}
	}
}

// Add accepts path. The bool is false when the file was already kept.
func (c *Catalog) Add(path string) (Entry, bool, error) {
	decision, err := intake.Accept(path)
	if err != nil {
		return Entry{}, false, err
	}
	for _, e := range c.entries {
		if same(e.Path, decision.OpenPath) || same(e.Source, decision.Source) || same(e.Path, decision.Source) {
			c.selected = e.ID
			return e, false, nil
		}
	}
	e := Entry{
		ID:        c.nextID(),
		Source:    decision.Source,
		Path:      decision.OpenPath,
		Name:      decision.Name,
		Kind:      decision.Kind,
		Companion: decision.Companion,
		Added:     c.now(),
	}
	c.entries = append(c.entries, e)
	c.selected = e.ID
	return e, true, nil
}

// Remove deletes id. The nearest remaining entry becomes selected.
func (c *Catalog) Remove(id string) {
	next := make([]Entry, 0, len(c.entries))
	drop := -1
	for i, e := range c.entries {
		if e.ID == id {
			drop = i
			continue
		}
		next = append(next, e)
	}
	if drop < 0 {
		return
	}
	c.entries = next
	if c.selected != id {
		return
	}
	c.selected = ""
	if len(c.entries) == 0 {
		return
	}
	i := drop
	if i >= len(c.entries) {
		i = len(c.entries) - 1
	}
	c.selected = c.entries[i].ID
}

func same(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func atoi(s string) int {
	if s == "" {
		return 0
	}
	n := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
