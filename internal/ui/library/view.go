// Package library draws the database page: the file list and the detail
// of the focused file. An open worker belongs to its file, not to a
// separate page. A worker whose path is not in the catalog is still a row.
// It does not decide what idalib can load; it shows what the worker said.
package library

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/catalog"
	"jig/internal/desk"
	"jig/internal/pack"
	"jig/internal/product"
	"jig/internal/session"
	"jig/internal/ui/chrome"
	"jig/internal/ui/theme"
)

// View renders the list column and the detail column.
func View(c *ui.Context, d *desk.Desk) {
	rows := libraryRows(d)
	if len(rows) == 0 {
		empty(c, d)
		return
	}
	focus := focused(d, rows)
	list(c, d, rows, focus)
	detail(c, d, focus)
}

type libRow struct {
	entry  catalog.Entry
	sess   *session.Session
	orphan bool
}

func (r libRow) key() string {
	if r.orphan && r.sess != nil {
		return "sess-" + r.sess.ID
	}
	return r.entry.ID
}

func (r libRow) path() string {
	if r.orphan && r.sess != nil {
		return firstPath(r.sess.SourcePath, r.sess.DatabasePath)
	}
	return r.entry.Path
}

func libraryRows(d *desk.Desk) []libRow {
	entries := d.Catalog.Entries()
	rows := make([]libRow, 0, len(entries)+4)
	taken := map[string]bool{}
	for _, e := range entries {
		s := sessionFor(d, e.Path, e.Source)
		if s != nil {
			taken[s.ID] = true
		}
		rows = append(rows, libRow{entry: e, sess: s})
	}
	for _, s := range d.Sessions.Sessions() {
		if taken[s.ID] {
			continue
		}
		name := s.TargetName
		if name == "" {
			name = filepath.Base(firstPath(s.SourcePath, s.DatabasePath))
		}
		rows = append(rows, libRow{
			orphan: true,
			sess:   s,
			entry: catalog.Entry{
				Name:   name,
				Source: s.SourcePath,
				Path:   firstPath(s.SourcePath, s.DatabasePath),
			},
		})
	}
	return rows
}

func focused(d *desk.Desk, rows []libRow) libRow {
	if d.SelectedSession != "" {
		for _, r := range rows {
			if r.orphan && r.sess != nil && r.sess.ID == d.SelectedSession {
				return r
			}
		}
	}
	if e, ok := d.Catalog.Selected(); ok {
		for _, r := range rows {
			if !r.orphan && r.entry.ID == e.ID {
				return r
			}
		}
	}
	return rows[0]
}

func matches(r libRow, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(r.entry.Name), q) || strings.Contains(strings.ToLower(r.path()), q)
}

// state is what a row and its detail say about one file.
type state struct {
	tone   chrome.Tone
	label  string
	reason string
}

const (
	labelReady    = "分析会话运行中"
	labelStarting = "启动中"
	labelStopping = "关闭中"
	labelFailed   = "打开失败"
	labelMissing  = "文件缺失"
	labelIdle     = "未打开"
)

func stateOf(d *desk.Desk, r libRow) state {
	switch {
	case !r.orphan && r.entry.Missing():
		return state{chrome.ToneFault, labelMissing, "磁盘上已找不到这个文件。它可能被移走或删除了。"}
	case r.sess != nil && r.sess.State == session.StateStopping:
		return state{chrome.ToneBusy, labelStopping, ""}
	case r.sess != nil && r.sess.State == session.StateLaunching,
		r.sess == nil && !r.orphan && d.Launching(r.entry.ID):
		return state{chrome.ToneBusy, labelStarting, ""}
	case r.sess != nil && r.sess.State == session.StateError:
		return state{chrome.ToneFault, labelFailed, r.sess.Error}
	case r.sess != nil:
		return state{chrome.ToneLive, labelReady, ""}
	}
	if msg, ok := d.Failure(r.entry.ID); ok && !r.orphan {
		return state{chrome.ToneFault, labelFailed, msg}
	}
	return state{chrome.ToneIdle, labelIdle, ""}
}

func empty(c *ui.Context, d *desk.Desk) {
	chrome.EmptyCard(c, "数据库还是空的", "把文件拖进窗口，或手动选择。任何文件都可以入库。",
		func() { chrome.Mark(c, 56) },
		func() {
			ui.Column(c).AlignItems(ui.Center).Gap(14).Children(func() {
				if chrome.PrimaryButton(c, "添加文件").Clicked() && d.RequestAdd != nil {
					d.RequestAdd()
				}
				chrome.Notice(c, d)
			})
		})
}

