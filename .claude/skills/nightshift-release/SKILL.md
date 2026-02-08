---
name: nightshift-release
description: Release a new version of the nightshift Go project using goreleaser. Use when the user asks to cut a release, push a new version, bump the version, publish a release, or verify a release succeeded. Covers version bumping, git tagging, goreleaser dry-run validation, pushing, and post-release verification via GitHub Actions and `gh`.
---

# Release Nightshift

Cut, publish, and verify releases for the nightshift Go project using goreleaser.

## Pre-flight

1. Ensure working tree is clean (`git status`).
2. Determine the **next version** by checking `git tag --sort=-v:refname | head -1` and asking the user what kind of bump (major/minor/patch) if not specified.
3. Confirm goreleaser is installed locally: `goreleaser --version`. If missing, prompt user to `brew install goreleaser`.

## Release Workflow

### 1. Bump version in source

Edit `cmd/nightshift/commands/root.go` — update the `Version` variable to the new version (without the `v` prefix):

```go
Version = "X.Y.Z"
```

### 2. Validate goreleaser config

Run a local dry-run to catch config errors before pushing:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

- `check` validates `.goreleaser.yml` syntax.
- `--snapshot --clean` builds all artifacts locally without publishing. Inspect the `dist/` directory to confirm binaries were produced for all expected OS/arch combos (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64).

Clean up after: `rm -rf dist/`.

### 3. Commit, tag, and push

```bash
git add cmd/nightshift/commands/root.go
git commit -m "Bump version to vX.Y.Z"
git tag vX.Y.Z
git push origin main
git push origin vX.Y.Z
```

The `v`-prefixed tag triggers the GitHub Actions release workflow (`.github/workflows/release.yml`).

### 4. Verify release

After pushing the tag, verify the release pipeline succeeded:

```bash
# Watch the Actions run (blocks until complete)
gh run list --workflow=release.yml --limit 1
gh run watch           # watch the most recent run

# Once complete, confirm the release exists
gh release view vX.Y.Z
```

Check that:
- The GitHub Actions run completed successfully.
- The release on GitHub lists all expected artifacts (tar.gz for each OS/arch + checksums.txt).
- The changelog was auto-generated and excludes docs/test/ci/chore commits.

If the homebrew cask was configured, also verify `gh api repos/marcus/homebrew-tap/contents/Casks/nightshift.rb` returns the updated cask.

### Troubleshooting

| Problem | Fix |
|---|---|
| `goreleaser check` fails | Fix `.goreleaser.yml` syntax per error message |
| Snapshot builds but CI fails | Compare local Go version (`go version`) with `go.mod`; ensure `CGO_ENABLED=0` |
| Release exists but missing artifacts | Re-run: delete the release (`gh release delete vX.Y.Z`), delete the tag (`git push --delete origin vX.Y.Z && git tag -d vX.Y.Z`), re-tag and push |
| Homebrew tap not updated | Check `HOMEBREW_TAP_TOKEN` secret is set in repo settings |
