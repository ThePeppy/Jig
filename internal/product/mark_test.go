package product

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMarkReadsAtIconSizes(t *testing.T) {
	app := DrawMark(64, MarkApp)
	corner := app.NRGBAAt(0, 0)
	if corner.A != 255 || corner.R > 40 {
		t.Fatalf("field corner %+v", corner)
	}
	mid := app.NRGBAAt(32, 28)
	if int(mid.R)+int(mid.G)+int(mid.B) <= int(corner.R)+int(corner.G)+int(corner.B) {
		t.Fatalf("prism %+v not lighter than field %+v", mid, corner)
	}

	tpl := DrawMark(32, MarkTemplate)
	if tpl.NRGBAAt(0, 0).A != 0 {
		t.Fatal("template corner should be clear")
	}
	opaque := 0
	for y := range 32 {
		for x := range 32 {
			if tpl.NRGBAAt(x, y).A > 200 {
				opaque++
			}
		}
	}
	if opaque < 40 {
		t.Fatalf("template stroke missing: %d opaque pixels", opaque)
	}
}

func TestIconFileMatchesMark(t *testing.T) {
	got, err := os.ReadFile(filepath.Join("..", "..", "resources", "icon.png"))
	if err != nil {
		t.Fatal(err)
	}
	want := PNG(1024, MarkApp)
	if !bytes.Equal(got, want) {
		t.Fatalf("resources/icon.png drifted from DrawMark (%d vs %d bytes)", len(got), len(want))
	}
}
