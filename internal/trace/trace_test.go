package trace

import (
	"testing"
	"time"
)

func TestTraceStoreAppendAndBounded(t *testing.T) {
	store := NewStore(3)

	store.Append(Entry{Function: "op1", Target: "target1", Duration: 10 * time.Millisecond})
	store.Append(Entry{Function: "op2", Target: "target2", Duration: 20 * time.Millisecond})
	store.Append(Entry{Function: "op3", Target: "target3", Duration: 30 * time.Millisecond})
	store.Append(Entry{Function: "op4", Target: "target4", Duration: 40 * time.Millisecond})

	entries := store.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Newest first
	if entries[0].Function != "op4" {
		t.Fatalf("expected op4 to be first, got %s", entries[0].Function)
	}

	if entries[0].FormatDuration() == "" {
		t.Fatalf("expected non-empty formatted duration")
	}

	store.Clear()
	if len(store.Entries()) != 0 {
		t.Fatalf("expected 0 entries after clear")
	}
}
