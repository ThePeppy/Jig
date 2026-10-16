// Package theme is the Jig surface: cool gray ground, white plates, ink marks.
package theme

import "github.com/egoist/mygo/ui"

const (
	WindowHex = "#E7E8EC"
	TitleBarH = 42
	ListW     = 264
	Gap       = 0
	Inset     = 0
	StatusH   = 30
	RowH      = 38
	// PaneR is 0: the system window corner is the only outer corner.
	PaneR = 0
	// CardR is a content-layer group, not a second window corner.
	CardR = 8
	// ControlR is a 28pt capsule. Toolbar controls use glass instead.
	ControlR = 14
	PillR    = 0

	FontUI   = "Helvetica Neue, PingFang SC"
	FontMono = "SF Mono, Menlo"
)

var (
	// Ground is cool and clearly separate from the white plates.
	Window    = ui.Hex(WindowHex)
	Pane      = ui.Hex("#FFFFFF")
	Card      = ui.Hex("#FFFFFF")
	Well      = ui.Hex("#F4F5F7")
	CardHover = ui.Hex("#F3F4F6")
	Active    = ui.Hex("#EEF0F3")
	Hover     = ui.Hex("#F6F7F9")
	Elevated  = ui.Hex("#FFFFFF")
	CodeBg    = ui.Hex("#F4F5F7")

	Ink       = ui.Hex("#14171C")
	Paper     = ui.Hex("#FFFFFF")
	Text      = Ink
	TextMuted = ui.Hex("#5C6570")
	Muted     = ui.Hex("#5C6570")
	Faint     = ui.Hex("#8B919A")

	// On is ink. The only chromatic color is fault.
	Live          = Ink
	LivePressed   = ui.Hex("#000000")
	LiveBg        = ui.Hex("#F4F5F7")
	LiveBorder    = ui.Hex("#D7DAE0")
	Run           = Ink
	RunBg         = LiveBg
	RunBorder     = LiveBorder
	Warn          = Ink
	WarnBg        = LiveBg
	WarnBorder    = LiveBorder
	Trouble       = ui.Hex("#9B2C2C")
	TroubleBg     = ui.Hex("#F8EEEE")
	TroubleBorder = ui.Hex("#9B2C2C")
	Idle          = Faint
	IdleBg        = ui.Hex("#F4F5F7")
	IdleBorder    = ui.Hex("#C5C9D1")

	Border       = ui.Hex("#E3E5EA")
	BorderSubtle = ui.Hex("#EEEFF2")
	BorderStrong = ui.Hex("#C8CCD4")
	Fill         = ui.Hex("#F4F5F7")
	FillHover    = ui.Hex("#EEF0F3")

	PrimaryBg    = Ink
	PrimaryHover = ui.Hex("#2A2E35")
	PrimaryText  = Paper
)

// Jig returns the light theme. No orange, no beige.
func Jig() *ui.Theme {
	t := ui.LightTheme()
	t.Background = Window
	t.Surface = Pane
	t.SurfaceHover = Hover
	t.SurfacePressed = Active
	t.Text = Text
	t.TextMuted = TextMuted
	t.Accent = Ink
	t.AccentHover = PrimaryHover
	t.AccentPressed = Ink
	t.AccentText = Paper
	t.Danger = Trouble
	t.Success = Ink
	t.Warning = Trouble
	t.Focus = ui.RGBA(20, 23, 28, 0.28)
	t.Selection = ui.RGBA(20, 23, 28, 0.10)
	t.Border = Border
	t.Scrollbar = ui.RGBA(20, 23, 28, 0.28)
	t.Inverse = Ink
	t.InverseText = Paper
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13
	t.Font = FontUI
	return t
}
