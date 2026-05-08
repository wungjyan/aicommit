# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o aicommit .

# Build with version info
go build -ldflags "-X github.com/wungjyan/aicommit/cmd.version=x.y.z -X github.com/wungjyan/aicommit/cmd.commit=$(git rev-parse --short HEAD) -X github.com/wungjyan/aicommit/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o aicommit .

# Run all tests
go test ./...

# Run a single test
go test ./internal/git/ -run TestGetStagedDiff -v

# Install locally
go install .
```

## Architecture

The main flow is: `cmd/root.go:run()` → git → AI → prompt → git commit.

```
main.go
  └── cmd/
        ├── root.go     — root command; orchestrates the full flow
        └── ai.go       — `aicommit ai [--setup]` subcommand for config management

internal/
  ├── ai/
  │     ├── provider.go — Provider interface (Generate)
  │     └── openai.go   — OpenAIProvider: HTTP client for OpenAI-compatible APIs
  │                       Ping() sends a minimal /chat/completions request for verification
  ├── config/
  │     └── config.go   — reads/writes ~/.aicommit/config.json
  ├── git/
  │     ├── diff.go     — IsGitRepo, GetStagedDiff, TruncateDiff (80KB default cap)
  │     └── commit.go   — Commit(message)
  ├── prompt/
  │     └── prompt.go   — ValidateMessage (Conventional Commits regex),
  │                       Confirm (interactive: Enter/e/r/q; edit re-validates),
  │                       editMessage (runs $EDITOR via sh -c for arg parsing)
  └── ui/
        └── ui.go       — ANSI colors + Spinner; auto-disabled when stdout is not a TTY
```

### Key design decisions

**Config priority**: env vars (`OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`) override `~/.aicommit/config.json`, which overrides built-in defaults. This lets CI/power users override without touching the config file.

**Error handling**: `cmd/root.go:Execute()` sets `SilenceErrors` and `SilenceUsage` on the cobra root command, then prints errors via `ui.Error()` for colored output. All sentinel errors (`ErrNotConfigured`, `ErrAPIKeyInvalid`, `ErrRateLimited`, `ErrNoStagedChanges`, `ErrNotGitRepo`) are defined in their respective packages and checked with `errors.Is`.

**AI provider**: `OpenAIProvider` speaks the OpenAI chat completions API directly over `net/http` (no SDK). The base URL is configurable so any OpenAI-compatible endpoint works (DeepSeek, OpenRouter, Bailian, etc.). The request URL is always `baseURL + "/chat/completions"`, so `baseURL` must end at `/v1` (no trailing slash — trimmed on construction).

**Setup wizard** (`aicommit ai --setup`): presents preset providers with correct base URLs and models pre-filled, then calls `Ping()` (minimal `/chat/completions` request) to verify before saving. This catches wrong URLs and invalid keys before they cause confusing errors at commit time.

**Message validation**: `Confirm` accepts a `valid bool` parameter. When validation fails, the commit option is hidden — the user must edit or regenerate. After editing, the main loop re-validates the edited message before allowing commit, preventing invalid messages from slipping through.

**Editor support**: `editMessage` runs the `$EDITOR` / `$VISUAL` value through `sh -c`, so arguments (e.g. `code --wait`) and paths with spaces are handled correctly.
