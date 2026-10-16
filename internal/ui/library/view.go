package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"app/internal/catalog"
	"app/internal/desk"
	"app/internal/intake"
	"app/internal/product"
	"app/internal/ui/chrome"
	"app/internal/ui/theme"
)

// View renders the binary & database library or the centered empty state.
func View(c *ui.Context, d *desk.Desk) {
	entries := d.Catalog.Entries()
	if len(entries) == 0 {
		empty(c, d)
		return
	}
	ui.Row(c).Grow(1).Gap(0).AlignItems(ui.Stretch).Children(func() {
		list(c, d, entries)
		detail(c, d)
	})
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

				// Soft slate workflow track
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

				// Tactile IDA runtime card
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

func list(c *ui.Context, d *desk.Desk, entries []catalog.Entry) {
	pane := chrome.Sidebar(c)

	openCount := 0
	for _, e := range entries {
		if _, ok := d.Sessions.FindByPath(e.Path); ok {
			openCount++
		}
	}

	pane.Children(func() {
		ui.Row(c).Height(36).Padding(0, 14).BorderWidth(0, 0, 1, 0).BorderColor(theme.BorderSubtle).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("共 %d 项", len(entries))).FontSize(11).TextColor(theme.TextMuted)
			ui.Spacer(c)
			if openCount > 0 {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Background(theme.Live)
					ui.Text(c, fmt.Sprintf("%d 已打开", openCount)).FontSize(11).TextColor(theme.Text)
				})
			}
		})

		ui.Scroll(c).Grow(1).Padding(6, 6).Gap(3).Children(func() {
			for _, e := range entries {
				curr, hasSelected := d.Catalog.Selected()
				row(c, d, e, hasSelected && e.ID == curr.ID)
			}
		})
	})
}

func row(c *ui.Context, d *desk.Desk, e catalog.Entry, on bool) {
	_, hasSession := d.Sessions.FindByPath(e.Path)
	accent := tileAccent(e, hasSession)

	item := ui.Row(c).
		Key("lib-row-"+e.ID).
		MinHeight(52).
		Padding(7, 10).
		Radius(theme.CardR).
		Cursor(ui.CursorPointer).
		Transition(chrome.ColorTransition).
		AlignItems(ui.Center).
		Gap(10).
		Focusable().
		Label(e.Name)

	if item.Clicked() {
		d.Catalog.Select(e.ID)
		on = true
	}

	chrome.StyleSidebarRow(item, on, accent)

	item.Children(func() {
		chrome.CartridgeTile(c, extShort(e.Path), accent, false)

		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				nameColor := theme.TextMuted
				if on {
					nameColor = theme.Text
				}
				ui.Text(c, e.Name).FontSize(13).FontWeight(500).TextColor(nameColor).Grow(1).SingleLine()

				if e.Missing() {
					chrome.Stage(c, "bad", "文件缺失")
				} else if hasSession {
					chrome.Stage(c, "live", "已打开")
				} else if e.Companion {
					ui.Text(c, "+db").FontSize(10).Font(theme.FontMono).TextColor(theme.Muted)
				}
			})

			ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
				ui.Text(c, kindLabel(e)).FontSize(11).TextColor(theme.Muted)
				ui.Text(c, "·").FontSize(11).TextColor(theme.Muted)
				ui.Text(c, product.ShortPath(filepath.Dir(e.Source))).FontSize(11).Font("SF Mono").TextColor(theme.Muted).Grow(1).SingleLine()
			})
		})
	})
}

