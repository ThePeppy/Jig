// Package theme is the Jig surface.
// The window is dark glass. The rail is the frame: a thin tint, so the
// desktop frost shows through. The desk is a less transparent plate of
// the same material. Cards sit darker on that plate. No theme switch.
// Indigo marks an action. Fault is oxide red. No orange, no beige.
package theme

import "github.com/egoist/mygo/ui"

const (
	// WindowHex is the color before the first frame, close to the desk.
	WindowHex = "#1C1C1A"
	TitleBarH = 42
	NavW      = 200
	ListW     = 264
	Gap       = 0
	Inset     = 0
	StatusH   = 32
	RowH      = 32
	// PaneR is 0: the system window corner is the only outer corner.
	PaneR = 0
	// CardR is a content-layer group, not a second window corner.
	CardR = 8
	// ControlR is the radius of a control. Sharp enough to read on a large screen.
	ControlR = 6
	PillR    = 8

	FontUI   = "Inter, SF Pro Text, PingFang SC"
	FontMono = "SF Mono, Menlo"
)

var (
	Window = ui.Hex(WindowHex)
	// Sidebar is the rail where the window has no material.
	Sidebar   = ui.Hex("#2A2E28")
	Canvas    = ui.Hex("#1E1E1C")
	Pane      = ui.Hex("#262624")
	Card      = Pane
	Well      = ui.Hex("#171716")
	CardHover = ui.Hex("#30302C")
	Active    = ui.Hex("#3A3E36")
	Hover     = ui.Hex("#32362E")
	Elevated  = ui.Hex("#2E2E2A")
	CodeBg    = ui.Hex("#171716")

	// The rail is the frame: low alpha, the frost shows.
	// The desk is the same material, much less transparent.
	RailGlass           = ui.Hex("#141614").Alpha(0.32)
	DeskGlass           = ui.Hex("#2A2A28").Alpha(0.78)
	CardGlass           = ui.Hex("#161614").Alpha(0.62)
	PlateGlass          = ui.Hex("#121210").Alpha(0.50)
	BandGlass           = ui.Hex("#242422").Alpha(0.84)
	RailHoverGlass      = ui.Hex("#FFFFFF").Alpha(0.10)
	RailActiveGlass     = ui.Hex("#FFFFFF").Alpha(0.18)
	DeskHoverGlass      = ui.Hex("#FFFFFF").Alpha(0.06)
	DeskActiveGlass     = ui.Hex("#FFFFFF").Alpha(0.12)
	ControlGlass        = ui.Hex("#FFFFFF").Alpha(0.08)
	PrimaryGlass        = ui.Hex("#FFFFFF").Alpha(0.14)
	PrimaryHoverGlass   = ui.Hex("#FFFFFF").Alpha(0.22)
	PrimaryPressedGlass = ui.Hex("#FFFFFF").Alpha(0.08)
	HairlineGlass       = ui.Hex("#FFFFFF").Alpha(0.16)
	RuleGlass           = ui.Hex("#FFFFFF").Alpha(0.08)
	FaultGlass          = ui.Hex("#9B2C2C").Alpha(0.28)

	Ink       = ui.Hex("#F2F0EC")
	Paper     = ui.Hex("#F4F1EA")
	Text      = Ink
	TextMuted = ui.Hex("#A39E96")
	Muted     = ui.Hex("#A39E96")
	Faint     = ui.Hex("#7C7870")

	Accent        = ui.Hex("#8B93E0")
	AccentHover   = ui.Hex("#A2A8EA")
	AccentPressed = ui.Hex("#6E78D4")

	Live          = Accent
	LivePressed   = AccentPressed
	LiveBg        = ui.Hex("#8B93E0").Alpha(0.16)
	LiveBorder    = ui.Hex("#8B93E0").Alpha(0.40)
	Run           = Ink
	RunBg         = Well
	RunBorder     = ui.Hex("#3A3A36")
	Warn          = Ink
	WarnBg        = Well
	WarnBorder    = ui.Hex("#3A3A36")
	Trouble       = ui.Hex("#E07A72")
	TroubleBg     = ui.Hex("#9B2C2C").Alpha(0.28)
	TroubleBorder = ui.Hex("#E07A72")
	Idle          = Faint
	IdleBg        = Well
	IdleBorder    = ui.Hex("#3A3A36")

	Border       = ui.Hex("#3A3A36")
	BorderSubtle = ui.Hex("#2E2E2A")
	BorderStrong = ui.Hex("#4A4A46")
	Fill         = Well
	FillHover    = Hover

	PrimaryBg    = ui.Hex("#3A3C38")
	PrimaryHover = ui.Hex("#4A4C48")
	PrimaryText  = Paper
)

// Jig returns the dark glass theme. No orange, no beige, no theme switch.
func Jig() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = Window
	t.Surface = Pane
	t.SurfaceHover = Hover
	t.SurfacePressed = Active
	t.Text = Text
	t.TextMuted = TextMuted
	t.Accent = Accent
	t.AccentHover = AccentHover
	t.AccentPressed = AccentPressed
	t.AccentText = Paper
	t.Danger = Trouble
	t.Success = Accent
	t.Warning = Trouble
	t.Focus = Accent.Alpha(0.35)
	t.Selection = Accent.Alpha(0.12)
	t.Border = Border
	t.Scrollbar = ui.RGBA(242, 240, 236, 0.32)
	t.Inverse = ui.Hex("#1C1C1A")
	t.InverseText = Paper
	t.Radius = ControlR
	t.Spacing = 4
	t.FontSize = 13
	t.Font = FontUI
	return t
}
