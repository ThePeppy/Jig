// Package appview is the window content.
package appview

import (
	"time"

	"github.com/egoist/mygo/ui"
	"jig/internal/desk"
	"jig/internal/ui/chrome"
	"jig/internal/ui/library"
	"jig/internal/ui/settings"
	"jig/internal/ui/theme"
	"jig/internal/ui/trace"
)

// New returns the view Mygo draws for d.
func New(d *desk.Desk) func(*ui.Context) {
	return func(c *ui.Context) {
		c.SetTheme(theme.Jig())
		chrome.Shell(c, d, func() {
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
