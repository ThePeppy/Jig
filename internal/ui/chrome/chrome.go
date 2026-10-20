// Package chrome draws the window frame and the parts every page shares:
// the rail, pills, keys, cards, fact grids and the empty card.
// The system blur is the ground. The rail is the frame: a faint tint, so
// the frost shows through. The desk is a dense plate of the same
// material. It does not paint a shader, a second window corner, or a
// theme switch, and it does not hold page state.
package chrome

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/product"
	"jig/internal/ui/theme"
)

// MCPURL is the endpoint an agent connects to.
const MCPURL = "http://127.0.0.1:9401/mcp"

// MCPAddr is the MCP listener shown beside the service state.
const MCPAddr = "127.0.0.1:9401"

// HubAddr is the worker hub.
const HubAddr = "127.0.0.1:9399"

// PageTransition is a short fade when the page or the selection changes.
var PageTransition = ui.ElementTransition{
	Duration: 100 * time.Millisecond,
	Ease:     ui.EaseOut,
	Enter:    &ui.Motion{Opacity: 0},
}

// Shell is the window: the rail, then the desk with a draggable title
// strip over content. content lays out the desk's columns in a row.
func Shell(c *ui.Context, d *desk.Desk, content func()) {
	h := titleBarH(c)
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		rail(c, d, h)
		desk := ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).
			BorderWidth(0, 0, 0, 1).BorderColor(theme.Rule)
		if c.Vibrancy() {
			desk.Background(theme.DeskGlass)
		} else {
			desk.Background(theme.DeskSolid)
		}
		desk.Children(func() {
			ui.Box(c).Height(h).FillWidth().DragWindow()
			ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(content)
		})
	})
}

func titleBarH(c *ui.Context) float32 {
	if h := c.TitleBar().Height; h > theme.TitleBarH {
		return h
	}
	return theme.TitleBarH
}

func rail(c *ui.Context, d *desk.Desk, h float32) {
	col := ui.Column(c).Width(theme.NavW).AlignSelf(ui.Stretch).Padding(0, 12, 16, 12).Gap(4)
	if c.Vibrancy() {
		col.Background(theme.RailGlass)
	} else {
		col.Background(theme.RailSolid)
	}
	col.Children(func() {
		ui.Box(c).Height(h + 12).FillWidth().DragWindow()
		ui.Row(c).Padding(0, 10, 22, 10).Gap(10).AlignItems(ui.Center).Children(func() {
			Mark(c, 26)
			ui.Text(c, product.Name).FontSize(17).FontWeight(600).LetterSpacing(-0.17).TextColor(theme.Text)
		})
		navItem(c, d, desk.TabLibrary, "数据库", IconLibrary)
		navItem(c, d, desk.TabTrace, "记录", IconTrace)
		navItem(c, d, desk.TabSettings, "设置", IconSettings)
		ui.Spacer(c)
		serviceCard(c, d)
	})
}

func navItem(c *ui.Context, d *desk.Desk, id desk.Tab, label string, icon *ui.SVG) {
	on := d.Tab == id
	btn := ui.Row(c).
		Key(id).
		Height(36).
		Padding(0, 10).
		Radius(theme.NavR).
		Gap(10).
		AlignItems(ui.Center).
		Cursor(ui.CursorPointer).
		Focusable().
		Label(label)
	switch {
	case on:
		btn.Background(theme.NavActive)
	case btn.Hovered():
		btn.Background(theme.NavHover)
	}
	if btn.Clicked() {
		d.Tab = id
		d.Notice = ""
	}
	col, weight := theme.TextSoft, 400
	if on {
		col, weight = theme.Text, 500
	}
	btn.Children(func() {
		ui.Icon(c, icon).Size(16, 16).TextColor(col)
		ui.Text(c, label).FontSize(13).FontWeight(weight).TextColor(col)
	})
}

// serviceCard is the rail's footer: whether both local listeners are up.
func serviceCard(c *ui.Context, d *desk.Desk) {
	ui.Column(c).Padding(12).Radius(12).Background(theme.RailCardVeil).Border(1, theme.Hairline).Gap(6).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if d.Listening {
				Dot(c, theme.Live, 8).Shadow(0, 0, 0, 3, theme.LiveHalo)
				ui.Text(c, "服务运行中").FontSize(13).FontWeight(500).TextColor(theme.Text)
			} else {
				Dot(c, theme.Idle, 8)
				ui.Text(c, "服务未启动").FontSize(13).FontWeight(500).TextColor(theme.Text)
			}
		})
		ui.Text(c, MCPAddr).FontSize(11).Font(theme.FontMono).TextColor(theme.Muted)
	})
}