func list(c *ui.Context, d *desk.Desk, rows []libRow, focus libRow) {
	col := ui.Column(c).Width(theme.ListW).AlignSelf(ui.Stretch).Padding(20, 12).Gap(14).
		BorderWidth(0, 1, 0, 0).BorderColor(theme.Rule)
	col.Children(func() {
		ui.Row(c).Padding(0, 8).AlignItems(ui.Center).Children(func() {
			chrome.Heading(c, "数据库")
			ui.Spacer(c)
			if chrome.PrimaryButton(c, "添加文件").Clicked() && d.RequestAdd != nil {
				d.RequestAdd()
			}
		})
		search(c, d)
		if d.Notice != "" {
			ui.Column(c).Padding(0, 8).Children(func() { chrome.Notice(c, d) })
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Gap(2).Children(func() {
			shown := 0
			for _, r := range rows {
				if !matches(r, d.Query) {
					continue
				}
				shown++
				row(c, d, r, r.key() == focus.key())
			}
			if shown == 0 {
				ui.Text(c, "没有匹配的文件").FontSize(12).TextColor(theme.Muted).Padding(10, 12)
			}
		})
		ui.Row(c).Padding(0, 8).Children(func() {
			ui.Row(c).Grow(1).Padding(14).Radius(theme.RowR).
				Border(1, theme.Dash).BorderStyle(ui.BorderDashed).Center().
				Children(func() {
					ui.Text(c, "拖入文件，直接入库").FontSize(13).TextColor(theme.Muted)
				})
		})
	})
}

func search(c *ui.Context, d *desk.Desk) {
	ui.Row(c).Padding(0, 8).Children(func() {
		ui.Row(c).Grow(1).Height(36).Padding(0, 12).Radius(theme.ControlR).Background(theme.FieldVeil).
			Gap(8).AlignItems(ui.Center).Cursor(ui.CursorText).
			Children(func() {
				ui.Icon(c, chrome.IconSearch).Size(14, 14).TextColor(theme.Muted)
				ui.TextInputBase(c.Key("lib-search"), &d.Query).
					Placeholder("搜索").
					Label("搜索数据库").
					FontSize(13).
					TextColor(theme.Text).
					Grow(1)
			})
	})
}

func row(c *ui.Context, d *desk.Desk, r libRow, on bool) {
	st := stateOf(d, r)
	item := ui.Row(c).
		Key("lib-row-"+r.key()).
		Padding(10, 12).
		Radius(theme.RowR).
		Gap(12).
		AlignItems(ui.Center).
		Cursor(ui.CursorPointer).
		Focusable().
		Label(r.entry.Name)
	if item.Clicked() {
		selectRow(d, r)
		on = true
	}
	switch {
	case on:
		item.Background(theme.RowActive)
	case item.Hovered():
		item.Background(theme.RowHover)
	}
	item.Children(func() {
		chrome.ToneDot(c, st.tone, 8)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, r.entry.Name).FontSize(13).FontWeight(500).TextColor(theme.Text).SingleLine()
			chrome.Mono(c, product.ShortPath(r.path()), 11.5, theme.Muted)
		})
	})
}

func detail(c *ui.Context, d *desk.Desk, r libRow) {
	st := stateOf(d, r)
	ui.Scroll(c).
		Key("lib-detail-"+r.key()).
		Grow(1).
		MinWidth(0).
		Padding(28, 32).
		Gap(24).
		Transition(chrome.PageTransition).
		Children(func() {
			ui.Column(c).Children(func() {
				chrome.Pill(c, st.tone, st.label)
				ui.Text(c, r.entry.Name).FontSize(30).FontWeight(600).LetterSpacing(-0.6).TextColor(theme.Text).
					SingleLine().Padding(12, 0, 4, 0)
				chrome.Mono(c, product.ShortPath(r.path()), 12, theme.Muted)
				ui.Row(c).Gap(10).Padding(18, 0, 0, 0).Children(func() { actions(c, d, r, st) })
			})
			body(c, d, r, st)
		})
}

