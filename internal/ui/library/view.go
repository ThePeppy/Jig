// Package library draws the file list. An open worker belongs to its file,
// not to a separate page. A worker whose path is not in the catalog is still a row.
package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/catalog"
	"jig/internal/desk"
	"jig/internal/product"
	"jig/internal/session"
	"jig/internal/ui/chrome"
	"jig/internal/ui/theme"
)

// View renders the library, including any open headless worker.
func View(c *ui.Context, d *desk.Desk) {
	rows := libraryRows(d)
	if len(rows) == 0 {
		empty(c, d)
		return
	}
	ui.Row(c).Grow(1).Gap(0).AlignItems(ui.Stretch).Children(func() {
		list(c, d, rows)
		if focus, ok := focused(d, rows); ok {
			detail(c, d, focus)
		}
	})
}

type libRow struct {
	entry  catalog.Entry
	sess   *session.Session
	orphan bool
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

func focused(d *desk.Desk, rows []libRow) (libRow, bool) {
	if len(rows) == 0 {
		return libRow{}, false
	}
	if d.SelectedSession != "" {
		for _, r := range rows {
			if r.orphan && r.sess != nil && r.sess.ID == d.SelectedSession {
				return r, true
			}
		}
	}
	if e, ok := d.Catalog.Selected(); ok {
		for _, r := range rows {
			if !r.orphan && r.entry.ID == e.ID {
				return r, true
			}
		}
	}
	return rows[0], true
}

func sameFocus(a, b libRow) bool {
	if a.orphan || b.orphan {
		return a.orphan && b.orphan && a.sess != nil && b.sess != nil && a.sess.ID == b.sess.ID
	}
	return a.entry.ID != "" && a.entry.ID == b.entry.ID
}

func empty(c *ui.Context, d *desk.Desk) {
	pane := chrome.Pane(c).Grow(1)
	st := d.Runtime()
	idaReady := d.IdaDir != "" || st.Found

	pane.Children(func() {
		ui.Column(c).Grow(1).Center().Padding(32, 32).Transition(chrome.DetailTransition).Children(func() {
			ui.Column(c).Width(500).AlignItems(ui.Center).Gap(18).Children(func() {
				chrome.HeroEmblem(c, chrome.IconLibrary, theme.Text)

				ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "还没有文件").FontSize(18).FontWeight(500).TextColor(theme.Text)
						chrome.Stage(c, "idle", "空")
					})

					ui.Text(c, "把任意二进制或已有数据库拖进窗口,或从下面添加。文件进入库,不会立刻打开。").
						FontSize(13).
						TextColor(theme.TextMuted).
						TextAlign(ui.Center)
				})

				ui.Row(c).Padding(2, 0, 6, 0).Children(func() {
					btn := chrome.PrimaryButton(c, "添加文件", chrome.IconPlus)
					if btn.Clicked() && d.RequestAdd != nil {
						d.RequestAdd()
					}
				})

				chrome.Plate(c).FillWidth().Padding(14, 18).Children(func() {
					ui.Row(c).Justify(ui.Center).Gap(12).AlignItems(ui.Center).Children(func() {
						chrome.Stage(c, "idle", "入库")
						chrome.Join(c)
						if idaReady {
							chrome.Stage(c, "ok", "指定目录")
						} else {
							chrome.Stage(c, "idle", "指定目录")
						}
						chrome.Join(c)
						chrome.Stage(c, "idle", "打开")
					})
				})

				chrome.Card(c).FillWidth().Padding(12, 16).Children(func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, chrome.IconFolder).Size(14, 14).TextColor(theme.TextMuted)
						ui.Text(c, "IDA 目录").FontSize(12).FontWeight(500).TextColor(theme.Text)
						ui.Text(c, idaLine(d)).FontSize(12).Font("SF Mono").TextColor(theme.TextMuted).Grow(1).SingleLine()
						pick := chrome.GhostButton(c, "指定", nil, false)
						if pick.Clicked() && d.RequestIdaDir != nil {
							d.RequestIdaDir()
						}
					})
				})
			})
		})

		chrome.Status(c, d, "空", "idle")
	})
}

