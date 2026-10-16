package intake

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcceptsAnyRegularFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"firmware", "rom.bin", "app.dex", "pkg.apk", "sample.i64", "sample.id0", "empty.elf"} {
		path := filepath.Join(dir, name)
		var body []byte
		if name != "empty.elf" {
			body = []byte("x")
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Accept(path)
		if err != nil {
			t.Fatalf("Accept(%s) %v", name, err)
		}
		if got.OpenPath != path || got.Source != path || got.Name != name {
			t.Fatalf("Accept(%s) = %+v", name, got)
		}
	}
}

func TestDoesNotRewriteToCompanion(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "firmware.bin")
	db := filepath.Join(dir, "firmware.i64")
	if err := os.WriteFile(bin, []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(db, []byte("idb"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Accept(bin)
	if err != nil {
		t.Fatal(err)
	}
	if got.OpenPath != bin || got.Name != "firmware.bin" {
		t.Fatalf("%+v", got)
	}
}

func TestRejectsClosedInputs(t *testing.T) {
	dir := t.TempDir()
	if _, err := Accept(dir); err != ErrDirectory {
		t.Fatalf("dir err %v", err)
	}
	if _, err := Accept(filepath.Join(dir, "missing.bin")); err != ErrMissing {
		t.Fatalf("missing err %v", err)
	}
	if _, err := Accept("  "); err != ErrMissing {
		t.Fatalf("blank err %v", err)
	}
}
