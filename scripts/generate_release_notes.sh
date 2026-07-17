#!/usr/bin/env bash

set -euo pipefail

current_tag="${1:-}"
output_file="${2:-release-notes.md}"

if [[ -z "${current_tag}" ]]; then
  echo "usage: $0 <current-tag> [output-file]" >&2
  exit 1
fi

repo_slug="${GITHUB_REPOSITORY:-}"
if [[ -z "${repo_slug}" ]]; then
  remote_url="$(git remote get-url origin 2>/dev/null || true)"
  repo_slug="$(printf '%s' "${remote_url}" | sed -E 's#^git@github.com:##; s#^https://github.com/##; s#\.git$##')"
fi

previous_tag="$(git tag --sort=-version:refname | grep -Fxv "${current_tag}" | head -n 1 || true)"

if [[ -n "${previous_tag}" ]]; then
  range="${previous_tag}..${current_tag}"
  compare_url="https://github.com/${repo_slug}/compare/${previous_tag}...${current_tag}"
else
  range="${current_tag}"
  compare_url="https://github.com/${repo_slug}/commits/${current_tag}"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

write_section() {
  local title="$1"
  local pattern="$2"
  local file="$3"

  git log --no-merges --pretty=format:'- %s (%h)' "${range}" -- \
    > /dev/null 2>&1 || true

  git log --no-merges --pretty=format:'- %s (%h)' "${range}" \
    | grep -E "${pattern}" > "${file}" || true

  if [[ -s "${file}" ]]; then
    {
      echo "## ${title}"
      echo
      cat "${file}"
      echo
    } >> "${output_file}"
  fi
}

other_file="${tmpdir}/other.txt"

cat > "${output_file}" <<'EOF'
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

write_section "Features" '^[[:space:]]*-[[:space:]]*feat(\(.+\))?:' "${tmpdir}/features.txt"
write_section "Fixes" '^[[:space:]]*-[[:space:]]*fix(\(.+\))?:' "${tmpdir}/fixes.txt"
{
  echo "## Full Changelog"
  echo
  if [[ -n "${previous_tag}" ]]; then
    echo "[${previous_tag}...${current_tag}](${compare_url})"
  else
    echo "[View commit history](${compare_url})"
  fi
  echo
} >> "${output_file}"
