package trace

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	internalTrace "jig/internal/trace"
	"jig/internal/ui/chrome"
	"jig/internal/ui/theme"
)

// View renders the agent call audit view.
func View(c *ui.Context, d *desk.Desk) {
	entries := d.Traces.Entries()
	if len(entries) == 0 {
		empty(c, d)
		return
	}

	latest := entries[0]
	errCount := 0
	for _, e := range entries {
		if e.Status == "error" {
			errCount++
		}
	}

	pane := chrome.Pane(c).Grow(1)
	pane.Children(func() {
		// Unboxed editorial header
		ui.Row(c).Padding(22, 28, 18, 28).BorderWidth(0, 0, 1, 0).BorderColor(theme.BorderSubtle).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "调用审计记录").FontSize(18).FontWeight(500).TextColor(theme.Text)
			chrome.Stage(c, "done", fmt.Sprintf("%d 条记录", len(entries)))
			if errCount > 0 {
				chrome.Stage(c, "bad", fmt.Sprintf("%d 异常", errCount))
			}
			ui.Spacer(c)

			clearBtn := chrome.GhostButton(c, "清空", nil, false).Label("清空记录")
			if clearBtn.Clicked() {
				d.Traces.Clear()
			}
		})

		ui.Scroll(c).Grow(1).Padding(20, 28).Gap(18).Children(func() {
			// 3-Segment Horizontal Telemetry Readout Strip
			chrome.Card(c).Children(func() {
				ui.Row(c).Dividers(1, theme.BorderSubtle).AlignItems(ui.Stretch).Children(func() {
					chrome.ReadoutCell(c, "最新调用", latest.Function, latest.Target, theme.Run)
					chrome.ReadoutCell(c, "最近耗时", latest.FormatDuration(), latest.Timestamp.Format("15:04:05"), theme.Live)
					statusText := "全部正常返回"
					statusCol := theme.Live
					if errCount > 0 {
						statusText = fmt.Sprintf("%d 次调用异常", errCount)
						statusCol = theme.Trouble
					}
					chrome.ReadoutCell(c, "通道状态", statusText, "MCP JSON-RPC 2.0", statusCol)
				})
			})

			// Latest Call Pipeline & Summary Dock
			chrome.Plate(c).Padding(16, 20).Gap(10).Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					chrome.Stage(c, "done", "发出")
					chrome.Join(c)
					chrome.Stage(c, "done", "已送到")
					chrome.Join(c)
					if latest.Status == "error" {
						chrome.Stage(c, "bad", "异常")
					} else {
						chrome.Stage(c, "ok", "已返回")
					}
					ui.Spacer(c)
					ui.Text(c, latest.Timestamp.Format("15:04:05")+" · "+latest.Target).FontSize(11).Font("SF Mono").TextColor(theme.Muted)
				})
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, latest.Function).FontSize(13).FontWeight(500).Font(theme.FontMono).TextColor(theme.Text)
					ui.Text(c, "—").FontSize(12).TextColor(theme.Faint)
					ui.Text(c, latest.Summary).FontSize(12).TextColor(theme.TextMuted).Grow(1).SingleLine()
				})
			})

			// Chronological Call Sheet
			ui.Column(c).Gap(8).Children(func() {
				chrome.Kicker(c, "调用时间线", len(entries))
				chrome.Card(c).Dividers(1, theme.BorderSubtle).Children(func() {
					for _, e := range entries {
						row(c, e)
					}
				})
			})
		})

		chrome.Status(c, d, fmt.Sprintf("%d 条调用记录", len(entries)), "done")
	})
}

func empty(c *ui.Context, d *desk.Desk) {
	pane := chrome.Pane(c).Grow(1)
	pane.Children(func() {
		ui.Column(c).Grow(1).Center().Padding(32, 32).Children(func() {
			ui.Column(c).Width(480).AlignItems(ui.Center).Gap(16).Children(func() {
				chrome.HeroEmblem(c, chrome.IconTrace, theme.Text)

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "还没有调用").FontSize(18).FontWeight(500).TextColor(theme.Text)
					chrome.Stage(c, "idle", "空")
				})

				ui.Text(c, "打开文件之后,Agent 的调用会落在这里。").
					FontSize(13).
					TextColor(theme.TextMuted).
					TextAlign(ui.Center)
			})
		})

		chrome.Status(c, d, "空", "idle")
	})
}

func row(c *ui.Context, e internalTrace.Entry) {
	item := ui.Row(c).MinHeight(40).Padding(0, 16).Gap(14).AlignItems(ui.Center)
	if item.Hovered() {
		item.Background(theme.CardHover)
	}

	ok := e.Status != "error"
	item.Children(func() {
		dot := theme.Ink
		if !ok {
			dot = theme.Trouble
		}
		ui.Box(c).Size(6, 6).Background(dot)

		ui.Text(c, e.Timestamp.Format("15:04:05")).FontSize(11).Font("SF Mono").TextColor(theme.Muted).Width(64)
		ui.Text(c, e.Function).FontSize(12).FontWeight(500).Font(theme.FontMono).TextColor(theme.Text).Width(164).SingleLine()
		ui.Text(c, e.Target).FontSize(12).TextColor(theme.Text).Width(156).SingleLine()
		ui.Text(c, e.Summary).FontSize(12).TextColor(theme.TextMuted).Grow(1).SingleLine()

		ui.Row(c).Width(96).Gap(8).Justify(ui.End).AlignItems(ui.Center).Children(func() {
			chrome.LatencyBar(c, e.Duration, ok)
			ui.Text(c, e.FormatDuration()).FontSize(11).Font("SF Mono").TextColor(theme.TextMuted).Width(46).TextAlign(ui.End)
		})
	})
}
