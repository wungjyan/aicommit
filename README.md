# aicommit

[中文](README.zh-CN.md)

AI-powered Git commit message generator. Reads your staged changes and uses an AI model to generate a [Conventional Commits](https://www.conventionalcommits.org/) message.

## Features

- Generates commit messages following Conventional Commits specification
- Uses the OpenAI Chat Completions API — compatible with OpenAI, DeepSeek, OpenRouter, Bailian, and any provider that implements this interface
- Interactive setup wizard with built-in provider presets
- Edit, regenerate, or reject generated messages before committing
- Validates message format and re-checks after manual edits
- Environment variable overrides for CI and power users

## Install

**macOS / Linux (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.ps1 | iex
```

**npm:**

```bash
npm i -g @wungjyan/aicommit
```

**go install:**

```bash
go install github.com/wungjyan/aicommit@latest
```

> **Note:** Please use only one installation method. Multiple installations may cause PATH conflicts.

**Build from source:**

```bash
git clone https://github.com/wungjyan/aicommit.git
cd aicommit
go build -o aicommit .
```

## Quick Start

**1. Set up your AI provider:**

```bash
aicommit ai --setup
```

The wizard shows built-in presets (OpenAI, DeepSeek, OpenRouter, Bailian) with correct URLs and models pre-filled, then verifies the connection before saving.

**2. Generate a commit message:**

```bash
git add .
aicommit
```

The tool reads your staged diff, sends it to the AI, and shows the generated message. Choose:

| Key | Action |
|-----|--------|
| `Enter` | Commit with the generated message |
| `e` | Edit the message in your `$EDITOR` |
| `r` | Regenerate a new message |
| `q` | Quit without committing |

## Configuration

Config is stored at `~/.aicommit/config.json`. You can view it with:

```bash
aicommit ai
```

### Environment Variables

Environment variables take priority over the config file:

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | API key |
| `OPENAI_BASE_URL` | Base URL (e.g. `https://api.deepseek.com/v1`) |
| `OPENAI_MODEL` | Model name (e.g. `deepseek-chat`) |
| `AICOMMIT_LANGUAGE` | Commit message language (e.g. `中文`, `English`) |

This is useful for CI or when you want to override settings without changing the config file.

### Language

By default, commit messages are generated in English. You can change the output language in the setup wizard:

```bash
aicommit ai --setup
# Select "Commit output language" → choose English, 中文, or enter a custom language
```

Any language is supported — just type its name (e.g. `日本語`, `Français`).

## API Compatibility

aicommit communicates with AI services using the **OpenAI Chat Completions API** (`POST /chat/completions`). Any provider that implements this interface is supported — you just need the correct base URL and model name.

> Anthropic, Google Gemini, and other non-OpenAI-compatible APIs are **not** supported directly. If your provider exposes a Chat Completions-compatible endpoint, it will work.

## Supported Providers

The setup wizard includes presets for:

- **OpenAI** — `https://api.openai.com/v1` / `gpt-4o-mini`
- **DeepSeek** — `https://api.deepseek.com/v1` / `deepseek-chat`
- **OpenRouter** — `https://openrouter.ai/api/v1` / `openai/gpt-4o-mini`
- **Bailian** — `https://dashscope.aliyuncs.com/compatible-mode/v1` / `qwen3.5-plus`
- **Custom** — any OpenAI-compatible endpoint

## License

MIT