func actions(c *ui.Context, d *desk.Desk, r libRow, st state) {
	switch st.label {
	case labelReady:
		if d.Saving(r.sess.ID) {
			chrome.BusyButton(c, "保存中", chrome.SizeLarge)
		} else if chrome.Button(c, "保存", chrome.SizeLarge).Clicked() {
			d.SaveSession(r.sess.ID)
		}
		if chrome.Button(c, "关闭会话", chrome.SizeLarge).Clicked() {
			_ = d.CloseSession(r.sess.ID)
		}
		return
	case labelStarting, labelStopping:
		return
	case labelFailed:
		if !r.orphan && chrome.PrimaryButton(c, "重试").Clicked() {
			_ = d.StartSession(r.entry)
		}
	case labelIdle:
		if chrome.PrimaryButton(c, "启动分析").Clicked() {
			_ = d.StartSession(r.entry)
		}
	}
	if !r.orphan && chrome.Button(c, "从数据库移除", chrome.SizeMedium).Clicked() {
		selectRow(d, r)
		d.RemoveSelected()
	}
}

func body(c *ui.Context, d *desk.Desk, r libRow, st state) {
	switch st.label {
	case labelFailed, labelMissing:
		reason := st.reason
		if reason == "" {
			reason = "worker 没有带回原因。"
		}
		chrome.FaultCard(c, "原因", reason)
	case labelReady:
		ui.Grid(c).Columns(2).Gap(16).Children(func() {
			infoCard(c, r.sess)
			agentCard(c, d)
			packCard(c, d, r.sess)
			recentCard(c, d, r)
		})
	case labelStarting, labelStopping:
		ui.Grid(c).Columns(2).Gap(16).Children(func() {
			progressCard(c, d, r, st)
			agentCard(c, d)
		})
	default:
		ui.Grid(c).Columns(2).Gap(16).Children(func() {
			fileCard(c, r)
			recentCard(c, d, r)
		})
	}
}

func infoCard(c *ui.Context, s *session.Session) {
	m := s.Metadata
	proc := strings.ToUpper(metaString(m, "processor"))
	if bits := metaInt(m, "bitness"); proc != "" && bits > 0 {
		proc = fmt.Sprintf("%s · %d 位", proc, bits)
	}
	auto := "—"
	switch {
	case m == nil:
	case metaBool(m, "autoAnalysisComplete"):
		auto = "已完成"
	case metaBool(m, "autoAnalysisRunning"):
		auto = "进行中"
	default:
		auto = "未开始"
	}
	chrome.Card(c, "数据库信息", func() {
		chrome.Facts(c, 80, 10,
			chrome.Fact{Label: "格式", Value: orDash(metaString(m, "fileType"))},
			chrome.Fact{Label: "处理器", Value: orDash(proc)},
			chrome.Fact{Label: "基址", Value: orDash(metaString(m, "imageBase")), Mono: true},
			chrome.Fact{Label: "自动分析", Value: auto},
		)
	})
}

func agentCard(c *ui.Context, d *desk.Desk) {
	chrome.Card(c, "代理连接", func() {
		ui.Row(c).Height(40).Padding(0, 6, 0, 12).Radius(theme.ControlR).Background(theme.Well).
			Gap(10).AlignItems(ui.Center).Children(func() {
			chrome.Mono(c, chrome.MCPURL, 12, theme.Text).Grow(1).MinWidth(0)
			chrome.CopyButton(c, d, "agent-url", "复制", chrome.MCPURL, chrome.SizeInline)
		})
		ui.Row(c).Gap(8).Padding(14, 0, 0, 0).AlignItems(ui.Center).Children(func() {
			if d.Listening {
				chrome.Dot(c, theme.Live, 8)
				ui.Text(c, "仅本机可访问").FontSize(13).TextColor(theme.TextSoft)
			} else {
				chrome.Dot(c, theme.Idle, 8)
				ui.Text(c, "尚未监听").FontSize(13).TextColor(theme.TextSoft)
			}
		})
	})
}

func packCard(c *ui.Context, d *desk.Desk, s *session.Session) {
	p := d.Sessions.PackOf(s.ID)
	chrome.Card(c, "分析包", func() {
		if p == nil {
			ui.Text(c, "未挂载").FontSize(13).TextColor(theme.TextSoft)
			ui.Text(c, "项目旁的 pack.json。挂上后，Agent 多出包里的工具。").FontSize(13).TextColor(theme.Muted).Wrap().Padding(10, 0, 0, 0)
			ui.Row(c).Padding(14, 0, 0, 0).Children(func() {
				if chrome.Button(c, "挂载", chrome.SizeSmall).Clicked() && d.RequestPack != nil {
					d.RequestPack(s.ID)
				}
			})
			return
		}
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			chrome.Mono(c, filepath.Base(p.Path), 12, theme.Text).Grow(1).MinWidth(0)
			ui.Text(c, "已挂载").FontSize(13).TextColor(theme.TextSoft)
		})
		ui.Text(c, packKinds(p.Tools)).FontSize(13).TextColor(theme.TextSoft).SingleLine().Padding(10, 0, 0, 0)
		ui.Row(c).Padding(14, 0, 0, 0).Children(func() {
			if chrome.Button(c, "卸下", chrome.SizeSmall).Clicked() {
				d.DetachPack(s.ID)
			}
		})
	})
}

