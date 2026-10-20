package menubar

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/product"
)

// Bar is the menu-bar status item. A click opens the panel under it; a
// second click, Escape or a click elsewhere closes it.
type Bar struct {
	d     *desk.Desk
	show  func()
	quit  func()
	tray  *mygo.Tray
	panel *mygo.Window
}

// Open adds the status item. Call it once the app is ready.
// show brings the main window forward. quit ends the process.
func Open(d *desk.Desk, show, quit func()) (*Bar, error) {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon:           product.PNG(32, product.MarkTemplate),
		IconIsTemplate: runtime.GOOS == "darwin",
		ToolTip:        product.Name,
	})
	if err != nil {
		return nil, err
	}
	b := &Bar{d: d, show: show, quit: quit, tray: tray}
	tray.OnClick(b.toggle)
	return b, nil
}

func (b *Bar) toggle() {
	if b.panel != nil {
		b.panel.Close()
		return
	}
	w := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      b.tray,
		Placement: mygo.PlacementBottom,
		Gap:       6,
		Width:     PanelW,
		Height:    PanelHeight(b.d),
		Focusable: true,
		Shadow:    true,
		Content: ui.View(Panel(b.d, Actions{
			Show:    b.show,
			Quit:    b.quit,
			Dismiss: b.dismiss,
		})),
	})
	b.panel = w
	w.OnClosed(func() {
		if b.panel == w {
			b.panel = nil
		}
	})
}

func (b *Bar) dismiss() {
	if b.panel != nil {
		b.panel.Close()
	}
}

// Refresh redraws an open panel at the height its rows need.
func (b *Bar) Refresh() {
	if b == nil {
		return
	}
	mygo.RunOnMain(func() {
		if b.panel == nil || b.panel.IsDestroyed() {
			return
		}
		b.panel.SetContentSize(PanelW, PanelHeight(b.d))
		b.panel.Invalidate()
	})
}

// Close removes the status item and its panel.
func (b *Bar) Close() {
	if b == nil || b.tray == nil {
		return
	}
	b.dismiss()
	b.tray.Destroy()
	b.tray = nil
}
