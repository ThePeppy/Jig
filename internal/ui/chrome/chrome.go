// Package chrome draws the window frame: plates, keys, and the one signal mark.
package chrome

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"app/internal/desk"
	"app/internal/ui/theme"
)

var (
	// DetailTransition is a short fade when the selected record changes.
	DetailTransition = ui.ElementTransition{
		Duration: 100 * time.Millisecond,
		Ease:     ui.EaseOut,
		Enter:    &ui.Motion{Opacity: 0},
	}

	// ColorTransition fades fill and border. No scale, no slide.
	ColorTransition = ui.ElementTransition{
		Duration: 100 * time.Millisecond,
		Ease:     ui.EaseOut,
		Colors:   true,
	}
)

// Title is the toolbar. Tabs and the add key are Liquid Glass capsules.
// The window's own corner clips this band; it is not a second frame.
func Title(c *ui.Context, d *desk.Desk) {
	bar := c.TitleBar()
	h := bar.Height
	if h < theme.TitleBarH {
		h = theme.TitleBarH
	}

	leftPad := bar.Left
	if leftPad < 80 {
		leftPad = 80
	}

	ui.Row(c).
		Height(h).
		Padding(0, bar.Right+12, 0, leftPad).
		Background(theme.Window).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.Border).
		Gap(8).
		AlignItems(ui.Center).
		DragWindow().
		Children(func() {
			sessCount := len(d.Sessions.Sessions())
			ui.Row(c).
				Height(32).
				Padding(2).
				Gap(2).
				Radius(16).
				Material(glass.Glass{}).
				AlignItems(ui.Center).
				Children(func() {
					tab(c, d, desk.TabLibrary, "数据库", false)
					tab(c, d, desk.TabSessions, "会话", sessCount > 0)
					tab(c, d, desk.TabTrace, "记录", false)
					tab(c, d, desk.TabSettings, "设置", false)
				})

			plus := ui.Box(c).
				Size(32, 32).
				Radius(16).
				Material(glass.Glass{Interactive: true}).
				Cursor(ui.CursorPointer).
				Center().
				Focusable().
				Label("添加文件")
			if plus.Clicked() && d.RequestAdd != nil {
				d.RequestAdd()
			}
			plus.Children(func() {
				ui.Icon(c, IconPlus).Size(12, 12).TextColor(theme.Text)
			})

			ui.Spacer(c)

			st := d.Runtime()
			on := d.IdaDir != "" || st.Found
			label := "127.0.0.1:9401"
			if on {
				label = "idalib · 127.0.0.1:9401"
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				mark(c, on, theme.Live)
				ui.Text(c, label).FontSize(11).Font(theme.FontMono).TextColor(theme.TextMuted)
			})
		})
}

func tab(c *ui.Context, d *desk.Desk, id desk.Tab, label string, liveDot bool) {
	on := d.Tab == id
	btn := ui.Row(c).
		Key(id).
		Height(28).
		Padding(0, 12).
		Radius(14).
		Gap(6).
		AlignItems(ui.Center).
		Cursor(ui.CursorPointer).
		Focusable().
		Label(label)
	if on {
		btn.Background(theme.Pane)
	}
	if btn.Clicked() {
		d.Tab = id
		d.Notice = ""
	}
	textColor := theme.TextMuted
	if on {
		textColor = theme.Text
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(textColor)
		if liveDot {
			mark(c, true, theme.Live)
		}
	})
}

// Pane is the content surface. It has no corner of its own: the system
// window corner is the only outer corner.
func Pane(c *ui.Context) ui.Element {
	return ui.Column(c).
		Background(theme.Pane).
		Clip().
		AlignSelf(ui.Stretch)
}

// Sidebar is the edge-to-edge list. A hairline splits it from the detail.
func Sidebar(c *ui.Context) ui.Element {
	return Pane(c).
		Width(theme.ListW).
		Background(theme.Well).
		BorderWidth(0, 1, 0, 0).
		BorderColor(theme.Border)
}

// Card is a ruled group on a plate. The line groups it; there is no shadow.
func Card(c *ui.Context) ui.Element {
	return ui.Column(c).
		Background(theme.Pane).
		Radius(theme.CardR).
		Border(1, theme.Border)
}

// Plate is a recessed spec field.
func Plate(c *ui.Context) ui.Element {
	return ui.Column(c).
		Background(theme.Well).
		Radius(theme.CardR).
		Border(1, theme.Border)
}

