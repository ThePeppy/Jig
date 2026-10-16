# Jig

本机逆向工作台。MyGo 画窗口；分析在独立进程 `jig-idalib` 里，链接用户已安装的 IDA 9.3 `libidalib`。Agent 只走本进程 MCP：`http://127.0.0.1:9401/mcp`。

## 不许破的边界

- 窗口进程不 `init_library`、不加载 `libidalib`、不链 IDA、不启动 Python、不读旧仓库 `supper-ida-mcp-tools`，也不写 `ida-pro-mcp`。
- IDA 许可和库不打进应用。SDK 头只在 gitignore 的 `third_party/ida-sdk`（`v9.3.0-release` 的 `src/include`）。worker 二进制 `native/idalib-worker/jig-idalib` 同样不提交。
- 一个 worker 进程一份数据库。
- 扩展名不是接纳白名单，也不做 `fileAssociations`。选择器无类型过滤，含隐藏文件；另一条路径是拖入窗口。
- 不装 IDA 插件，不做插件重装、代理修复、IDA 窗口页。界面文案不得出现「重新安装」「IDA 窗口」。
- 深色玻璃固定，简体中文。不加工具还没接上的主题开关。侧栏是外框，透明度高，磨砂露出来；内容区是同一材质上更实的半透明板。
- 一个能力一个包。`main.go` 只有窗口、文件选择器、拖入。业务不进 `main`。

## 地图

| 路径 | 职责 |
| --- | --- |
| `main.go` | 窗口、`Take`、选 IDA 目录。对话框在 goroutine，结果经 `Window.Update` 回主线程。 |
| `internal/product` | 名称 `Jig`，数据目录 `~/.jig`，路径缩写成 `~`。棱柱标在 `DrawMark`。 |
| `internal/intake` | 路径能不能入库。无 UI，无磁盘。 |
| `internal/catalog` | 内存库。`Path` 是 idalib 要打开的路径，`Source` 是用户选的文件，`Name` 永远是用户看到的文件名。 |
| `internal/desk` | 窗口状态。只把库条目和 `idaDir` 写入 `~/.jig/state.json`。会话和记录不落盘。 |
| `internal/idalib` | 探测安装目录和 `libidalib` 是否存在。不加载。 |
| `internal/session` | worker 生命周期 + TCP hub `127.0.0.1:9399`。 |
| `internal/mcp` | MCP HTTP `127.0.0.1:9401`，`/mcp` 与 `/health`。 |
| `internal/trace` | 内存环形记录，默认 200 条，新的在前。 |
| `internal/ui/theme` | 颜色和度量。改表面先改这里。 |
| `internal/ui/chrome` | 标题栏、分栏、卡片、状态标记。页面复用，不另画一套。 |
| `internal/ui/{library,trace,settings}` | 三个标签页。打开的 worker 画在数据库页，不另开会话页。 |
| `internal/ui/appview` | 三页组合。 |
| `native/idalib-worker` | C++17 worker。工具实现在 `tools.cpp`。 |

## 接纳

`intake.Accept` 只判断路径是不是已存在的普通文件。空路径、不存在、目录、非常规文件拒绝。扩展名不是白名单，也不是黑名单：`.id0`、零字节、无扩展名都入库。不把旁边的 `.i64` / `.idb` 改写成打开路径。

能不能装载不在窗口进程里问。worker 在 `init_library` 之后：

- 扩展名不带点交给 `is_database_ext`。为真就直接 `open_database`，不要先当普通文件探装载器。9.3 里 `.id0` 也会返回真。
- 否则 `open_linput` + `build_loaders_list`。列表空就报「没有装载器接受这个文件」，不创建数据库。
- `open_database` 返回 0 才算打开。非 0 时把 `ui_msg` 收进错误串。`msg_get_lines` 在 idalib 里是空的，不要靠它。头文件不列举错误码。
- `open_database` 的契约是：数据库不存在就新建并装载输入。不要在 Go 里再猜伴随库。

窗口文案：入库失败走 `intake.Message`；打开失败用 worker 带回的句子。重复添加只选中已有条目。

## 进程与协议

`Desk.StartServers` 拉起两个回环监听。端口写死，不是设置项。`Listening` 为两者都成功。

