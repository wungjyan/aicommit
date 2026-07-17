# CLAUDE.md

Development notes for this repository.

## Commands

```bash
# Build
go build -o aicommit .

# Build with version info
go build -ldflags "-X github.com/wungjyan/aicommit/cmd.version=x.y.z -X github.com/wungjyan/aicommit/cmd.commit=$(git rev-parse --short HEAD) -X github.com/wungjyan/aicommit/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o aicommit .

# Test and static analysis
go test ./...
go vet ./...

# Install locally
go install .
```

## Architecture

`main.go` calls `cmd.Execute()`, which creates a fresh Cobra command tree with injected input, stdout, stderr, Git, configuration, provider, and UI dependencies.

```text
cmd/
  root.go          root command, modes, version command, terminal checks
  run.go           CommitWorkflow: diff -> provider -> validate -> confirm/commit
  config*.go       config display, setup, set, check, and path commands
  errors.go        usage/AI/validation error categories and exit-code mapping
  adapters.go      production adapters and stream wiring

internal/
  ai/              OpenAI-compatible provider plus retained experimental CLI providers
  config/          persisted config and effective environment resolution
  git/             staged diff, truncation, and commit execution
  prompt/          validation plus injected confirmation/editor streams
  ui/              injected stderr diagnostics, color, and spinner
```

## Key Decisions

**Effective configuration:** `AICOMMIT_BACKEND` and `AICOMMIT_LANGUAGE` apply to all implemented backends. For the OpenAI backend, `OPENAI_API_KEY`, `OPENAI_BASE_URL`, and `OPENAI_MODEL` override the saved config. Resolution follows environment, config file, then defaults. Older config files without `backend` mean `openai`.

**Backends:** The OpenAI-compatible API is the only backend exposed in the setup flow and is used for normal releases. Codex and Claude providers remain in the codebase for future latency work, but their choices and `--backend` flag are hidden from public help. If working on those retained providers, do not read or persist CLI credentials.

**Configuration commands:** `aicommit config` displays effective values and sources. `config setup` configures the OpenAI-compatible API and output language. `config set` makes field-level non-interactive changes, and `config check` checks API connectivity without modifying config.

**Workflow:** `CommitWorkflow` truncates the staged diff before every provider sees it. Regeneration reuses that same bounded diff. Every generated or edited message is validated before commit.

**Output:** stdout is data-only (`--dry-run`, config display/JSON/path, version). Spinner, status, confirmation prompts, warnings, and errors use stderr through `internal/ui.UI`.

**Exit codes:** `0` is success, `1` is a general runtime failure, `2` is invalid usage, `3` is an AI backend failure, and `4` is an invalid automatically generated message. Preserve AI sentinel errors with `%w` so `cmd.ExitCode` can classify them.

**Editor:** `$EDITOR`, then `$VISUAL`, then `vim` is invoked through `sh -c` to support editor arguments such as `code --wait`. The editor's streams are injected by the command adapter.

## Releases

`CHANGELOG.md` is the source of GitHub Release notes. Keep user-visible changes
under `## Unreleased`; categories inside that section use `###` headings.

```bash
bash scripts/release.sh 0.1.7 --dry-run
bash scripts/release.sh 0.1.7
```

The release helper requires a clean tree, a valid unused version, and non-empty
notes. It freezes the Unreleased section, commits `CHANGELOG.md`, and creates an
annotated tag. The tag workflow runs `go test ./...`, builds all six native
assets, extracts the matching CHANGELOG section into the GitHub Release, then
publishes npm.

The npm package is a Node-based downloader and launcher for the native binary.
Its published package version is set from the release tag; `npm/postinstall.js`
must download that same version's asset, never GitHub's latest release.