func list(c *ui.Context, d *desk.Desk, rows []libRow) {
	pane := chrome.Sidebar(c)
	openCount := 0
	for _, r := range rows {
		if r.sess != nil {
			openCount++
		}
	}
	focus, _ := focused(d, rows)

	pane.Children(func() {
		ui.Row(c).Height(36).Padding(0, 14).BorderWidth(0, 0, 1, 0).BorderColor(theme.BorderSubtle).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("共 %d 项", len(rows))).FontSize(11).TextColor(theme.TextMuted)
			ui.Spacer(c)
			if openCount > 0 {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Background(theme.Live)
					ui.Text(c, fmt.Sprintf("%d 已打开", openCount)).FontSize(11).TextColor(theme.Text)
				})
			}
		})

		ui.Scroll(c).Grow(1).Padding(6, 6).Gap(3).Children(func() {
			for _, r := range rows {
				row(c, d, r, sameFocus(r, focus))
			}
		})
	})
}

func row(c *ui.Context, d *desk.Desk, r libRow, on bool) {
	accent := rowAccent(d, r)
	key := r.entry.ID
	if r.orphan && r.sess != nil {
		key = "sess-" + r.sess.ID
	}

	item := ui.Row(c).
		Key("lib-row-"+key).
		MinHeight(52).
		Padding(7, 10).
		Radius(theme.CardR).
		Cursor(ui.CursorPointer).
		Transition(chrome.ColorTransition).
		AlignItems(ui.Center).
		Gap(10).
		Focusable().
		Label(r.entry.Name)

	if item.Clicked() {
		selectRow(d, r)
		on = true
	}

	chrome.StyleSidebarRow(item, on, accent)

	item.Children(func() {
		chrome.CartridgeTile(c, extShort(tilePath(r)), accent, false)

		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				nameColor := theme.TextMuted
				if on {
					nameColor = theme.Text
				}
				ui.Text(c, r.entry.Name).FontSize(13).FontWeight(500).TextColor(nameColor).Grow(1).SingleLine()

				if !r.orphan && r.entry.Missing() {
					chrome.Stage(c, "bad", "文件缺失")
				} else if stopping(r) {
					chrome.LiveStage(c, "关闭中")
				} else if starting(d, r) {
					chrome.LiveStage(c, "启动中")
				} else if r.sess != nil {
					chrome.Stage(c, stateTone(r.sess.State), stateLabel(r.sess.State))
				}
			})

			ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
				if r.orphan && r.sess != nil {
					ui.Text(c, r.sess.ID).FontSize(11).Font("SF Mono").TextColor(theme.Muted)
					ui.Text(c, "·").FontSize(11).TextColor(theme.Muted)
					ui.Text(c, r.sess.Elapsed()).FontSize(11).Font("SF Mono").TextColor(theme.Muted)
					return
				}
				ui.Text(c, "文件").FontSize(11).TextColor(theme.Muted)
				ui.Text(c, "·").FontSize(11).TextColor(theme.Muted)
				ui.Text(c, product.ShortPath(filepath.Dir(r.entry.Source))).FontSize(11).Font("SF Mono").TextColor(theme.Muted).Grow(1).SingleLine()
			})
		})
	})
}