// ReadoutCell is one cell of a spec strip. The value is type, not a colored lamp.
func ReadoutCell(c *ui.Context, label, value, sub string, _ ui.Color) {
	ui.Column(c).Grow(1).Basis(0).Padding(12, 16).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(theme.Muted)
		ui.Text(c, value).FontSize(15).FontWeight(500).Font(theme.FontMono).TextColor(theme.Text).SingleLine()
		if sub != "" {
			ui.Text(c, sub).FontSize(11).TextColor(theme.Faint).SingleLine()
		}
	})
}

// PathSlotCard is a path spec: role, file name, full path.
func PathSlotCard(c *ui.Context, roleLabel, fileName, fullPath, badge string, badgeTone string) {
	Card(c).Grow(1).Basis(0).Padding(12, 16).Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, roleLabel).FontSize(12).FontWeight(500).TextColor(theme.TextMuted)
			ui.Spacer(c)
			if badge != "" {
				Stage(c, badgeTone, badge)
			}
		})
		ui.Text(c, fileName).FontSize(14).FontWeight(500).TextColor(theme.Text).SingleLine()
		ui.Text(c, fullPath).FontSize(12).Font(theme.FontMono).TextColor(theme.TextMuted).SingleLine()
	})
}

// PrimaryButton is an ink key. Press does not scale.
func PrimaryButton(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	btn := ui.Row(c).
		Height(28).
		Padding(0, 12).
		Radius(theme.ControlR).
		Background(theme.PrimaryBg).
		Cursor(ui.CursorPointer).
		Transition(ColorTransition).
		Gap(6).
		Center().
		Focusable().
		Label(label)
	if btn.Hovered() {
		btn.Background(theme.PrimaryHover)
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(11, 11).TextColor(theme.PrimaryText)
		}
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(theme.PrimaryText)
	})
	return btn
}

// GhostButton is a hairline key. Fault is oxide text, not a tinted pill.
func GhostButton(c *ui.Context, label string, icon *ui.SVG, danger bool) ui.Element {
	btn := ui.Row(c).
		Height(28).
		Padding(0, 12).
		Radius(theme.ControlR).
		Background(theme.Pane).
		Border(1, theme.BorderStrong).
		Cursor(ui.CursorPointer).
		Transition(ColorTransition).
		Gap(6).
		Center().
		Focusable().
		Label(label)
	txtCol := theme.Text
	iconCol := theme.TextMuted
	if danger {
		txtCol = theme.Trouble
		iconCol = theme.Trouble
		btn.BorderColor(theme.Trouble)
	}
	if btn.Hovered() {
		btn.Background(theme.Hover)
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(12, 12).TextColor(iconCol)
		}
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(txtCol)
	})
	return btn
}

// CartridgeTile is a square format plate. A 2px bar appears only for on or fault.
func CartridgeTile(c *ui.Context, tag string, accent ui.Color, large bool) {
	w, h := float32(28), float32(28)
	fontSize := float32(9)
	if large {
		w, h = 40, 40
		fontSize = 11
	}
	if len(tag) > 4 {
		tag = tag[:4]
	}
	box := ui.Box(c).Size(w, h).Radius(theme.CardR).Background(theme.Well).Border(1, theme.BorderStrong).Center()
	if accent == theme.Trouble {
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 2}, accent, 0)
		})
	}
	box.Children(func() {
		ui.Text(c, tag).FontSize(fontSize).FontWeight(500).Font(theme.FontMono).TextColor(theme.Text)
	})
}

// StyleSidebarRow marks the selected row with a fill and a 2px signal bar.
func StyleSidebarRow(item ui.Element, on bool, _ ui.Color) {
	item.Transition(ColorTransition)
	if on {
		item.Background(theme.Active)
	} else if item.Hovered() {
		item.Background(theme.Hover)
	} else {
		item.Background(ui.Transparent)
	}
	if on {
		item.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 2, H: r.H}, theme.Live, 0)
		})
	}
}

// HeroEmblem is a square plate for an empty state. Not a reticle.
func HeroEmblem(c *ui.Context, icon *ui.SVG, accent ui.Color) {
	col := theme.Text
	if accent == theme.Live || accent == theme.Trouble {
		col = accent
	}
	ui.Box(c).Size(36, 36).Radius(theme.CardR).Background(theme.Well).Border(1, theme.BorderStrong).Center().Children(func() {
		ui.Icon(c, icon).Size(16, 16).TextColor(col)
	})
}

