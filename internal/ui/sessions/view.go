package sessions

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"app/internal/desk"
	"app/internal/product"
	"app/internal/session"
	"app/internal/ui/chrome"
	"app/internal/ui/theme"
)

// View renders the background idalib headless worker status or active sessions.
func View(c *ui.Context, d *desk.Desk) {
	listItems := d.Sessions.Sessions()
	if len(listItems) == 0 {
		empty(c, d)
		return
	}

	ui.Row(c).Grow(1).Gap(0).AlignItems(ui.Stretch).Children(func() {
		list(c, d, listItems)
		selected := listItems[0]
		for _, s := range listItems {
			if s.ID == d.SelectedSession {
				selected = s
				break
			}
		}
		detail(c, d, selected, len(listItems))
	})
}

func empty(c *ui.Context, d *desk.Desk) {
	pane := chrome.Pane(c).Grow(1)
	pane.Children(func() {
		ui.Column(c).Grow(1).Center().Padding(32, 32).Transition(chrome.DetailTransition).Children(func() {
			ui.Column(c).Width(520).AlignItems(ui.Center).Gap(18).Children(func() {
				chrome.HeroEmblem(c, chrome.IconSessions, theme.Text)

				ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "还没有会话").FontSize(18).FontWeight(500).TextColor(theme.Text)
						chrome.Stage(c, "idle", "空")
					})

					ui.Text(c, "库里的文件还没交给 idalib。这里只显示本机无界面会话,不显示 IDA 窗口。").
						FontSize(13).
						TextColor(theme.TextMuted).
						TextAlign(ui.Center)
				})

				// Visual local conduit diagram in a harmonious slate card
				chrome.Plate(c).FillWidth().Padding(16, 20).Children(func() {
					ui.Row(c).Justify(ui.Center).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Column(c).AlignItems(ui.Center).Gap(2).Children(func() {
							ui.Text(c, "AI Agent").FontSize(12).FontWeight(500).TextColor(theme.Text)
							ui.Text(c, "MCP 客户端").FontSize(11).TextColor(theme.TextMuted)
						})
						chrome.ConduitBridge(c, "127.0.0.1:9401", d.Listening)
						ui.Column(c).AlignItems(ui.Center).Gap(2).Children(func() {
							ui.Text(c, "MCP 服务").FontSize(12).FontWeight(500).TextColor(theme.Text)
							ui.Text(c, "本机中心").FontSize(11).TextColor(theme.TextMuted)
						})
						chrome.ConduitBridge(c, "127.0.0.1:9399", d.Listening)
						ui.Column(c).AlignItems(ui.Center).Gap(2).Children(func() {
							ui.Text(c, "Worker").FontSize(12).FontWeight(500).TextColor(theme.Text)
							ui.Text(c, "idalib 进程").FontSize(11).TextColor(theme.TextMuted)
						})
					})
				})

				if d.IdaDir != "" {
					chrome.Card(c).FillWidth().Children(func() {
						chrome.Line(c, "IDA 路径", product.ShortPath(d.IdaDir))
					})
				}
			})
		})

		chrome.Status(c, d, "空", "idle")
	})
}

func list(c *ui.Context, d *desk.Desk, items []*session.Session) {
	pane := chrome.Sidebar(c)
	pane.Children(func() {
		ui.Row(c).Height(36).Padding(0, 14).BorderWidth(0, 0, 1, 0).BorderColor(theme.BorderSubtle).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%d 个会话", len(items))).FontSize(11).TextColor(theme.TextMuted)
			ui.Spacer(c)
			ui.Text(c, "无界面").FontSize(11).TextColor(theme.TextMuted)
		})

		ui.Scroll(c).Grow(1).Padding(6, 6).Gap(3).Children(func() {
			selectedID := items[0].ID
			for _, s := range items {
				if s.ID == d.SelectedSession {
					selectedID = s.ID
					break
				}
			}
			for _, s := range items {
				row(c, d, s, s.ID == selectedID)
			}
		})
	})
}

func row(c *ui.Context, d *desk.Desk, s *session.Session, on bool) {
	tone := "ok"
	accent := theme.Live
	if s.State == session.StateError {
		tone = "bad"
		accent = theme.Trouble
	} else if s.State == session.StateClosed {
		tone = "idle"
		accent = theme.Idle
	}

	item := ui.Row(c).
		Key("sess-row-"+s.ID).
		MinHeight(52).
		Padding(7, 10).
		Radius(theme.CardR).
		Cursor(ui.CursorPointer).
		Transition(chrome.ColorTransition).
		AlignItems(ui.Center).
		Gap(10).
		Focusable().
		Label(s.TargetName)

	if item.Clicked() {
		d.SelectedSession = s.ID
		on = true
	}

	chrome.StyleSidebarRow(item, on, accent)

	item.Children(func() {
		chrome.CartridgeTile(c, extCode(s.DatabasePath), accent, false)

		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				nameCol := theme.TextMuted
				if on {
					nameCol = theme.Text
				}
				ui.Text(c, s.TargetName).FontSize(13).FontWeight(500).TextColor(nameCol).Grow(1).SingleLine()
				chrome.Stage(c, tone, stateLabel(s.State))
			})

			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, s.ID).FontSize(11).Font("SF Mono").TextColor(theme.Muted)
				ui.Text(c, "·").FontSize(11).TextColor(theme.Muted)
				ui.Text(c, s.Elapsed()).FontSize(11).Font("SF Mono").TextColor(theme.Muted)
			})
		})
	})
}

