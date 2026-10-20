// Package trace draws the record page: the call list with a filter, and
// the input and result of the selected call. It reads the in-memory
// store; it does not keep or write records itself.
package trace

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	internalTrace "jig/internal/trace"
	"jig/internal/ui/chrome"
	"jig/internal/ui/theme"
)

// columns are the record table: mark, tool, database, latency, time.
var columns = []ui.Track{ui.Fixed(20), ui.Fr(1.4), ui.Fr(1.2), ui.Fixed(70), ui.Fixed(80)}

// View renders the record list and the selected record.
func View(c *ui.Context, d *desk.Desk) {
	entries := d.Traces.Entries()
	if len(entries) == 0 {
		chrome.EmptyCard(c, "还没有调用", "打开文件之后，Agent 的调用会落在这里。",
			func() { ui.Icon(c, chrome.IconTrace).Size(40, 40).TextColor(theme.Muted) }, nil)
		return
	}
	shown := make([]internalTrace.Entry, 0, len(entries))
	for _, e := range entries {
		if keep(d.TraceFilter, e) {
			shown = append(shown, e)
		}
	}
	sel, ok := selected(d, shown)
	list(c, d, shown, sel.ID)
	if ok {
		detail(c, sel)
	}
}

func keep(f desk.TraceFilter, e internalTrace.Entry) bool {
	switch f {
	case desk.TraceOK:
		return !e.Failed()
	case desk.TraceFailed:
		return e.Failed()
	}
	return true
}

func selected(d *desk.Desk, shown []internalTrace.Entry) (internalTrace.Entry, bool) {
	for _, e := range shown {
		if e.ID == d.SelectedTrace {
			return e, true
		}
	}
	if len(shown) > 0 {
		return shown[0], true
	}
	return internalTrace.Entry{}, false
}

func list(c *ui.Context, d *desk.Desk, shown []internalTrace.Entry, sel string) {
	ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Padding(24, 20, 20, 24).Gap(16).
		BorderWidth(0, 1, 0, 0).BorderColor(theme.Rule).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				chrome.Heading(c, "记录")
				ui.Spacer(c)
				if chrome.Button(c, "清空", chrome.SizeMedium).Clicked() {
					d.Traces.Clear()
					d.SelectedTrace = ""
				}
			})
			filter(c, d)
			ui.Grid(c).ColumnTracks(columns...).Padding(0, 14).Children(func() {
				ui.Box(c)
				head(c, "工具", ui.Start)
				head(c, "数据库", ui.Start)
				head(c, "耗时", ui.Start)
				head(c, "时间", ui.End)
			})
			ui.Scroll(c).Grow(1).MinHeight(0).Gap(2).Children(func() {
				now := d.Now()
				for _, e := range shown {
					row(c, d, e, e.ID == sel, now)
				}
				if len(shown) == 0 {
					ui.Text(c, "没有符合条件的记录").FontSize(12).TextColor(theme.Muted).Padding(12, 14)
				}
			})
			ui.Text(c, fmt.Sprintf("只保留最近 %d 条，不写入磁盘", d.Traces.Max())).FontSize(12).TextColor(theme.Muted)
		})
}

func head(c *ui.Context, label string, align ui.Align) {
	ui.Text(c, label).FontSize(11.5).FontWeight(500).TextColor(theme.Muted).TextAlign(align)
}

func filter(c *ui.Context, d *desk.Desk) {
	ui.Row(c).Padding(3).Radius(17).Background(theme.SegmentVeil).Gap(4).AlignSelf(ui.Start).Children(func() {
		for _, f := range []struct {
			id    desk.TraceFilter
			label string
		}{{desk.TraceAll, "全部"}, {desk.TraceOK, "成功"}, {desk.TraceFailed, "失败"}} {
			on := d.TraceFilter == f.id
			seg := ui.Row(c).Key(fmt.Sprintf("trace-filter-%d", f.id)).Height(28).Padding(0, 14).Radius(14).Center().
				Cursor(ui.CursorPointer).Focusable().Label(f.label)
			if seg.Clicked() {
				d.TraceFilter = f.id
				on = true
			}
			col := theme.TextSoft
			if on {
				seg.Background(theme.SegmentOn)
				col = theme.Text
			} else if seg.Hovered() {
				seg.Background(theme.RowHover)
			}
			seg.Children(func() {
				ui.Text(c, f.label).FontSize(12).FontWeight(500).TextColor(col)
			})
		}
	})
}

func row(c *ui.Context, d *desk.Desk, e internalTrace.Entry, on bool, now time.Time) {
	item := ui.Grid(c).ColumnTracks(columns...).AlignItems(ui.Center).
		Key("trace-row-"+e.ID).Height(44).Padding(0, 14).Radius(theme.RowR).
		Cursor(ui.CursorPointer).Focusable().Label(e.Function)
	if item.Clicked() {
		d.SelectedTrace = e.ID
		on = true
	}
	switch {
	case on:
		item.Background(theme.RowActive)
	case item.Hovered():
		item.Background(theme.RowHover)
	}
	item.Children(func() {
		mark(c, e)
		chrome.Mono(c, e.Function, 12.5, theme.Text).MinWidth(0).Padding(0, 8, 0, 0)
		ui.Text(c, e.Target).FontSize(13).TextColor(theme.TextSoft).SingleLine().MinWidth(0).Padding(0, 8, 0, 0)
		ui.Text(c, e.FormatDuration()).FontSize(12).TextColor(theme.Muted).SingleLine()
		ui.Text(c, chrome.Ago(now, e.Timestamp)).FontSize(12).TextColor(theme.Muted).TextAlign(ui.End).SingleLine()
	})
}

func mark(c *ui.Context, e internalTrace.Entry) {
	switch {
	case e.Failed():
		chrome.ToneDot(c, chrome.ToneFault, 8)
	case e.Status == "running":
		chrome.ToneDot(c, chrome.ToneBusy, 8)
	default:
		chrome.ToneDot(c, chrome.ToneLive, 8)
	}
}

func detail(c *ui.Context, e internalTrace.Entry) {
	ui.Scroll(c).Key("trace-detail-"+e.ID).Width(theme.DetailW).AlignSelf(ui.Stretch).Padding(28, 24).Gap(20).
		Transition(chrome.PageTransition).
		Children(func() {
			ui.Column(c).Children(func() {
				if e.Failed() {
					chrome.Pill(c, chrome.ToneFault, "失败")
				} else {
					chrome.Pill(c, chrome.ToneLive, "成功")
				}
				ui.Text(c, e.Function).FontSize(20).FontWeight(600).Font(theme.FontMono).TextColor(theme.Text).
					SingleLine().Padding(12, 0, 4, 0)
				ui.Text(c, e.Target+" · "+e.FormatDuration()).FontSize(13).TextColor(theme.Muted).SingleLine()
			})
			args := e.Args
			if args == "" {
				args = "{}"
			}
			code(c, "入参", args)
			result := e.Result
			if result == "" {
				result = e.Summary
			}
			code(c, "结果", result)
		})
}

func code(c *ui.Context, title, text string) {
	chrome.Card(c, title, func() {
		ui.Text(c, text).FontSize(12).Font(theme.FontMono).LineHeight(1.6).TextColor(theme.CodeInk).Wrap().Selectable()
	})
}