// Mark draws the brand emblem at size DIPs.
func Mark(c *ui.Context, size float32) ui.Element {
	return ui.Icon(c, IconBrand).Size(size, size).TextColor(theme.Accent)
}

// Dot is a status mark. The color carries the state; text stays light.
func Dot(c *ui.Context, col ui.Color, size float32) ui.Element {
	return ui.Box(c).Size(size, size).Radius(size / 2).Background(col)
}

// BusyDot breathes while work is still going.
func BusyDot(c *ui.Context, size float32) ui.Element {
	box := Dot(c, theme.Accent, size)
	return box.Opacity(0.35 + 0.65*box.Loop("busy", 1200*time.Millisecond, ui.EaseInOut))
}

// Tone is the state a pill or a row mark shows.
type Tone int

const (
	ToneIdle Tone = iota
	ToneLive
	ToneBusy
	ToneFault
)

// ToneDot draws the row mark for tone.
func ToneDot(c *ui.Context, tone Tone, size float32) {
	switch tone {
	case ToneLive:
		Dot(c, theme.Live, size)
	case ToneBusy:
		BusyDot(c, size)
	case ToneFault:
		Dot(c, theme.Fault, size)
	default:
		Dot(c, theme.Idle, size)
	}
}

// Pill is a state chip: tinted ground, dot, short label.
func Pill(c *ui.Context, tone Tone, label string) {
	bg, ink := theme.ControlVeil, theme.TextSoft
	switch tone {
	case ToneLive:
		bg, ink = theme.LiveBg, theme.LiveInk
	case ToneBusy:
		bg, ink = theme.BusyBg, theme.BusyInk
	case ToneFault:
		bg, ink = theme.FaultBg, theme.FaultInk
	}
	ui.Row(c).Height(24).Padding(0, 10).Radius(12).Background(bg).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Start).Children(func() {
		ToneDot(c, tone, 6)
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(ink)
	})
}

// Size is the height class of a key.
type Size int

const (
	// SizeSmall sits inside a card.
	SizeSmall Size = iota
	// SizeMedium heads a column or a card.
	SizeMedium
	// SizeLarge is the action row under a title.
	SizeLarge
	// SizeInline sits inside a well, beside the text it acts on.
	SizeInline
)

func keyBox(c *ui.Context, key, label string, size Size) ui.Element {
	h, pad := float32(32), float32(14)
	r := h / 2
	switch size {
	case SizeSmall:
		h, r = 30, 15
	case SizeLarge:
		h, pad, r = 36, 18, 18
	case SizeInline:
		h, pad, r = 28, 12, 8
	}
	return ui.Row(c).
		Key("key-"+key).
		Height(h).
		Padding(0, pad).
		Radius(r).
		Center().
		Cursor(ui.CursorPointer).
		Focusable().
		Label(label)
}

// PrimaryButton is the one light key of a view.
func PrimaryButton(c *ui.Context, label string) ui.Element {
	btn := keyBox(c, label, label, SizeMedium).Background(theme.Primary)
	if btn.Pressed() {
		btn.Opacity(0.85)
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(13).FontWeight(600).TextColor(theme.Ink)
	})
	return btn
}

// Button is a veiled key.
func Button(c *ui.Context, label string, size Size) ui.Element {
	return veiled(c, label, label, label, size)
}

func veiled(c *ui.Context, key, label, shown string, size Size) ui.Element {
	btn := keyBox(c, key, label, size).Background(theme.ControlVeil)
	if btn.Hovered() {
		btn.Background(theme.ControlHover)
	}
	font, weight := float32(13), 500
	switch size {
	case SizeSmall:
		font = 12
	case SizeInline:
		font, weight = 12, 400
	}
	btn.Children(func() {
		ui.Text(c, shown).FontSize(font).FontWeight(weight).TextColor(theme.Text)
	})
	return btn
}

// BusyButton is a key while its work runs. It takes no click.
func BusyButton(c *ui.Context, label string, size Size) {
	h, pad := float32(32), float32(14)
	if size == SizeLarge {
		h, pad = 36, 18
	}
	ui.Row(c).Height(h).Padding(0, pad).Radius(h / 2).Background(theme.ControlVeil).Gap(8).Center().Label(label).Children(func() {
		BusyDot(c, 6)
		ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(theme.TextSoft)
	})
}

// CopyButton writes text to the clipboard and says so for a moment.
func CopyButton(c *ui.Context, d *desk.Desk, key, label, text string, size Size) {
	shown := label
	if left := d.CopiedFor(key); left > 0 {
		shown = "已复制"
		c.After(left)
	}
	if veiled(c, key, label, shown, size).Clicked() {
		c.Services().WriteClipboard(text)
		d.MarkCopied(key)
	}
}

