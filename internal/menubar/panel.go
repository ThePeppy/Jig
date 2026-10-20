// Package menubar is the status item in the menu bar and the panel under
// it. The panel lists analysis sessions with a close key, offers files
// that can be opened, brings the main window forward and quits. It does
// not create the main window and does not load idalib.
package menubar

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"

	"jig/internal/catalog"
	"jig/internal/desk"
	"jig/internal/product"
	"jig/internal/session"
	"jig/internal/ui/chrome"
	"jig/internal/ui/theme"
)

// PanelW is the panel width in DIPs.
const PanelW = 304

const (
	panelPad   = 8
	headerH    = 54
	sectionH   = 25
	itemH      = 36
	ruleGap    = 6
	maxListH   = 8 * itemH
	panelFixed = 2*panelPad + headerH + 1 + ruleGap + // header and its rule
		2*(1+2*ruleGap) + 3*itemH // two rules around the window keys, three keys
)

// Actions are what the panel asks of the app.
type Actions struct {
	// Show brings the main window forward.
	Show func()
	// Quit ends the process.
	Quit func()
	// Dismiss closes the panel.
	Dismiss func()
}

type live struct {
	id, name string
	state    session.State
}

type openable struct {
	entry catalog.Entry
	label string
}

func lives(d *desk.Desk) []live {
	var out []live
	covered := map[string]bool{}
	for _, s := range d.Sessions.Sessions() {
		if s.State == session.StateClosed {
			continue
		}
		name := s.TargetName
		if name == "" {
			name = filepath.Base(firstPath(s.SourcePath, s.DatabasePath))
		}
		if name == "" || name == "." {
			name = s.ID
		}
		out = append(out, live{id: s.ID, name: name, state: s.State})
		covered[clean(s.SourcePath)] = true
		covered[clean(s.DatabasePath)] = true
	}
	for _, e := range d.Catalog.Entries() {
		if d.Launching(e.ID) && !covered[clean(e.Path)] && !covered[clean(e.Source)] {
			out = append(out, live{id: "launch-" + e.ID, name: e.Name, state: session.StateLaunching})
		}
	}
	return out
}

func openables(d *desk.Desk) []openable {
	entries := d.Catalog.Entries()
	labels := disambiguate(entries)
	var out []openable
	for _, e := range entries {
		if e.Missing() || d.Launching(e.ID) || sessionFor(d, e) != nil {
			continue
		}
		out = append(out, openable{entry: e, label: labels[e.ID]})
	}
	return out
}

// PanelHeight is the panel height for the current sessions and files.
func PanelHeight(d *desk.Desk) int {
	return panelFixed + listHeight(len(lives(d)), len(openables(d)))
}

func listHeight(nLive, nOpen int) int {
	h := sectionH + max(nLive, 1)*itemH
	if nOpen > 0 {
		h += sectionH + nOpen*itemH
	}
	return min(h, sectionH+maxListH)
}

// Panel is the view inside the tray flyout.
func Panel(d *desk.Desk, a Actions) func(*ui.Context) {
	return func(c *ui.Context) {
		c.SetTheme(theme.Jig())
		c.Root().Background(ui.Transparent)
		show := func() {
			a.Dismiss()
			a.Show()
		}
		c.OnShortcut(ui.Cmd, ui.KeyJ, show)
		c.OnShortcut(ui.Cmd, ui.KeyQ, a.Quit)

		ls, opens := lives(d), openables(d)
		ui.Column(c).Fill().Padding(panelPad).Radius(16).Background(theme.PanelSolid).Children(func() {
			header(c, d)
			rule(c, 0, ruleGap)
			ui.Scroll(c).Height(float32(listHeight(len(ls), len(opens)))).Children(func() {
				section(c, "分析会话")
				for _, l := range ls {
					sessionItem(c, d, l)
				}
				if len(ls) == 0 {
					ui.Row(c).Height(itemH).Padding(0, 10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "没有打开的会话").FontSize(13).TextColor(theme.Muted)
					})
				}
				if len(opens) > 0 {
					section(c, "打开分析")
					for _, o := range opens {
						openItem(c, d, o, a.Show)
					}
				}
			})
			rule(c, ruleGap, ruleGap)
			if key(c, "打开 Jig 窗口", "⌘J", true) {
				show()
			}
			if key(c, "关闭全部会话", "", len(ls) > 0) {
				d.CloseAll()
			}
			rule(c, ruleGap, ruleGap)
			if key(c, "退出 "+product.Name, "⌘Q", true) {
				a.Quit()
			}
		})
	}
}

