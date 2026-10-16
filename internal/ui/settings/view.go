package settings

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"jig/internal/desk"
	"jig/internal/idalib"
	"jig/internal/product"
	"jig/internal/ui/chrome"
	"jig/internal/ui/theme"
)

// View draws the settings groups.
func View(c *ui.Context, d *desk.Desk) {
	ui.Row(c).Grow(1).Gap(0).AlignItems(ui.Stretch).Children(func() {
		list(c, d)
		body(c, d)
	})
}

func list(c *ui.Context, d *desk.Desk) {
	pane := chrome.Sidebar(c)
	portMeta := "尚未监听"
	if d.Listening {
		portMeta = "监听中"
	}
	pane.Children(func() {
		ui.Row(c).Height(32).Padding(0, 12).BorderWidth(0, 0, 1, 0).BorderColor(chrome.Wash(c, "rule")).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "系统选项").FontSize(11).TextColor(theme.TextMuted)
		})

		ui.Scroll(c).Grow(1).Padding(4, 6).Gap(2).Children(func() {
			item(c, d, desk.SettingsIDA, "IDA", "目录", "安装路径与运行时")
			item(c, d, desk.SettingsPorts, "端口", portMeta, "MCP 与 Worker")
			item(c, d, desk.SettingsAgents, "Agent", "客户端连接", "Claude、Codex、OpenCode")
			item(c, d, desk.SettingsAppearance, "界面", "中文", "深色玻璃")
		})
	})
}

func item(c *ui.Context, d *desk.Desk, id desk.SettingsSection, title, meta, sub string) {
	on := d.Section == id
	row := ui.Column(c).
		Key(fmt.Sprintf("set-row-%d", id)).
		MinHeight(40).
		Padding(6, 10).
		Radius(theme.ControlR).
		Cursor(ui.CursorPointer).
		Transition(chrome.ColorTransition).
		Justify(ui.Center).
		Gap(3).
		Focusable().
		Label(title)

	if row.Clicked() {
		d.Section = id
		on = true
	}

	chrome.StyleSidebarRow(c, row, on, theme.PrimaryBg)

	titleColor := theme.TextMuted
	if on {
		titleColor = theme.Text
	}

	row.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(titleColor).Grow(1)
			ui.Text(c, meta).FontSize(11).TextColor(theme.TextMuted)
		})
		ui.Text(c, sub).FontSize(11).TextColor(theme.Muted).SingleLine()
	})
}

func body(c *ui.Context, d *desk.Desk) {
	pane := chrome.Pane(c).Grow(1)
	pane.Children(func() {
		ui.Column(c).
			Key(fmt.Sprintf("set-body-%d", d.Section)).
			Grow(1).
			Transition(chrome.DetailTransition).
			Children(func() {
				switch d.Section {
				case desk.SettingsPorts:
					ports(c, d)
				case desk.SettingsAgents:
					agents(c, d)
				case desk.SettingsAppearance:
					appearance(c)
				default:
					ida(c, d)
				}
			})
		chrome.Status(c, d, "设置", "idle")
	})
}

func ida(c *ui.Context, d *desk.Desk) {
	st := d.Runtime()
	ready := d.IdaDir != "" || st.Found

	ui.Column(c).Padding(16, 24, 12, 24).BorderWidth(0, 0, 1, 0).BorderColor(chrome.Wash(c, "rule")).Gap(4).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "IDA").FontSize(15).FontWeight(500).TextColor(theme.Text)
			if ready {
				chrome.Stage(c, "ok", "就绪")
			} else {
				chrome.Stage(c, "idle", "未设置")
			}
			ui.Spacer(c)

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				pick := chrome.PrimaryButton(c, "选择目录...", nil).Label("选择目录")
				if pick.Clicked() && d.RequestIdaDir != nil {
					d.RequestIdaDir()
				}

				refresh := chrome.GhostButton(c, "重新探测", nil, false)
				if refresh.Clicked() {
					d.RefreshRuntime()
				}
			})
		})

		ui.Text(c, "Jig 加载本机 libidalib，不启动 IDA 界面，也不启动 Python。").FontSize(13).TextColor(theme.TextMuted)
	})

	ui.Scroll(c).Grow(1).Padding(16, 24).Gap(12).Children(func() {
		// 3-Segment Horizontal Readout Strip
		chrome.Card(c).Children(func() {
			ui.Row(c).Dividers(1, chrome.Wash(c, "rule")).AlignItems(ui.Stretch).Children(func() {
				idalibVal := "未找到"
				idalibCol := theme.Idle
				if st.Library != "" {
					idalibVal = "可加载"
					idalibCol = theme.Live
				}
				chrome.ReadoutCell(c, "idalib", idalibVal, "无界面逆向引擎", idalibCol)
				chrome.ReadoutCell(c, "库文件", libraryTitle(st), "本机 libidalib，不打进应用", theme.Run)
				chrome.ReadoutCell(c, "数据目录", "~/.jig", "状态与样本索引存储", theme.TextMuted)
			})
		})

		// Dual-Slot Path Cards
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			badge := "未设置"
			tone := "idle"
			if ready {
				badge = "已探测"
				tone = "ok"
			}
			chrome.PathSlotCard(c, "目录", installTitle(d), installLine(d), badge, tone).Grow(1).Basis(0)
			chrome.ConduitBridge(c, "Headless", ready)
			chrome.PathSlotCard(c, "libidalib", libraryTitle(st), libraryLine(st), libraryBadge(st), libraryTone(st)).Grow(1).Basis(0)
		})

		// Harmonious Architecture Explanation Card
		chrome.Plate(c).Padding(16, 18).Gap(8).Children(func() {
			ui.Text(c, "无插件直连").FontSize(13).FontWeight(500).TextColor(theme.Text)
			ui.Text(c, "不往 IDA 的 plugins 目录装插件。打开后由原生 worker 调用本机 libidalib 和 Hex-Rays，不启动 Python。").FontSize(12).TextColor(theme.TextMuted)
		})
	})
}

