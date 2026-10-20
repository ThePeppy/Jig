package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/menubar"
	"jig/internal/product"
	"jig/internal/session"
	"jig/internal/ui/appview"
	"jig/internal/ui/theme"
)

// shell owns the main window. Closing it leaves the process up; the menu
// bar and the Dock bring it back. Quit is the only path that exits.
type shell struct {
	d        *desk.Desk
	w        *mygo.Window
	bar      *menubar.Bar
	quitting bool
	shut     bool
}

func main() {
	dir, err := product.DataDir()
	if err != nil {
		log.Fatal(err)
	}
	d, err := desk.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	s := &shell{d: d}
	mygo.Theme.SetSource(mygo.ThemeDark)
	// A listener, even an empty one, keeps the process after the last window.
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnActivate(func(bool) {
		if !s.quitting {
			s.show()
		}
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		s.quitting = true
		s.shutdownSessions()
	})
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		if s.bar != nil {
			s.bar.Close()
		}
	})
	mygo.App.WhenReady(func() {
		d.NotifyUI = s.poke
		d.Post = s.post
		d.StartServers()
		bar, err := menubar.Open(d, s.show, mygo.App.Quit)
		if err != nil {
			log.Print("菜单栏: ", err)
		}
		s.bar = bar
		s.show()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func (s *shell) show() {
	if s.quitting {
		return
	}
	if s.w != nil && !s.w.IsDestroyed() {
		if s.w.IsMinimized() {
			s.w.Restore()
		}
		s.w.Show()
		s.w.Focus()
		return
	}
	w := mygo.NewWindow(mygo.WindowOptions{
		Title:                product.Name,
		Width:                1280,
		Height:               800,
		MinWidth:             1080,
		MinHeight:            680,
		UseContentSize:       true,
		TitleBarStyle:        mygo.TitleBarHidden,
		TitleBarHeight:       theme.TitleBarH,
		TrafficLightPosition: &mygo.Point{X: 18, Y: 15},
		// Under-window blur is the ground. The rail tints it lightly.
		Vibrancy:        mygo.VibrancyUnderWindow,
		BackgroundColor: theme.WindowHex,
		StateKey:        "main",
		Content:         ui.View(appview.New(s.d)),
	})
	s.w = w
	s.d.RequestAdd = func() { pickFiles(w, s.d) }
	s.d.RequestIdaDir = func() { pickIdaDir(w, s.d) }
	s.d.RequestPack = func(id string) { pickPack(w, s.d, id) }
	w.OnFileDrop(func(e *mygo.FileDropEvent) {
		paths := append([]string(nil), e.Paths...)
		w.Update(func() { s.d.Take(paths, nil) })
	})
	w.OnClosed(func() {
		if s.w == w {
			s.w = nil
		}
	})
}

func (s *shell) post(fn func()) {
	go mygo.RunOnMain(func() {
		fn()
		s.refreshBar()
		s.redraw()
	})
}

func (s *shell) poke() {
	go mygo.RunOnMain(func() {
		s.refreshBar()
		s.redraw()
	})
}

func (s *shell) refreshBar() {
	if s.bar != nil {
		s.bar.Refresh()
	}
}

func (s *shell) redraw() {
	if s.w != nil && !s.w.IsDestroyed() {
		s.w.Invalidate()
	}
}

// shutdownSessions asks live workers to pack before the process exits.
// It does not wait: a disconnect also packs, and waiting would freeze quit.
func (s *shell) shutdownSessions() {
	if s.shut {
		return
	}
	s.shut = true
	for _, sess := range s.d.Sessions.Sessions() {
		if sess.State == session.StateStopping || sess.State == session.StateClosed {
			continue
		}
		_ = s.d.CloseSession(sess.ID)
	}
}

func pickFiles(w *mygo.Window, d *desk.Desk) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:          w,
			Title:           "添加文件",
			ButtonLabel:     "添加",
			Multiple:        true,
			ShowHiddenFiles: true,
		})
		w.Update(func() { d.Take(paths, err) })
	}()
}

func pickIdaDir(w *mygo.Window, d *desk.Desk) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    w,
			Title:     "选择 IDA 安装目录",
			Directory: true,
		})
		w.Update(func() {
			if err != nil || len(paths) == 0 {
				return
			}
			d.SetIdaDir(paths[0])
		})
	}()
}

func pickPack(w *mygo.Window, d *desk.Desk, id string) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:          w,
			Title:           "挂载分析包",
			ButtonLabel:     "挂载",
			ShowHiddenFiles: true,
		})
		w.Update(func() {
			if err != nil || len(paths) == 0 {
				return
			}
			d.AttachPack(id, paths[0])
		})
	}()
}