func detail(c *ui.Context, d *desk.Desk, r libRow) {
	accent := rowAccent(d, r)
	st := d.Runtime()
	idaReady := d.IdaDir != "" || st.Found
	missing := !r.orphan && r.entry.Missing()
	key := r.entry.ID
	if r.orphan && r.sess != nil {
		key = "sess-" + r.sess.ID
	}
	headerPath := r.entry.Path
	if r.orphan && r.sess != nil {
		headerPath = firstPath(r.sess.SourcePath, r.sess.DatabasePath)
	}

	pane := chrome.Pane(c).Grow(1)
	pane.Children(func() {
		ui.Row(c).
			Key("lib-head-"+key).
			Padding(22, 28, 18, 28).
			BorderWidth(0, 0, 1, 0).
			BorderColor(theme.BorderSubtle).
			Gap(14).
			AlignItems(ui.Center).
			Transition(chrome.DetailTransition).
			Children(func() {
				chrome.CartridgeTile(c, extShort(tilePath(r)), accent, true)

				ui.Column(c).Grow(1).Gap(5).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, r.entry.Name).FontSize(18).FontWeight(500).TextColor(theme.Text).SingleLine()

						if missing {
							chrome.Stage(c, "bad", "文件缺失")
						} else if stopping(r) {
							chrome.LiveStage(c, "关闭中")
						} else if starting(d, r) {
							chrome.LiveStage(c, "启动中")
						} else if r.sess != nil {
							chrome.Stage(c, stateTone(r.sess.State), stateLabel(r.sess.State))
						} else {
							chrome.Stage(c, "idle", "未打开")
						}
					})

					ui.Text(c, product.ShortPath(headerPath)).FontSize(12).Font("SF Mono").TextColor(theme.TextMuted).SingleLine()
				})

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					switch {
					case stopping(r):
						chrome.BusyGhost(c, "关闭中", true)
					case r.sess != nil:
						closeBtn := chrome.GhostButton(c, "关闭会话", nil, true)
						if closeBtn.Clicked() {
							_ = d.CloseSession(r.sess.ID)
						}
					case starting(d, r):
						chrome.BusyButton(c, "启动中")
					case !missing:
						launch := chrome.PrimaryButton(c, "启动分析", chrome.IconPlay)
						if launch.Clicked() {
							_ = d.StartSession(r.entry)
						}
					}
					if !r.orphan {
						del := chrome.GhostButton(c, "移出", nil, false)
						if del.Clicked() {
							d.RemoveSelected()
						}
					}
				})
			})

		ui.Scroll(c).
			Key("lib-body-"+key).
			Grow(1).
			Padding(20, 28).
			Gap(18).
			Transition(chrome.DetailTransition).
			Children(func() {
				if missing {
					ui.Row(c).Padding(8, 12).Radius(theme.CardR).Background(theme.TroubleBg).Border(1, theme.Trouble).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, chrome.IconAlert).Size(14, 14).TextColor(theme.Trouble)
						ui.Text(c, "文件缺失: 磁盘上已找不到该路径").FontSize(12).TextColor(theme.Trouble)
					})
				}
				workCard(c, d, r)

				if r.sess != nil && r.sess.State != session.StateLaunching {
					sessionBody(c, r.sess)
				}

				if !r.orphan {
					chrome.Card(c).Children(func() {
						ui.Row(c).Dividers(1, theme.BorderSubtle).AlignItems(ui.Stretch).Children(func() {
							chrome.ReadoutCell(c, "类型", "文件", "装载与报错由 idalib 决定", accent)
							chrome.ReadoutCell(c, "大小", formatFileSize(r.entry.Path), filepath.Base(r.entry.Path), theme.Run)

							ui.Column(c).Grow(1.3).Basis(0).Padding(14, 18).Gap(7).Justify(ui.Center).Children(func() {
								ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
									chrome.Stage(c, "done", "入库")
									chrome.Join(c)
									if idaReady {
										chrome.Stage(c, "done", "指定目录")
									} else {
										chrome.Stage(c, "idle", "指定目录")
									}
									chrome.Join(c)
									if r.sess != nil {
										chrome.Stage(c, "live", "打开")
									} else {
										chrome.Stage(c, "idle", "打开")
									}
								})
								ui.Text(c, "打开时询问装载器，失败带回 idalib 的错误。").FontSize(11).TextColor(theme.TextMuted).SingleLine()
							})
						})
					})
				}

				if r.sess == nil || samePath(r.sess.SourcePath, r.sess.DatabasePath) || r.sess.DatabasePath == "" {
					openBadge := "未打开"
					openTone := "idle"
					switch {
					case stopping(r):
						openBadge = "关闭中"
						openTone = "live"
					case starting(d, r):
						openBadge = "启动中"
						openTone = "live"
					case r.sess != nil:
						openBadge = "已挂载"
						openTone = "live"
					}
					chrome.PathSlotCard(c, "路径", r.entry.Name, product.ShortPath(r.entry.Path), openBadge, openTone)
				}

				callLog(c, d, r)
			})

		statusLabel := "未打开"
		statusTone := "idle"
		switch {
		case stopping(r):
			statusLabel = "关闭中"
			statusTone = "live"
		case starting(d, r):
			statusLabel = "启动中"
			statusTone = "live"
		case r.sess != nil:
			statusLabel = stateLabel(r.sess.State)
			statusTone = stateTone(r.sess.State)
		}
		chrome.Status(c, d, statusLabel, statusTone)
	})
}