func ports(c *ui.Context, d *desk.Desk) {
	ui.Column(c).Padding(16, 24, 12, 24).BorderWidth(0, 0, 1, 0).BorderColor(chrome.Wash(c, "rule")).Gap(4).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "端口").FontSize(15).FontWeight(500).TextColor(theme.Text)
			if d.Listening {
				chrome.Stage(c, "ok", "监听中")
			} else {
				chrome.Stage(c, "idle", "尚未监听")
			}
		})
		if d.Listening {
			ui.Text(c, "MCP HTTP 与 Worker TCP 已在本机回环地址监听。").FontSize(13).TextColor(theme.TextMuted)
		} else {
			ui.Text(c, "MCP 与 Worker 在本机监听。目前尚未监听。").FontSize(13).TextColor(theme.TextMuted)
		}
	})

	ui.Scroll(c).Grow(1).Padding(16, 24).Gap(12).Children(func() {
		chrome.Card(c).Children(func() {
			ui.Row(c).Dividers(1, chrome.Wash(c, "rule")).AlignItems(ui.Stretch).Children(func() {
				chrome.ReadoutCell(c, "MCP", "127.0.0.1:9401", "对外 Agent HTTP 端点", theme.Run)
				chrome.ReadoutCell(c, "Worker", "127.0.0.1:9399", "内部 idalib 进程通信", theme.Live)
			})
		})

		chrome.Plate(c).Padding(16, 20).Children(func() {
			ui.Row(c).Justify(ui.Center).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Column(c).AlignItems(ui.Center).Gap(3).Children(func() {
					ui.Text(c, "AI Agent").FontSize(13).FontWeight(500).TextColor(theme.Text)
					ui.Text(c, "Claude / Codex").FontSize(11).TextColor(theme.TextMuted)
				})
				chrome.ConduitBridge(c, "127.0.0.1:9401", d.Listening)
				ui.Column(c).AlignItems(ui.Center).Gap(3).Children(func() {
					ui.Text(c, "Jig").FontSize(13).FontWeight(500).TextColor(theme.Text)
					ui.Text(c, "本机中心").FontSize(11).TextColor(theme.TextMuted)
				})
				chrome.ConduitBridge(c, "127.0.0.1:9399", d.Listening)
				ui.Column(c).AlignItems(ui.Center).Gap(3).Children(func() {
					ui.Text(c, "idalib Worker").FontSize(13).FontWeight(500).TextColor(theme.Text)
					ui.Text(c, "无头分析进程").FontSize(11).TextColor(theme.TextMuted)
				})
			})
		})
	})
}

