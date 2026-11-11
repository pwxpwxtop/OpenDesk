# OpenDesk

用「Windows 桌面」的方式管理远程 Linux 主机：通过 SSH 连接后，每个会话会打开一个独立的远程桌面窗口，桌面上有壁纸、可拖动的图标、任务栏、右键菜单，以及控制台、文件管理、系统监控、文件预览等应用。

技术栈：Wails v3 (alpha) + Go + Vue 3 + Vite + PrimeVue + xterm.js。

## 功能

### 连接管理（主窗口）

- 左侧菜单：服务器 / 活动会话
- 右侧服务器卡片网格，支持新增、编辑、删除、搜索
- 认证方式：密码或 SSH 私钥（含带 passphrase 的私钥），连接信息保存在本机
- 点击卡片即连接，成功后自动打开该会话的远程桌面窗口
- 关闭某个会话的最后一个窗口时，SSH 连接自动断开

### 远程桌面窗口

- 壁纸：15 套内置渐变配色 + 可上传本地图片
- 桌面图标：控制台、文件管理、系统监控、壁纸；上传的文件/文件夹也会以图标形式出现在桌面
- 图标可拖动并自动吸附网格，位置与上传文件列表按会话持久化；右键「整理桌面图标」可一键复位
- 应用窗口：拖动、八向缩放、最大化、最小化、关闭，任务栏切换
- 任务栏：开始菜单、已打开窗口、连接状态、时钟
- 桌面右键：打开应用、更换壁纸、上传/粘贴本地文件、整理图标、关闭全部窗口、断开连接

### 控制台

- xterm.js 终端，基于 SSH PTY，支持窗口缩放、颜色、`Ctrl+C` 等常用控制
- 连接断开时给出提示

### 文件管理

- SFTP 浏览、面包屑路径、点击路径可编辑（`Ctrl+L`）
- 新建文件夹、重命名、删除（多选批量）、上传、下载
- 目录/文件名筛选、隐藏文件开关、按名称/大小/时间排序
- 键盘快捷键：

| 快捷键 | 作用 |
| --- | --- |
| `Enter` | 打开目录 / 预览文件 |
| `Backspace` / `Alt+←` | 返回上级目录 |
| `F5` / `Ctrl+R` | 刷新 |
| `Ctrl+L` | 编辑路径 |
| `↑` `↓` / `Shift+↑↓` | 上下选择 / 连续多选 |
| `Ctrl+A` | 全选 |
| `F2` | 重命名 |
| `Delete` | 删除 |
| `Ctrl+N` | 新建文件夹 |
| `Ctrl+U` | 上传文件 |
| `Ctrl+V` | 粘贴本地文件 |
| `Esc` | 取消选择 / 关闭菜单 / 清空筛选 |

- 上传时窗口底部显示进度条（文件名、百分比、已传/总大小、目标目录、完成数量）

### 文件传输

三种方式，都会写入远程目录并可在桌面看到图标：

1. **拖拽**：从资源管理器把文件/文件夹拖进桌面窗口，落到远程 `~/Desktop`
2. **选择框**：桌面右键「上传本地文件…」或文件管理「上传」，支持多选与整个目录
3. **剪贴板**：资源管理器 `Ctrl+C` 后在桌面按 `Ctrl+V`（写入 `~/Desktop`），或在文件管理器按 `Ctrl+V`（写入当前目录）
   - 实现方式：直接读取 Windows 剪贴板的 `CF_HDROP` 文件列表，因此不受浏览器粘贴限制

文件夹会递归镜像到远端，单个文件失败不影响其它文件。

### 文件预览

在文件管理中双击文件（或选中后按 `Enter`、右键「预览」），或在桌面上双击文件图标：

| 类型 | 行为 |
| --- | --- |
| 视频 mp4/webm/ogv/mov/m4v/mkv/avi | 内置播放器播放，可拖动进度（HTTP Range 流式读取） |
| 音频 mp3/wav/ogg/flac/m4a/aac | 内置音频播放器 |
| 图片 png/jpg/gif/svg/webp/bmp/ico/avif | 居中显示，自适应缩放 |
| PDF | 内嵌 PDF 阅读器 |
| 文本/代码 txt/md/log/json/yml/go/py/js… | 只读预览，**超过 1MB 不支持查看** |
| 其它 | 提示不支持在线查看，可下载 |

