<p align="center">
  <img src="assets/icon.svg" alt="Coder Logo" width="128" height="128" />
</p>

<h1 align="center">Coder Suite</h1>

<p align="center">
  终端优先的 AI 开发套件：单步 Diff 编辑器与自主编程智能体。
</p>

<p align="center">
  <a href="https://github.com/sokinpui/coder.nvim"><b>coder.nvim</b></a> (Neovim 插件) &nbsp;•&nbsp;
  <a href="https://github.com/sokinpui/coder.flutter"><b>coder.flutter</b></a> (跨平台 GUI 客户端)
</p>

<p align="center">
  <a href="README.md">English</a>
</p>

## Quick Install

一键安装 `coder` 与 `co`：

```bash
curl -fsSL https://raw.githubusercontent.com/sokinpui/coder/main/quickinstall.sh | bash
```

如需同时安装所有独立子工具（`itf`、`sf`、`pcat`、`pti`），可追加 `--all`：

```bash
curl -fsSL https://raw.githubusercontent.com/sokinpui/coder/main/quickinstall.sh | bash -s -- --all
```

## Install with Go

```bash
go install github.com/sokinpui/coder/cmd/coder@latest
go install github.com/sokinpui/coder/cmd/co@latest
```

## Local Build

```bash
git clone https://github.com/sokinpui/coder.git
cd coder
./install.sh
```

## Positioning: Coder

**`coder`** 是一个**单步 AI 代码编辑器**，专为希望对项目上下文和代码改动拥有绝对控制权的开发者设计。

- **人机协同（Human-in-the-loop）**：自主挑选并控制输入给模型的精确上下文文件。
- **统一 Diff 应用**：模型输出标准 Unified Diff 与文件生命周期操作，通过 `itf` 一键安全打补丁。
- **安全可逆**：每一次修改均可审查，并可通过 `/undo` 随时回滚撤销。

## Positioning: Co

**`co`** 是一个**自主编程智能体（Autonomous Coding Agent）**。它在交互式工具调用循环中运作，能够自主解决复杂工程任务。

- **自主工具执行**：自主读取文件、执行精确字符串替换、创建文件以及运行 Shell 命令。
- **编辑前必读守卫（Read-before-edit Guard）**：严禁模型在未通过 `read` 查看文件的情况下进行盲目文本替换。
- **视觉与 PDF 多模态上下文**：原生支持本地图片与 PDF 文档的多模态视觉输入。

## Starting Coder and Co

启动单步编辑器：

```bash
coder [files...]
```

> 详细使用手册、CLI 标志、快捷键与命令列表：[**docs/coder/README.zh.md**](docs/coder/README.zh.md)

启动自主编程智能体：

```bash
co [prompt]
```

> 详细智能体手册、工具架构与工作流：[**docs/coagent/README.zh.md**](docs/coagent/README.zh.md)

## Endpoint Configuration

Coder 支持连接任何兼容 OpenAI 协议的 API 端点。可通过环境变量设置 API 密钥：

```bash
export CODER_API_KEY="your-api-key"
```

在全局配置文件 `~/.config/coder/config.yaml` 或项目本地配置 `.coder/config.yaml` 中配置端点与模型：

```yaml
server:
  url: http://localhost:9001/v1
  protocol: responses # "responses" (/v1/responses) 或 "chat" (/v1/chat/completions)

title:
  modelcode: aisrp/gemini-flash-lite-latest

coder:
  modelcode: aisrp/gemini-flash-latest
  reasoningeffort: high
  context:
    dirs: ["."]

agent:
  modelcode: aisrp/gemini-flash-lite-latest
  reasoningeffort: high
  max_iterations: 50
```

## Coder Suite

- **[coder.nvim](https://github.com/sokinpui/coder.nvim)**：直接集成到 Neovim 编辑器缓冲区的插件。
- **[coder.flutter](https://github.com/sokinpui/coder.flutter)**：跨平台 GUI 客户端（桌面端、移动端、Web），通过 WebSocket 连接（`coder --ws`）。
- **[itf](./pkg/itf/README.md)**：Insert To File — 补丁解析与应用工具，具备事务级 undo/redo 撤销重做能力。
- **[sf](./pkg/sf/README.md)**：Search Fast — 遵循 `.gitignore` 的高并发并行目录检索器。
- **[pti](./pkg/pti/README.md)**：PDF To Image — 将 PDF 文档按需渲染为高质量图片供视觉模型理解。
- **[pcat](./pkg/pcat/README.md)**：Prompt Cat — 将文件拼接并格式化为带语法高亮的 Markdown 代码块。
