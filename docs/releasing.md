# Releasing aicommit

Releases are driven by Git tags and curated notes in [CHANGELOG.md](../CHANGELOG.md).
The tag is the build version; the corresponding CHANGELOG section is the GitHub
Release body.

## Maintain Unreleased notes

Add every user-visible change to the `## Unreleased` section before it is
released. Group entries by a useful heading such as Added, Changed, Fixed, or
Security. Do not wait until after the tag is pushed to write the notes.

## Verify before releasing

Run the relevant checks first:

```bash
go test ./...
go build -o aicommit .
```

## Prepare a version

Preview the notes without changing the repository:

```bash
bash scripts/release.sh 0.1.7 --dry-run
```

When ready, create the release commit and annotated tag:

```bash
bash scripts/release.sh 0.1.7
```

The script requires a clean working tree, a valid unused version, and a
non-empty `Unreleased` section. It moves those notes under
`## 0.1.7 - YYYY-MM-DD`, commits the CHANGELOG, and creates `v0.1.7`.

Push only after reviewing the local commit and tag:

```bash
git push origin <branch>
git push origin v0.1.7
```

Alternatively, add `--push` to the release command.

## What CI publishes

The pushed tag triggers GitHub Actions. It runs `go test ./...`, builds
macOS, Linux, and Windows binaries for amd64 and arm64, creates the GitHub
Release from the matching CHANGELOG section plus installation instructions, and
then publishes the npm wrapper.

The npm package version is set from the tag during publishing. Its postinstall
script downloads the Release asset with the same version, so installing a pinned
npm version installs the matching native binary.
