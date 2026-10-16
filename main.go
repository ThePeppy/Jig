package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"app/internal/desk"
	"app/internal/product"
	"app/internal/ui/appview"
	"app/internal/ui/theme"
)

func main() {
	dir, err := product.DataDir()
	if err != nil {
		log.Fatal(err)
	}
	d, err := desk.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:                product.Name,
			Width:                1280,
			Height:               800,
			MinWidth:             1080,
			MinHeight:            680,
			UseContentSize:       true,
			TitleBarStyle:        mygo.TitleBarHidden,
			TitleBarHeight:       theme.TitleBarH,
			TrafficLightPosition: &mygo.Point{X: 12, Y: 14},
			BackgroundColor:      theme.WindowHex,
			StateKey:             "main",
			Content:              ui.View(appview.New(d)),
		})
		d.NotifyUI = func() { w.Update(func() {}) }
		d.StartServers()
		d.RequestAdd = func() { pickFiles(w, d) }
		d.RequestIdaDir = func() { pickIdaDir(w, d) }
		w.OnFileDrop(func(e *mygo.FileDropEvent) {
			paths := append([]string(nil), e.Paths...)
			w.Update(func() { d.Take(paths, nil) })
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
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