func detail(c *ui.Context, d *desk.Desk, s *session.Session, total int) {
	pane := chrome.Pane(c).Grow(1)
	pane.Children(func() {
		// Tactile specimen header
		ui.Row(c).
			Key("sess-head-"+s.ID).
			Padding(22, 28, 18, 28).
			BorderWidth(0, 0, 1, 0).
			BorderColor(theme.BorderSubtle).
			Gap(14).
			AlignItems(ui.Center).
			Transition(chrome.DetailTransition).
			Children(func() {
				chrome.CartridgeTile(c, extCode(s.DatabasePath), theme.Live, true)

				ui.Column(c).Grow(1).Gap(5).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, s.TargetName).FontSize(18).FontWeight(500).TextColor(theme.Text).SingleLine()
						chrome.Stage(c, "ok", stateLabel(s.State))
					})

					ui.Text(c, product.ShortPath(s.DatabasePath)).FontSize(12).Font("SF Mono").TextColor(theme.TextMuted).SingleLine()
				})

				closeBtn := chrome.GhostButton(c, "关闭会话", nil, true)
				if closeBtn.Clicked() {
					_ = d.CloseSession(s.ID)
				}
			})

		// Workspace body
		ui.Scroll(c).
			Key("sess-body-"+s.ID).
			Grow(1).
			Padding(20, 28).
			Gap(18).
			Transition(chrome.DetailTransition).
			Children(func() {
				// 3-Segment Horizontal Readout Strip (Session ID, PID, Elapsed)
				chrome.Card(c).Children(func() {
					ui.Row(c).Dividers(1, theme.BorderSubtle).AlignItems(ui.Stretch).Children(func() {
						chrome.ReadoutCell(c, "会话 ID", s.ID, "本机实例句柄", theme.Run)
						chrome.ReadoutCell(c, "进程 PID", fmt.Sprintf("%d", s.PID), "idalib Python 进程", theme.Live)
						chrome.ReadoutCell(c, "运行时长", s.Elapsed(), "无界面驻留中", theme.Live)
					})
				})

				// Dual-Slot Side-by-Side Path Routing Module (Source ──▸ Database)
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					chrome.PathSlotCard(c, "样本路径", filepath.Base(s.SourcePath), product.ShortPath(s.SourcePath), "已连接", "done")
					chrome.ConduitBridge(c, "127.0.0.1:9399", true)
					chrome.PathSlotCard(c, "数据库", filepath.Base(s.DatabasePath), product.ShortPath(s.DatabasePath), stateLabel(s.State), "ok")
				})

				type callRow struct {
					fn, durStr, sum string
					dur             time.Duration
					ok              bool
				}
				var calls []callRow
				for _, tr := range d.Traces.Entries() {
					if tr.Target == s.TargetName || tr.Target == s.ID {
						calls = append(calls, callRow{
							fn:     tr.Function,
							durStr: tr.FormatDuration(),
							sum:    tr.Summary,
							dur:    tr.Duration,
							ok:     tr.Status != "error",
						})
					}
				}

				ui.Column(c).Gap(8).Children(func() {
					chrome.Kicker(c, "这次打开以来", len(calls))
					if len(calls) > 0 {
						chrome.Card(c).Dividers(1, theme.BorderSubtle).Children(func() {
							for _, tr := range calls {
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
					} else {
						chrome.Plate(c).MinHeight(72).Center().Padding(16, 20).Children(func() {
							ui.Column(c).AlignItems(ui.Center).Gap(6).Children(func() {
								ui.Text(c, "等待外部 Agent 调用").FontSize(13).FontWeight(500).TextColor(theme.Text)
								ui.Text(c, "会话已就绪，MCP 请求送达后将在此实时显示执行耗时与返回摘要。").FontSize(12).TextColor(theme.TextMuted)
							})
						})
					}
				})
			})

		chrome.Status(c, d, fmt.Sprintf("%d 个会话活跃中", total), "ok")
	})
}

func extCode(path string) string {
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		return "BIN"
	}
	if len(ext) > 4 {
		return ext[:4]
	}
	return ext
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