func agents(c *ui.Context, d *desk.Desk) {
	snippet := `{"mcpServers":{"jig":{"url":"http://127.0.0.1:9401/mcp"}}}`

	ui.Column(c).Padding(16, 24, 12, 24).BorderWidth(0, 0, 1, 0).BorderColor(chrome.Wash(c, "rule")).Gap(4).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Agent 客户端连接").FontSize(15).FontWeight(500).TextColor(theme.Text)
			ui.Spacer(c)

			copyBtn := chrome.PrimaryButton(c, "复制 MCP JSON 配置", chrome.IconCopy).Label("复制配置")
			if copyBtn.Clicked() {
				mygo.Clipboard.WriteText(snippet)
				d.Notice = "已复制 MCP 配置 JSON"
			}
		})
		ui.Text(c, "在外部 AI Agent 中配置 MCP 服务端点即可直接调用 idalib 进行逆向分析。").FontSize(13).TextColor(theme.TextMuted)
	})

	ui.Scroll(c).Grow(1).Padding(16, 24).Gap(12).Children(func() {
		chrome.Card(c).Children(func() {
			ui.Row(c).Dividers(1, chrome.Wash(c, "rule")).AlignItems(ui.Stretch).Children(func() {
				chrome.ReadoutCell(c, "端点 URL", "http://127.0.0.1:9401/mcp", "本机回环 HTTP 入口", theme.Live)
				chrome.ReadoutCell(c, "协议", "Streamable HTTP (SSE) / JSON-RPC", "标准 MCP 传输层", theme.Run)
			})
		})

		// Code Editor Window Card with Line Numbers (harmonious slate surface)
		ui.Column(c).Gap(8).Children(func() {
			chrome.Kicker(c, "配置片段")
			chrome.Card(c).Clip().Children(func() {
				ui.Row(c).Height(32).Padding(0, 14).Background(chrome.Wash(c, "plate")).BorderWidth(0, 0, 1, 0).BorderColor(chrome.Wash(c, "hairline")).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "mcp.json").FontSize(11).Font("SF Mono").TextColor(theme.Text)
					ui.Spacer(c)
					ui.Text(c, "Claude · Codex · OpenCode").FontSize(11).TextColor(theme.TextMuted)
				})

				codeLines := []struct {
					no   string
					text string
					col  ui.Color
				}{
					{"1", `{`, theme.TextMuted},
					{"2", `  "mcpServers": {`, theme.Text},
					{"3", `    "jig": {`, theme.Text},
					{"4", `      "url": "http://127.0.0.1:9401/mcp"`, theme.Text},
					{"5", `    }`, theme.Text},
					{"6", `  }`, theme.Text},
					{"7", `}`, theme.TextMuted},
				}
				ui.Column(c).Padding(14, 18).Gap(4).Children(func() {
					for _, ln := range codeLines {
						ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
							ui.Text(c, ln.no).FontSize(11).Font("SF Mono").TextColor(theme.Muted).Width(16)
							ui.Text(c, ln.text).FontSize(12).Font("SF Mono").TextColor(ln.col)
						})
					}
				})
			})
		})
	})
}

func appearance(c *ui.Context) {
	ui.Column(c).Padding(16, 24, 12, 24).BorderWidth(0, 0, 1, 0).BorderColor(chrome.Wash(c, "rule")).Gap(4).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "外观").FontSize(15).FontWeight(500).TextColor(theme.Text)
			chrome.Stage(c, "done", "深色玻璃")
		})
		ui.Text(c, "深色玻璃，中文。侧栏更透，内容区更实。不提供主题开关。").FontSize(13).TextColor(theme.TextMuted)
	})

	ui.Scroll(c).Grow(1).Padding(16, 24).Gap(12).Children(func() {
		chrome.Card(c).Children(func() {
			ui.Row(c).Dividers(1, chrome.Wash(c, "hairline")).AlignItems(ui.Stretch).Children(func() {
				chrome.ReadoutCell(c, "语言", "简体中文", "界面显示语言", theme.Text)
				chrome.ReadoutCell(c, "外观", "深色玻璃", "侧栏更透，内容更实", theme.Text)
			})
		})
	})
}

func installTitle(d *desk.Desk) string {
	if d.IdaDir != "" {
		return filepath.Base(d.IdaDir)
	}
	st := d.Runtime()
	if st.Found {
		return filepath.Base(st.Install)
	}
	return "未指定安装目录"
}

func libraryTitle(st idalib.Status) string {
	if st.Library != "" {
		return filepath.Base(st.Library)
	}
	return "未找到"
}

func libraryLine(st idalib.Status) string {
	if st.Library != "" {
		return product.ShortPath(st.Library)
	}
	return "安装目录里没有 libidalib"
}

func libraryBadge(st idalib.Status) string {
	if st.Library != "" {
		return "已探测"
	}
	return "缺失"
}

func libraryTone(st idalib.Status) string {
	if st.Library != "" {
		return "ok"
	}
	return "idle"
}

func installLine(d *desk.Desk) string {
	if d.IdaDir != "" {
		return product.ShortPath(d.IdaDir)
	}
	st := d.Runtime()
	if st.Found {
		return "探测到 " + product.ShortPath(st.Install)
	}
	return "未设置"
}
