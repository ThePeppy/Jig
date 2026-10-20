// Package settings draws the settings page: the IDA install, the local
// listeners and where Jig keeps its data. Ports are fixed, so they are
// shown, not edited. There is no theme switch.
package settings

import (
	"net/url"

	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/product"
	"jig/internal/ui/chrome"
	"jig/internal/ui/theme"
)

// View renders the three setting cards.
func View(c *ui.Context, d *desk.Desk) {
	ui.Scroll(c).Grow(1).MinWidth(0).Padding(24, 32).Gap(16).Children(func() {
		ui.Column(c).Padding(0, 0, 4, 0).Children(func() { chrome.Heading(c, "设置") })
		ida(c, d)
		service(c, d)
		data(c, d)
	})
}

func group(c *ui.Context, title string, action func(), facts ...chrome.Fact) {
	chrome.Card(c, "", func() {
		ui.Row(c).AlignItems(ui.Center).Padding(0, 0, 14, 0).Children(func() {
			ui.Text(c, title).FontSize(12).FontWeight(500).TextColor(theme.Muted)
			ui.Spacer(c)
			action()
		})
		chrome.Facts(c, 96, 12, facts...)
	})
}

func ida(c *ui.Context, d *desk.Desk) {
	st := d.Runtime()
	dir := "未设置"
	switch {
	case d.IdaDir != "":
		dir = product.ShortPath(d.IdaDir)
	case st.Found:
		dir = product.ShortPath(st.Install)
	}
	lib := chrome.Fact{Label: "libidalib", Value: "未找到", Dotted: true}
	if st.Library != "" {
		lib.Value, lib.Tone = "已找到", chrome.ToneLive
	}
	group(c, "IDA 安装", func() {
		if chrome.Button(c, "选择目录", chrome.SizeMedium).Clicked() && d.RequestIdaDir != nil {
			d.RequestIdaDir()
		}
	},
		chrome.Fact{Label: "目录", Value: dir, Mono: true},
		lib,
		chrome.Fact{Label: "说明", Value: "许可和库来自你本机的 IDA，不随应用分发", Soft: true},
	)
}

func service(c *ui.Context, d *desk.Desk) {
	state := chrome.Fact{Label: "状态", Value: "尚未监听", Dotted: true}
	if d.Listening {
		state.Value, state.Tone = "两个端口均在监听", chrome.ToneLive
	}
	group(c, "本地服务", func() {
		chrome.CopyButton(c, d, "settings-url", "复制地址", chrome.MCPURL, chrome.SizeMedium)
	},
		chrome.Fact{Label: "MCP", Value: chrome.MCPURL, Mono: true},
		chrome.Fact{Label: "中心", Value: chrome.HubAddr, Mono: true},
		state,
	)
}

func data(c *ui.Context, d *desk.Desk) {
	group(c, "数据", func() {
		if chrome.Button(c, "在访达中显示", chrome.SizeMedium).Clicked() && d.Dir() != "" {
			c.Services().OpenURL((&url.URL{Scheme: "file", Path: d.Dir()}).String())
		}
	},
		chrome.Fact{Label: "位置", Value: product.ShortPath(d.Dir()), Mono: true},
		chrome.Fact{Label: "保存内容", Value: "只保存库条目和 IDA 目录。会话与记录不落盘", Soft: true},
	)
}
