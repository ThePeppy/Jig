package appview

import (
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/session"
	"jig/internal/trace"
	"jig/internal/ui/chrome"
)

func TestWindowShowsLibraryThenOtherPages(t *testing.T) {
	d := openDesk(t)
	tt := ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("数据库还是空的") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if tt.HasText("重新安装") || tt.HasText("IDA 窗口") {
		t.Fatalf("old GUI copy leaked: %q", tt.Texts())
	}

	path := writeFile(t, "firmware.bin")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("firmware.bin") || !tt.HasText("未打开") || !tt.HasText("启动分析") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("从数据库移除"); err != nil {
		t.Fatal(err)
	}
	if len(d.Catalog.Entries()) != 0 || !tt.HasText("数据库还是空的") {
		t.Fatalf("entries %d texts %q", len(d.Catalog.Entries()), tt.Texts())
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
	if !strings.Contains(joined, "尚未监听") || !strings.Contains(joined, "IDA 安装") || strings.Contains(joined, "重新安装") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestMissingFileShowsInTheLibrary(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "gone.dex")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("文件缺失") || !tt.HasText("gone.dex") || !tt.HasText("磁盘上已找不到") || tt.HasText("启动分析") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestSearchNarrowsTheList(t *testing.T) {
	d := openDesk(t)
	for _, name := range []string{"firmware.bin", "libnet.so"} {
		if err := d.AddPath(writeFile(t, name)); err != nil {
			t.Fatal(err)
		}
	}
	tt := ui.NewTester(New(d), 1280, 800)
	if err := tt.Click("搜索数据库"); err != nil {
		t.Fatal(err)
	}
	tt.Type("libnet")
	tt.Frame()
	if _, ok := tt.Find("firmware.bin"); ok {
		t.Fatalf("filtered row still shown: %q", tt.Texts())
	}
	if _, ok := tt.Find("libnet.so"); !ok {
		t.Fatalf("matching row missing: %q", tt.Texts())
	}
	tt.Type("zzz")
	tt.Frame()
	if !tt.HasText("没有匹配的文件") {
		t.Fatalf("no-match line missing: %q", tt.Texts())
	}
}

func TestSwitchTabsMultipleTimesAndRender(t *testing.T) {
	d := openDesk(t)
	view := New(d)
	tt := ui.NewTester(view, 1280, 800)
	tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 84, Right: 14})
	tabs := []string{"数据库", "记录", "设置", "记录", "数据库", "记录", "设置", "记录"}
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
		if img := ui.Render(view, 1280, 800, 2); img == nil {
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
		Metadata: map[string]any{
			"fileType": "ELF64 for ARM64", "processor": "arm", "bitness": float64(64),
			"imageBase": "0x400000", "autoAnalysisComplete": true,
		},
	})
	d.Traces.Append(trace.Entry{
		Function: "ida_decompile",
		Target:   "active_target.bin",
		Duration: 1200 * time.Millisecond,
		Status:   "ok",
		Args:     `{"address": "0x401a30"}`,
		Result:   "int sub_401a30(int a1)",
	})

	view := New(d)
	tt := ui.NewTester(view, 1280, 800)
	tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 84, Right: 14})
	for _, want := range []string{"active_target.bin", "分析会话运行中", "关闭会话", "ARM · 64 位", "0x400000", "已完成", chrome.MCPURL, "未挂载", "ida_decompile"} {
		if !tt.HasText(want) {
			t.Fatalf("library missing %q: %q", want, tt.Texts())
		}
	}

	if err := tt.Click("记录"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"1.2 秒", "入参", "0x401a30", "int sub_401a30"} {
		if !tt.HasText(want) {
			t.Fatalf("record missing %q: %q", want, tt.Texts())
		}
	}
	if err := tt.Click("数据库"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("关闭会话"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("active_target.bin") || len(d.Sessions.Sessions()) != 0 {
		t.Fatalf("session still visible: sessions %d texts %q", len(d.Sessions.Sessions()), tt.Texts())
	}
}

func TestRecordFilterAndSelection(t *testing.T) {
	d := openDesk(t)
	d.Traces.Append(trace.Entry{Function: "ida_xrefs", Target: "a.bin", Status: "ok", Args: `{"address": "0x10"}`})
	d.Traces.Append(trace.Entry{Function: "ida_get_function", Target: "a.bin", Status: "error", Result: "not in a function: 0x20"})
	d.Tab = desk.TabTrace
	tt := ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("not in a function") {
		t.Fatalf("newest record not selected: %q", tt.Texts())
	}
	if err := tt.Click("成功"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("ida_get_function"); ok {
		t.Fatalf("failed call shown under 成功: %q", tt.Texts())
	}
	if !tt.HasText(`"address": "0x10"`) {
		t.Fatalf("detail did not follow the filter: %q", tt.Texts())
	}
	if err := tt.Click("全部"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("ida_xrefs"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(`"address": "0x10"`) || tt.HasText("not in a function") {
		t.Fatalf("selection did not move: %q", tt.Texts())
	}
	if err := tt.Click("清空"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("还没有调用") {
		t.Fatalf("clear left records: %q", tt.Texts())
	}
}

func TestSettingsCopyAndReveal(t *testing.T) {
	d := openDesk(t)
	now := time.Date(2026, 10, 20, 13, 0, 0, 0, time.Local)
	d.Now = func() time.Time { return now }
	d.Tab = desk.TabSettings
	tt := ui.NewTester(New(d), 1280, 800)
	if err := tt.Click("复制地址"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != chrome.MCPURL || !tt.HasText("已复制") {
		t.Fatalf("clipboard %q texts %q", tt.Clipboard(), tt.Texts())
	}
	now = now.Add(2 * time.Second)
	tt.Frame()
	if tt.HasText("已复制") {
		t.Fatal("copy note did not clear")
	}
	if err := tt.Click("在访达中显示"); err != nil {
		t.Fatal(err)
	}
	want := (&url.URL{Scheme: "file", Path: d.Dir()}).String()
	if !slices.Contains(tt.OpenedURLs(), want) {
		t.Fatalf("opened %q want %q", tt.OpenedURLs(), want)
	}
}

func TestCatalogFileShowsItsWorker(t *testing.T) {
	d := openDesk(t)
	openPath := writeFile(t, "firmware.bin")
	idlePath := writeFile(t, "idle.bin")
	if err := d.AddPath(openPath); err != nil {
		t.Fatal(err)
	}
	if err := d.AddPath(idlePath); err != nil {
		t.Fatal(err)
	}
	d.Sessions.Register(&session.Session{
		ID:           "s-firm",
		TargetName:   "firmware.bin",
		SourcePath:   openPath,
		DatabasePath: openPath + ".i64",
		PID:          42,
		State:        session.StateReady,
		StartedAt:    time.Now().Add(-time.Minute),
	})

	tt := ui.NewTester(New(d), 1280, 800)
	if err := tt.Click("firmware.bin"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("分析会话运行中") || !tt.HasText("保存") || tt.HasText("启动分析") {
		t.Fatalf("open file texts %q", tt.Texts())
	}
	if err := tt.Click("idle.bin"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("未打开") || !tt.HasText("启动分析") || tt.HasText("关闭会话") {
		t.Fatalf("idle file texts %q", tt.Texts())
	}
}

func TestPackAttachedShowsAndDetaches(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "firmware.bin")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	d.Sessions.Register(&session.Session{ID: "s-pack", TargetName: "firmware.bin", SourcePath: path, State: session.StateReady})
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codec.tsv"), []byte("category\tpacket_id\tpacket_name\thandler_va\tfield_index\tdepth\treader\tsize\tcall_va\treader_va\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pack.json"), []byte(`{"id":"p","tools":[{"name":"ida_p_fields","kind":"fields_from_calls","config":{"codec":"codec.tsv"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	asked := ""
	d.RequestPack = func(id string) { asked = id }
	tt := ui.NewTester(New(d), 1280, 800)
	if err := tt.Click("挂载"); err != nil {
		t.Fatal(err)
	}
	if asked != "s-pack" {
		t.Fatalf("pack picker asked for %q", asked)
	}
	d.AttachPack(asked, dir)
	tt.Frame()
	if !tt.HasText("已挂载") || !tt.HasText("pack.json") || !tt.HasText("fields_from_calls") {
		t.Fatalf("attached pack missing: %q", tt.Texts())
	}
	if err := tt.Click("卸下"); err != nil {
		t.Fatal(err)
	}
	if d.Sessions.PackOf("s-pack") != nil || !tt.HasText("未挂载") {
		t.Fatalf("detach did not take: %q", tt.Texts())
	}
}

func TestStartFailureOffersRetry(t *testing.T) {
	d := openDesk(t)
	t.Setenv("JIG_IDALIB_WORKER", filepath.Join(t.TempDir(), "missing-worker"))
	if err := d.AddPath(writeFile(t, "firmware.bin")); err != nil {
		t.Fatal(err)
	}
	got := make(chan func(), 1)
	d.Post = func(fn func()) { got <- fn }

	tt := ui.NewTester(New(d), 1280, 800)
	if err := tt.Click("启动分析"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("启动中") || !tt.HasText("正在拉起 worker") {
		t.Fatalf("launch progress missing: %q", tt.Texts())
	}
	select {
	case fn := <-got:
		fn()
	case <-time.After(5 * time.Second):
		t.Fatal("launch did not finish")
	}
	tt.Frame()
	if !tt.HasText("打开失败") || !tt.HasText("原因") || !tt.HasText("missing-worker") || !tt.HasText("重试") {
		t.Fatalf("failure missing: %q", tt.Texts())
	}
	if err := tt.Click("重试"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("打开失败") || !tt.HasText("启动中") {
		t.Fatalf("retry did not restart: %q", tt.Texts())
	}
	(<-got)()
}

func TestStoppingSessionShowsThePack(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "firmware.bin")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	d.Sessions.Register(&session.Session{ID: "s-stop", TargetName: "firmware.bin", SourcePath: path, DatabasePath: path, PID: 42, State: session.StateStopping})
	tt := ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("关闭中") || !tt.HasText("正在收拢数据库") || tt.HasText("关闭会话") {
		t.Fatalf("close process missing: %q", tt.Texts())
	}
}

func TestLaunchingSessionShowsTheOpenStep(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "firmware.bin")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	d.Sessions.Register(&session.Session{ID: "s-open", TargetName: "firmware.bin", SourcePath: path, DatabasePath: path, PID: 77, State: session.StateLaunching, StartedAt: time.Now()})
	tt := ui.NewTester(New(d), 1280, 800)
	if !tt.HasText("启动中") || !tt.HasText("正在打开数据库") {
		t.Fatalf("open step missing: %q", tt.Texts())
	}
}

func TestCloseClearsTheReadyState(t *testing.T) {
	d := openDesk(t)
	path := writeFile(t, "firmware.bin")
	if err := d.AddPath(path); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"s-ready", "s-other"} {
		d.Sessions.Register(&session.Session{ID: id, TargetName: "firmware.bin", SourcePath: path, DatabasePath: path, PID: 951149, State: session.StateReady})
	}
	tt := ui.NewTester(New(d), 1280, 800)
	if err := tt.Click("关闭会话"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("分析会话运行中") || tt.HasText("关闭会话") || !tt.HasText("未打开") || !tt.HasText("session.close") {
		t.Fatalf("ready state stuck: %q", tt.Texts())
	}
	if len(d.Sessions.Sessions()) != 0 {
		t.Fatalf("sessions still live: %d", len(d.Sessions.Sessions()))
	}
	if got := d.Traces.Entries()[0]; got.Function != "session.close" || got.Summary != "数据库已收拢" {
		t.Fatalf("close record %+v", got)
	}
}

func TestRailFrostStaysClearerThanTheDesk(t *testing.T) {
	d := openDesk(t)
	tt := ui.NewTester(New(d), 1280, 800)
	tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 84, Right: 14})
	tt.SetVibrancy(true)
	img := tt.Image()
	// Away from labels and the hairline. The rail is 200 DIP wide.
	rail := img.RGBAAt(40, 520)
	desk := img.RGBAAt(640, 120)
	if int(desk.A) < int(rail.A)+80 {
		t.Fatalf("rail %v is not clearly more transparent than the desk %v", rail, desk)
	}
}

// TestRenderScreens writes the design boards as PNGs when JIG_RENDER_DIR is set.
func TestRenderScreens(t *testing.T) {
	outDir := os.Getenv("JIG_RENDER_DIR")
	save := func(name string, view func(*ui.Context)) {
		tt := ui.NewTester(view, 1280, 800)
		tt.SetTitleBar(ui.TitleBar{Height: 42, Left: 84, Right: 14})
		tt.SetScale(2)
		img := tt.Image()
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
	now := time.Date(2026, 10, 20, 14, 28, 36, 0, time.Local)
	d.Now = func() time.Time { return now }
	d.Listening = true
	view := New(d)
	save("jig_empty.png", view)

	dir := t.TempDir()
	firmware := filepath.Join(dir, "firmware_v2.bin")
	libnet := filepath.Join(dir, "libnet.so")
	helper := filepath.Join(dir, "HelperTool")
	dex := filepath.Join(dir, "classes.dex")
	for _, p := range []string{firmware, libnet, helper, dex} {
		_ = os.WriteFile(p, make([]byte, 48200), 0o644)
		_ = d.AddPath(p)
	}
	d.Sessions.Register(&session.Session{
		ID: "s-7f9a02", TargetName: "firmware_v2.bin", SourcePath: firmware, DatabasePath: firmware + ".i64",
		PID: 48210, State: session.StateReady,
		Metadata: map[string]any{
			"fileType": "ELF 64 位", "processor": "arm", "bitness": float64(64),
			"imageBase": "0x400000", "autoAnalysisComplete": true,
		},
	})
	d.Sessions.Register(&session.Session{ID: "s-3c11b8", TargetName: "libnet.so", SourcePath: libnet, PID: 48305, State: session.StateReady})
	for i, e := range []trace.Entry{
		{Function: "ida_open", Target: "firmware_v2.bin", Duration: 4800 * time.Millisecond, Args: "{\n  \"path\": \"~/Samples/firmware_v2.bin\"\n}"},
		{Function: "ida_list_functions", Target: "firmware_v2.bin", Duration: 400 * time.Millisecond},
		{Function: "ida_get_function", Target: "libnet.so", Duration: 60 * time.Second, Status: "error", Result: "worker 调用超时"},
		{Function: "ida_rename", Target: "libnet.so", Duration: 100 * time.Millisecond},
		{Function: "ida_xrefs", Target: "firmware_v2.bin", Duration: 300 * time.Millisecond},
		{Function: "ida_decompile", Target: "firmware_v2.bin", Duration: 1200 * time.Millisecond,
			Args: "{\n  \"address\": \"0x401a30\",\n  \"create\": false\n}", Result: "int sub_401a30(int a1)\n{\n  return a1 + 1;\n}"},
	} {
		e.Timestamp = now.Add(-[]time.Duration{12, 9, 6, 3, 1, 0}[i] * time.Minute)
		if e.Status == "" {
			e.Status = "ok"
		}
		d.Traces.Append(e)
	}
	d.Catalog.Select("1")
	save("jig_library.png", view)

	d.Catalog.Select("3")
	save("jig_library_idle.png", view)

	t.Setenv("JIG_IDALIB_WORKER", filepath.Join(dir, "missing-worker"))
	_ = d.StartSession(d.Catalog.Entries()[3])
	save("jig_library_failed.png", view)

	d.Tab = desk.TabTrace
	save("jig_trace.png", view)

	d.Tab = desk.TabSettings
	save("jig_settings.png", view)
}

func openDesk(t *testing.T) *desk.Desk {
	t.Helper()
	d, err := desk.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
