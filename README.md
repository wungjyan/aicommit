# aicommit

[中文](README.zh-CN.md)

`aicommit` generates [Conventional Commits](https://www.conventionalcommits.org/) messages from staged Git changes through an OpenAI-compatible API.

## Install

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.sh | sh
```

The installer writes to `~/.local/bin` and never requests administrator access. If that directory is not in your `PATH`, it prints the command to add it for your shell. To use another user-writable directory, set `AICOMMIT_INSTALL_DIR` on `sh`:

```bash
curl -fsSL https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.sh | AICOMMIT_INSTALL_DIR="$HOME/bin" sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.ps1 | iex
```

The PowerShell installer writes to `%LOCALAPPDATA%\aicommit\bin` and adds that directory to the user-level `PATH`; it does not require administrator access.

**npm:**

```bash
npm i -g @wungjyan/aicommit
```

**Go:**

```bash
go install github.com/wungjyan/aicommit@latest
```

Build from source with `go build -o aicommit .`.

## Uninstall

For a binary installed by the shell or PowerShell installer, remove the binary while keeping your API configuration:

```bash
aicommit uninstall
```

To also remove `~/.aicommit` and its saved configuration, confirm the prompt or use `--yes` in a script:

```bash
aicommit uninstall --purge
aicommit uninstall --purge --yes
```

For an npm installation, use `npm uninstall -g @wungjyan/aicommit` instead.

For a Go installation, use Go's configured binary directory:

```bash
go_bin="$(go env GOBIN)"
[ -n "$go_bin" ] || go_bin="$(go env GOPATH)/bin"
rm -f "$go_bin/aicommit"
```

## Quick Start

Configure a backend once, then use the normal Git workflow:

```bash
# Configure the OpenAI-compatible API once.
aicommit config setup

# Stage only the changes intended for this commit.
git add .

# Generate a message, then choose to commit, edit, regenerate, or quit.
aicommit
```

The default command reads `git diff --cached`, generates a message, and lets you commit, edit, regenerate, or quit.

| Key | Action |
| --- | --- |
| `Enter` | Commit the generated message |
| `e` | Edit it in `$EDITOR`, then validate again |
| `r` | Regenerate using the same staged diff |
| `q` | Quit without committing |

Only staged changes are used. Stage the intended files before invoking `aicommit`.

## AI Backends

`aicommit config setup` offers API settings and a separate output-language group. The OpenAI-compatible API is the only backend currently exposed, and its connectivity is checked before saving.

### OpenAI-Compatible API

Choose **OpenAI-compatible API** in the setup wizard. Presets are available for OpenAI, DeepSeek, OpenRouter, Bailian, and a custom endpoint. A custom endpoint must include its API version path when required, usually `/v1`.

The backend calls `POST <base-url>/chat/completions`. It works with services that implement the OpenAI Chat Completions API.

### Experimental CLI Integrations (Temporarily Hidden)

Codex CLI and Claude Code CLI integrations remain in the codebase for future evaluation, but they are intentionally not selectable in the current release. Their startup and generation latency is not reliable enough for a command that should return a commit message quickly.

Use an OpenAI-compatible API for all normal usage. The CLI integrations will be reconsidered only when they can provide a consistently fast experience.

## Configuration

Configuration is stored at `~/.aicommit/config.json`. Inspect the final effective values and their sources with:

```bash
aicommit config
aicommit config --json
aicommit config path
```

`config` shows each value's source: `environment`, `config`, or `default`. API keys are always masked, including JSON output.

### Environment Variables

Non-empty environment variables override the saved configuration.

| Variable | Applies to | Description |
| --- | --- | --- |
| `AICOMMIT_LANGUAGE` | All | Generated commit-message language |
| `OPENAI_API_KEY` | OpenAI-compatible API | API key |
| `OPENAI_BASE_URL` | OpenAI-compatible API | Base URL, such as `https://api.deepseek.com/v1` |
| `OPENAI_MODEL` | OpenAI-compatible API | Model name, such as `deepseek-chat` |

The default OpenAI base URL is `https://api.openai.com/v1`; the default model is `gpt-4o-mini`. Existing config files without a `backend` field are treated as `openai`.

### Non-Interactive Updates

Use `config set` for scripts or precise field changes. It preserves fields that are not supplied.

```bash
aicommit config set --api-key "$OPENAI_API_KEY"
aicommit config set --base-url https://api.deepseek.com/v1 --model deepseek-chat
aicommit config set --language 中文
```

Add `--check` before saving to verify OpenAI-compatible API connectivity. A failed check does not overwrite the existing configuration.

## Commands

| Command | Description |
| --- | --- |
| `aicommit` | Generate interactively and optionally commit |
| `aicommit --dry-run` | Print a generated message without prompting or committing |
| `aicommit --yes`, `aicommit -y` | Generate, validate, and commit without prompting |
| `aicommit --edit`, `aicommit -e` | Open the editor immediately after generation, then continue interactively |
| `aicommit --yes --edit` | Edit, validate, and commit without a confirmation prompt |
| `aicommit --no-color` | Disable ANSI color output |
| `aicommit config` | Show effective configuration and sources |
| `aicommit config setup` | Run the interactive setup wizard |
| `aicommit config set [flags]` | Persist one or more configuration fields |
| `aicommit config check` | Check API connectivity without changing config |
| `aicommit config path` | Print the absolute configuration path |
| `aicommit -v`, `aicommit --version`, `aicommit version` | Print version, commit, and build time |
| `aicommit uninstall` | Remove the installed binary and keep configuration |
| `aicommit uninstall --purge` | Also remove saved configuration (asks for confirmation) |

All commands reject unexpected positional arguments. `--dry-run` cannot be combined with `--yes` or `--edit`. `--edit` requires stdin, stdout, and stderr to be terminals. In a non-interactive environment, use `--dry-run` or `--yes`.

## Automation and CI

Use `--dry-run` when another command consumes the message:

```bash
message="$(aicommit --dry-run)"
printf '%s\n' "$message"
```

Use `--yes` to commit in a controlled CI job:

```bash
export OPENAI_API_KEY="$CI_OPENAI_API_KEY"
export OPENAI_MODEL=gpt-4o-mini

git add --all
aicommit --yes
```

`--yes` and `--dry-run` never read confirmation input.

## Output and Exit Codes

Data is written to stdout so scripts can consume it reliably. Progress, confirmation prompts, warnings, and errors are written to stderr.

| stdout data | stderr diagnostics |
| --- | --- |
| `--dry-run` message | spinner and status messages |
| `config` and `config --json` | setup and confirmation prompts |
| `config path` | warnings and errors |
| `version` | connection-check status |

| Code | Meaning |
| --- | --- |
| `0` | Success, including an interactive quit |
| `1` | General runtime error, such as Git, config, editor, or commit failure |
| `2` | Invalid command usage or flag combination |
| `3` | AI configuration, authentication, connection, or provider response failure |
| `4` | Generated message is invalid in an automatic mode |

For an invalid `--dry-run` result, stdout still contains the unmodified generated message, while stderr contains the validation error and the command exits `4`.

## Security and Data Handling

The staged diff is sent to the configured OpenAI-compatible API. Review staged changes before using a backend with data you should not share.

API keys are stored with restricted file permissions and masked in all displayed configuration.

## Breaking Change

The legacy `aicommit ai` and `aicommit ai --setup` commands have been removed. Use `aicommit config` and `aicommit config setup` instead. Codex CLI and Claude Code CLI integrations are also temporarily hidden while their interactive latency is evaluated.

## License

MIT
