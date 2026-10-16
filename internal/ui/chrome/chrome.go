// Package chrome draws the window frame.
// The system blur is the ground. The rail is the frame: a thin tint, so
// the frost shows through. The desk is a less transparent plate of the
// same material. It does not paint a shader, and it does not draw a
// second window corner or a theme switch.
package chrome

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/ui/theme"
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

// Wash is one depth of the window glass. Over vibrancy it is a
// specialized tint, so the desktop frost shows through at that
// region's opacity. Elsewhere it is the solid token of the same role.
func Wash(c *ui.Context, role string) ui.Color {
	if c.Vibrancy() {
		switch role {
		case "rail":
			return theme.RailGlass
		case "desk":
			return theme.DeskGlass
		case "railHover":
			return theme.RailHoverGlass
		case "railActive":
			return theme.RailActiveGlass
		case "hover":
			return theme.DeskHoverGlass
		case "active":
			return theme.DeskActiveGlass
		case "card":
			return theme.CardGlass
		case "plate":
			return theme.PlateGlass
		case "band":
			return theme.BandGlass
		case "control":
			return theme.ControlGlass
		case "primary":
			return theme.PrimaryGlass
		case "primaryHover":
			return theme.PrimaryHoverGlass
		case "primaryPressed":
			return theme.PrimaryPressedGlass
		case "fault":
			return theme.FaultGlass
		case "rule":
			return theme.RuleGlass
		case "hairline":
			return theme.HairlineGlass
		case "field":
			return theme.DeskGlass
		}
	}
	switch role {
	case "rail":
		return theme.Sidebar
	case "desk", "field", "band":
		return theme.Canvas
	case "railHover", "hover":
		return theme.Hover
	case "railActive", "active":
		return theme.Active
	case "card", "control":
		return theme.Card
	case "plate":
		return theme.Well
	case "primary":
		return theme.PrimaryBg
	case "primaryHover":
		return theme.AccentHover
	case "primaryPressed":
		return theme.AccentPressed
	case "fault":
		return theme.TroubleBg
	case "rule":
		return theme.BorderSubtle
	case "hairline":
		return theme.Border
	}
	return theme.Canvas
}

func primaryInk(c *ui.Context) ui.Color {
	return theme.PrimaryText
}

// Shell is the window. The blur is the ground. The rail and the desk
// each paint their own tint, so the two regions do not collapse into
// one wash.
func Shell(c *ui.Context, d *desk.Desk, content func()) {
	bar := c.TitleBar()
	h := bar.Height
	if h < theme.TitleBarH {
		h = theme.TitleBarH
	}
	leftPad := bar.Left
	if leftPad < 80 {
		leftPad = 80
	}
	rightPad := bar.Right
	if rightPad < 12 {
		rightPad = 12
	}
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		rail(c, d, h, leftPad)
		ui.Box(c).Width(1).Background(Wash(c, "hairline"))
		desk := ui.Column(c).Grow(1).AlignSelf(ui.Stretch).Background(Wash(c, "desk"))
		desk.Children(func() {
			contentBar(c, d, h, rightPad)
			content()
		})
	})
}

func rail(c *ui.Context, d *desk.Desk, h, leftPad float32) {
	col := ui.Column(c).Width(theme.NavW).AlignSelf(ui.Stretch).Background(Wash(c, "rail"))
	col.Children(func() {
		ui.Row(c).
			Height(h).
			Padding(0, 14, 0, leftPad).
			AlignItems(ui.Center).
			Gap(8).
			DragWindow().
			Children(func() {
				ui.Icon(c, IconBrand).Size(14, 14).TextColor(theme.Accent)
				ui.Text(c, "Jig").FontSize(13).FontWeight(600).TextColor(theme.Text)
			})

		ui.Column(c).Padding(6, 8, 0, 8).Gap(1).Children(func() {
			navItem(c, d, desk.TabLibrary, "数据库", IconLibrary, len(d.Sessions.Sessions()) > 0)
			navItem(c, d, desk.TabTrace, "记录", IconTrace, false)
			navItem(c, d, desk.TabSettings, "设置", IconSettings, false)
		})

		ui.Spacer(c)
		addRow(c, d)
	})
}

func addRow(c *ui.Context, d *desk.Desk) {
	ui.Row(c).
		Height(theme.StatusH).
		Padding(0, 8).
		BorderWidth(1, 0, 0, 0).
		BorderColor(Wash(c, "hairline")).
		AlignItems(ui.Center).
		Children(func() {
			add := ui.Row(c).
				Grow(1).
				Height(26).
				Padding(0, 8).
				Radius(theme.ControlR).
				Gap(8).
				AlignItems(ui.Center).
				Cursor(ui.CursorPointer).
				Focusable().
				Label("添加文件")
			if add.Hovered() {
				add.Background(Wash(c, "railHover"))
			}
			if add.Clicked() && d.RequestAdd != nil {
				d.RequestAdd()
			}
			add.Children(func() {
				ui.Icon(c, IconPlus).Size(13, 13).TextColor(theme.TextMuted)
				ui.Text(c, "添加文件").FontSize(13).TextColor(theme.Text)
			})
		})
}