Worker 查找：`JIG_IDALIB_WORKER` 优先。否则从可执行文件目录和 cwd 各自起，含自身最多 5 层找 `native/idalib-worker/jig-idalib`，再在这两处旁边找 `jig-idalib`。启动参数：`--center-host` `--center-port` `--path` `--instance-id` `--ida-dir`。`IDADIR` 设为解析后的安装目录（`.app` 落到 `Contents/MacOS`）。

帧：4 字节大端长度 + JSON，上限 64MiB。类型：`hello`、`tool_call`、`tool_result`、`shutdown`。worker 先打开数据库再连 hub，`hello` 带 `target.get_metadata`。`shutdown` 或连接断开后必须 `save_database(..., DBFL_KILL)` 把摊开的 `.id0/.id1/.id2/.nam/.til` 收回 `.i64` 再退出。不要 `Kill`：SIGKILL 会把库留在摊开状态。

MCP 名到 worker 名：

- 中心自己处理：`ida_list_targets`、`ida_open`、`ida_close_target`。`ida_open` 必须先过 `intake.Accept`。
- `ida_ping_target` → `target.ping`
- `ida_get_metadata` → `target.get_metadata`
- `ida_save_target` → `session.save`；另有 `session.start_auto_analysis` / `session.stop_auto_analysis` / `session.close`
- `ida_list_functions` 等 → `analysis.*`（`list_functions` `get_function` `decompile` `disassemble` `xrefs` `list_strings` `list_imports` `get_bytes` `search_text` `rename` `set_comment`）

对外清单是 `toolDefinitions()`。未公开的名字不要擅自写进 schema，也不要删掉已有分发。worker 调用超时 60s。同一 `OpenPath` 已有会话则复用。

探测只看文件在不在：已存目录优先，否则只扫平台应用目录里名称含 `IDA` 的安装（macOS 为 `/Applications/*.app`）。不是全盘搜索。

## 界面

MyGo `v0.5.2`。内容区 1280×800，最小 1080×680。`TitleBarHidden`，高 `theme.TitleBarH`（42）。系统画交通灯，不画假的。窗口圆角由系统裁切；内容不留第二套外圆角，不留分栏间隙。

左侧导航宽 `theme.NavW`（200），是窗口外框：低透明度罩层，系统磨砂从这里透出。内容区是同一材质、透明度更低的半透明板。卡片再压深一档。不要把外框铺成和内容差不多的实底。没有材质才用实色回退。故障只用氧化红。状态是圆点，字保持浅色。

三个目的地：数据库、记录、设置。打开的 worker 就在对应文件的详情里，不另开一页。「数据库」是库，不是扩展名过滤。

颜色只来自 `theme`。深色固定。不用橙色，不用米色。不加工具还没接上的主题开关。

UI 字符串中文，短。空态、按钮、错误句保持现有语气。时钟用 `d.Now()`，测试会冻住它。

MyGo 寿命：不把 `ui.Element` 或 parts 存进状态；稳定身份用 `c.Key` / `Bind`。构建函数会多次执行，副作用只放在点击和显式动作里，不放在无条件绘制路径。慢 I/O 离开 UI 线程，用 `Window.Update` 回写。改 API 前先看本仓库锁定的 mygo 源码，不要凭记忆。

## 改代码

- 包注释用英文，写清边界（做什么、明确不做什么）。给用户看的错误用中文。
- `Desk` 是主线程状态。hub、MCP、worker 退出经 `OnChange` → `NotifyUI` 刷新，不直接改 UI。
- 库的磁盘只在 `desk.Save`。catalog 不自己写文件。
- 新页面复用 `chrome` 的 Pane / Sidebar / Card / Stage / Status，不新造视觉语言。
- 不给选择器加扩展名过滤器，也不在 Go 里按扩展名拒绝附属文件或改写伴随库。不把零字节、无扩展名、固件、dex、apk 挡在库外。
- 注释与实现不一致时改注释。`ConduitBridge` 在接通时用 `theme.Live`，不是橙色。

## 验证

```text
go test ./...
go tool mygo vet .
```

改共享状态加 `-race`。UI 回归用 `ui.NewTester`：元素消失再出现再多走几帧，并覆盖焦点、点击、快捷键。截图测试只在设置了 `JIG_RENDER_DIR` 时写 PNG。

worker 不在 `go test` 里。改 `native/idalib-worker` 后：

```text
make -C native/idalib-worker
```

需要本机 IDA 9.3 和 `third_party/ida-sdk`。没有这两者就说明没跑到 idalib，不要把 Go 测试通过写成打开/反编译验收。