func header(c *ui.Context, d *desk.Desk) {
	ui.Row(c).Height(headerH).Padding(10, 10, 12, 10).Gap(10).AlignItems(ui.Center).Children(func() {
		chrome.Mark(c, 24)
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Text(c, product.Name).FontSize(14).FontWeight(600).TextColor(theme.Text)
			chrome.Mono(c, chrome.MCPAddr, 11, theme.Muted)
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if d.Listening {
				chrome.Dot(c, theme.Live, 7)
				ui.Text(c, "运行中").FontSize(12).TextColor(theme.LiveInk)
			} else {
				chrome.Dot(c, theme.Idle, 7)
				ui.Text(c, "未启动").FontSize(12).TextColor(theme.TextSoft)
			}
		})
	})
}

func rule(c *ui.Context, above, below float32) {
	ui.Box(c).Height(1).FillWidth().Background(theme.Hairline).Margin(above, 6, below, 6)
}

func section(c *ui.Context, title string) {
	ui.Row(c).Height(sectionH).Padding(6, 10, 4, 10).AlignItems(ui.End).Children(func() {
		ui.Text(c, title).FontSize(11.5).FontWeight(500).TextColor(theme.Muted)
	})
}

func sessionItem(c *ui.Context, d *desk.Desk, l live) {
	item := ui.Row(c).Key("panel-live-"+l.id).Height(itemH).Padding(0, 6, 0, 10).Radius(theme.ControlR).Gap(10).AlignItems(ui.Center)
	if item.Hovered() {
		item.Background(theme.NavActive)
	}
	item.Children(func() {
		switch l.state {
		case session.StateLaunching, session.StateStopping:
			chrome.BusyDot(c, 7)
		case session.StateError:
			chrome.Dot(c, theme.Fault, 7)
		default:
			chrome.Dot(c, theme.Live, 7)
		}
		ui.Text(c, l.name).FontSize(13).TextColor(theme.Text).SingleLine().Grow(1).MinWidth(0)
		switch l.state {
		case session.StateLaunching:
			ui.Text(c, "启动中").FontSize(12).TextColor(theme.Muted).Padding(0, 6, 0, 0)
		case session.StateStopping:
			ui.Text(c, "关闭中").FontSize(12).TextColor(theme.Muted).Padding(0, 6, 0, 0)
		default:
			x := ui.Row(c).Size(28, 28).Radius(8).Center().Cursor(ui.CursorPointer).Focusable().Label("关闭 " + l.name + " 会话")
			col := theme.TextSoft
			if x.Hovered() {
				x.Background(theme.NavActive)
				col = theme.Text
			}
			if x.Clicked() {
				_ = d.CloseSession(l.id)
			}
			x.Children(func() { ui.Icon(c, chrome.IconClose).Size(12, 12).TextColor(col) })
		}
	})
}

func openItem(c *ui.Context, d *desk.Desk, o openable, show func()) {
	item := ui.Row(c).Key("panel-open-"+o.entry.ID).Height(itemH).Padding(0, 10).Radius(theme.ControlR).Gap(10).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Focusable().Label("打开 " + o.label)
	hover := item.Hovered()
	if hover {
		item.Background(theme.NavActive)
	}
	if item.Clicked() {
		d.Tab = desk.TabLibrary
		_ = d.StartSession(o.entry)
		show()
	}
	item.Children(func() {
		chrome.Dot(c, theme.Idle, 7)
		ui.Text(c, o.label).FontSize(13).TextColor(theme.TextSoft).SingleLine().Grow(1).MinWidth(0)
		if hover {
			ui.Text(c, "打开").FontSize(12).TextColor(theme.Muted)
		}
	})
}

func key(c *ui.Context, label, hint string, enabled bool) bool {
	btn := ui.Row(c).Key("panel-key-"+label).Height(itemH).Padding(0, 10).Radius(theme.ControlR).AlignItems(ui.Center).Label(label)
	col := theme.Muted
	if enabled {
		col = theme.Text
		btn.Cursor(ui.CursorPointer).Focusable()
		if btn.Hovered() {
			btn.Background(theme.NavActive)
		}
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(col).Grow(1)
		if hint != "" {
			ui.Text(c, hint).FontSize(12).TextColor(theme.Muted)
		}
	})
	return enabled && btn.Clicked()
}

func sessionFor(d *desk.Desk, e catalog.Entry) *session.Session {
	for _, s := range d.Sessions.Sessions() {
		if same(s.SourcePath, e.Path) || same(s.DatabasePath, e.Path) || same(s.SourcePath, e.Source) || same(s.DatabasePath, e.Source) {
			return s
		}
	}
	return nil
}

func disambiguate(entries []catalog.Entry) map[string]string {
	count := map[string]int{}
	for _, e := range entries {
		count[e.Name]++
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e.ID] = e.Name
		if count[e.Name] > 1 {
			out[e.ID] = e.Name + " — " + product.ShortPath(e.Path)
		}
	}
	return out
}

func firstPath(paths ...string) string {
	for _, p := range paths {
		if p != "" {
			return p
		}
	}
	return ""
}

func clean(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func same(a, b string) bool {
	return a != "" && b != "" && clean(a) == clean(b)
}
