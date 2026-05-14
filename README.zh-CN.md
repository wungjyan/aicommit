# aicommit

[English](README.md)

基于 AI 的 Git commit message 生成工具。读取暂存区的 diff，自动生成符合 [Conventional Commits](https://www.conventionalcommits.org/) 规范的提交信息。

## 功能特性

- 自动生成符合 Conventional Commits 规范的提交信息
- 基于 OpenAI Chat Completions API，兼容 OpenAI、DeepSeek、OpenRouter、百炼等所有实现了该接口的服务
- 交互式配置向导，内置主流 Provider 预设
- 提交前可编辑、重新生成或放弃生成结果
- 自动校验格式，编辑后二次验证，防止非法消息入库
- 环境变量覆盖，适配 CI 和高级用户

## 安装

**macOS / Linux（推荐）：**

```bash
curl -fsSL https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.sh | sh
```

**Windows（PowerShell）：**

```powershell
irm https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.ps1 | iex
```

**npm：**

```bash
npm i -g @wungjyan/aicommit
```

**go install：**

```bash
go install github.com/wungjyan/aicommit@latest
```

> **注意：** 请只选择一种安装方式，多种方式同时安装可能导致 PATH 冲突。

**从源码构建：**

```bash
git clone https://github.com/wungjyan/aicommit.git
cd aicommit
go build -o aicommit .
```

## 快速开始

**第一步：配置 AI 服务**

```bash
aicommit ai --setup
```

向导会列出内置预设（OpenAI、DeepSeek、OpenRouter、百炼），Base URL 和 Model 已自动填好，验证连接成功后保存配置。

**第二步：生成并提交**

```bash
git add .
aicommit
```

工具读取暂存区 diff，发送给 AI，展示生成的提交信息。你可以：

| 按键 | 操作 |
|------|------|
| `Enter` | 直接提交 |
| `e` | 在 `$EDITOR` 中编辑 |
| `r` | 重新生成 |
| `q` | 放弃，不提交 |

## 配置说明

配置文件路径：`~/.aicommit/config.json`，查看当前配置：

```bash
aicommit ai
```

### 环境变量

环境变量优先级高于配置文件：

| 变量名 | 说明 |
|--------|------|
| `OPENAI_API_KEY` | API Key |
| `OPENAI_BASE_URL` | 接口地址（如 `https://api.deepseek.com/v1`） |
| `OPENAI_MODEL` | 模型名称（如 `deepseek-chat`） |

适合在 CI 中使用，或临时切换配置而不想修改文件。

## API 兼容性

aicommit 使用 **OpenAI Chat Completions API**（`POST /chat/completions`）与 AI 服务通信。任何实现了该接口的 Provider 均可使用，只需填入对应的 Base URL 和模型名称即可。

> 不兼容 Anthropic、Google Gemini 等非 OpenAI 格式的 API。如果你的服务商提供了 Chat Completions 兼容接口，则可以正常使用。

## 支持的 Provider

配置向导内置以下预设：

| Provider | Base URL | 默认模型 |
|----------|----------|----------|
| OpenAI | `https://api.openai.com/v1` | `gpt-4o-mini` |
| DeepSeek | `https://api.deepseek.com/v1` | `deepseek-chat` |
| OpenRouter | `https://openrouter.ai/api/v1` | `openai/gpt-4o-mini` |
| 百炼 | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `qwen3.5-plus` |
| 自定义 | 手动输入 | 手动输入 |

任何实现了 OpenAI 兼容接口的服务均可使用。

## 许可证

MIT
