package desk

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"jig/internal/session"
	"jig/internal/trace"
)

func TestTakePersistsAndRejectsADirectory(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "firmware.bin")
	if err := os.WriteFile(bin, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	d.Take([]string{bin, root}, nil)
	if d.Notice == "" {
		t.Fatal("expected a notice for the directory")
	}
	if len(d.Catalog.Entries()) != 1 {
		t.Fatalf("entries %+v", d.Catalog.Entries())
	}

	again, err := Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	entries := again.Catalog.Entries()
	if len(entries) != 1 || entries[0].Name != "firmware.bin" {
		t.Fatalf("%+v", entries)
	}
}

func TestAddPathSelectsLibrary(t *testing.T) {
	root := t.TempDir()
	d, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	d.NotifyUI = func() { n++ }
	d.Tab = TabSettings
	path := filepath.Join(root, "rom.img")
	if err := os.WriteFile(path, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	if d.Tab != TabLibrary {
		t.Fatalf("tab %v", d.Tab)
	}
	sidecar := filepath.Join(root, "nope.id0")
	if err := os.WriteFile(sidecar, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(sidecar); err != nil {
		t.Fatal(err)
	}
	if len(d.Catalog.Entries()) != 2 {
		t.Fatalf("entries %d", len(d.Catalog.Entries()))
	}
	if n != 2 {
		t.Fatalf("notify %d", n)
	}
}

func TestCloseTraceLeavesPackingUntilTheWorkerExits(t *testing.T) {
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &session.Session{
		ID:         "s-stop",
		TargetName: "firmware.bin",
		SourcePath: "/tmp/firmware.bin",
		State:      session.StateStopping,
		PID:        97100,
	}
	d.Sessions.Register(s)
	d.Traces.Append(trace.Entry{
		Function:  "session.close",
		Target:    "firmware.bin",
		Duration:  638 * time.Microsecond,
		Status:    "ok",
		Summary:   "正在收拢数据库",
		Timestamp: time.Now().Add(-2 * time.Second),
	})

	d.settleCloseTraces()
	if got := d.Traces.Entries()[0].Summary; got != "正在收拢数据库" {
		t.Fatalf("settled while packing: %s", got)
	}

	s.State = session.StateClosed
	d.settleCloseTraces()
	got := d.Traces.Entries()[0]
	if got.Summary != "数据库已收拢" {
		t.Fatalf("summary %s", got.Summary)
	}
	if got.Duration < time.Second {
		t.Fatalf("duration still the request blip: %s", got.Duration)
	}
}
