# aicommit

[English](README.md)

`aicommit` 通过 OpenAI 兼容 API，根据 Git 暂存区变更生成符合 [Conventional Commits](https://www.conventionalcommits.org/) 规范的提交信息。

## 安装

**macOS / Linux：**

```bash
curl -fsSL https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.sh | sh
```

安装脚本会写入 `~/.local/bin`，不会请求管理员权限。如果该目录尚未加入 `PATH`，脚本会打印对应 shell 的配置提示。需要使用其他可写的用户目录时，请将 `AICOMMIT_INSTALL_DIR` 传给 `sh`：

```bash
curl -fsSL https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.sh | AICOMMIT_INSTALL_DIR="$HOME/bin" sh
```

**Windows（PowerShell）：**

```powershell
irm https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.ps1 | iex
```

PowerShell 安装脚本会写入 `%LOCALAPPDATA%\aicommit\bin`，并将该目录加入用户级 `PATH`；不需要管理员权限。

**npm：**

```bash
npm i -g @wungjyan/aicommit
```

**Go：**

```bash
go install github.com/wungjyan/aicommit@latest
```

从源码构建：`go build -o aicommit .`。

## 卸载

通过 Shell 或 PowerShell 安装脚本安装的二进制，可使用下列命令删除，同时保留 API 配置：

```bash
aicommit uninstall
```

如需同时删除 `~/.aicommit` 及已保存的配置，请在提示中确认；脚本中可使用 `--yes`：

```bash
aicommit uninstall --purge
aicommit uninstall --purge --yes
```

通过 npm 安装时，请改用 `npm uninstall -g @wungjyan/aicommit`。

通过 Go 安装时，请根据 Go 配置的二进制目录删除：

```bash
go_bin="$(go env GOBIN)"
[ -n "$go_bin" ] || go_bin="$(go env GOPATH)/bin"
rm -f "$go_bin/aicommit"
```

## 快速开始

先配置一次后端，再使用正常的 Git 工作流：

```bash
# 首次使用时，配置一次 OpenAI 兼容 API。
aicommit config setup

# 仅暂存准备提交的改动。
git add .

# 生成提交信息后，可选择提交、编辑、重新生成或退出。
aicommit
```

默认命令会读取 `git diff --cached`，生成提交信息，并允许你提交、编辑、重新生成或退出。

| 按键 | 操作 |
| --- | --- |
| `Enter` | 提交生成的信息 |
| `e` | 在 `$EDITOR` 中编辑，然后重新校验 |
| `r` | 使用同一份暂存区 diff 重新生成 |
| `q` | 不提交并退出 |

工具只读取暂存区变更。调用前请先暂存需要提交的文件。

## AI 后端

`aicommit config setup` 将 API 设置和输出语言设置分开。当前只开放 OpenAI 兼容 API 后端，并会在保存前检查其连通性。

### OpenAI 兼容 API

在向导中选择 **OpenAI-compatible API**。它提供 OpenAI、DeepSeek、OpenRouter、百炼和自定义端点预设。自定义端点在需要时必须包含 API 版本路径，通常为 `/v1`。

此后端调用 `POST <base-url>/chat/completions`，支持实现了 OpenAI Chat Completions API 的服务。

### 实验性 CLI 集成（暂时隐藏）

Codex CLI 和 Claude Code CLI 的集成实现仍保留在代码中，供后续评估；但当前版本不会在配置流程中提供选择。它们的启动和生成耗时无法稳定满足“快速返回一条提交信息”的体验要求。

日常使用请配置 OpenAI 兼容 API。只有当 CLI 集成能提供稳定的快速体验后，才会重新开放。

## 配置

配置文件位于 `~/.aicommit/config.json`。可通过以下命令查看最终生效值和来源：

```bash
aicommit config
aicommit config --json
aicommit config path
```

`config` 会显示每项值的来源：`environment`、`config` 或 `default`。API Key 在文本和 JSON 输出中都会被掩码。

### 环境变量

非空环境变量的优先级高于已保存的配置。

| 变量 | 适用后端 | 说明 |
| --- | --- | --- |
| `AICOMMIT_LANGUAGE` | 全部 | 生成提交信息所使用的语言 |
| `OPENAI_API_KEY` | OpenAI 兼容 API | API Key |
| `OPENAI_BASE_URL` | OpenAI 兼容 API | 接口地址，例如 `https://api.deepseek.com/v1` |
| `OPENAI_MODEL` | OpenAI 兼容 API | 模型名称，例如 `deepseek-chat` |

OpenAI 默认地址是 `https://api.openai.com/v1`，默认模型是 `gpt-4o-mini`。旧配置文件缺少 `backend` 字段时，会按 `openai` 处理。

### 非交互修改

脚本或精确修改配置时使用 `config set`。未指定的字段会保留原值。

```bash
aicommit config set --api-key "$OPENAI_API_KEY"
aicommit config set --base-url https://api.deepseek.com/v1 --model deepseek-chat
aicommit config set --language 中文
```

增加 `--check` 可在保存前检查 OpenAI 兼容 API 连通性。检查失败不会覆盖已有配置。

## 命令参考

| 命令 | 说明 |
| --- | --- |
| `aicommit` | 交互式生成并选择是否提交 |
| `aicommit --dry-run` | 生成并输出提交信息，不询问、不提交 |
| `aicommit --yes`、`aicommit -y` | 生成、校验并直接提交，不询问 |
| `aicommit --edit`、`aicommit -e` | 生成后立即打开编辑器，然后进入交互流程 |
| `aicommit --yes --edit` | 编辑、校验后直接提交，不显示确认提示 |
| `aicommit --no-color` | 禁用 ANSI 颜色输出 |
| `aicommit config` | 显示最终生效配置及来源 |
| `aicommit config setup` | 运行交互式配置向导 |
| `aicommit config set [flags]` | 持久化修改一个或多个配置字段 |
| `aicommit config check` | 检查 API 连通性，不修改配置 |
| `aicommit config path` | 输出配置文件绝对路径 |
| `aicommit -v`、`aicommit --version`、`aicommit version` | 输出版本、提交和构建时间 |
| `aicommit uninstall` | 删除已安装二进制，保留配置 |
| `aicommit uninstall --purge` | 同时删除已保存配置（会请求确认） |

所有命令都拒绝额外位置参数。`--dry-run` 不能与 `--yes` 或 `--edit` 一起使用。`--edit` 要求 stdin、stdout 和 stderr 都是终端。非交互环境必须使用 `--dry-run` 或 `--yes`。

## 自动化与 CI

需要让其他命令读取提交信息时使用 `--dry-run`：

```bash
message="$(aicommit --dry-run)"
printf '%s\n' "$message"
```

在受控 CI 任务中使用 `--yes` 自动提交：

```bash
export OPENAI_API_KEY="$CI_OPENAI_API_KEY"
export OPENAI_MODEL=gpt-4o-mini

git add --all
aicommit --yes
```

`--yes` 和 `--dry-run` 都不会读取确认输入。

## 输出与退出码

数据写入 stdout，方便脚本稳定消费；进度、确认提示、警告和错误写入 stderr。

| stdout 数据 | stderr 诊断信息 |
| --- | --- |
| `--dry-run` 生成的信息 | spinner 和状态信息 |
| `config` 和 `config --json` | 配置向导和确认提示 |
| `config path` | 警告和错误 |
| `version` | 连接检查状态 |

| 退出码 | 含义 |
| --- | --- |
| `0` | 成功，包括交互时主动退出 |
| `1` | 通用运行错误，例如 Git、配置、编辑器或提交失败 |
| `2` | 命令用法或 flag 组合错误 |
| `3` | AI 配置、鉴权、连接或 Provider 响应失败 |
| `4` | 自动模式下生成的信息格式不合法 |

当 `--dry-run` 生成非法信息时，stdout 仍会原样输出生成结果，stderr 会显示校验错误，进程退出码为 `4`。

## 安全与数据处理

暂存区 diff 会发送给配置的 OpenAI 兼容 API。使用前请确认暂存内容可以交给对应后端处理。

API Key 以受限文件权限保存，并且在所有配置展示中被掩码。

## 破坏性变更

旧命令 `aicommit ai` 和 `aicommit ai --setup` 已移除。请改用 `aicommit config` 和 `aicommit config setup`。Codex CLI 和 Claude Code CLI 集成也会在评估交互延迟期间暂时隐藏。

## 许可证

MIT
