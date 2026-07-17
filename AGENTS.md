# AGENTS.md

Guidance for agents maintaining `aicommit`, a Go CLI that generates and
optionally commits Conventional Commit messages from staged Git changes.

## Toolchain and Commands

- The module declares Go `1.26.2` in `go.mod`.
- Keep dependencies managed through Go modules. The direct application
  dependencies are Cobra and `golang.org/x/term`; the OpenAI-compatible client
  uses `net/http` directly.
- Run formatting on changed Go files with `gofmt -w <files>` before testing.

```bash
# Build the CLI locally
go build -o aicommit .

# Build a release-style binary with version metadata
go build -ldflags "-X github.com/wungjyan/aicommit/cmd.version=x.y.z -X github.com/wungjyan/aicommit/cmd.commit=$(git rev-parse --short HEAD) -X github.com/wungjyan/aicommit/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o aicommit .

# Run the complete test suite
go test ./...

# Run focused package tests
go test ./internal/ai -run TestGenerate -v
go test ./internal/git -run TestGetStagedDiff -v
go test ./internal/prompt -run TestValidateMessage -v

# Install the current checkout
go install .
```

## Release Commands

User-visible release notes live in `CHANGELOG.md`. Add changes under
`## Unreleased`, then use the release helper:

```bash
# Preview the notes without writing, committing, or tagging.
bash scripts/release.sh 0.1.7 --dry-run

# Freeze Unreleased, commit CHANGELOG, and create an annotated v0.1.7 tag.
bash scripts/release.sh 0.1.7
```

The non-dry-run command requires a clean working tree and a non-empty
`Unreleased` section. Pushing the tag triggers GitHub Actions.

Tests create temporary Git repositories and invoke the local `git` executable.
They do not call an external AI service: provider behavior is tested with
`httptest` servers. The editor test is skipped on Windows.

## Runtime Flow

`main.go` calls `cmd.Execute()`. The default command follows this sequence:

```text
cmd.NewRootCommand
  -> CommitWorkflow.runWithOptions
  -> git.IsGitRepo
  -> git.GetStagedDiff
  -> git.TruncateDiff (80 KiB default)
  -> config.LoadConfig
  -> ai.NewProvider
  -> provider.Generate
  -> prompt.ValidateMessage / prompt.Confirm loop
  -> git.Commit
```

`cmd/root.go` owns top-level command construction and terminal checks;
`cmd/run.go` owns the commit workflow; and `cmd/config*.go` owns
configuration display and setup. Keep package-level code focused on its
boundary rather than moving CLI control flow into `internal/*` packages.

## Package Responsibilities

```text
cmd/
  root.go       root command, version command, terminal checks
  run.go        diff -> generation -> validation -> confirmation/commit workflow
  config*.go    configuration display, setup, set, check, and path commands
  errors.go     error categories and exit-code mapping

internal/
  ai/           Provider interface and OpenAI-compatible HTTP implementation
  config/       ~/.aicommit/config.json persistence
  git/          repository detection, staged diff, truncation, git commit
  prompt/       Conventional Commit validation and interactive confirmation/editor
  ui/           terminal colors, status messages, and spinner

npm/
  bin/          Node launcher for the downloaded native binary
  postinstall.js fetches the GitHub release asset matching the npm package version

scripts/
  install.*     shell and PowerShell installers
  release.sh    validates, freezes, commits, and tags a CHANGELOG release
  generate_release_notes.sh extracts the matching CHANGELOG section for a tag
```

## Configuration and Provider Contract

`config.Config` persists `backend`, `api_key`, `base_url`, `model`, and
`language` in `~/.aicommit/config.json`. The config directory is created with mode `0700`
and the file with mode `0600`.

For every provider setting, a non-empty environment value wins over the saved
config value:

| Setting | Environment variable | Default |
| --- | --- | --- |
| Backend | `AICOMMIT_BACKEND` | `openai` |
| API key | `OPENAI_API_KEY` | required; no default |
| Base URL | `OPENAI_BASE_URL` | `https://api.openai.com/v1` |
| Model | `OPENAI_MODEL` | `gpt-4o-mini` |
| Message language | `AICOMMIT_LANGUAGE` | English behavior |

The currently exposed backend is OpenAI-compatible API. Codex and Claude CLI
providers remain implemented for future evaluation but are intentionally hidden
from the interactive setup flow.

The OpenAI provider trims trailing slashes and always calls
`<base-url>/chat/completions`. A configured endpoint must therefore include
its version path where applicable (usually `/v1`). `Ping` deliberately uses a
minimal request to that same endpoint instead of `GET /models`, because not
all compatible providers implement the latter.

`Generate` has a 60-second HTTP client timeout. Preserve and wrap sentinel
errors with `%w` when extending failures: `ai.ErrNotConfigured`,
`ai.ErrAPIKeyInvalid`, `ai.ErrRateLimited`, and `ai.ErrRequestFailed` are part
of the CLI's error-handling contract.

## Git, Prompt, and UI Behavior

- `git.GetStagedDiff` uses `git diff --cached`; an empty diff returns
  `git.ErrNoStagedChanges`.
- `git.TruncateDiff` is byte-based and adds a truncation notice. Preserve that
  bounded-input behavior before sending a diff to a provider.
- `prompt.ValidateMessage` validates only the header: allowed types are
  `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`,
  and `build`; headers are limited to 72 characters.
- An invalid generated or edited message must never be committable. The main
  loop re-validates after edits, and `Confirm` hides the commit option when
  `valid` is false.
- `editMessage` honors `$EDITOR`, then `$VISUAL`, then `vim`. It invokes the
  editor with `sh -c` so editor arguments such as `code --wait` work.
- `ui` disables ANSI escapes and the animated spinner when stdout is not a
  terminal. Retain usable non-TTY output for automation and pipes.

`cmd.Execute` suppresses Cobra's default error output and formats errors with
`ui.Error`; callers still receive the original error for a non-zero process
exit in `main.go`.

## Change Guidelines

- Add or update focused tests with behavior changes. Use `httptest` for HTTP
  paths and temporary repositories for Git behavior.
- Avoid network calls in tests and do not depend on a user's real config,
  editor, or Git repository.
- Keep the generated-message prompt in `internal/ai/openai.go` aligned with
  `prompt.ValidateMessage`; changing supported types or header rules usually
  requires changing both.
- Keep README documentation in English and Chinese in sync when changing a
  user-visible command, configuration option, supported provider, or install
  path.
- The npm package is a distribution wrapper, not an alternate implementation.
  Release asset names must remain `aicommit-<platform>-<arch>[.exe]` to match
  `npm/postinstall.js`. The postinstall script must download the asset for its
  own npm package version, never an unpinned latest release.
- Release notes are curated in `CHANGELOG.md`; do not infer GitHub Release
  content from commit prefixes. Keep `README.md`, `README.zh-CN.md`, and the
  two release guides in sync when changing the release workflow.