### 系统监控

- 主机名、内核、运行时长、负载、CPU 使用率与核心数、内存/交换分区、磁盘占用、进程列表
- 数据通过 SSH 执行系统命令采集，定时刷新

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 18+
- Wails v3 CLI（`wails3`）

### 开发

```bash
wails3 dev
```

或分开运行前端与后端：

```bash
cd frontend && npm install && npm run dev
```

### 构建

```bash
wails3 build
```

生成可执行文件（当前模块名为 `changeme`，产物即 `changeme.exe`）。Windows 图标资源由 `wails3 generate syso` 生成到项目根目录，普通 `go build .` 也会带上图标。

### 重新生成前端绑定

Go 服务方法/模型变更后需要重新生成：

```bash
wails3 generate bindings -ts
```

生成目录：`frontend/bindings/`。

## 项目结构

```
.
├── main.go                  # 应用入口：服务注册、窗口创建、事件注册、资源处理器
├── sshservice.go            # SSH/SFTP/监控/上传/预览的后端服务
├── preview.go               # /preview 流式预览路由 + 文本读取（1MB 限制）
├── clipboard_windows.go     # 读取 Windows 剪贴板文件列表（Ctrl+V）
├── clipboard_other.go       # 非 Windows 平台的占位实现
├── build/                   # 图标、平台打包配置与 Taskfile
└── frontend/
    ├── bindings/            # Wails 生成的 TypeScript/JavaScript 绑定
    └── src/
        ├── views/
        │   ├── ServersView.vue        # 主窗口：服务器卡片 + 活动会话
        │   └── DesktopSessionView.vue # 远程桌面窗口
        ├── components/
        │   ├── desktop/               # 窗口框架、桌面图标、任务栏、右键菜单
        │   └── apps/                  # 控制台、文件管理、系统监控、壁纸、预览
        ├── composables/useFileManager.js  # 文件管理状态与操作
        ├── stores/                    # connections / sessions / desktop / transfers
        ├── services/ssh.js            # 绑定封装 + 事件总线
        ├── config/                    # 应用元数据、壁纸配色
        └── styles/app.css             # 全局设计变量与桌面样式
```

## 实现要点

- **多窗口**：`OpenSessionWindow` 通过 `application.Get().Window.NewWithOptions` 为每个会话创建原生窗口，URL 为 `/#/desktop/<sessionId>`。
- **事件广播**：`ssh:output`、`ssh:closed`、`ssh:upload`、`ssh:upload:summary` 由后端 emit，所有窗口均可订阅；上传进度据此汇总到 `transfers` store。
- **视频流式播放**：`preview.go` 用 SFTP 打开远程文件并交给 `http.ServeContent`，天然支持 `Range`，无需先完整下载。
- **桌面状态**：每个原生窗口是独立渲染进程，`desktop` store 只管理本窗口的窗口栈与图标；图标布局、壁纸、上传文件列表存 `localStorage`。
- **拖放**：窗口开启 `EnableFileDrop`，用 `data-file-drop-target` / `data-drop-dir` 标注投放区域，Go 侧在 `WindowFilesDropped` 中读取投放目录。

## 本地数据与安全

- 服务器配置（含密码）保存在 `localStorage`，**密码为明文**，仅适合本机个人使用
- SSH 目前使用 `ssh.InsecureIgnoreHostKey()`，不校验 known_hosts
- 壁纸、图标布局、上传文件图标同样保存在 `localStorage`

## 已知限制

- 文本类预览上限 1MB，超出需下载
- 剪贴板粘贴仅在 Windows 有效
- 视频能否播放取决于 WebView2 支持的编码（H.264/VP9 通常可用，部分容器如 mkv 可能无法播放）
- 尚不支持在应用内复制远程文件回 Windows 资源管理器（可用「下载」）
- 拖拽上传的落地目录固定为 `~/Desktop`，文件管理内拖放则落到当前目录