func detail(c *ui.Context, d *desk.Desk) {
	e, ok := d.Catalog.Selected()
	if !ok {
		return
	}

	_, hasSession := d.Sessions.FindByPath(e.Path)
	accent := tileAccent(e, hasSession)
	st := d.Runtime()
	idaReady := d.IdaDir != "" || st.Found

	pane := chrome.Pane(c).Grow(1)
	pane.Children(func() {
		// Tactile specimen header
		ui.Row(c).
			Key("lib-head-"+e.ID).
			Padding(22, 28, 18, 28).
			BorderWidth(0, 0, 1, 0).
			BorderColor(theme.BorderSubtle).
			Gap(14).
			AlignItems(ui.Center).
			Transition(chrome.DetailTransition).
			Children(func() {
				chrome.CartridgeTile(c, extShort(e.Path), accent, true)

				ui.Column(c).Grow(1).Gap(5).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, e.Name).FontSize(18).FontWeight(500).TextColor(theme.Text).SingleLine()

						if e.Missing() {
							chrome.Stage(c, "bad", "文件缺失")
						} else if hasSession {
							chrome.Stage(c, "live", "会话活跃中")
						} else {
							chrome.Stage(c, "idle", "未打开")
						}
					})

					ui.Text(c, product.ShortPath(e.Path)).FontSize(12).Font("SF Mono").TextColor(theme.TextMuted).SingleLine()
				})

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if !e.Missing() {
						if hasSession {
							jump := chrome.PrimaryButton(c, "查看会话", nil)
							if jump.Clicked() {
								d.Tab = desk.TabSessions
							}
						} else {
							launch := chrome.PrimaryButton(c, "启动分析", chrome.IconPlay)
							if launch.Clicked() {
								_ = d.StartSession(e)
							}
						}
					}

					del := chrome.GhostButton(c, "移出", nil, false)
					if del.Clicked() {
						d.RemoveSelected()
					}
				})
			})

		// Main inspector body with smooth transition on entry switch
		ui.Scroll(c).
			Key("lib-body-"+e.ID).
			Grow(1).
			Padding(20, 28).
			Gap(18).
			Transition(chrome.DetailTransition).
			Children(func() {
				if e.Missing() {
					ui.Row(c).Padding(8, 12).Radius(theme.CardR).Background(theme.TroubleBg).Border(1, theme.Trouble).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, chrome.IconAlert).Size(14, 14).TextColor(theme.Trouble)
						ui.Text(c, "文件缺失: 磁盘上已找不到该路径").FontSize(12).TextColor(theme.Trouble)
					})
				}

				// 3-Segment Horizontal Instrument Readout Strip (Type, Size, Pipeline Stage)
				chrome.Card(c).Children(func() {
					ui.Row(c).Dividers(1, theme.BorderSubtle).AlignItems(ui.Stretch).Children(func() {
						chrome.ReadoutCell(c, "类型", kindLabel(e), "按内容加载，不限扩展名", accent)
						chrome.ReadoutCell(c, "大小", formatFileSize(e.Path), filepath.Base(e.Path), theme.Run)

						// Pipeline cell
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
								if hasSession {
									chrome.Stage(c, "live", "打开")
								} else {
									chrome.Stage(c, "idle", "打开")
								}
							})
							if e.Companion {
								ui.Text(c, "旁边已有数据库,打开时用它,不再灌入原文件。").FontSize(11).TextColor(theme.TextMuted).SingleLine()
							} else {
								ui.Text(c, "无界面模式 · 按需建函数").FontSize(11).TextColor(theme.Muted).SingleLine()
							}
						})
					})
				})

				// Dual-Slot Side-by-Side Path Routing Module (Source ──▸ Open Target)
				bridgeLabel := "直接打开"
				if e.Companion {
					bridgeLabel = "伴随库直通"
				}
				openBadge := "待命"
				openTone := "idle"
				if hasSession {
					openBadge = "已挂载"
					openTone = "live"
				} else if e.Companion {
					openBadge = "+db"
					openTone = "run"
				}
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					chrome.PathSlotCard(c, "来源", filepath.Base(e.Source), product.ShortPath(e.Source), "原始文件", "done")
					chrome.ConduitBridge(c, bridgeLabel, hasSession || e.Companion)
					chrome.PathSlotCard(c, "打开", filepath.Base(e.Path), product.ShortPath(e.Path), openBadge, openTone)
				})

				// Chronological call log for this target (or harmonious slate standby card)
				type callRow struct {
					fn, durStr, sum string
					dur             time.Duration
					ok              bool
				}
				var fileTraces []callRow
				for _, tr := range d.Traces.Entries() {
					if tr.Target == e.Name {
						fileTraces = append(fileTraces, callRow{
							fn:     tr.Function,
							durStr: tr.FormatDuration(),
							sum:    tr.Summary,
							dur:    tr.Duration,
							ok:     tr.Status != "error",
						})
					}
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
					} else {
						chrome.Plate(c).MinHeight(72).Center().Padding(16, 20).Children(func() {
							ui.Column(c).AlignItems(ui.Center).Gap(6).Children(func() {
								ui.Text(c, "尚无针对该文件的调用记录").FontSize(13).FontWeight(500).TextColor(theme.Text)
								ui.Text(c, "点击右上角「启动分析」拉起 idalib 会话后，外部 Agent 的工具调用将在此实时呈现。").FontSize(12).TextColor(theme.TextMuted)
							})
						})
					}
				})
			})

		statusLabel := "未打开"
		statusTone := "idle"
		if hasSession {
			statusLabel = "会话活跃中"
			statusTone = "live"
		}
		chrome.Status(c, d, statusLabel, statusTone)
	})
}

func tileAccent(e catalog.Entry, hasSession bool) ui.Color {
	if e.Missing() {
		return theme.Trouble
	}
	if hasSession {
		return theme.Live
	}
	return theme.Muted
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

func kindLabel(e catalog.Entry) string {
	if e.Kind == intake.KindDatabase {
		if e.Companion {
			return "伴随数据库"
		}
		return "数据库"
	}
	return "文件"
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
