# Releasing Goliath

Goliath is distributed through four channels, all driven by a single
[GoReleaser](https://goreleaser.com) pipeline (`.goreleaser.yaml`) that runs in
GitHub Actions (`.github/workflows/release.yml`) whenever a `v*` tag is pushed.

| Channel | Install command | Artifact pushed by |
|---|---|---|
| GitHub Releases | download from Releases page | `goliath_<ver>_<os>_<arch>.tar.gz` (+ `.zip` on Windows) + `checksums.txt` |
| `go install` | `go install github.com/igbanam/goliath/cmd/goliath@latest` | nothing — served by the Go module proxy from the public repo + tag |
| Homebrew (macOS/Linux) | `brew install igbanam/tap/goliath` | `Formula/goliath.rb` → `igbanam/homebrew-tap` |
| Scoop (Windows) | `scoop bucket add goliath https://github.com/igbanam/scoop-bucket && scoop install goliath` | `goliath.json` → `igbanam/scoop-bucket` |

## Versioning: ZeroVer until v1.0.0

Until **feature parity with Polylith is confirmed**, Goliath is versioned with
ZeroVer: `v0.1.0`, `v0.2.0`, `v0.3.0`, …

During the ZeroVer phase, `.goreleaser.yaml` is configured so that:

- every GitHub release is flagged **Pre-release** (`release.prerelease: "true"`)
- no 0.x release becomes the **latest** release (`release.make_latest: "false"`)
- Homebrew and Scoop still publish (0.x versions in the formula/manifest
  correctly signal instability to users)

### Cutting the first stable release (v1.0.0)

When Polylith feature parity is confirmed:

1. Edit `.goreleaser.yaml`:
   ```yaml
   release:
     prerelease: auto      # was "true"
     make_latest: "true"   # was "false"
   ```
2. Commit and push to `main`.
3. Tag and push:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
4. From then on, standard semver applies (`v1.1.0`, `v2.0.0`, …). Use `-rc1`
   suffixes for release candidates — GoReleaser's `prerelease: auto` will mark
   those as pre-releases automatically.

### Cutting a ZeroVer release

```bash
git tag v0.2.0
git push origin v0.2.0
```

The `release` workflow triggers automatically. It will:

1. Build `goliath` for linux/darwin/windows × amd64/arm64 (CGO disabled).
2. Create tar.gz archives (zip for Windows) including LICENSE, README.md,
   and docs/commands.md.
3. Generate `checksums.txt` (sha256) and create the GitHub Release
   (flagged **Pre-release** during ZeroVer).
4. Push `Formula/goliath.rb` to `igbanam/homebrew-tap`.
5. Push `goliath.json` to `igbanam/scoop-bucket`.

`go install ...@v0.2.0` works immediately after the tag is pushed — no extra
step needed.

## One-time repository setup (already done, kept for reference)

1. **Main repo** — `github.com/igbanam/goliath` (public). Required for
   `go install` and GitHub Releases.
2. **Homebrew tap** — `github.com/igbanam/homebrew-tap` (public). GoReleaser
   pushes the formula on its default branch.
3. **Scoop bucket** — `github.com/igbanam/scoop-bucket` (public). GoReleaser
   pushes the manifest on its default branch.

## Secrets setup (already done, kept for reference)

| Secret | Where | Value | Scope |
|---|---|---|---|
| `GITHUB_TOKEN` | automatic | provided by GitHub Actions | `permissions: contents: write` in the workflow — used for the GitHub Release only |
| `GH_PAT` | `igbanam/goliath` repo secrets | see note below | classic PAT with `repo` scope — used to push to `homebrew-tap` and `scoop-bucket` (the default Actions token cannot write to other repos) |

The `GH_PAT` was seeded with the local `gh` token. **Rotate it**: create a
dedicated classic PAT at https://github.com/settings/tokens (only `repo`
scope), then run:

```bash
gh secret set GH_PAT --repo igbanam/goliath   # paste the new PAT when prompted
```

and revoke the broad token afterwards.

## Verifying a release locally before tagging

```bash
# Validate config
goreleaser check

# Full dry run: builds everything into ./dist, no publish
goreleaser release --snapshot --clean --skip=publish

# Inspect what would be produced
ls -la dist/
```

CI also runs `goreleaser check` and a full snapshot release on every PR
(`.github/workflows/ci.yml`), so config or cross-compile regressions are
caught before a tag is ever pushed.

## Troubleshooting

- **Tap/Scoop push fails with "resource not accessible by integration"** —
  the `GH_PAT` secret is missing, expired, or lacks `repo` scope. The default
  `GITHUB_TOKEN` cannot push to a different repository; a PAT is required.
- **Release is not marked pre-release** — check `release.prerelease` in
  `.goreleaser.yaml` (should be `"true"` during ZeroVer).
- **`goliath version` prints `dev`** — the binary was built with plain
  `go build`; version is injected via `-ldflags` by GoReleaser. Expected.
- **Homebrew audit complains about 0.x version** — normal under ZeroVer;
  formulas in third-party taps may use 0.x versions.
