# AGENTS.md

This file documents repository-specific guidance for Codex, Claude, and other coding agents working in `myxb`.

## Mission

Maintain and improve `myxb`, a Go CLI for logging into Tsinglan School's Xiaobao system, reading student scores, and calculating GPA with local credential storage and self-update support.

When making changes, optimize for:

- keeping GPA calculation behavior correct
- preserving login and update flows
- keeping the CLI buildable on Windows, macOS, and Linux
- leaving behind clear documentation for the next round of work

## Read First

Before changing behavior, skim these files:

- `README.md` for user-facing usage
- `CLAUDE.md` for the existing architecture summary
- `API_DOCUMENTATION.md` for endpoint behavior
- `GPA_CALCULATION.md` for grading rules and weighting details
- `.github/workflows/release.yml` for release packaging and asset naming
- `install.ps1` and `install.sh` for install/update expectations

## Tech Stack

- Language: Go `1.24.2`
- CLI framework: `github.com/urfave/cli/v3`
- Release/update dependencies:
  - `github.com/rhysd/go-github-selfupdate/selfupdate`
  - `github.com/blang/semver`
- Output/UI helpers:
  - `github.com/jedib0t/go-pretty/v6`

There are currently no Go test files in the repo. `go test ./...` is still useful as a compile-level safety check.

## Repository Map

- `cmd/myxb/`
  - `main.go`: CLI entrypoint, command registration, version banner, auto-update notification
  - `login.go`: interactive login flow
  - `gpa.go`: GPA calculation command and display logic
  - `update.go`: self-update command and update notification logic
  - `colors.go`: terminal styling helpers
- `internal/`
  - `api/`: Xiaobao API wrappers
  - `auth/`: password hashing helpers
  - `client/`: HTTP client and cookie handling
  - `config/`: credential persistence under `~/.myxb/config.json`
  - `models/`: API request/response models
- `pkg/gpa/`
  - `calculator.go`: GPA calculation pipeline
  - `score_mapping.go`: score mapping and course classification loading
  - `score_mapping.json`: embedded GPA mapping data
  - `course_classification.json`: embedded weighted/unweighted course lists

## Important Project Facts

- The release version is injected at build time with `-ldflags "-X main.version=..."`. Without that, the binary reports `Dev`.
- `pkg/gpa/*.json` files are embedded into the binary. Any change there requires a rebuild.
- The CLI supports `login`, `logout`, and `update`, plus the default GPA command.
- Saved credentials live outside the repo in the user's home directory. Be careful not to log or expose them while debugging.
- The `update` command depends on GitHub Releases existing in the expected repo and on release assets being present for the current OS/architecture.

## Working Rules For Agents

- Prefer small, focused changes over large refactors unless the task clearly requires one.
- Preserve existing CLI behavior unless the task explicitly asks for behavior changes.
- If you change GPA logic, also review whether `GPA_CALCULATION.md`, `README.md`, or embedded classification/mapping files must be updated.
- If you change API request/response handling, review `API_DOCUMENTATION.md` and affected model structs together.
- If you change release asset names, update all dependent places together:
  - `.github/workflows/release.yml`
  - `install.ps1`
  - `install.sh`
  - any update-related assumptions in `cmd/myxb/update.go`
- Do not commit generated binaries. The repo ignores `*.exe`, and release binaries are produced by CI.

## Local Development Workflow

Typical local loop:

1. Inspect the files involved and confirm how the current command flow works.
2. Make the code change.
3. Run `gofmt` on touched Go files.
4. Run `go test ./...`.
5. Build the CLI locally.
6. If the change affects live behavior, do the smallest safe manual verification against the real service.

Common commands on Windows PowerShell:

```powershell
go test ./...
go build -o myxb.exe ./cmd/myxb
.\myxb.exe help
.\myxb.exe login
.\myxb.exe
.\myxb.exe -t
```

Common commands on macOS/Linux:

```bash
go test ./...
go build -o myxb ./cmd/myxb
./myxb help
./myxb login
./myxb
./myxb -t
```

Notes:

