# Coder - 用户手册

[English](README.md)

Coder 是一款面向终端工作流、以人为主导的交互式 TUI AI 代码编辑器。它基于确定性的单步编辑模式：由你提供精确的文件上下文，AI 给出标准 Unified Diff 及文件生命周期变更建议，并通过 `itf` 工具实现一键审查与打补丁。

## CLI 用法

```bash
coder [files...]          # 启动交互式 TUI，将指定文件载入上下文
coder -c                  # 以纯对话模式启动交互式 TUI（不加载文件上下文）
coder -p "prompt" [files] # 携带初始提示词启动会话
coder -e -p "prompt"      # 非交互式单次执行，结果直接输出到标准输出 (stdout)
coder -C [files...]       # 打印构建好的 Prompt 和上下文到 stdout（便于调试）
coder --config            # 在 $EDITOR 中编辑本地 (.coder/config.yaml) 或全局配置
coder --config -g         # 在 $EDITOR 中编辑全局配置 (~/.config/coder/config.yaml)
coder --headless          # 作为 Headless JSON-RPC 服务运行（基于标准输入输出）
coder --port 9005         # 在指定 TCP 端口运行 Headless JSON-RPC 服务
coder --ws --port 9005    # 运行支持 WebSocket/HTTP 的 Headless 服务
```

## 核心工作流

1. **选择上下文**：在命令行启动时指定文件（如 `coder main.go internal/`），或在 TUI 中使用 `/file <路径>` 或 `@<路径>` 动态添加。
2. **审查指示**：Coder 会加载对应文件和目录并遵循排除规则。PDF 文档会自动转为视觉多模态上下文。
3. **生成变更**：模型以 Unified Diff、新建文件、删除或重命名代码块的形式输出变更方案。
4. **应用补丁**：按 `Ctrl+A` 或运行 `/itf` 自动打补丁到本地文件系统中。如需撤销，运行 `/undo` 即可安全回滚。

## 快捷键

| 快捷键         | 操作说明                                                        |
| :------------- | :-------------------------------------------------------------- |
| `Ctrl+J`       | 发送消息 / 提交命令                                             |
| `Ctrl+E`       | 在外部编辑器（`$EDITOR`）中编辑当前提示词                       |
| `Ctrl+V`       | 从系统剪贴板粘贴（支持纯文本和图片）                           |
| `Ctrl+A`       | 应用最近一条 AI 回复中的代码变更（通过 `itf`）                  |
| `Ctrl+H`       | 打开历史对话选择器                                             |
| `Ctrl+N`       | 开启新对话会话                                                 |
| `Ctrl+B`       | 打开原子消息（Atomic Messages）遮罩层以进行分支会话创建         |
| `Ctrl+F`       | 检索当前上下文文件并在外部编辑器中打开                         |
| `Ctrl+T`       | 检索项目文件/目录并添加到上下文（等同于 `/file`）               |
| `Ctrl+L`       | 快速查看当前上下文摘要信息（等同于 `/list`）                    |
| `Ctrl+U` / `D` | 向上 / 向下半页滚动对话内容                                     |
| `Ctrl+Z`       | 挂起应用切至后台终端                                           |
| `Esc`          | 打开 **原子消息（Atomic Messages）** 遮罩层                     |
| `Ctrl+C`       | 清空当前输入行（空行连续按两次退出应用）                       |
| `Tab`          | 自动补全命令与路径参数                                         |

## 斜杠命令

- `/file [paths...]`（别名：`@`）：将文件或目录追加到当前活动上下文中。
- `/exclude [paths...]`：从当前活动上下文中剔除指定路径。
- `/list`：显示当前上下文中包含的项目文件及 PDF 文档列表。
- `/clear_context`：清空上下文中所有的文件和文档。
- `/itf [args]`：将最近一条 AI 响应中的代码变更通过 `itf` 应用到文件。
- `/undo`：撤销最近一次由 `itf` 实施的文件变更。
- `/chat`：将当前会话模式切换为普通聊天模式。
- `/coding`：将当前会话模式切换为编码模式。
- `/model [name]`：在线切换生成模型，或列出可用模型列表。
- `/new`：重置当前对话并开启新会话，保留当前配置。
- `/history`：浏览并加载已保存的会话历史记录。
- `/active`：查看并快速切换当前内存中的活动会话。
- `/rename [title]`：重命名当前会话标题。
- `/shell [cmd]`（别名：`!`）：运行交互式终端命令或启动子 Shell。
- `/config [reload]`：查看当前生效配置，或从磁盘重新加载配置。
- `/help`：展示帮助界面。
- `/quit`（别名：`/q`）：退出程序。

## 原子消息遮罩层 (`Esc`)

按 `Esc` 键可进入原子消息管理层，针对对话栈中的单条消息进行精确操作：

- `j` / `k`：在消息列表之间上下移动光标。
- `v`：进入/退出多选模式。
- `o` / `O`：在多选状态下交换光标端与锚点端。
- `y`：将选中的消息文本或图片复制至系统剪贴板。
- `d`：删除选中的单条或多条消息。
- `a`：应用该条或其上方最近一条 AI 响应中的代码变更（通过 `itf`）。
- `e`：在外部编辑器（`$EDITOR`）中编辑选中的用户提示词。
- `r`：以选中的提示词为起点重新生成回复。
- `b`：以选中消息为切入点分支生成新会话。
- `Esc` / `Ctrl+C`：退出遮罩层回到对话。

## 配置说明

配置文件位于全局 `~/.config/coder/config.yaml` 或项目根目录 `.coder/config.yaml`：

```yaml
coder:
  modelcode: aisrp/gemini-flash-latest
  reasoningeffort: high
  context:
    files: []
    dirs: ["."]
    exclusions: []
  keymap:
    submit: ctrl+j
    editor: ctrl+e
    applyitf: ctrl+a
```
