package menubar

import (
	"runtime"

	"github.com/egoist/mygo"

	"jig/internal/desk"
	"jig/internal/product"
)

// Bar is the menu-bar status item. Refresh rebuilds it from desk.
type Bar struct {
	d    *desk.Desk
	show func()
	quit func()
	tray *mygo.Tray
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
	b.Refresh()
	return b, nil
}

// Refresh rebuilds the menu and the count beside the icon.
func (b *Bar) Refresh() {
	if b == nil || b.tray == nil {
		return
	}
	mygo.RunOnMain(b.refresh)
}

func (b *Bar) refresh() {
	if b.tray == nil {
		return
	}
	b.tray.SetTitle(Title(b.d))
	b.tray.SetMenu(mygo.NewMenu(menuItems(Items(b.d, b.show, b.quit))))
}

// Close removes the status item.
func (b *Bar) Close() {
	if b == nil || b.tray == nil {
		return
	}
	b.tray.Destroy()
	b.tray = nil
}

func menuItems(items []Item) []*mygo.MenuItem {
	out := make([]*mygo.MenuItem, 0, len(items))
	for _, it := range items {
		if it.Separator {
			out = append(out, mygo.Separator())
			continue
		}
		row := &mygo.MenuItem{ID: it.ID, Label: it.Label, Disabled: it.Disabled}
		if len(it.Children) > 0 {
			row.Submenu = menuItems(it.Children)
		}
		if it.Run != nil {
			fn := it.Run
			row.Click = func(*mygo.MenuItem, *mygo.Window) { fn() }
		}
		out = append(out, row)
	}
	return out
}