- `go test ./...` currently reports "no test files" for packages, but it still validates that packages compile.
- Manual login/GPA verification requires a real Xiaobao account and network access.
- Avoid running update-related flows unless you actually need to validate release behavior.

## Submission And Commit Workflow

Observed repository history uses conventional-style commits such as:

- `feat(gpa): ...`
- `fix(...): ...`
- `refactor(...): ...`
- `chore(...): ...`

Follow that style unless the maintainer asks for something different.

Recommended workflow:

1. Keep each commit scoped to one logical change.
2. Use a conventional commit message, with scope when helpful.
3. Make sure the repo still passes `go test ./...` and a local build before committing.
4. If a change affects runtime behavior, update the relevant documentation in the same commit or same PR.
5. If this is part of a release train, work on the appropriate `dev/x.y.z` branch when that branch exists, then merge back to `main`.

Good commit examples:

- `feat(gpa): show task details with the -t flag`
- `fix(login): handle captcha retry after failed authentication`
- `chore(release): update installer and release notes template`

## Packaging And Release Workflow

This project's packaging flow is tag-driven.

### Local build packaging

For a quick local binary:

```powershell
go build -o myxb.exe ./cmd/myxb
```

If you need a versioned release-like local build, inject the version explicitly:

```powershell
go build -ldflags "-X main.version=1.0.6" -o myxb.exe ./cmd/myxb
```

### Official release packaging

Official release packaging happens through GitHub Actions in `.github/workflows/release.yml`.

Trigger:

- pushing a tag that matches `v*`, for example `v1.0.6`

What CI does:

1. Builds binaries for:
   - `linux/amd64`
   - `linux/arm64`
   - `windows/amd64`
   - `darwin/amd64`
   - `darwin/arm64`
2. Injects the version from the tag into `main.version`
3. Produces assets named:
   - `myxb_linux_amd64`
   - `myxb_linux_arm64`
   - `myxb_windows_amd64.exe`
   - `myxb_darwin_amd64`
   - `myxb_darwin_arm64`
4. Generates `SHA256SUMS`
5. Creates a draft GitHub Release with those files attached

### Release checklist

Before tagging:

1. Confirm `main` contains the intended fixes.
2. Run `go test ./...`.
3. Run a local build.
4. If release behavior changed, verify `install.ps1`, `install.sh`, and `cmd/myxb/update.go` still match the asset layout.
5. Update user-facing docs if commands or behavior changed.

Tag and publish:

1. Create and push a semantic version tag like `v1.0.6`.
2. Wait for the `Build Release` workflow to finish.
3. Open the draft GitHub Release.
4. Review generated release notes and fill in the `Features`, `Changes / Bug Fixes`, and `Chores` sections.
5. Sanity-check attached assets and `SHA256SUMS`.
6. Publish the release.

### Why asset naming stability matters

The following pieces depend on release assets being present and consistently named:

- `install.ps1` downloads `myxb_windows_amd64.exe` or `myxb_windows_386.exe` if that asset ever exists again
- `install.sh` selects `myxb_<os>_<arch>`
- `myxb update` depends on GitHub Releases being available for the current platform

If asset naming or supported architectures change, update the workflow and installers together.

## Verification Checklist For Code Changes

Use this checklist before wrapping up work:

- `gofmt` run on all edited Go files
- `go test ./...` passes
- local build succeeds
- docs updated if behavior changed
- embedded JSON files rebuilt if they were edited
- no secrets, tokens, or saved credentials were added to the repo

For higher-risk changes, also verify one or more of:

- `login` still works
- default GPA calculation still runs
- `-t` task display still renders correctly
- `update` still reports version information correctly

## Known Pitfalls

- GPA calculation depends on proportion normalization and filtering out null scores.
- Course classification has explicit lists and keyword-based fallbacks; check both before changing behavior.
- Embedded JSON changes do nothing until the binary is rebuilt.
- Release builds inject the version; a plain local build shows `Dev`.
- Update/install behavior is easy to break if release assets, repo owner/name, or workflow outputs drift apart.
