package session

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionManagerLifecycle(t *testing.T) {
	m := NewManager()

	s := &Session{
		ID:           "test-session-1",
		TargetName:   "firmware.bin",
		SourcePath:   "/tmp/firmware.bin",
		DatabasePath: "/tmp/firmware.i64",
		PID:          12345,
		State:        StateReady,
	}

	m.Register(s)

	if len(m.Sessions()) != 1 {
		t.Fatalf("expected 1 session, got %d", len(m.Sessions()))
	}

	found, ok := m.FindByPath("/tmp/firmware.bin")
	if !ok || found.ID != "test-session-1" {
		t.Fatalf("expected to find session by source path")
	}

	foundDB, okDB := m.FindByPath("/tmp/firmware.i64")
	if !okDB || foundDB.ID != "test-session-1" {
		t.Fatalf("expected to find session by database path")
	}

	if err := m.Close("test-session-1"); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	if len(m.Sessions()) != 0 {
		t.Fatalf("expected 0 sessions after close")
	}
}

func TestSessionLaunchValidation(t *testing.T) {
	m := NewManager()
	_, err := m.Launch(context.Background(), "", "", "", "")
	if err == nil {
		t.Fatal("expected error launching with empty target path")
	}
}

func TestLaunchDoesNotStartAWorker(t *testing.T) {
	m := NewManager()
	dir := t.TempDir()
	_, err := m.Launch(context.Background(), "sample", filepath.Join(dir, "sample"), filepath.Join(dir, "sample"), dir)
	if err == nil || !strings.Contains(err.Error(), "libidalib") {
		t.Fatal(err)
	}
	if len(m.Sessions()) != 0 {
		t.Fatal("launch registered a session")
	}
}

func TestShortSessionID(t *testing.T) {
	m := NewManager()
	m.Register(&Session{ID: "aaaaaaaa"})
	id, err := m.newID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 8 || id == "aaaaaaaa" {
		t.Fatalf("id %q", id)
	}
	for _, c := range id {
		if c < '0' || (c > '9' && c < 'a') || c > 'f' {
			t.Fatalf("id %q", id)
		}
	}
}

func TestWorkerCandidatesPreferTheBundle(t *testing.T) {
	exe := filepath.Join("/Applications", "Jig.app", "Contents", "MacOS", "Jig")
	wd := filepath.Join("/work", "Jig")
	got := workerCandidates(exe, wd)
	want := filepath.Join("/Applications", "Jig.app", "Contents", "Resources", "jig-idalib")
	if len(got) == 0 || got[0] != want {
		t.Fatalf("first candidate %q, want %s", got, want)
	}
	repo := filepath.Join(wd, "native", "idalib-worker", "jig-idalib")
	if !slicesContains(got, repo) {
		t.Fatalf("checkout walk missing: %v", got)
	}
}

func TestWorkerCandidatesWalkFromAPackageDir(t *testing.T) {
	wd := filepath.Join("/work", "Jig", "internal", "session")
	got := workerCandidates("/tmp/go-build123/b001/exe", wd)
	want := filepath.Join("/work", "Jig", "native", "idalib-worker", "jig-idalib")
	if !slicesContains(got, want) {
		t.Fatalf("%v", got)
	}
	if bundleResources("/tmp/go-build123/b001/exe") != "" {
		t.Fatal("a go test binary is not a bundle")
	}
}

func slicesContains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