func workCard(c *ui.Context, d *desk.Desk, r libRow) {
	if stopping(r) {
		chrome.Work(c, "关闭中", "", "正在收拢数据库。完成后回到未打开。", []string{"通知", "收拢", "离开"}, 1)
		return
	}
	if !starting(d, r) {
		return
	}
	step := 0
	detail := "正在拉起 worker。"
	if r.sess != nil && r.sess.State == session.StateLaunching {
		step = 1
		detail = "正在打开数据库，等待连上。"
	}
	meta := ""
	if t, ok := d.LaunchStarted(r.entry.ID); ok {
		meta = waitLabel(t)
	}
	chrome.Work(c, "启动中", meta, detail, []string{"拉起", "打开", "就绪"}, step)
}

func starting(d *desk.Desk, r libRow) bool {
	if stopping(r) {
		return false
	}
	if r.sess != nil && r.sess.State == session.StateLaunching {
		return true
	}
	if r.orphan || r.sess != nil && r.sess.State == session.StateReady {
		return false
	}
	return d.Launching(r.entry.ID)
}

func stopping(r libRow) bool {
	return r.sess != nil && r.sess.State == session.StateStopping
}

func waitLabel(t time.Time) string {
	d := time.Since(t).Round(time.Second)
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

func sessionBody(c *ui.Context, s *session.Session) {
	chrome.Card(c).Children(func() {
		ui.Row(c).Dividers(1, theme.BorderSubtle).AlignItems(ui.Stretch).Children(func() {
			chrome.ReadoutCell(c, "会话 ID", s.ID, "本机实例句柄", theme.Run)
			chrome.ReadoutCell(c, "进程 PID", fmt.Sprintf("%d", s.PID), "分析进程", theme.Live)
			chrome.ReadoutCell(c, "运行时长", s.Elapsed(), "无界面驻留中", theme.Live)
		})
	})

	src := s.SourcePath
	db := s.DatabasePath
	if src == "" {
		src = db
	}
	if db == "" || samePath(src, db) {
		return
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		chrome.PathSlotCard(c, "样本路径", filepath.Base(src), product.ShortPath(src), "已连接", "done").Grow(1).Basis(0)
		chrome.ConduitBridge(c, "127.0.0.1:9399", true)
		chrome.PathSlotCard(c, "数据库", filepath.Base(db), product.ShortPath(db), stateLabel(s.State), stateTone(s.State)).Grow(1).Basis(0)
	})
}

