// Package appview is the window content.
package appview

import (
	"time"

	"app/internal/desk"
	"app/internal/ui/chrome"
	"app/internal/ui/library"
	"app/internal/ui/sessions"
	"app/internal/ui/settings"
	"app/internal/ui/theme"
	"app/internal/ui/trace"
	"github.com/egoist/mygo/ui"
)

// New returns the view Mygo draws for d.
func New(d *desk.Desk) func(*ui.Context) {
	return func(c *ui.Context) {
		c.SetTheme(theme.Jig())
		ui.Column(c).Fill().Background(theme.Pane).Children(func() {
			chrome.Title(c, d)
			ui.Column(c).
				Key(d.Tab).
				Grow(1).
				Transition(ui.ElementTransition{
					Duration: 100 * time.Millisecond,
					Ease:     ui.EaseOut,
					Enter:    &ui.Motion{Opacity: 0},
				}).
				Children(func() {
					switch d.Tab {
					case desk.TabSessions:
						sessions.View(c, d)
					case desk.TabTrace:
						trace.View(c, d)
					case desk.TabSettings:
						settings.View(c, d)
					default:
						library.View(c, d)
					}
				})
		})
	}
}
