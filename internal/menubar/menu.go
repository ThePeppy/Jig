// Package menubar is the status item in the menu bar. It opens and closes
// analysis sessions and quits the app. It does not create the window and
// does not load idalib.
package menubar

import (
	"path/filepath"
	"strconv"

	"jig/internal/catalog"
	"jig/internal/desk"
	"jig/internal/product"
	"jig/internal/session"
)

// Item is one row of the status menu. A separator has no label. Children
// make a submenu. Run is nil when the row cannot be chosen.
type Item struct {
	ID        string
	Label     string
	Disabled  bool
	Separator bool
	Children  []Item
	Run       func()
}

// Items is the status menu for the current library and sessions.
// show brings the main window forward. quit ends the process.
func Items(d *desk.Desk, show, quit func()) []Item {
	return []Item{
		{ID: "show", Label: "显示主界面", Run: show},
		{Separator: true},
		{ID: "open", Label: "打开分析", Children: openItems(d, show)},
		{ID: "close", Label: "关闭分析", Children: closeItems(d)},
		{Separator: true},
		{ID: "quit", Label: "退出 " + product.Name, Run: quit},
	}
}

// Title is the count drawn beside the icon. Empty when nothing is open.
func Title(d *desk.Desk) string {
	n := openCount(d)
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func openItems(d *desk.Desk, show func()) []Item {
	entries := d.Catalog.Entries()
	labels := disambiguate(entries)
	var items []Item
	for _, e := range entries {
		if e.Missing() || d.Launching(e.ID) || sessionFor(d, e) != nil {
			continue
		}
		entry := e
		items = append(items, Item{
			ID:    "open-" + e.ID,
			Label: labels[e.ID],
			Run: func() {
				d.Tab = desk.TabLibrary
				_ = d.StartSession(entry)
				if show != nil {
					show()
				}
			},
		})
	}
	if len(items) == 0 {
		return []Item{{ID: "open-empty", Label: "没有可打开的文件", Disabled: true}}
	}
	return items
}

func closeItems(d *desk.Desk) []Item {
	var items []Item
	covered := map[string]bool{}
	for _, s := range d.Sessions.Sessions() {
		name := s.TargetName
		if name == "" {
			name = filepath.Base(firstPath(s.SourcePath, s.DatabasePath))
		}
		if name == "" {
			name = s.ID
		}
		id := s.ID
		st := s.State
		items = append(items, Item{
			ID:       "close-" + id,
			Label:    sessionLabel(name, st),
			Disabled: st == session.StateLaunching || st == session.StateStopping,
			Run: func() {
				if st == session.StateLaunching || st == session.StateStopping {
					return
				}
				_ = d.CloseSession(id)
			},
		})
		covered[clean(s.SourcePath)] = true
		covered[clean(s.DatabasePath)] = true
	}
	for _, e := range d.Catalog.Entries() {
		if !d.Launching(e.ID) || covered[clean(e.Path)] || covered[clean(e.Source)] {
			continue
		}
		items = append(items, Item{
			ID:       "close-launch-" + e.ID,
			Label:    e.Name + " · 启动中",
			Disabled: true,
		})
	}
	if len(items) == 0 {
		return []Item{{ID: "close-empty", Label: "没有打开的会话", Disabled: true}}
	}
	return items
}

func openCount(d *desk.Desk) int {
	n := 0
	seen := map[string]bool{}
	for _, s := range d.Sessions.Sessions() {
		if s.State == session.StateStopping {
			continue
		}
		n++
		seen[clean(s.SourcePath)] = true
		seen[clean(s.DatabasePath)] = true
	}
	for _, e := range d.Catalog.Entries() {
		if d.Launching(e.ID) && !seen[clean(e.Path)] && !seen[clean(e.Source)] {
			n++
		}
	}
	return n
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

func sessionLabel(name string, st session.State) string {
	switch st {
	case session.StateLaunching:
		return name + " · 启动中"
	case session.StateStopping:
		return name + " · 关闭中"
	case session.StateError:
		return name + " · 失败"
	default:
		return name
	}
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
