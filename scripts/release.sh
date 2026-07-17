#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <version> [--push] [--dry-run]" >&2
  exit 1
}

fail() {
  echo "$1" >&2
  exit 1
}

version=""
push=false
dry_run=false

for arg in "$@"; do
  case "$arg" in
    --push) push=true ;;
    --dry-run) dry_run=true ;;
    -* ) usage ;;
    *)
      if [[ -n "$version" ]]; then
        usage
      fi
      version="$arg"
      ;;
  esac
done

[[ -n "$version" ]] || usage
version="${version#v}"

if [[ ! "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
  fail "invalid version: $version"
fi

tag="v$version"
changelog="CHANGELOG.md"

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || fail "not inside a Git repository"
[[ -f "$changelog" ]] || fail "missing $changelog"

if [[ "$dry_run" != true ]] && [[ -n "$(git status --porcelain)" ]]; then
  fail "working tree is not clean; commit or stash changes before releasing"
fi

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
  fail "tag $tag already exists"
fi

unreleased_line="$(grep -nE '^##[[:space:]]+Unreleased([[:space:]]|$)' "$changelog" | head -n 1 | cut -d: -f1 || true)"
[[ -n "$unreleased_line" ]] || fail "missing '## Unreleased' section in $changelog"

next_heading_line="$(awk -v start="$unreleased_line" 'NR > start && /^##[[:space:]]+/ { print NR; exit }' "$changelog")"
if [[ -z "$next_heading_line" ]]; then
  next_heading_line="$(wc -l < "$changelog")"
  next_heading_line=$((next_heading_line + 1))
fi

if ! awk -v start="$unreleased_line" -v end="$next_heading_line" '
  NR > start && NR < end && $0 !~ /^[[:space:]]*$/ && $0 !~ /^[[:space:]]*#/ { found = 1 }
  END { exit(found ? 0 : 1) }
' "$changelog"; then
  fail "the '## Unreleased' section is empty; add release notes before releasing"
fi

notes="$(sed -n "$((unreleased_line + 1)),$((next_heading_line - 1))p" "$changelog" | sed '/^[[:space:]]*$/d')"
date="$(date +%Y-%m-%d)"

echo "Releasing $tag ($date)"
echo
echo "Release notes:"
echo "----------------------------------------"
printf '%s\n' "$notes"
echo "----------------------------------------"

if [[ "$dry_run" == true ]]; then
  echo "Dry run — no files changed, no commit, and no tag created."
  exit 0
fi

tmpfile="$(mktemp)"
trap 'rm -f "$tmpfile"' EXIT

awk -v line="$unreleased_line" -v version="$version" -v date="$date" '
  NR == line {
    print
    print ""
    print "## " version " - " date
    next
  }
  { print }
' "$changelog" > "$tmpfile"
mv "$tmpfile" "$changelog"

git add "$changelog"
git commit -m "chore(release): prepare $tag"
git tag -a "$tag" -m "aicommit $tag"

branch="$(git branch --show-current)"
if [[ "$push" == true ]]; then
  git push origin "$branch"
  git push origin "$tag"
  echo "Published $tag; GitHub Actions will build the release."
else
  echo "Created release commit and $tag locally."
  echo "To publish:"
  echo "  git push origin $branch"
  echo "  git push origin $tag"
fi
