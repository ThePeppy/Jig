package menubar

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/session"
)

type calls struct{ show, quit, dismiss int }

func (k *calls) actions() Actions {
	return Actions{
		Show:    func() { k.show++ },
		Quit:    func() { k.quit++ },
		Dismiss: func() { k.dismiss++ },
	}
}

func panel(d *desk.Desk, k *calls) *ui.Tester {
	return ui.NewTester(Panel(d, k.actions()), PanelW, PanelHeight(d))
}

func TestEmptyPanel(t *testing.T) {
	d := openDesk(t)
	k := &calls{}
	tt := panel(d, k)
	if !tt.HasText("没有打开的会话") || tt.HasText("打开分析") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("关闭全部会话"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("退出 Jig"); err != nil {
		t.Fatal(err)
	}
	if k.quit != 1 {
		t.Fatal("quit was not called")
	}
}

func TestOpenFromThePanelStartsAndShowsTheWindow(t *testing.T) {
	d := openDesk(t)
	t.Setenv("JIG_IDALIB_WORKER", filepath.Join(t.TempDir(), "missing-worker"))
	if err := d.AddPath(writeFile(t, "firmware.bin")); err != nil {
		t.Fatal(err)
	}
	d.Tab = desk.TabSettings
	d.Post = func(func()) {}
	before := PanelHeight(d)
	k := &calls{}
	tt := panel(d, k)
	if err := tt.Click("打开 firmware.bin"); err != nil {
		t.Fatal(err)
	}
	if k.show != 1 || d.Tab != desk.TabLibrary || !d.Launching("1") {
		t.Fatalf("show %d tab %v launching %v", k.show, d.Tab, d.Launching("1"))
	}
	tt.Frame()
	if !tt.HasText("启动中") || tt.HasText("打开分析") {
		t.Fatalf("launching row missing: %q", tt.Texts())
	}
	if PanelHeight(d) >= before {
		t.Fatalf("height %d did not drop the open section from %d", PanelHeight(d), before)
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
	tt := panel(d, &calls{})
	if tt.HasText("gone.bin") {
		t.Fatalf("missing file offered: %q", tt.Texts())
	}
}

func TestCloseFromThePanel(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "libfoo.dylib")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	d.Sessions.Register(&session.Session{ID: "s-ready", TargetName: "libfoo.dylib", SourcePath: path, State: session.StateReady})
	d.Sessions.Register(&session.Session{ID: "s-stop", TargetName: "packing.bin", SourcePath: filepath.Join(t.TempDir(), "packing.bin"), State: session.StateStopping})
	tt := panel(d, &calls{})
	if _, ok := tt.Find("关闭 packing.bin 会话"); ok {
		t.Fatal("stopping session offers a close key")
	}
	if !tt.HasText("关闭中") {
		t.Fatalf("stopping row missing: %q", tt.Texts())
	}
	if err := tt.Click("关闭 libfoo.dylib 会话"); err != nil {
		t.Fatal(err)
	}
	if _, ok := d.Sessions.Get("s-ready"); ok {
		t.Fatal("session still open")
	}
	tt = panel(d, &calls{})
	if _, ok := tt.Find("打开 libfoo.dylib"); !ok {
		t.Fatalf("closed file not offered again: %q", tt.Texts())
	}
}

func TestCloseAllFromThePanel(t *testing.T) {
	d := openDesk(t)
	for _, id := range []string{"a", "b"} {
		d.Sessions.Register(&session.Session{ID: id, TargetName: id + ".bin", SourcePath: filepath.Join(t.TempDir(), id+".bin"), State: session.StateReady})
	}
	tt := panel(d, &calls{})
	if err := tt.Click("关闭全部会话"); err != nil {
		t.Fatal(err)
	}
	if n := len(d.Sessions.Sessions()); n != 0 {
		t.Fatalf("%d sessions left", n)
	}
}

func TestDuplicateNamesCarryThePath(t *testing.T) {
	d := openDesk(t)
	a := writeFile(t, "firmware.bin")
	b := writeFile(t, "firmware.bin")
	if err := d.AddPath(a); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(b); err != nil {
		t.Fatal(err)
	}
	ops := openables(d)
	if len(ops) != 2 || ops[0].label == ops[1].label {
		t.Fatalf("labels %+v", ops)
	}
}

func TestShortcutShowsTheWindow(t *testing.T) {
	d := openDesk(t)
	k := &calls{}
	tt := panel(d, k)
	tt.Key(ui.Cmd, ui.KeyJ)
	if k.show != 1 || k.dismiss != 1 {
		t.Fatalf("show %d dismiss %d", k.show, k.dismiss)
	}
}

// TestRenderPanel writes the panel as a PNG when JIG_RENDER_DIR is set.
func TestRenderPanel(t *testing.T) {
	outDir := os.Getenv("JIG_RENDER_DIR")
	if outDir == "" {
		t.Skip("JIG_RENDER_DIR not set")
	}
	d := openDesk(t)
	d.Listening = true
	for _, name := range []string{"firmware_v2.bin", "libnet.so"} {
		d.Sessions.Register(&session.Session{ID: name, TargetName: name, SourcePath: "/s/" + name, State: session.StateReady})
	}
	tt := panel(d, &calls{})
	tt.SetScale(2)
	f, err := os.Create(filepath.Join(outDir, "jig_menubar.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
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
