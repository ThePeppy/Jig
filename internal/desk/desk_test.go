package desk

import (
	"os"
	"path/filepath"
	"testing"
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
	if err := d.AddPath(sidecar); err == nil {
		t.Fatal("sidecar accepted")
	}
	if d.Notice == "" {
		t.Fatal("missing notice")
	}
}
