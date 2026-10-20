// Package theme is the Jig surface.
// The window is dark glass. The rail is the frame: a faint tint, so the
// desktop frost shows through. The desk is a much less transparent plate
// of the same material. Cards are a light veil on the desk. No theme
// switch. Green is live, periwinkle is focus and work, fault is oxide red.
// No orange, no beige.
package theme

import "github.com/egoist/mygo/ui"

const (
	// WindowHex is the color before the first frame, the desk's ground.
	WindowHex = "#10151C"
	TitleBarH = 42
	NavW      = 200
	ListW     = 340
	DetailW   = 380

	NavR     = 10
	RowR     = 12
	CardR    = 16
	ControlR = 10

	FontUI   = "Inter, SF Pro Text, PingFang SC"
	FontMono = "JetBrains Mono, SF Mono, Menlo"
)

var (
	Window = ui.Hex(WindowHex)

	Text     = ui.Hex("#EEF1F5")
	TextSoft = ui.Hex("#A7B0BC")
	Muted    = ui.Hex("#8892A0")
	CodeInk  = ui.Hex("#C9D1DC")
	// Ink is the dark text on a primary key.
	Ink = ui.Hex("#10151C")

	Accent  = ui.Hex("#8CA8FF")
	Live    = ui.Hex("#52D6A4")
	LiveInk = ui.Hex("#BFF1DE")
	LiveBg  = ui.Hex("#52D6A4").Alpha(0.12)
	// LiveHalo rings the service dot.
	LiveHalo    = ui.Hex("#52D6A4").Alpha(0.18)
	Fault       = ui.Hex("#D8604C")
	FaultInk    = ui.Hex("#F4C4BB")
	FaultBg     = ui.Hex("#D8604C").Alpha(0.16)
	FaultBorder = ui.Hex("#D8604C").Alpha(0.35)
	BusyInk     = ui.Hex("#D3DDFF")
	BusyBg      = ui.Hex("#8CA8FF").Alpha(0.14)
	Idle        = ui.Hex("#5A6472")

	Primary = Text

	// Veils are white over whatever is below: frost or the solid fallback.
	RailCardVeil = ui.Hex("#FFFFFF").Alpha(0.045)
	CardVeil     = ui.Hex("#FFFFFF").Alpha(0.04)
	EmptyVeil    = ui.Hex("#FFFFFF").Alpha(0.025)
	RowHover     = ui.Hex("#FFFFFF").Alpha(0.05)
	SegmentVeil  = RowHover
	NavHover     = ui.Hex("#FFFFFF").Alpha(0.06)
	FieldVeil    = NavHover
	Rule         = ui.Hex("#FFFFFF").Alpha(0.07)
	Hairline     = ui.Hex("#FFFFFF").Alpha(0.08)
	ControlVeil  = Hairline
	RowActive    = ui.Hex("#FFFFFF").Alpha(0.09)
	NavActive    = ui.Hex("#FFFFFF").Alpha(0.10)
	SegmentOn    = ui.Hex("#FFFFFF").Alpha(0.12)
	ControlHover = ui.Hex("#FFFFFF").Alpha(0.13)
	Dash         = ui.Hex("#FFFFFF").Alpha(0.18)
	DashStrong   = ui.Hex("#FFFFFF").Alpha(0.20)
	Well         = ui.Hex("#000000").Alpha(0.28)

	// Over vibrancy the rail is a faint cool tint and the desk a dense one.
	RailGlass = ui.Hex("#141B26").Alpha(0.30)
	DeskGlass = ui.Hex("#0E1217").Alpha(0.74)
	// Without a material the same two regions are solid.
	RailSolid = ui.Hex("#151A21")
	DeskSolid = ui.Hex("#0F1319")
	// PanelSolid is the menu-bar panel. It has no material behind it, so
	// it stays nearly opaque instead of showing the desktop unblurred.
	PanelSolid = ui.Hex("#1A1F28").Alpha(0.97)
)

// Jig returns the dark glass theme. No orange, no beige, no theme switch.
func Jig() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = Window
	t.Surface = DeskSolid
	t.SurfaceHover = ControlHover
	t.SurfacePressed = SegmentOn
	t.Text = Text
	t.TextMuted = Muted
	t.Accent = Accent
	t.AccentHover = ui.Hex("#A3B9FF")
	t.AccentPressed = ui.Hex("#7592F0")
	t.AccentText = Ink
	t.Danger = Fault
	t.Success = Live
	t.Warning = Fault
	t.Focus = Accent
	t.Selection = Accent.Alpha(0.28)
	t.Border = Hairline
	t.Scrollbar = ui.RGBA(238, 241, 245, 0.28)
	t.Inverse = ui.Hex("#1A1F28")
	t.InverseText = Text
	t.Radius = ControlR
	t.Spacing = 4
	t.FontSize = 13
	t.Font = FontUI
	return t
}
