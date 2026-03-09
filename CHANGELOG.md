# Changelog

All notable changes to vismo will be documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- `vismo get <resource>[/<name>]` command - fetches any Kubernetes resource and prints its YAML
- kubectl-compatible resource resolution: short names (`deploy`, `sts`, `svc`, `po`, `cm`, `ing`, `ds`, `rs`, `cj`, `pvc`, `pv`, `ns`, `sa`, `no`), plural and singular forms, and slash syntax (`deploy/api`)
- GVR resolution via the Kubernetes discovery API with a built-in short name table as fallback
- `-n` / `--namespace` flag and `-A` / `--all-namespaces` flag, matching kubectl conventions
- `--kubeconfig` flag with fallback to `KUBECONFIG` env var, `~/.kube/config`, and in-cluster config
- Source detection engine - identifies which tool manages a resource by inspecting labels and annotations:
  - **Helm** - `app.kubernetes.io/managed-by=Helm`, `meta.helm.sh/release-name`, `helm.sh/chart`
  - **ArgoCD** - `argocd.argoproj.io/app-name`, `argocd.argoproj.io/tracking-id`
  - **Kustomize** - `app.kubernetes.io/managed-by=kustomize`, `kustomize.toolkit.fluxcd.io/*`
  - **Flux** - `kustomize.toolkit.fluxcd.io/name`, `helm.toolkit.fluxcd.io/name`
  - **kubectl** - `kubectl.kubernetes.io/last-applied-configuration`
- `--field` flag - traces ownership of any dot-separated field path (e.g. `spec.replicas`) using `managedFields`, sorted newest-first
- Request-scoped in-memory cache to avoid duplicate API calls within a single command execution
- `version` command - prints the version injected at build time via `-ldflags`
- `--log-level` flag (`debug`, `info`, `warn`, `error`) - default is `warn` (silent); `debug` prints every internal step
- Notice printed to stderr when `--log-level=debug` or `--log-level=info` is active
- Structured logging via `slog` throughout all packages
- `TextFormatter` - human-readable output with TTY-aware ANSI section headers
- Banner embedded in the binary via `//go:embed`
- `Makefile` with `build`, `test`, `lint`, and `run` targets
- `.golangci.yml` with `errcheck`, `govet`, `staticcheck`, and `unused` linters
- GitHub Actions CI workflow - runs `go vet`, `go test -race`, build smoke-test, and `golangci-lint` on every push and pull request
- GitHub Actions Release workflow - builds statically linked binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64` on every `v*.*.*` tag; publishes a GitHub Release with auto-generated changelog