func packKinds(tools []pack.Tool) string {
	kinds := make([]string, 0, len(tools))
	for _, t := range tools {
		if !slices.Contains(kinds, t.Kind) {
			kinds = append(kinds, t.Kind)
		}
	}
	if len(kinds) == 0 {
		return "没有工具"
	}
	return strings.Join(kinds, " · ")
}

func recentCard(c *ui.Context, d *desk.Desk, r libRow) {
	type call struct {
		fn string
		at time.Time
	}
	var calls []call
	for _, tr := range d.Traces.Entries() {
		if traceMatches(tr.Target, r) {
			calls = append(calls, call{tr.Function, tr.Timestamp})
			if len(calls) == 3 {
				break
			}
		}
	}
	now := d.Now()
	chrome.Card(c, "最近记录", func() {
		if len(calls) == 0 {
			ui.Text(c, "还没有调用").FontSize(13).TextColor(theme.Muted)
			return
		}
		ui.Column(c).Gap(10).Children(func() {
			for _, cl := range calls {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					chrome.Mono(c, cl.fn, 12, theme.Text).Grow(1).MinWidth(0)
					ui.Text(c, chrome.Ago(now, cl.at)).FontSize(12).Font(theme.FontMono).TextColor(theme.Muted)
				})
			}
		})
	})
}

func progressCard(c *ui.Context, d *desk.Desk, r libRow, st state) {
	line := "正在收拢数据库。完成后回到未打开。"
	var since time.Time
	if st.label == labelStarting {
		line = "正在拉起 worker。"
		if r.sess != nil {
			line = "正在打开数据库，等待连上。"
			since = r.sess.StartedAt
		} else if t, ok := d.LaunchStarted(r.entry.ID); ok {
			since = t
		}
	}
	chrome.Card(c, "进度", func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			chrome.BusyDot(c, 8)
			ui.Text(c, line).FontSize(13).TextColor(theme.Text).Wrap().Grow(1)
		})
		if !since.IsZero() {
			ui.Text(c, "已等待 "+waitLabel(d.Now().Sub(since))).FontSize(12).Font(theme.FontMono).TextColor(theme.Muted).Padding(10, 0, 0, 0)
		}
	})
}

func fileCard(c *ui.Context, r libRow) {
	size, modified := "—", "—"
	if info, err := os.Stat(r.entry.Path); err == nil {
		size = formatSize(info.Size())
		modified = info.ModTime().Format("2006-01-02 15:04")
	}
	chrome.Card(c, "文件信息", func() {
		chrome.Facts(c, 80, 10,
			chrome.Fact{Label: "大小", Value: size},
			chrome.Fact{Label: "修改时间", Value: modified},
			chrome.Fact{Label: "目录", Value: product.ShortPath(filepath.Dir(r.entry.Path)), Mono: true},
		)
	})
}

func waitLabel(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

func formatSize(sz int64) string {
	switch {
	case sz >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(sz)/(1<<20))
	case sz >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(sz)/(1<<10))
	default:
		return fmt.Sprintf("%d B", sz)
	}
}

func traceMatches(target string, r libRow) bool {
	if target == "" {
		return false
	}
	if target == r.entry.Name {
		return true
	}
	return r.sess != nil && (target == r.sess.ID || target == r.sess.TargetName)
}

func selectRow(d *desk.Desk, r libRow) {
	if r.orphan && r.sess != nil {
		d.SelectedSession = r.sess.ID
		return
	}
	d.Catalog.Select(r.entry.ID)
	if r.sess != nil {
		d.SelectedSession = r.sess.ID
		return
	}
	d.SelectedSession = ""
}

func sessionFor(d *desk.Desk, paths ...string) *session.Session {
	for _, s := range d.Sessions.Sessions() {
		for _, p := range paths {
			if samePath(s.SourcePath, p) || samePath(s.DatabasePath, p) {
				return s
			}
		}
	}
	return nil
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func firstPath(paths ...string) string {
	for _, p := range paths {
		if p != "" {
			return p
		}
	}
	return ""
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func metaString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func metaBool(m map[string]any, key string) bool {
	b, _ := m[key].(bool)
	return b
}

func metaInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}