// LatencyBar is a static proportion. Length is the data; color is only fault.
func LatencyBar(c *ui.Context, d time.Duration, ok bool) {
	ratio := float32(d.Milliseconds()) / 250.0
	if ratio < 0.08 {
		ratio = 0.08
	}
	if ratio > 1 {
		ratio = 1
	}
	barCol := theme.Ink
	if !ok {
		barCol = theme.Trouble
	}
	ui.Box(c).Size(36, 3).Background(theme.Well).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * ratio, H: r.H}, barCol, 0)
	})
}

// ConduitBridge is a hairline with an arrow. Active turns the arrow orange.
func ConduitBridge(c *ui.Context, protocol string, active bool) {
	col := theme.Faint
	if active {
		col = theme.Live
	}
	ui.Column(c).Width(88).AlignItems(ui.Center).Justify(ui.Center).Gap(4).Children(func() {
		if protocol != "" {
			ui.Text(c, protocol).FontSize(10).Font(theme.FontMono).TextColor(theme.Muted).SingleLine()
		}
		ui.Box(c).FillWidth().Height(8).Draw(func(p *ui.Painter, r ui.Rect) {
			cy := r.Y + r.H/2
			p.Line(r.X, cy, r.X+r.W-6, cy, 1, theme.BorderStrong)
			ax := r.X + r.W - 1
			p.StrokePolyline([][2]float32{{ax - 4, cy - 3}, {ax, cy}, {ax - 4, cy + 3}}, 1, col)
		})
	})
}

// Stage is a square mark plus an ink label. The mark carries the state, not the words.
func Stage(c *ui.Context, tone, label string) {
	filled := false
	markCol := theme.Faint
	switch tone {
	case "live", "ok", "run", "active", "warn":
		filled = true
		markCol = theme.Live
	case "bad", "trouble":
		filled = true
		markCol = theme.Trouble
	case "done":
		filled = true
		markCol = theme.Ink
	}
	ui.Row(c).Height(20).Gap(6).AlignItems(ui.Center).Children(func() {
		mark(c, filled, markCol)
		ui.Text(c, label).FontSize(12).TextColor(theme.Text)
	})
}

// Join is the hairline between stages.
func Join(c *ui.Context) {
	ui.Box(c).Width(16).Height(1).Background(theme.BorderStrong)
}

// Kicker is a section label, with an optional count.
func Kicker(c *ui.Context, title string, count ...int) {
	ui.Row(c).Padding(0, 2).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(12).FontWeight(500).TextColor(theme.TextMuted)
		if len(count) > 0 && count[0] > 0 {
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d", count[0])).FontSize(11).Font(theme.FontMono).TextColor(theme.TextMuted)
		}
	})
}

// Status is the bottom strip: state mark, then the clock.
func Status(c *ui.Context, d *desk.Desk, label, tone string) {
	ui.Row(c).Height(theme.StatusH).Padding(0, 22).BorderWidth(1, 0, 0, 0).BorderColor(theme.Border).Gap(10).AlignItems(ui.Center).Children(func() {
		if d.Notice != "" {
			Stage(c, "bad", d.Notice)
		} else {
			Stage(c, tone, label)
		}
		ui.Spacer(c)
		ui.Text(c, d.Now().Format("15:04")).FontSize(11).Font(theme.FontMono).TextColor(theme.TextMuted)
	})
}

// Line is a spec row: fixed label, monospace value.
func Line(c *ui.Context, label, value string) {
	ui.Row(c).MinHeight(36).Padding(0, 16).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(theme.TextMuted).Width(84)
		ui.Text(c, value).FontSize(12).Font(theme.FontMono).TextColor(theme.Text).Grow(1).SingleLine()
	})
}

func mark(c *ui.Context, filled bool, col ui.Color) {
	box := ui.Box(c).Size(6, 6)
	if filled {
		box.Background(col)
		return
	}
	box.Border(1, theme.BorderStrong)
}

// CallMark colors a finished call. Orange is reserved for something that is on.
func CallMark(ok bool) ui.Color {
	if ok {
		return theme.Ink
	}
	return theme.Trouble
}
