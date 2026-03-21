# Contributing

## Pull requests

- **Title:** Use [Conventional Commits](https://www.conventionalcommits.org/) so history and release notes stay readable. Examples: `feat(agent): add hook limit`, `fix(cli): handle EOF`, `chore(ci): bump Go version`.
- **Checks:** The `CI` workflow runs Go lint (`golangci-lint`), Go tests (with coverage on Linux), Rust `fmt` / `clippy` on `phantom-cli` + `phantom-client` (not the Tauri crate), Rust `llvm-cov` for those packages, full build including eBPF on Ubuntu, macOS build/test, and Linux BPF e2e (`make test-e2e-mr`: Rust CLI, shell scripts, and extended Go e2e including `E2E_SCENARIOS`). `PR checks` validates the PR title.
- **Merge:** Squash merge is recommended so the default commit message matches the PR title.

## Branch protection (maintainers)

In GitHub **Settings → Branches**, for `main` / `master` consider:

- Require a pull request before merging
- Require status checks to pass before merging. Enable every job from the **CI** workflow (note matrix jobs appear per OS, e.g. `Lint (ubuntu-latest)`), plus **Conventional PR title** from **PR checks**
- Optionally require branches to be up to date before merging

Exact job names appear in the Actions tab and may include matrix suffixes (e.g. OS); pick the checks that match your workflow run.

## Coverage reports

- Linux **Go** coverage is uploaded as a workflow artifact (`go-coverage`). Optional: add a repository secret `CODECOV_TOKEN` to push **Go** and **Rust** reports to [Codecov](https://codecov.io/) (flags `go` and `rust`).

## Releases

- Tag a version with `v*` (e.g. `v0.2.0`) to trigger the **Release** workflow. It runs a **Preflight** job (`go test ./...`, Rust tests for `phantom-cli` / `phantom-client`), then builds cross-compiled `phantom-agent` binaries and `phantom-cli` for Linux amd64, and creates a GitHub Release with those assets.

## Local development

- `make proto` — regenerate Go protobufs (needs `protoc`).
- `make desktop-install` / `make desktop-dev` — Phantom Desktop (Tauri); `make desktop-build` for release binary (see [src/desktop/README.md](src/desktop/README.md)).
- `make test-e2e-ci` — Go e2e only (`E2E_HTTP10`, `E2E_NETWORK`, `E2E_SCENARIOS`). `make test-e2e-mr` — full Linux MR e2e (CLI + scripts + `test-e2e-ci`). See [README.md](README.md) and [docs/testing.md](docs/testing.md).