func callLog(c *ui.Context, d *desk.Desk, r libRow) {
	type callRow struct {
		fn, durStr, sum string
		dur             time.Duration
		ok              bool
	}
	var fileTraces []callRow
	for _, tr := range d.Traces.Entries() {
		if !traceMatches(tr.Target, r) {
			continue
		}
		fileTraces = append(fileTraces, callRow{
			fn:     tr.Function,
			durStr: tr.FormatDuration(),
			sum:    tr.Summary,
			dur:    tr.Duration,
			ok:     tr.Status != "error",
		})
	}

	ui.Column(c).Gap(8).Children(func() {
		chrome.Kicker(c, "这次打开以来", len(fileTraces))
		if len(fileTraces) > 0 {
			chrome.Card(c).Dividers(1, theme.BorderSubtle).Children(func() {
				for _, tr := range fileTraces {
					ui.Row(c).MinHeight(40).Padding(0, 16).Gap(14).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(6, 6).Background(chrome.CallMark(tr.ok))
						ui.Text(c, tr.fn).FontSize(12).FontWeight(500).Font(theme.FontMono).TextColor(theme.Text).Width(156).SingleLine()
						ui.Row(c).Width(96).Gap(8).AlignItems(ui.Center).Children(func() {
							chrome.LatencyBar(c, tr.dur, tr.ok)
							ui.Text(c, tr.durStr).FontSize(11).Font("SF Mono").TextColor(theme.TextMuted)
						})
						ui.Text(c, tr.sum).FontSize(12).TextColor(theme.TextMuted).Grow(1).SingleLine()
					})
				}
			})
			return
		}
		chrome.Plate(c).MinHeight(72).Center().Padding(16, 20).Children(func() {
			ui.Column(c).AlignItems(ui.Center).Gap(6).Children(func() {
				if r.sess != nil {
					ui.Text(c, "等待外部 Agent 调用").FontSize(13).FontWeight(500).TextColor(theme.Text)
					ui.Text(c, "MCP 请求送达后将在此显示耗时与摘要。").FontSize(12).TextColor(theme.TextMuted)
					return
				}
				ui.Text(c, "尚无针对该文件的调用记录").FontSize(13).FontWeight(500).TextColor(theme.Text)
				ui.Text(c, "点击右上角「启动分析」拉起 idalib 后，外部 Agent 的工具调用将在此实时呈现。").FontSize(12).TextColor(theme.TextMuted)
			})
		})
	})
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

func tilePath(r libRow) string {
	if r.orphan && r.sess != nil {
		return firstPath(r.sess.DatabasePath, r.sess.SourcePath)
	}
	return r.entry.Path
}

func rowAccent(d *desk.Desk, r libRow) ui.Color {
	if starting(d, r) || stopping(r) {
		return theme.Live
	}
	if r.sess != nil && r.sess.State == session.StateError {
		return theme.Trouble
	}
	if !r.orphan && r.entry.Missing() {
		return theme.Trouble
	}
	if r.sess != nil {
		return theme.Live
	}
	return theme.Muted
}

func stateLabel(s session.State) string {
	switch s {
	case session.StateLaunching:
		return "启动中"
	case session.StateReady:
		return "就绪"
	case session.StateStopping:
		return "关闭中"
	case session.StateClosed:
		return "已关闭"
	case session.StateError:
		return "失败"
	default:
		return string(s)
	}
}

func stateTone(s session.State) string {
	switch s {
	case session.StateError:
		return "bad"
	case session.StateClosed:
		return "idle"
	default:
		return "live"
	}
}

func extShort(path string) string {
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		return "BIN"
	}
	if len(ext) > 4 {
		return ext[:4]
	}
	return ext
}

func formatFileSize(path string) string {
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		ext = "BIN"
	}
	info, err := os.Stat(path)
	if err != nil {
		return ext + " 文件"
	}
	sz := info.Size()
	switch {
	case sz >= 1024*1024:
		return fmt.Sprintf("%.2f MB · %s 文件", float64(sz)/(1024*1024), ext)
	case sz >= 1024:
		return fmt.Sprintf("%.1f KB · %s 文件", float64(sz)/1024, ext)
	default:
		return fmt.Sprintf("%d B · %s 文件", sz, ext)
	}
}

func idaLine(d *desk.Desk) string {
	if d.IdaDir != "" {
		return product.ShortPath(d.IdaDir)
	}
	st := d.Runtime()
	if st.Found {
		return "探测到 " + product.ShortPath(st.Install)
	}
	return "未设置"
}