// Card is a group on the desk: a veil, a hairline, a quiet title.
func Card(c *ui.Context, title string, body func()) ui.Element {
	card := ui.Column(c).Padding(18, 20).Radius(theme.CardR).Background(theme.CardVeil).Border(1, theme.Hairline).MinWidth(0)
	card.Children(func() {
		if title != "" {
			cardTitle(c, title)
		}
		body()
	})
	return card
}

// cardTitle is the label at the top of a card.
func cardTitle(c *ui.Context, title string) {
	ui.Text(c, title).FontSize(12).FontWeight(500).TextColor(theme.Muted).Padding(0, 0, 12, 0)
}

// Fact is one label and value of a fact grid.
type Fact struct {
	Label string
	Value string
	// Mono sets the value in the code face.
	Mono bool
	// Soft dims a value that explains rather than states.
	Soft bool
	// Tone, when not ToneIdle or when Dotted, puts a dot before the value.
	Tone   Tone
	Dotted bool
}

// Facts lays out a label column of labelW and a value column.
func Facts(c *ui.Context, labelW, gap float32, facts ...Fact) {
	ui.Grid(c).ColumnTracks(ui.Fixed(labelW), ui.Fr(1)).GapY(gap).AlignItems(ui.Center).Children(func() {
		for _, f := range facts {
			ui.Text(c, f.Label).FontSize(13).TextColor(theme.Muted)
			ui.Row(c).Gap(8).AlignItems(ui.Center).MinWidth(0).Children(func() {
				if f.Dotted || f.Tone != ToneIdle {
					ToneDot(c, f.Tone, 8)
				}
				factValue(c, f)
			})
		}
	})
}

// factValue is the value cell of one fact.
func factValue(c *ui.Context, f Fact) {
	t := ui.Text(c, f.Value).FontSize(13).TextColor(theme.Text).SingleLine().Grow(1).MinWidth(0)
	if f.Mono {
		t.Font(theme.FontMono).FontSize(12.5)
	}
	if f.Soft {
		t.TextColor(theme.TextSoft)
	}
}

// Heading is the title of a column.
func Heading(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(20).FontWeight(600).LetterSpacing(-0.2).TextColor(theme.Text)
}

// Mono is a line in the code face.
func Mono(c *ui.Context, text string, size float32, col ui.Color) ui.Element {
	return ui.Text(c, text).FontSize(size).Font(theme.FontMono).TextColor(col).SingleLine()
}

// EmptyCard is the dashed card in the middle of an empty page.
func EmptyCard(c *ui.Context, title, body string, emblem, action func()) {
	ui.Column(c).Grow(1).Center().Padding(40).Transition(PageTransition).Children(func() {
		ui.Column(c).Width(520).Padding(56, 40).Radius(24).
			Border(1.5, theme.DashStrong).BorderStyle(ui.BorderDashed).
			Background(theme.EmptyVeil).AlignItems(ui.Center).Gap(14).
			Children(func() {
				emblem()
				ui.Text(c, title).FontSize(22).FontWeight(600).LetterSpacing(-0.22).TextColor(theme.Text).Padding(6, 0, 0, 0)
				ui.Text(c, body).FontSize(13).LineHeight(1.6).TextColor(theme.TextSoft).TextAlign(ui.Center).Wrap()
				if action != nil {
					ui.Row(c).Padding(10, 0, 0, 0).Children(action)
				}
			})
	})
}

// FaultCard is a card whose hairline is oxide: what went wrong, in words.
func FaultCard(c *ui.Context, title, reason string) {
	Card(c, title, func() {
		ui.Text(c, reason).FontSize(13).LineHeight(1.6).TextColor(theme.Text).Wrap().Selectable()
	}).Border(1, theme.FaultBorder)
}

// Notice is the desk's last refusal, as an oxide strip.
func Notice(c *ui.Context, d *desk.Desk) {
	if d.Notice == "" {
		return
	}
	ui.Row(c).Padding(10, 12).Radius(theme.ControlR).Background(theme.FaultBg).Border(1, theme.FaultBorder).Gap(8).AlignItems(ui.Center).Children(func() {
		Dot(c, theme.Fault, 6)
		ui.Text(c, d.Notice).FontSize(12).TextColor(theme.FaultInk).Wrap().Grow(1)
	})
}

// Ago is how long before now t was, in the short form a list shows.
func Ago(now, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d/time.Hour))
	default:
		return t.Format("01-02 15:04")
	}
}