func navItem(c *ui.Context, d *desk.Desk, id desk.Tab, label string, icon *ui.SVG, liveDot bool) {
	on := d.Tab == id
	btn := ui.Row(c).
		Key(id).
		Height(30).
		Padding(0, 8).
		Radius(theme.ControlR).
		Gap(8).
		AlignItems(ui.Center).
		Cursor(ui.CursorPointer).
		Focusable().
		Label(label)
	switch {
	case on:
		btn.Background(Wash(c, "railActive"))
	case btn.Hovered():
		btn.Background(Wash(c, "railHover"))
	}
	if btn.Clicked() {
		d.Tab = id
		d.Notice = ""
	}
	iconCol := theme.Faint
	textCol := theme.TextMuted
	weight := 400
	if on {
		iconCol = theme.Accent
		textCol = theme.Text
		weight = 500
	}
	btn.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(iconCol)
		ui.Text(c, label).FontSize(13).FontWeight(weight).TextColor(textCol).Grow(1)
		if liveDot {
			mark(c, true, theme.Live)
		}
	})
}

func contentBar(c *ui.Context, d *desk.Desk, h, rightPad float32) {
	ui.Row(c).
		Height(h).
		Padding(0, rightPad+16, 0, 24).
		AlignItems(ui.Center).
		DragWindow().
		BorderWidth(0, 0, 1, 0).
		BorderColor(Wash(c, "hairline")).
		Children(func() {
			ui.Text(c, sectionName(d.Tab)).FontSize(13).FontWeight(500).TextColor(theme.Text)
			ui.Spacer(c)
			endpoint(c, d)
		})
}

func sectionName(tab desk.Tab) string {
	switch tab {
	case desk.TabTrace:
		return "记录"
	case desk.TabSettings:
		return "设置"
	default:
		return "数据库"
	}
}

func endpoint(c *ui.Context, d *desk.Desk) {
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
}

// Pane is the desk. Over a material it draws no fill. The system corner
// is the only outer corner.
func Pane(c *ui.Context) ui.Element {
	col := ui.Column(c).Clip().AlignSelf(ui.Stretch)
	if !c.Vibrancy() {
		col.Background(theme.Canvas)
	}
	return col
}

// Sidebar is the edge-to-edge list. A hairline splits it from the detail.
func Sidebar(c *ui.Context) ui.Element {
	return Pane(c).
		Width(theme.ListW).
		BorderWidth(0, 1, 0, 0).
		BorderColor(Wash(c, "hairline"))
}

// Card is a lifted group on the material. The line groups it; there is no shadow.
func Card(c *ui.Context) ui.Element {
	return ui.Column(c).
		Background(Wash(c, "card")).
		Radius(theme.CardR).
		Border(1, Wash(c, "hairline"))
}

// Plate is a recessed spec field, a deeper veil of the same material.
func Plate(c *ui.Context) ui.Element {
	return ui.Column(c).
		Background(Wash(c, "plate")).
		Radius(theme.CardR).
		Border(1, Wash(c, "hairline"))
}

// ReadoutCell is one cell of a spec strip. The value is type, not a colored lamp.
func ReadoutCell(c *ui.Context, label, value, sub string, _ ui.Color) {
	ui.Column(c).Grow(1).Basis(0).Padding(10, 14).Gap(3).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(theme.Muted)
		ui.Text(c, value).FontSize(13).FontWeight(500).Font(theme.FontMono).TextColor(theme.Text).SingleLine()
		if sub != "" {
			ui.Text(c, sub).FontSize(11).TextColor(theme.Faint).SingleLine()
		}
	})
}

// PathSlotCard is a path spec: role, file name, full path.
// It sizes to its text. In a row, Grow(1).Basis(0) the result so cards share width.
func PathSlotCard(c *ui.Context, roleLabel, fileName, fullPath, badge string, badgeTone string) ui.Element {
	card := Card(c).Padding(12, 14).Gap(4)
	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, roleLabel).FontSize(12).FontWeight(500).TextColor(theme.TextMuted)
			ui.Spacer(c)
			if badge != "" {
				Stage(c, badgeTone, badge)
			}
		})
		ui.Text(c, fileName).FontSize(13).FontWeight(500).TextColor(theme.Text).SingleLine()
		ui.Text(c, fullPath).FontSize(12).Font(theme.FontMono).TextColor(theme.TextMuted).SingleLine()
	})
	return card
}

