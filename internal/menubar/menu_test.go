package menubar

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"jig/internal/desk"
	"jig/internal/session"
)

func TestEmptyMenu(t *testing.T) {
	d := openDesk(t)
	quit := 0
	items := Items(d, nil, func() { quit++ })
	if find(items, "quit").Label != "退出 Jig" {
		t.Fatalf("quit label %q", find(items, "quit").Label)
	}
	find(items, "quit").Run()
	if quit != 1 {
		t.Fatal("quit was not called")
	}
	if child := findChild(items, "open", "open-empty"); !child.Disabled || child.Label != "没有可打开的文件" {
		t.Fatalf("open empty %+v", child)
	}
	if child := findChild(items, "close", "close-empty"); !child.Disabled {
		t.Fatalf("close empty %+v", child)
	}
	if Title(d) != "" {
		t.Fatalf("title %q", Title(d))
	}
}

func TestOpenStartsTheSessionAndShowsTheWindow(t *testing.T) {
	d := openDesk(t)
	// Launch must not start a real worker.
	t.Setenv("JIG_IDALIB_WORKER", filepath.Join(t.TempDir(), "missing-worker"))
	path := writeFile(t, "firmware.bin")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	shown := 0
	d.Post = func(func()) {}
	item := findChild(Items(d, func() { shown++ }, nil), "open", "open-1")
	if item.Label != "firmware.bin" || item.Run == nil {
		t.Fatalf("%+v", item)
	}
	item.Run()
	if shown != 1 {
		t.Fatal("window was not shown")
	}
	if d.Tab != desk.TabLibrary {
		t.Fatalf("tab %v", d.Tab)
	}
	if !d.Launching("1") {
		t.Fatal("session was not started")
	}
	again := Items(d, nil, nil)
	if findChild(again, "open", "open-1").ID != "" {
		t.Fatal("launching file still offered as open")
	}
	busy := findChild(again, "close", "close-launch-1")
	if !busy.Disabled || busy.Label != "firmware.bin · 启动中" {
		t.Fatalf("launch row %+v", busy)
	}
	if Title(d) != "1" {
		t.Fatalf("title %q", Title(d))
	}
}

func TestMissingFileIsNotOffered(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "gone.bin")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if findChild(Items(d, nil, nil), "open", "open-empty").ID == "" {
		t.Fatal("missing file was offered")
	}
}

func TestCloseSessionFromTheMenu(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "libfoo.dylib")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	d.Sessions.Register(&session.Session{
		ID:         "s-ready",
		TargetName: "libfoo.dylib",
		SourcePath: path,
		State:      session.StateReady,
	})
	d.Sessions.Register(&session.Session{
		ID:         "s-stop",
		TargetName: "packing.bin",
		SourcePath: filepath.Join(t.TempDir(), "packing.bin"),
		State:      session.StateStopping,
	})
	items := Items(d, nil, nil)
	if findChild(items, "open", "open-1").ID != "" {
		t.Fatal("open session still listed under open")
	}
	ready := findChild(items, "close", "close-s-ready")
	if ready.Disabled || ready.Label != "libfoo.dylib" {
		t.Fatalf("ready %+v", ready)
	}
	stop := findChild(items, "close", "close-s-stop")
	if !stop.Disabled || stop.Label != "packing.bin · 关闭中" {
		t.Fatalf("stopping %+v", stop)
	}
	ready.Run()
	after := Items(d, nil, nil)
	if findChild(after, "close", "close-s-ready").ID != "" {
		t.Fatal("closed session still listed")
	}
	if findChild(after, "open", "open-1").Label != "libfoo.dylib" {
		t.Fatal("closed file was not offered again")
	}
	if Title(d) != "" {
		t.Fatalf("title after close %q", Title(d))
	}
}

func TestDuplicateNamesCarryThePath(t *testing.T) {
	d := openDesk(t)
	a := writeFile(t, "firmware.bin")
	other := filepath.Join(t.TempDir(), "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	b := filepath.Join(other, "firmware.bin")
	if err := os.WriteFile(b, []byte{2}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(a); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(b); err != nil {
		t.Fatal(err)
	}
	items := Items(d, nil, nil)
	if findChild(items, "open", "open-1").Label == findChild(items, "open", "open-2").Label {
		t.Fatal("duplicate names were not disambiguated")
	}
}

func TestShowFocusesWithoutStarting(t *testing.T) {
	d := openDesk(t)
	shown := 0
	find(Items(d, func() { shown++ }, nil), "show").Run()
	if shown != 1 {
		t.Fatal("show was not called")
	}
}

func openDesk(t *testing.T) *desk.Desk {
	t.Helper()
	d, err := desk.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d.Now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	return d
}

func writeFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func find(items []Item, id string) Item {
	for _, it := range items {
		if it.ID == id {
			return it
		}
	}
	return Item{}
}

func findChild(items []Item, parent, id string) Item {
	for _, it := range find(items, parent).Children {
		if it.ID == id {
			return it
		}
	}
	return Item{}
}
