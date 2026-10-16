package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddKeepsAnyFileAndDedupes(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "firmware")
	if err := os.WriteFile(bin, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	c := New()
	first, added, err := c.Add(bin)
	if err != nil || !added || first.Kind != "file" {
		t.Fatalf("added=%v err=%v entry=%+v", added, err, first)
	}
	again, added, err := c.Add(bin)
	if err != nil || added || again.ID != first.ID {
		t.Fatalf("dedupe added=%v err=%v id=%s", added, err, again.ID)
	}
	if len(c.Entries()) != 1 {
		t.Fatalf("entries %d", len(c.Entries()))
	}
}

func TestRemoveSelectsNeighbor(t *testing.T) {
	dir := t.TempDir()
	c := New()
	var ids []string
	for _, name := range []string{"a.bin", "b.dex", "c.apk"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte{1}, 0o644); err != nil {
			t.Fatal(err)
		}
		e, _, err := c.Add(path)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, e.ID)
	}
	c.Select(ids[1])
	c.Remove(ids[1])
	got, ok := c.Selected()
	if !ok || got.ID != ids[2] {
		t.Fatalf("selected %+v ok=%v", got, ok)
	}
	c.Remove(ids[2])
	got, ok = c.Selected()
	if !ok || got.ID != ids[0] {
		t.Fatalf("selected %+v ok=%v", got, ok)
	}
}

func TestMissingAfterDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.bin")
	if err := os.WriteFile(path, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	c := New()
	e, _, err := c.Add(path)
	if err != nil {
		t.Fatal(err)
	}
	if e.Missing() {
		t.Fatal("file exists")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if !c.Entries()[0].Missing() {
		t.Fatal("expected missing")
	}
}

func TestRestoreRepairsDuplicateIDsAndAdvancesSequence(t *testing.T) {
	dir := t.TempDir()
	c := New()
	c.Restore([]Entry{
		{ID: "1", Name: "df_game_r.i64", Path: "/a/df_game_r.i64", Source: "/a/df_game_r.i64"},
		{ID: "1", Name: "df_guild_r.i64", Path: "/b/df_guild_r.i64", Source: "/b/df_guild_r.i64"},
	})
	entries := c.Entries()
	if len(entries) != 2 || entries[0].ID == entries[1].ID {
		t.Fatalf("expected distinct IDs after restore, got %+v", entries)
	}
	c.Select(entries[1].ID)
	sel, ok := c.Selected()
	if !ok || sel.Name != "df_guild_r.i64" {
		t.Fatalf("expected df_guild_r.i64 selected, got %+v", sel)
	}

	thirdPath := filepath.Join(dir, "third.bin")
	if err := os.WriteFile(thirdPath, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	third, added, err := c.Add(thirdPath)
	if err != nil || !added || third.ID == entries[0].ID || third.ID == entries[1].ID {
		t.Fatalf("expected non-colliding ID for third entry, got %+v", third)
	}
}