// PrimaryButton is the action key. Over a material it is a light veil
// with light text, not a solid chip. Without a material it is a dark key.
func PrimaryButton(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	ink := primaryInk(c)
	btn := ui.Row(c).
		Height(28).
		Padding(0, 12).
		Radius(theme.ControlR).
		Background(Wash(c, "primary")).
		Cursor(ui.CursorPointer).
		Gap(6).
		Center().
		Focusable().
		Label(label)
	if c.Vibrancy() {
		btn.Border(1, theme.Accent.Alpha(0.35))
	}
	if btn.Pressed() {
		btn.Background(Wash(c, "primaryPressed"))
	} else if btn.Hovered() {
		btn.Background(Wash(c, "primaryHover"))
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(11, 11).TextColor(ink)
		}
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(ink)
	})
	return btn
}

// GhostButton is a hairline key on the material. Fault is oxide text, not a tinted pill.
func GhostButton(c *ui.Context, label string, icon *ui.SVG, danger bool) ui.Element {
	btn := ui.Row(c).
		Height(28).
		Padding(0, 10).
		Radius(theme.ControlR).
		Border(1, Wash(c, "hairline")).
		Cursor(ui.CursorPointer).
		Gap(6).
		Center().
		Focusable().
		Label(label)
	if !c.Vibrancy() {
		btn.Background(theme.Card)
	}
	txtCol := theme.Text
	iconCol := theme.TextMuted
	if danger {
		txtCol = theme.Trouble
		iconCol = theme.Trouble
	}
	if btn.Hovered() {
		btn.Background(Wash(c, "hover"))
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(12, 12).TextColor(iconCol)
		}
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(txtCol)
	})
	return btn
}

// BusyButton is the action key while it works. It does not take another click.
func BusyButton(c *ui.Context, label string) ui.Element {
	ink := primaryInk(c)
	btn := ui.Row(c).
		Height(28).
		Padding(0, 12).
		Radius(theme.ControlR).
		Background(Wash(c, "primary")).
		Gap(6).
		Center().
		Label(label)
	if c.Vibrancy() {
		btn.Border(1, theme.Accent.Alpha(0.35))
	}
	btn.Children(func() {
		spin(c, label, ink)
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(ink)
	})
	return btn
}

// BusyGhost is a hairline key that is working. Fault stays oxide, not a tint.
func BusyGhost(c *ui.Context, label string, danger bool) ui.Element {
	txt := theme.Text
	if danger {
		txt = theme.Trouble
	}
	btn := ui.Row(c).
		Height(28).
		Padding(0, 10).
		Radius(theme.ControlR).
		Border(1, Wash(c, "hairline")).
		Gap(6).
		Center().
		Label(label)
	if !c.Vibrancy() {
		btn.Background(theme.Card)
	}
	btn.Children(func() {
		spin(c, label, txt)
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(txt)
	})
	return btn
}

// LiveStage is a breathing mark plus a label, for work that is still going.
func LiveStage(c *ui.Context, label string) {
	ui.Row(c).Height(20).Gap(6).AlignItems(ui.Center).Children(func() {
		pulse(c)
		ui.Text(c, label).FontSize(12).TextColor(theme.Text)
	})
}

// Work is the process of a start or a close. Earlier steps are done.
// The mark breathes and the bar moves so the wait is not a freeze.
func Work(c *ui.Context, title, meta, detail string, steps []string, active int) {
	Card(c).Padding(12, 14).Gap(8).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			pulse(c)
			ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(theme.Text)
			ui.Spacer(c)
			if meta != "" {
				ui.Text(c, meta).FontSize(12).Font(theme.FontMono).TextColor(theme.TextMuted)
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			for i, step := range steps {
				if i > 0 {
					Join(c)
				}
				tone := "idle"
				if i < active {
					tone = "done"
				} else if i == active {
					tone = "live"
				}
				Stage(c, tone, step)
			}
		})
		sweep(c)
		ui.Text(c, detail).FontSize(12).TextColor(theme.TextMuted)
	})
}

func spin(c *ui.Context, key string, col ui.Color) {
	icon := ui.Icon(c, IconRefresh).Size(11, 11).TextColor(col)
	icon.Rotate(icon.Loop(key, 800*time.Millisecond, ui.Linear) * 360)
}

func pulse(c *ui.Context) {
	box := ui.Box(c).Size(6, 6).Radius(3).Background(theme.Live)
	box.Opacity(0.4 + 0.6*box.Loop("pulse", 1200*time.Millisecond, ui.EaseInOut))
}

