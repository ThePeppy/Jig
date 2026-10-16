package idalib

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProbeUsesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	st := Probe(dir, t.TempDir())
	if !st.Found || st.Missing || st.Install != dir {
		t.Fatalf("%+v", st)
	}
	st = Probe(filepath.Join(dir, "gone"), dir)
	if st.Found || !st.Missing {
		t.Fatalf("%+v", st)
	}
}

func TestProbeFindsProfessionalApp(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("bundle names are macOS")
	}
	root := t.TempDir()
	for _, name := range []string{"IDA Home 9.4.app", "IDA Professional 9.0.app", "Notes.app"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	st := Probe("", root)
	if !st.Found || filepath.Base(st.Install) != "IDA Professional 9.0.app" {
		t.Fatalf("%+v", st)
	}
}
