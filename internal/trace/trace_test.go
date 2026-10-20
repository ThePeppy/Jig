package trace

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestTraceStoreAppendAndBounded(t *testing.T) {
	store := NewStore(3)

	store.Append(Entry{Function: "op1"})
	store.Append(Entry{Function: "op2"})
	store.Append(Entry{Function: "op3"})
	store.Append(Entry{Function: "op4"})

	entries := store.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Function != "op4" || entries[2].Function != "op2" {
		t.Fatalf("order %q %q", entries[0].Function, entries[2].Function)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.ID == "" || seen[e.ID] {
			t.Fatalf("id %q not unique", e.ID)
		}
		seen[e.ID] = true
	}

	store.Clear()
	if len(store.Entries()) != 0 {
		t.Fatalf("expected 0 entries after clear")
	}
	store.Append(Entry{Function: "op5"})
	if seen[store.Entries()[0].ID] {
		t.Fatal("id reused after clear")
	}
}

func TestFormatDurationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{
		{12 * time.Millisecond, "12 毫秒"},
		{99 * time.Millisecond, "99 毫秒"},
		{100 * time.Millisecond, "0.1 秒"},
		{1200 * time.Millisecond, "1.2 秒"},
		{60 * time.Second, "60 秒"},
	} {
		if got := (Entry{Duration: tc.d}).FormatDuration(); got != tc.want {
			t.Errorf("%v: got %q want %q", tc.d, got, tc.want)
		}
	}
}

func TestClipKeepsRunesWhole(t *testing.T) {
	s := strings.Repeat("a", DetailLimit-1) + "反编译"
	got := Clip(s)
	if !strings.HasSuffix(got, "\n…") || !utf8.ValidString(got) {
		t.Fatalf("clip tail %q", got[len(got)-8:])
	}
	if Clip("short") != "short" {
		t.Fatal("short text changed")
	}
}