func sweep(c *ui.Context) {
	ui.Box(c).Height(2).FillWidth().Background(Wash(c, "plate")).Radius(1).Draw(func(p *ui.Painter, r ui.Rect) {
		p.AnimationFrame()
		period := 1200 * time.Millisecond
		t := float32(p.Now().UnixNano()%int64(period)) / float32(period)
		if t > 0.5 {
			t = 1 - t
		}
		t *= 2
		w := r.W * 0.28
		if w < 18 {
			w = 18
		}
		if w > r.W {
			w = r.W
		}
		x := r.X + (r.W-w)*t
		p.Fill(ui.Rect{X: x, Y: r.Y, W: w, H: r.H}, theme.Live, 1)
	})
}

// CartridgeTile is a small format plate. A 2px bar appears only for a fault.
func CartridgeTile(c *ui.Context, tag string, accent ui.Color, large bool) {
	w, h := float32(24), float32(24)
	fontSize := float32(9)
	radius := float32(4)
	if large {
		w, h = 32, 32
		fontSize = 10
		radius = 6
	}
	if len(tag) > 4 {
		tag = tag[:4]
	}
	box := ui.Box(c).Size(w, h).Radius(radius).Background(Wash(c, "plate")).Border(1, Wash(c, "hairline")).Center()
	if accent == theme.Trouble {
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 2}, accent, 0)
		})
	}
	box.Children(func() {
		ui.Text(c, tag).FontSize(fontSize).FontWeight(500).Font(theme.FontMono).TextColor(theme.Text)
	})
}

// StyleSidebarRow marks the selected row with a deeper veil, hover with a lighter one.
func StyleSidebarRow(c *ui.Context, item ui.Element, on bool, _ ui.Color) {
	switch {
	case on:
		item.Background(Wash(c, "active"))
	case item.Hovered():
		item.Background(Wash(c, "hover"))
	default:
		item.Background(ui.Transparent)
	}
}

// HeroEmblem is a quiet mark for an empty state. Not a boxed icon.
func HeroEmblem(c *ui.Context, icon *ui.SVG, accent ui.Color) {
	col := theme.Faint
	if accent == theme.Live || accent == theme.Trouble {
		col = accent
	}
	ui.Icon(c, icon).Size(22, 22).TextColor(col)
}

// LatencyBar is a static proportion. Length is the data; color is accent or fault.
func LatencyBar(c *ui.Context, d time.Duration, ok bool) {
	ratio := float32(d.Milliseconds()) / 250.0
	if ratio < 0.08 {
		ratio = 0.08
	}
	if ratio > 1 {
		ratio = 1
	}
	barCol := theme.Live
	if !ok {
		barCol = theme.Trouble
	}
	ui.Box(c).Size(36, 3).Radius(1).Background(Wash(c, "plate")).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * ratio, H: r.H}, barCol, 1)
	})
}

// ConduitBridge is a hairline with an arrow. Active turns the arrow accent.
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
			p.Line(r.X, cy, r.X+r.W-6, cy, 1, Wash(c, "hairline"))
			ax := r.X + r.W - 1
			p.StrokePolyline([][2]float32{{ax - 4, cy - 3}, {ax, cy}, {ax - 4, cy + 3}}, 1, col)
		})
	})
}

// Stage is a circle mark plus an ink label. The mark carries the state, not the words.
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
	ui.Box(c).Width(16).Height(1).Background(Wash(c, "hairline"))
}

// Kicker is a section label, with an optional count.
func Kicker(c *ui.Context, title string, count ...int) {
	ui.Row(c).Padding(0, 2).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(11).FontWeight(500).TextColor(theme.TextMuted)
		if len(count) > 0 && count[0] > 0 {
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d", count[0])).FontSize(11).Font(theme.FontMono).TextColor(theme.TextMuted)
		}
	})
}

// Status is the bottom band of the desk: state mark, then the clock.
func Status(c *ui.Context, d *desk.Desk, label, tone string) {
	row := ui.Row(c).Height(theme.StatusH).Padding(0, 24).Background(Wash(c, "band")).BorderWidth(1, 0, 0, 0).BorderColor(Wash(c, "hairline")).Gap(8).AlignItems(ui.Center)
	row.Children(func() {
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
	ui.Row(c).MinHeight(32).Padding(0, 14).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(theme.TextMuted).Width(84)
		ui.Text(c, value).FontSize(12).Font(theme.FontMono).TextColor(theme.Text).Grow(1).SingleLine()
	})
}

func mark(c *ui.Context, filled bool, col ui.Color) {
	box := ui.Box(c).Size(6, 6).Radius(3)
	if filled {
		box.Background(col)
		return
	}
	box.Border(1, Wash(c, "hairline"))
}

// CallMark colors a finished call. Accent is on. Fault is oxide.
func CallMark(ok bool) ui.Color {
	if ok {
		return theme.Live
	}
	return theme.Trouble
}
