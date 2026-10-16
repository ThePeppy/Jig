package intake

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcceptsAnyRegularFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"firmware", "rom.bin", "app.dex", "pkg.apk", "image.img", "mod.sys", "lib.so"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte{0}, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Accept(path)
		if err != nil {
			t.Fatalf("Accept(%s) %v", name, err)
		}
		if got.Kind != KindFile || got.Companion || got.OpenPath != path {
			t.Fatalf("Accept(%s) = %+v", name, got)
		}
		if got.Name != name {
			t.Fatalf("name %q, want %q", got.Name, name)
		}
	}
}

func TestAcceptsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.elf")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Accept(path); err != nil {
		t.Fatal(err)
	}
}

func TestDatabaseKind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.i64")
	if err := os.WriteFile(path, []byte("idb"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Accept(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != KindDatabase || got.OpenPath != path || got.Companion {
		t.Fatalf("%+v", got)
	}
}

func TestCompanionDatabaseWins(t *testing.T) {
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
	if !got.Companion || got.OpenPath != db || got.Kind != KindDatabase || got.Name != "firmware.bin" {
		t.Fatalf("%+v", got)
	}
}

func TestSuffixCompanion(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "app.bin")
	db := bin + ".i64"
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
	if got.OpenPath != db {
		t.Fatalf("open %s, want %s", got.OpenPath, db)
	}
}

func TestRejectsClosedInputs(t *testing.T) {
	dir := t.TempDir()
	sidecar := filepath.Join(dir, "sample.id0")
	if err := os.WriteFile(sidecar, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Accept(sidecar); err != ErrSidecar {
		t.Fatalf("sidecar err %v", err)
	}
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
