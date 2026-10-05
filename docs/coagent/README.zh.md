# Co - 自主编程智能体

[English](README.md)

`co` 是一款基于大语言模型自主工具调用循环的终端智能体。与依赖人工审查 Unified Diff 的 `coder` 不同，`co` 能够自主探索项目工作区、读取文件、执行终端命令，并在推理过程中自主编辑源代码。

## CLI 用法

```bash
co                         # 在当前目录启动交互式智能体 TUI
co "修复测试用例失败"     # 传入初始任务直接启动智能体
co -p "重构生成器接口"     # 使用显式 -p 参数传入初始提示词
co -m gpt-4o               # 覆盖当前智能体的生成模型
co -i "自定义开发规范"     # 覆盖智能体系统指令
co -P responses            # 设定 API 协议（"responses" 或 "chat"）
co --completion bash       # 生成 Shell 自动补全脚本
```

## 工具生态与安全保护守卫

`co` 在最大迭代轮数（默认：50 轮）内自主执行工具循环，内置以下核心工具：

- **`read`**：从本地文件系统读取内容。
  - 纯文本与源代码：返回原始文本。
  - 图片（`.png`, `.jpg`, `.jpeg`, `.webp` 等）：作为多模态视觉上下文载入。
  - PDF 文档（`.pdf`）：按需光栅化渲染为图片消息载入视觉上下文。
- **`edit`**：基于精准块匹配替换文件内容（`old_string` -> `new_string`）。
  - **前置安全守卫（Guard Clause）**：严禁修改在当前会话中尚未被 `read` 读取过的文件。
  - 要求 `old_string` 在文件中具有唯一性，杜绝歧义覆盖。
- **`write`**：创建新文件或完全覆写现有文件，自动创建不存在的父级目录。
- **`bash`**：在宿主系统中执行 Shell 命令，默认超时时间 60 秒，自动捕获并截断标准输出与错误流。

## 直接注入文件上下文

在 `co` 的终端交互界面中，你可以通过 `@` 或 `/file` 命令直接将文件喂给智能体：

```text
@pkg/itf/patcher.go 解释这里的补丁匹配算法
/file internal/config/config.go
```

该操作会立刻触发 `read` 工具，并将解析后的文件内容直接插入到当前对话的消息流中。

## 快捷键

| 快捷键         | 操作说明                                                        |
| :------------- | :-------------------------------------------------------------- |
| `Ctrl+J`       | 发送消息 / 提交命令                                             |
| `Ctrl+E`       | 在外部编辑器（`$EDITOR`）中编辑当前提示词                       |
| `Ctrl+V`       | 从剪贴板粘贴文本或图片                                         |
| `Ctrl+T`       | 切换工具调用显示模式（**紧凑模式** vs. **展开详情**）          |
| `Ctrl+H`       | 打开历史对话选择器                                             |
| `Ctrl+N`       | 开启新智能体会话                                               |
| `Ctrl+B`       | 打开原子消息（Atomic Messages）遮罩层以创建分支会话             |
| `Ctrl+U` / `D` | 向上 / 向下半页滚动对话内容                                     |
| `Ctrl+Z`       | 挂起应用切至后台终端                                           |
| `Esc`          | 打开 **原子消息（Atomic Messages）** 遮罩层                     |
| `Ctrl+C`       | 取消当前执行中的智能体工具循环 / 清空输入行                    |
| `Tab`          | 自动补全命令与文件参数                                         |

## 配置说明

在 `config.yaml` 的 `agent:` 节点下配置智能体行为：

```yaml
agent:
  modelcode: aisrp/gemini-flash-lite-latest
  reasoningeffort: high
  max_iterations: 50
  keymap:
    submit: ctrl+j
    editor: ctrl+e
```
