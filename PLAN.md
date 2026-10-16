# Jig 原生应用计划

本目录是项目根。Mygo 绘制窗口，进程内完成库、会话和调用记录。不再以 IDA Pro 窗口为产品对象，不再安装或管理 IDA GUI 插件。

旧仓库 `supper-ida-mcp-tools` 不是本进程的依赖。本仓库不读取它的路径，也不启动它的 Python worker。

## 已确认的缺陷

`MainWindow` 的文件选择器把默认类型收成 `*.i64 *.idb *.exe *.dll *.sys *.so *.dylib *.bin *.elf`。库文案也把产品说成「数据库或 exe」。idalib 的 `open_database` 接受 IDA 装载器能读的任意路径，扩展名不是能力边界。固件、无扩展名、dex、apk、镜像因此进不了日常路径。

选择器里虽然有「所有文件」，产品仍按数据库和 exe 设计。这是产品缺陷，不是 IDA 的限制。

## 文件接纳

规则在 `internal/intake`。扩展名只用于展示和伴随数据库解析，绝不作为拒绝条件。

接纳：

- 任意已存在的普通文件，包括无扩展名。
- `.i64` / `.idb` 记为数据库。
- 非数据库文件若旁边已有伴随数据库，打开路径改记伴随文件，避免再次灌入。查找顺序与旧 `IdaDatabasePaths` 相同：`stem.i64`、`stem.idb`、`path+.i64`、`path+.idb`。列表名仍用来源文件名，不把用户选的二进制改写成 `.i64`。
- 零字节文件也接纳。能不能分析由 idalib 决定。

拒绝：

- 空路径、不存在的路径、目录、非常规文件。
- IDA 附属文件 `.id0` `.id1` `.id2` `.nam` `.til`。它们不是可打开的输入。提示改加数据库或原始二进制。

系统限制：macOS 不能把「所有文件」注册成 Finder 的打开方式。产品路径是窗口内选择器（不加过滤器，含隐藏文件）和拖入窗口。不把 `fileAssociations` 做成另一份白名单。

## 依赖边界

IDA 的许可和 `libidalib` 不能打进应用。分析在独立进程 `native/idalib-worker/jig-idalib` 里完成：链接本机 `libidalib.dylib` 和 `libida.dylib`，用 IDA 9.3 SDK 头文件调用函数列表、反汇编、交叉引用、字符串、导入、搜索、改名、注释和 Hex-Rays `decompile`。不链进窗口进程，不启动 Python，不回头找旧仓库。

一个进程一份数据库。SDK 头文件按 `v9.3.0-release` 取到 `third_party/ida-sdk`，不提交。运行时只探测安装目录，不在窗口进程里 `init_library`。

运行时只探测，不加载库：

- 已保存的安装目录优先。
- 否则在平台应用目录里找名称含 `IDA` 的安装（macOS 为 `/Applications/*.app`）。这是提示，不是全盘搜索。用户始终可以改目录。
- 探测只看 `libidalib` 文件在不在，不调用 `init_library`。

## 目录

```text
.
  main.go                         进程入口。窗口、文件选择器、拖入。
  mygo.json
  PLAN.md                         本文件。产品切完后删除。
  internal/product                名称、数据目录、路径显示。
  internal/intake                 文件接纳。无 UI。
  internal/catalog                库条目。无磁盘。
  internal/idalib                 安装目录与 libidalib 探测。不加载库。
  internal/desk                   窗口状态与 ~/.jig/state.json。
  internal/ui/theme               spec.json 的颜色和度量。
  internal/ui/chrome              标题栏、分栏、stage、状态条。
  internal/ui/library             库。
  internal/ui/sessions            会话。
  internal/ui/trace               记录。
  internal/ui/settings            设置。
  internal/ui/appview             四页组合。
```

一个能力一个包。`main.go` 不放业务。

## 窗口
- 内容区 1280×800，最小 1080×680。
- 系统绘制窗口按钮。`TitleBarHidden`，标题栏高 40。不画假的交通灯。
- 四个标签：数据库、会话、记录、设置。标签名「数据库」是库，不是扩展名过滤。
- 有左右栏时，列表宽 264，贴边，中间一条发丝线。不留间隙，不画第二套圆角。窗口圆角由系统裁切。工具栏里的标签和添加按钮是 Liquid Glass 胶囊。
- 状态是方块标记，不是变色的字。通电是墨色方块。故障是 `#9B2C2C`。不用橙色，不用米色。
- 浅色固定。不提供一个还没接上的主题开关。

本阶段会话页和记录页可以显示真实会话和调用。不把库里的文件伪装成已打开的会话。库里的文件可以添加、选择、移出，并写入 `~/.jig/state.json`。

设置页有四组：IDA（目录、重新探测）、端口（`127.0.0.1:9401` / `9399`，标明尚未监听）、Agent、界面（中文、浅色）。没有插件重装，没有代理修复按钮。

## 后续阶段

1. 原生 worker 已接上。窗口里打开一个真实文件，确认会话页能看到函数列表和反编译。
2. 本进程提供 MCP HTTP 与 worker TCP。设置页的端口从「尚未监听」改为真实状态。
3. 把 Codex / Claude / OpenCode 的配置写到本进程端点。不写旧的 `ida-pro-mcp` 桥。
4. 用真实二进制验收打开、调用和关闭。然后删除 `mcp-center` 桌面、GUI 插件安装器和 IDA 窗口页。

## 本轮验收

- `go test ./...` 通过。
- 无扩展名、`.bin`、`.dex`、`.apk` 可以入库；目录和 `.id0` 不行；伴随 `.i64` 被优先作为打开路径。
- 测试窗口能从空库加文件、移出，并切到会话、记录、设置。设置页不出现插件重装或 IDA 窗口。
