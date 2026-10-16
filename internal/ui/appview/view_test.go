package appview

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/session"
	"jig/internal/trace"
)

func TestWindowShowsLibraryThenOtherPages(t *testing.T) {
	d := openDesk(t)
	d.Now = func() time.Time { return time.Date(2026, 10, 16, 14, 2, 0, 0, time.Local) }
	tt := ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("还没有文件") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if tt.HasText("重新安装") || tt.HasText("IDA 窗口") {
		t.Fatalf("old GUI copy leaked: %q", tt.Texts())
	}

	path := filepath.Join(t.TempDir(), "firmware.bin")
	if err := os.WriteFile(path, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	tt = ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("firmware.bin") || !tt.HasText("未打开") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("移出"); err != nil {
		t.Fatal(err)
	}
	if len(d.Catalog.Entries()) != 0 || !tt.HasText("还没有文件") {
		t.Fatalf("entries %d texts %q", len(d.Catalog.Entries()), tt.Texts())
	}

	if err := tt.Click("会话"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("还没有会话") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("记录"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("还没有调用") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(joined, "尚未监听") || strings.Contains(joined, "重新安装") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestMissingFileShowsInTheLibrary(t *testing.T) {
	d := openDesk(t)
	path := filepath.Join(t.TempDir(), "gone.dex")
	if err := os.WriteFile(path, []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("文件缺失") || !tt.HasText("gone.dex") {
		t.Fatalf("texts %q", tt.Texts())
	}
}
func TestSwitchTabsMultipleTimesAndRender(t *testing.T) {
	d := openDesk(t)
	view := New(d)
	tt := ui.NewTester(view, 1280, 800)
	tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 84, Right: 14})
	tabs := []string{"数据库", "会话", "记录", "设置", "记录", "数据库", "记录", "会话", "记录"}
	for _, tabName := range tabs {
		if err := tt.Click(tabName); err != nil {
			t.Fatalf("click %s failed: %v", tabName, err)
		}
		for x := float32(100); x < 1000; x += 150 {
			for y := float32(50); y < 600; y += 150 {
				tt.Move(x, y)
				tt.Scroll(x, y, 0, 20)
			}
		}
		img := ui.Render(view, 1280, 800, 2)
		if img == nil {
			t.Fatalf("render returned nil on %s", tabName)
		}
	}
}

func TestActiveSessionsAndTracesRender(t *testing.T) {
	d := openDesk(t)
	d.Sessions.Register(&session.Session{
		ID:           "s-abc123",
		TargetName:   "active_target.bin",
		SourcePath:   "/tmp/active_target.bin",
		DatabasePath: "/tmp/active_target.i64",
		PID:          9999,
		State:        session.StateReady,
		StartedAt:    time.Now().Add(-5 * time.Minute),
	})

	d.Traces.Append(trace.Entry{
		Function: "session.launch",
		Target:   "active_target.bin",
		Duration: 15 * time.Millisecond,
		Status:   "ok",
		Summary:  "Worker 已就绪 (PID: 9999)",
	})

	view := New(d)
	tt := ui.NewTester(view, 1280, 800)
	tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 84, Right: 14})

	if err := tt.Click("会话"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("active_target.bin") || !tt.HasText("就绪") {
		t.Fatalf("sessions texts missing: %q", tt.Texts())
	}

	if err := tt.Click("记录"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("session.launch") || !tt.HasText("调用审计记录") {
		t.Fatalf("trace texts missing: %q", tt.Texts())
	}
}

func TestRenderAllStudioScreens(t *testing.T) {
	outDir := os.Getenv("JIG_RENDER_DIR")
	savePNG := func(name string, view func(*ui.Context)) {
		img := ui.Render(view, 1280, 800, 2)
		if img == nil {
			t.Fatalf("nil render for %s", name)
		}
		if outDir == "" {
			return
		}
		f, err := os.Create(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	}

	d := openDesk(t)
	d.Now = func() time.Time { return time.Date(2026, 10, 16, 14, 28, 36, 0, time.Local) }
	view := New(d)

	// 1. Empty Library
	savePNG("jig_ui_1_lib_empty.png", view)

	// 2. Populated library, two live sessions, one missing file.
	dir := t.TempDir()
	bin1 := filepath.Join(dir, "libil2cpp-arm64.so")
	db1 := filepath.Join(dir, "libil2cpp-arm64.i64")
	bin2 := filepath.Join(dir, "bootloader_v4.bin")
	bin3 := filepath.Join(dir, "kernel_cache.macho")
	gone := filepath.Join(dir, "legacy_driver.sys")
	_ = os.WriteFile(bin1, make([]byte, 48200), 0o644)
	_ = os.WriteFile(db1, make([]byte, 192400), 0o644)
	_ = os.WriteFile(bin2, make([]byte, 16384), 0o644)
	_ = os.WriteFile(bin3, make([]byte, 81920), 0o644)
	_ = os.WriteFile(gone, []byte{1, 2, 3}, 0o644)

	_ = d.AddPath(bin1)
	_ = d.AddPath(bin2)
	_ = d.AddPath(bin3)
	_ = d.AddPath(gone)
	_ = os.Remove(gone)
	d.Catalog.Select("1")

	d.Sessions.Register(&session.Session{
		ID:           "s-7f9a02",
		TargetName:   "libil2cpp-arm64.so",
		SourcePath:   bin1,
		DatabasePath: db1,
		PID:          48210,
		State:        session.StateReady,
		StartedAt:    time.Now().Add(-12*time.Minute - 34*time.Second),
	})
	d.Sessions.Register(&session.Session{
		ID:           "s-3c11b8",
		TargetName:   "bootloader_v4.bin",
		SourcePath:   bin2,
		DatabasePath: bin2,
		PID:          48305,
		State:        session.StateReady,
		StartedAt:    time.Now().Add(-3*time.Minute - 18*time.Second),
	})

	d.Traces.Append(trace.Entry{
		Timestamp: time.Date(2026, 10, 16, 14, 26, 10, 0, time.Local),
		Function:  "session.launch",
		Target:    "libil2cpp-arm64.so",
		Duration:  420 * time.Millisecond,
		Status:    "ok",
		Summary:   "Worker 已就绪 (PID: 48210)",
	})
	d.Traces.Append(trace.Entry{
		Timestamp: time.Date(2026, 10, 16, 14, 27, 4, 0, time.Local),
		Function:  "ida_list_functions",
		Target:    "libil2cpp-arm64.so",
		Duration:  38 * time.Millisecond,
		Status:    "ok",
		Summary:   "返回 14,280 个符号条目 (page 1/29)",
	})
	d.Traces.Append(trace.Entry{
		Timestamp: time.Date(2026, 10, 16, 14, 27, 19, 0, time.Local),
		Function:  "ida_decompile",
		Target:    "libil2cpp-arm64.so",
		Duration:  84 * time.Millisecond,
		Status:    "ok",
		Summary:   "sub_18004A2C0 @ 0x18004A2C0 · 生成 68 行伪代码",
	})
	d.Traces.Append(trace.Entry{
		Timestamp: time.Date(2026, 10, 16, 14, 28, 12, 0, time.Local),
		Function:  "ida_xrefs",
		Target:    "bootloader_v4.bin",
		Duration:  12 * time.Millisecond,
		Status:    "ok",
		Summary:   "命中 9 处交叉引用 → rom_verify_sig",
	})

	d.Tab = desk.TabLibrary
	savePNG("jig_ui_2_lib_populated.png", view)
	d.Catalog.Select("3")
	savePNG("jig_ui_2b_lib_unopened.png", view)
	d.Catalog.Select("1")

	d.Tab = desk.TabSessions
	savePNG("jig_ui_3_sessions_active.png", view)

	d.Tab = desk.TabTrace
	savePNG("jig_ui_4_trace_active.png", view)

	d.Tab = desk.TabSettings
	d.Section = desk.SettingsIDA
	savePNG("jig_ui_5_settings_ida.png", view)

	d.Section = desk.SettingsAgents
	savePNG("jig_ui_6_settings_agent.png", view)
}

func openDesk(t *testing.T) *desk.Desk {
	t.Helper()
	d, err := desk.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}
