#!/usr/bin/env bash

set -euo pipefail

current_tag="${1:-}"
output_file="${2:-release-notes.md}"
version="${current_tag#v}"
changelog="CHANGELOG.md"

if [[ -z "$current_tag" || "$version" == "$current_tag" ]]; then
  echo "usage: $0 <current-tag> [output-file]" >&2
  exit 1
fi
[[ -f "$changelog" ]] || { echo "missing $changelog" >&2; exit 1; }

repo_slug="${GITHUB_REPOSITORY:-}"
if [[ -z "$repo_slug" ]]; then
  remote_url="$(git remote get-url origin 2>/dev/null || true)"
  repo_slug="$(printf '%s' "$remote_url" | sed -E 's#^git@github.com:##; s#^https://github.com/##; s#\.git$##')"
fi

previous_tag="$(git tag --merged "$current_tag" --sort=-version:refname | grep -Fxv "$current_tag" | head -n 1 || true)"
if [[ -n "$previous_tag" ]]; then
  compare_url="https://github.com/$repo_slug/compare/$previous_tag...$current_tag"
else
  compare_url="https://github.com/$repo_slug/commits/$current_tag"
fi

awk -v version="$version" '
  function matches_version(heading) {
    return heading == version || index(heading, version " ") == 1 ||
      heading == "[" version "]" || index(heading, "[" version "] ") == 1
  }

  /^##[[:space:]]+/ {
    heading = $0
    sub(/^##[[:space:]]+/, "", heading)
    if (found) {
      exit
    }
    if (matches_version(heading)) {
      found = 1
    }
  }

  found {
    print
  }

  END {
    if (!found) {
      exit 2
    }
  }
' "$changelog" > "$output_file" || {
  echo "unable to find release notes for version $version in $changelog" >&2
  exit 1
}

cat >> "$output_file" <<'EOF'

## Installation

**macOS / Linux**
```bash
curl -fsSL https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.sh | sh
```

Installs to `~/.local/bin` without administrator access. The installer prints PATH setup instructions when needed.

**Windows (PowerShell)**
```powershell
irm https://raw.githubusercontent.com/wungjyan/aicommit/main/scripts/install.ps1 | iex
```

**npm**
```bash
npm install -g @wungjyan/aicommit
```

**Go**
```bash
go install github.com/wungjyan/aicommit@latest
```
EOF

{
  echo
  echo "## Full Changelog"
  echo
  if [[ -n "$previous_tag" ]]; then
    echo "[$previous_tag...$current_tag]($compare_url)"
  else
    echo "[View commit history]($compare_url)"
  fi
  echo
} >> "$output_file"
