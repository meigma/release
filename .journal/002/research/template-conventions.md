# `template-go` conventions for the release CLI

## Direct answer

`template-go` is a small Cobra/Viper CLI skeleton wrapped in a substantially more complete repository toolchain. The reusable command pattern is: a thin `main`, a fresh `*cobra.Command` built by `internal/cli.NewRootCommand`, explicit stdin/stdout/stderr injection, an injected non-global `*viper.Viper`, `ExecuteContext` with signal cancellation, and linker-injected `version`/`commit`/`date` metadata. The current starter has only a root command, one persistent flag, environment-variable support, and no config-file reader. It does **not** yet demonstrate the template's mandated strict hexagonal architecture: there are no port interfaces, adapters, Mockery configuration, generated mocks, or `mocks/` packages. It also has no `doc.go` files despite the repository rule requiring one per package. (`cmd/template-go/main.go:1-46`; `internal/cli/root.go:14-111`; `AGENTS.md:15-37,53-70`)

## 1. Tracked tree

The tree below is the complete tracked tree on the clean local `master`, corresponding to Git tree `2dc7b019e5689e50f53988ef6cfe91c973b7e16`; source: [GitHub recursive tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1). `CLAUDE.md` and `.claude/skills` are tracked symlinks (mode `120000` in that tree).

```text
.
├── .agents/
│   └── skills/
│       ├── apko/
│       │   ├── SKILL.md
│       │   └── references/apko-commands.md
│       ├── git/SKILL.md
│       ├── journal-sync/
│       │   ├── SKILL.md
│       │   └── agents/openai.yaml
│       ├── melange/
│       │   ├── SKILL.md
│       │   └── references/melange-commands.md
│       ├── mise/
│       │   ├── SKILL.md
│       │   └── references/mise-commands.md
│       ├── session-close/
│       │   ├── SKILL.md
│       │   └── references/session-artifacts.md
│       ├── session-continue/SKILL.md
│       ├── session-new/
│       │   ├── SKILL.md
│       │   └── references/notes-template.md
│       ├── session-setup/SKILL.md
│       └── worktrunk/
│           ├── SKILL.md
│           └── references/commands.md
├── .claude/
│   └── skills                         [symlink]
├── .github/
│   ├── dependabot.yml
│   ├── repository-settings.toml
│   ├── scripts/
│   │   ├── configure_github_repo.py
│   │   ├── stage_ghd_release_assets.py
│   │   ├── test_configure_github_repo.py
│   │   └── test_stage_ghd_release_assets.py
│   └── workflows/
│       ├── attest.yml
│       ├── ci.yml
│       ├── docs-pages.yml
│       ├── release-dry-run.yml
│       ├── release-please.yml
│       ├── release.yml
│       └── security-scan.yml
├── .gitignore
├── .golangci.yml
├── .goreleaser.yaml
├── .moon/
│   ├── toolchains.yml
│   └── workspace.yml
├── .release-please-manifest.json
├── .session.md
├── AGENTS.md
├── CHANGELOG.md
├── CLAUDE.md                              [symlink]
├── CONTRIBUTING.md
├── DELETE_ME.md
├── README.md
├── SECURITY.md
├── apko.yaml
├── cmd/
│   └── template-go/
│       └── main.go
├── docs/
│   ├── .gitignore
│   ├── .python-version
│   ├── docs/
│   │   ├── index.md
│   │   └── stylesheets/palette.css
│   ├── mkdocs.yml
│   ├── moon.yml
│   ├── pyproject.toml
│   └── uv.lock
├── ghd.toml
├── go.mod
├── go.sum
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   └── root_test.go
│   ├── config/
│   │   └── config.go
│   └── templateinfo/
│       ├── info.go
│       └── info_test.go
├── melange.yaml
├── mise.lock
├── mise.toml
├── moon.yml
├── release-please-config.json
└── scaffold/
    └── .journal/
        ├── INDEX.md
        ├── SKILLS.md
        └── TECH_NOTES.md
```

Directory purposes:

- `cmd/template-go/` is the process entrypoint: it owns signal handling, real process streams, linker variables, Cobra execution, error printing, and the exit code; command construction remains elsewhere. (`cmd/template-go/main.go:1-46`)
- `internal/cli/` owns Cobra/Viper wiring, root-command construction, stream/config/build-info injection, version rendering, config binding, output handling, and command-level tests. (`internal/cli/root.go:1-111`; `internal/cli/root_test.go:1-74`)
- `internal/config/` projects Viper values into the starter's runtime `Config` and supplies the default message when the resolved value is blank. (`internal/config/config.go:1-26`)
- `internal/templateinfo/` holds template identity/default text independently of Cobra and Viper. (`internal/templateinfo/info.go:1-8`; `internal/templateinfo/info_test.go:1-13`)
- `docs/` is a separate Python/uv/MkDocs Material Moon project; source lives under `docs/docs`, build output under `docs/build`, and the tracked navigation currently contains only a home page. (`docs/pyproject.toml:1-11`; `docs/mkdocs.yml:1-50`; `docs/moon.yml:1-58`)
- `.github/` contains repository/Dependabot policy, repository-bootstrap and release-asset Python utilities, and CI, Pages, security, release rehearsal, release publication, release-please, and isolated-attestation workflows. (`.github/workflows/ci.yml:1-81`; `.github/workflows/release.yml:1-25`; `.github/scripts/stage_ghd_release_assets.py:1-36`)
- `.agents/skills/` contains checked-in operational skills and references for APKO, Git, journal synchronization, Melange, mise, Worktrunk, and the session lifecycle; `.session.md` is the repository's declared operating protocol. (`AGENTS.md:1-11`; `.agents/skills/apko/SKILL.md`; `.agents/skills/git/SKILL.md`; `.agents/skills/session-setup/SKILL.md`)
- `.moon/` defines the root/docs workspace and deliberately leaves language/tool versions to system binaries provisioned by mise. (`.moon/workspace.yml:1-38`; `.moon/toolchains.yml:1-15`)
- `scaffold/.journal/` is the tracked seed copied into generated repositories for journal index, skills, and technical notes. (`scaffold/.journal/INDEX.md`; `scaffold/.journal/SKILLS.md`; `scaffold/.journal/TECH_NOTES.md`)

## 2. CLI wiring

### Dependencies

- Module: `github.com/meigma/template-go`; Go directive: `1.26.4`. (`go.mod:1-3`)
- The only direct module requirements are Cobra `github.com/spf13/cobra v1.10.2` and Viper `github.com/spf13/viper v1.21.0`. (`go.mod:5-8`)
- Fang and Charmbracelet packages are absent from `go.mod`; no production Go source imports `log/slog`. There is therefore no Fang, Lip Gloss/Huh/Charm Log, or `slog.Logger` wiring in the starter. (`go.mod:1-25`; `cmd/template-go/main.go:1-12`; `internal/cli/root.go:1-13`; `internal/config/config.go:1-10`; `internal/templateinfo/info.go:1-3`)
- `github.com/google/go-cmp v0.7.0` and `github.com/rogpeppe/go-internal v1.15.0` are indirect dependencies only; neither is imported by the tracked tests. (`go.mod:10-25`; `internal/cli/root_test.go:1-9`; `internal/templateinfo/info_test.go:1-5`)

### Constructors and command tree

Key signatures actually present:

```go
func NewRootCommand(options Options) *cobra.Command
func Load(vp *viper.Viper) Config
func Summary() string
```

(`internal/cli/root.go:39-40`; `internal/config/config.go:16-17`; `internal/templateinfo/info.go:5-6`)

`NewRootCommand` constructs a fresh command on every call. Its `Options` inject stdin, stdout, stderr, build metadata, and a concrete Viper instance; nil streams become noninteractive/discard streams, nil Viper becomes `viper.New()`, and blank build fields become `dev`/`none`/`unknown`. (`internal/cli/root.go:14-53,82-94`)

The constructed root has `Use: "template-go"`, a short description, a Cobra `Version`, `SilenceUsage: true`, `SilenceErrors: true`, a `PersistentPreRunE` that binds config, and a root `RunE` that loads config and prints one line. It sets all three Cobra streams and defines a persistent `--message` flag. (`internal/cli/root.go:55-80`)

There are **no subcommands** and no `AddCommand` call in the tracked Go tree; the only `*cobra.Command` literal is the root command. The architect must introduce the requested profile-driven operations and their constructors rather than following an existing subcommand example. (`internal/cli/root.go:39-80`; [complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1))

### Flags, environment, config, and precedence

The only setting is `message`. Cobra supplies its default from `templateinfo.Summary`; Viper uses prefix `TEMPLATE_GO`, maps `-` and `.` to `_`, calls `AutomaticEnv`, and binds both root persistent flags and the executing command's flags. (`internal/cli/root.go:76-79,96-109`)

The effective Viper order is `Set` > flag > environment > config > key/value store > default, per the Viper 1.21.0 documentation; this starter uses only bound flags and environment plus the pflag/default-message fallback. ([Viper v1.21.0 README, precedence section](https://github.com/spf13/viper/blob/v1.21.0/README.md#why-viper); `internal/cli/root.go:76-79,96-109`; `internal/config/config.go:16-26`)

No config file is selected or read: `initializeConfig` contains no `SetConfigFile`, `AddConfigPath`, or `ReadInConfig`; `config.Load` reads the resolved key directly with `vp.GetString`, trims it, and falls back when blank. Therefore the template does **not** provide a config-file location, format, parse-error policy, or config-file test. (`internal/cli/root.go:96-109`; `internal/config/config.go:16-26`)

Tests independently prove `--message` and `TEMPLATE_GO_MESSAGE`; there is no case setting both at once, so the combined flag-over-environment result is library-documented rather than directly tested here. (`internal/cli/root_test.go:38-74`; [Viper v1.21.0 README](https://github.com/spf13/viper/blob/v1.21.0/README.md#why-viper))

### Main and versioning

`main` is intentionally thin: `main()` delegates to `run()` and exits; `run()` creates a SIGINT/SIGTERM `signal.NotifyContext`, supplies real process streams and linker values to `NewRootCommand`, calls `ExecuteContext`, writes an execution error to stderr, and returns `0` or `1`. (`cmd/template-go/main.go:20-46`; `README.md:46-54`)

The binary has package-main variables `version`, `commit`, and `date` defaulting to `dev`, `none`, and `unknown`. These become `cli.BuildInfo`; the custom version template prints `template-go VERSION (COMMIT) built DATE`. The code does not call `runtime/debug.ReadBuildInfo`; “build info” in this starter means the explicit `cli.BuildInfo` struct. (`cmd/template-go/main.go:13-35`; `internal/cli/root.go:14-20,69-76`; `internal/cli/root_test.go:10-36`)

GoReleaser injects `main.version={{ .Version }}`, `main.commit={{ .FullCommit }}`, and `main.date={{ .CommitDate }}` with `-X`; it also uses `-s -w -buildid=`, `-trimpath`, and a reproducible module timestamp. (`.goreleaser.yaml:7-31`)

The Melange build mirrors those three `-X` assignments using release-supplied vars; its defaults are `0.0.0`/`none`/`unknown`, while the local ordinary Moon build runs plain `go build` and therefore retains the Go-source defaults. (`melange.yaml:1-7,24-39`; `moon.yml:54-59`)

## 3. Ports, adapters, and mocks

The repository instructions require strict hexagonal architecture, side-effect-free business logic, I/O behind ports, single-purpose adapters, generated Mockery mocks, and `mocks/` subpackages under adapter packages. (`AGENTS.md:15-37`)

What exists in the starter:

- Explicit process-boundary injection for `io.Reader`, `io.Writer`, and a fresh `*viper.Viper` makes the Cobra layer locally testable without replacing global stdin/stdout/config state. (`internal/cli/root.go:22-53`; `internal/cli/root_test.go:10-74`)
- Package separation is limited to command wiring (`internal/cli`), runtime config projection (`internal/config`), and static template identity (`internal/templateinfo`). (`internal/cli/root.go:1-13`; `internal/config/config.go:1-10`; `internal/templateinfo/info.go:1-3`)
- All external types are concrete except the standard-library `io.Reader`/`io.Writer` interfaces. There are no declared application/domain port interfaces, no adapter implementations, and no demonstrated interface-placement convention. (`internal/cli/root.go:14-37`; `internal/config/config.go:11-26`; `internal/templateinfo/info.go:1-8`)

What is absent from the complete tracked tree:

- `.mockery.yml` / `.mockery.yaml`;
- any `mocks/` subpackage;
- generated mocks;
- any Go `interface` declaration;
- adapter packages or an application/domain layer.

([complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1); `AGENTS.md:15-37`)

Accordingly, the architect must introduce the actual ports/adapters and their interfaces, Mockery configuration, and generated `mocks/` packages required by the product's real GitHub/filesystem/process/network I/O. The template supplies testable process wiring, not a finished hexagonal example. (`AGENTS.md:17-37`; `internal/cli/root.go:22-53`)

## 4. Testing conventions

### Libraries and files

No test framework is a direct `go.mod` requirement: Testify, Mockery, `go-internal/testscript`, and Testcontainers are absent. `go-cmp` and `go-internal` appear only as indirect requirements and are not imported by the tracked tests. (`go.mod:5-25`; `internal/cli/root_test.go:1-9`; `internal/templateinfo/info_test.go:1-5`)

Tracked Go tests:

- `internal/cli/root_test.go` has three single-scenario tests for version output, flag-driven output, and environment-driven output. They use standard `testing`, `bytes.Buffer`, direct `t.Fatalf` comparisons, injected `viper.New()`, and `ExecuteContext`; the first two call `t.Parallel`, while the environment test uses `t.Setenv` and is not parallel. (`internal/cli/root_test.go:1-74`)
- `internal/templateinfo/info_test.go` is a small parallel standard-library test asserting only that `Summary()` is nonempty. (`internal/templateinfo/info_test.go:1-13`)
- There is no `internal/config/config_test.go`, no `cmd/template-go/main_test.go`, no table-driven Go test, no golden file, and no `.txtar`/testscript fixture in the tracked tree. ([complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1))

Two additional tracked Python `unittest` files cover repository configuration and GoReleaser/ghd asset staging with fakes and temporary directories: `.github/scripts/test_configure_github_repo.py` and `.github/scripts/test_stage_ghd_release_assets.py`. (`.github/scripts/test_configure_github_repo.py:1-70`; `.github/scripts/test_stage_ghd_release_assets.py:1-122`)

### How tests are invoked

The root Moon `test` task runs `go test ./...`; the aggregate `check` depends on format, lint, build, Go test, and strict docs build. (`moon.yml:61-81`)

GoReleaser repeats `go test ./...` as a `before` hook. (`.goreleaser.yaml:5-8`)

`README.md` declares Moon the standard task front door (`root:format`, `root:lint`, `root:build`, `root:test`, `root:check`) and shows `go test ./...` for the starter CLI. (`README.md:33-54`)

There is no Makefile or justfile in the tracked tree. `mise.toml` contains one local container-image task, not a Go-test task; test orchestration belongs to Moon. (`mise.toml:48-72`; [complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1))

The Python test files are not referenced by root/docs Moon tasks or any tracked workflow; they exist but are not part of the declared aggregate CI gate. (`moon.yml:45-81`; `docs/moon.yml:30-58`; `.github/workflows/ci.yml:79-81`)

The repository rules call for unit, mock-adapter integration, and live-service/end-to-end layers; Mockery-generated mocks are mandatory under those rules. The existing starter tests cover only the small in-process CLI behavior and template constant. (`AGENTS.md:29-37`; `internal/cli/root_test.go:10-74`; `internal/templateinfo/info_test.go:5-13`)

## 5. Toolchain and quality gates

### Go and mise pins

Go is pinned consistently to `1.26.4` in `go.mod`, `mise.toml`, and `docs` uses Python `3.14.3`. (`go.mod:1-3`; `mise.toml:19-22`; `docs/.python-version:1`)

`mise.toml` pins:

| Tool | Version |
|---|---:|
| Go | `1.26.4` |
| Python | `3.14.3` |
| `golangci-lint` via Aqua | `2.12.2` |
| uv via Aqua | `0.11.0` |
| Moon via Aqua | `2.3.5` |
| Melange via Aqua | `0.54.0` |
| APKO via Aqua | `1.2.19` |
| Cosign via Aqua | `3.1.1` |

(`mise.toml:19-37`)

`GOTOOLCHAIN=local` forbids implicit Go toolchain download; mise lockfile use is enabled and locked/fail-closed, and `mise.lock` is committed. (`mise.toml:39-46`; `README.md:6-20`)

Moon runs `system` binaries supplied on `PATH` by mise rather than managing language toolchains itself. (`moon.yml:13-19`; `.moon/toolchains.yml:1-8`)

### Moon tasks

The root project declares file groups for Go sources, lint config, and release config, then exposes:

- `format`: `golangci-lint fmt --config .golangci.yml --diff`;
- `lint`: `golangci-lint run --config .golangci.yml ./... --show-stats=false`;
- `build`: `go build -o bin/template-go ./cmd/template-go`;
- `test`: `go test ./...`;
- `check`: depends on root format/lint/build/test plus `docs:build`, is uncached, and is marked `runInCI: true`.

(`moon.yml:21-81`)

The workspace has explicit `root` and `docs` projects, default project `root`, GitHub VCS/default branch `master`, and dependency installation disabled at the workspace pipeline level. (`.moon/workspace.yml:4-38`)

### golangci-lint

The file is golangci-lint schema version 2 and identifies itself as the strict configuration for `golangci-lint v2.12.2`. It enables `goimports` and `golines` formatters; `goimports.local-prefixes` is `github.com/meigma/template-go` and `golines.max-len` is 120. (`.golangci.yml:3-13,21-44`)

Enabled linters, exactly as configured:

`asasalint`, `asciicheck`, `bidichk`, `bodyclose`, `canonicalheader`, `copyloopvar`, `cyclop`, `depguard`, `dupl`, `durationcheck`, `embeddedstructfieldcheck`, `errcheck`, `errname`, `errorlint`, `exhaustive`, `exptostd`, `fatcontext`, `forbidigo`, `funcorder`, `funlen`, `gocheckcompilerdirectives`, `gochecknoglobals`, `gochecknoinits`, `gochecksumtype`, `gocognit`, `goconst`, `gocritic`, `gocyclo`, `godoclint`, `godot`, `gomoddirectives`, `goprintffuncname`, `gosec`, `govet`, `iface`, `ineffassign`, `intrange`, `iotamixing`, `loggercheck`, `makezero`, `mirror`, `mnd`, `modernize`, `musttag`, `nakedret`, `nestif`, `nilerr`, `nilnesserr`, `nilnil`, `noctx`, `nolintlint`, `nonamedreturns`, `nosprintfhostport`, `perfsprint`, `predeclared`, `promlinter`, `protogetter`, `reassign`, `recvcheck`, `revive`, `rowserrcheck`, `sloglint`, `spancheck`, `sqlclosecheck`, `staticcheck`, `testableexamples`, `testifylint`, `tparallel`, `unconvert`, `unparam`, `unqueryvet`, `unused`, `usestdlibvars`, `usetesting`, `wastedassign`, and `whitespace`. (`.golangci.yml:46-143`)

Notable nondefault settings:

- maximum same issue count 50; cyclomatic complexity 30 with package average 10; function length 100 lines/50 statements; cognitive complexity 20. (`.golangci.yml:15-19,172-179,288-306`)
- `depguard` denies old protobuf/UUID packages, `math/rand` outside tests, and `log` outside `main.go` in favor of `log/slog`. (`.golangci.yml:181-228`)
- type assertions are checked; exhaustive checking covers switches and maps; embedded mutexes are forbidden. (`.golangci.yml:229-246`)
- `godoclint` adds `no-unused-link` and `require-stdlib-doclink`; `govet` enables all analyzers except `fieldalignment`. (`.golangci.yml:322-341`)
- `nolint` directives must name the linter and normally include an explanation. (`.golangci.yml:373-382`)
- `sloglint` prohibits global loggers and requires context-aware calls when context is in scope. (`.golangci.yml:402-419`)
- Staticcheck enables all checks except ST1000, ST1016, and QF1008. (`.golangci.yml:420-435`)
- `testpackage` is deliberately disabled; tests use same-package/white-box style. Test-file exclusions disable `bodyclose`, `dupl`, `errcheck`, `funlen`, `goconst`, `gosec`, `noctx`, and `wrapcheck`. (`.golangci.yml:127-136,451-473`)

### CI workflow

`.github/workflows/ci.yml` has one job, `ci`, on pushes and pull requests to `master` or `main`. It uses `ubuntu-latest`, read-only contents permission, job-level `GOTOOLCHAIN=local`, and finishes with `moon ci --summary minimal`. (`.github/workflows/ci.yml:1-29,79-81`)

Its external actions are SHA-pinned:

- `actions/checkout` v7.0.0 at `9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0`, with full history and persisted credentials disabled. (`.github/workflows/ci.yml:29-34`)
- `jdx/mise-action` v4.2.0 at `e6a8b3978addb5a52f2b4cd9d91eafa7f0ab959d`, installing mise `2026.6.14` with cache enabled. (`.github/workflows/ci.yml:35-41`)
- `actions/cache` v6.0.0 at `2c8a9bd7457de244a408f35966fab2fb45fda9c8` for Go modules, Go build artifacts, golangci-lint, and uv downloads. (`.github/workflows/ci.yml:47-78`)

## 6. Documentation and release plumbing

### Docs

The docs project uses MkDocs with Material. `pyproject.toml` requires Python `>=3.14`, declares `mkdocs-material>=9.7.0`, and is non-package uv project. (`docs/pyproject.toml:1-11`)

`mkdocs.yml` is strict, points at `docs/docs`, builds to `docs/build`, uses the Material theme with light/dark palettes, search, common pymdown extensions, edit links, and a single `Home: index.md` navigation item. (`docs/mkdocs.yml:1-50`)

The docs Moon project uses system Python/uv, runs `uv sync --locked`, builds with `uv run --locked mkdocs build --strict`, and provides a local serve task on `127.0.0.1:8000`. (`docs/moon.yml:1-58`)

The repository rule requires all user-facing docs under `docs/` and a Diátaxis organization. The actual tracked site has only `docs/docs/index.md` and a palette stylesheet; there are no tutorial/how-to/reference/explanation sections yet. (`AGENTS.md:53-70`; `docs/mkdocs.yml:33-34`; `docs/docs/index.md:1-16`; [complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1))

The Pages workflow has `build` and `deploy` jobs. It builds via mise/Moon, uploads `docs/build`, and uses SHA-pinned `actions/configure-pages` v6.0.0 (`45bfe019…`), `actions/upload-pages-artifact` v5.0.0 (`fc324d35…`), and `actions/deploy-pages` v5.0.0 (`cd2ce8fc…`). (`.github/workflows/docs-pages.yml:20-79`)

### GoReleaser and ghd

`.goreleaser.yaml` is version 2 and defines one `template-go` build from `./cmd/template-go`, `CGO_ENABLED=0`, for Darwin/Linux × amd64/arm64. It emits raw binary-format archives named `template-go_VERSION_OS_ARCH`, one `checksums.txt`, and binary SBOMs. Changelog generation and direct release publication are disabled. (`.goreleaser.yaml:1-49`)

A `before` hook runs `go test ./...`; build flags provide trimpath, stripping, an empty build ID, the three linker values, and commit-timestamp normalization. (`.goreleaser.yaml:5-31`)

`ghd.toml` is the install manifest matching those four asset names. It defines package `template-go`, tag pattern `v${version}`, installed binary path `template-go`, and provenance signer workflow `meigma/template-go/.github/workflows/attest.yml`; its purpose is to let consumers install the released binary with `ghd` while verifying the expected attestation signer. (`ghd.toml:1-35`; `README.md:97-112`)

`.github/scripts/stage_ghd_release_assets.py` validates that signer, tag pattern, binary path, OS/architecture asset matrix, checksums, and GoReleaser artifact inventory before staging release assets. (`.github/scripts/stage_ghd_release_assets.py:33-36,69-70,125-181,224-228`)

### Release Please and publication workflow

`release-please-config.json` uses release type `go`, `v` tags without component names, forced tag creation, draft releases, and pre-1.0 minor/patch bump rules. Its single root package is `template-go`; it maintains `CHANGELOG.md` and version-bumps `melange.yaml` and `apko.yaml`. Visible changelog sections are Features, Bug Fixes, Performance, and Dependencies; Documentation and Chores are hidden. (`release-please-config.json:1-27`)

The manifest records root version `0.1.1`. (`.release-please-manifest.json:1-3`)

The Release Please workflow runs on `master` pushes or manual dispatch, obtains an app token with SHA-pinned `actions/create-github-app-token` v3.2.0 at `bcd2ba49218906704ab6c1aa796996da409d3eb1`, then runs SHA-pinned `googleapis/release-please-action` v5.0.0 at `45996ed1f6d02564a971a2fa1b5860e934307cf7` using the config and manifest files. (`.github/workflows/release-please.yml:1-38`)

The tag-triggered release workflow does not let GoReleaser publish directly: it invokes SHA-pinned `goreleaser/goreleaser-action` v7.2.2 at `5daf1e915a5f0af01ddbcd89a43b8061ff4f1a89` with `release --clean --skip=publish`, stages/validates the ghd assets, smoke-tests the release binary, and uploads assets itself. (`.github/workflows/release.yml:74-157`)

Its job graph is `resolve-release`, `binary-release-assets`, isolated reusable-workflow `attest-binaries`, matrix `melange-build`, `container-image-release`, isolated `attest-image`, and `release-inspection-summary`. (`.github/workflows/release.yml:25-27,74-75,159-179,260-261,395-410`)

Release rehearsal has `binary-release-dry-run`, matrix `melange-build-dry-run`, and `container-image-dry-run`; it uses the same GoReleaser action/SHA with `--skip=publish`. (`.github/workflows/release-dry-run.yml:19-21,48-56,133-134,202-206`)

### Godoc as observed

The repository rule says every function/type/field, exported or unexported, gets Godoc and every package gets a `doc.go`. (`AGENTS.md:53-65`)

Observed production code documents all exported types, exported fields, and exported functions (`BuildInfo`, `Options`, `NewRootCommand`, `Config`, `Load`, `Summary`). (`internal/cli/root.go:14-40`; `internal/config/config.go:11-17`; `internal/templateinfo/info.go:5-6`)

Observed production code does not document private functions such as `run`, `withDefaults`, `initializeConfig`, and `printLine`, and the tracked tree contains no `doc.go` in `main`, `cli`, `config`, or `templateinfo`. Thus the current code demonstrates exported-API comments, but it does not yet satisfy its own D1/D4 package-documentation rules; the new CLI must add those missing package docs rather than copy the omission. (`cmd/template-go/main.go:20-24`; `internal/cli/root.go:82-111`; `AGENTS.md:53-65`; [complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1))

## 7. Reusable bits

### Reuse the pattern essentially verbatim

These are stable template conventions rather than product behavior:

- The process skeleton in `cmd/template-go/main.go`: linker-variable defaults, `signal.NotifyContext`, explicit real streams, `ExecuteContext`, stderr error reporting, and numeric exit result. Adapt only the import path, command directory/binary name, and product-specific build-variable target if package `main` changes. (`cmd/template-go/main.go:13-46`)
- The constructor-boundary shape in `internal/cli/root.go`: one fresh root per call; injected `In`/`Out`/`Err`, `BuildInfo`, and non-global `*viper.Viper`; nil-safe stream/config defaults; `SetIn`/`SetOut`/`SetErr`; `SilenceUsage`/`SilenceErrors`. Adapt the command identity and behavior. (`internal/cli/root.go:14-67,76-80`)
- The build metadata contract (`Version`, `Commit`, `Date`), blank-value defaults, and custom `--version` test pattern. Adapt the rendered binary name. (`internal/cli/root.go:14-20,69-94`; `internal/cli/root_test.go:10-36`)
- The `system`-toolchain handoff from mise to Moon and the aggregate `moon ci --summary minimal` front door. (`moon.yml:13-19,69-81`; `.moon/toolchains.yml:1-8`; `.github/workflows/ci.yml:79-81`)
- The CI security posture: empty workflow permissions, job-scoped read permission, SHA-pinned actions, checkout credentials disabled, `GOTOOLCHAIN=local`, and mise-lock-based provisioning/caching. (`.github/workflows/ci.yml:12-39,47-81`)
- The strict docs build mechanism (`uv sync --locked`; `uv run --locked mkdocs build --strict`) if the new repository retains the same docs stack. (`docs/moon.yml:30-51`)

### Adapt rather than copy literally

- `go.mod`: change the module path; retain exact dependency versions only if the new CLI intentionally starts on Cobra `v1.10.2`, Viper `v1.21.0`, and Go `1.26.4`. (`go.mod:1-8`; `README.md:21-31`)
- `internal/cli/root.go`: change `Use`, `Short`, version-template name, environment prefix, root behavior, flags, and introduce the required profile-driven subcommands. The current root-only `message` behavior is template demonstration code. (`internal/cli/root.go:55-80,96-109`)
- `internal/config/config.go`: replace the one-field message projection; add any required config-file behavior explicitly because none exists. (`internal/config/config.go:11-26`; `internal/cli/root.go:96-109`)
- `internal/templateinfo/`: replace the starter identity/default message or remove the package if the product has no equivalent static metadata. (`internal/templateinfo/info.go:1-8`)
- `.golangci.yml`: preserve the configured linter policy, but change `goimports.local-prefixes` from `github.com/meigma/template-go` to the new module. (`.golangci.yml:33-44`)
- `moon.yml`: change project metadata, binary output/path, file groups, and release inputs while preserving the Moon task front door. (`moon.yml:1-81`; `README.md:21-31`)
- `mise.toml`/`mise.lock`: keep them paired; update pins deliberately and regenerate all four platform lock entries rather than hand-editing the lock. Rename/remove the `image-local` task if the product's container name or release shape differs. (`mise.toml:1-17,19-72`; `README.md:12-20`)
- `docs/mkdocs.yml`, `docs/pyproject.toml`, `docs/docs/index.md`: update repository/site/project names and build real Diátaxis content. (`docs/mkdocs.yml:1-9,33-34`; `docs/pyproject.toml:1-8`; `docs/docs/index.md:1-16`; `AGENTS.md:65-70`)
- `.goreleaser.yaml`, `ghd.toml`, `melange.yaml`, `apko.yaml`, Release Please files, and release workflows: all contain `template-go`, `meigma`, binary paths, asset patterns, signer workflow, image names, or version files that must match the new repository. The template itself explicitly requires these renames after bootstrap. (`README.md:21-31,97-112`; `.goreleaser.yaml:1-42`; `ghd.toml:1-35`; `release-please-config.json:1-27`; `melange.yaml:1-39`)
- `.github/scripts/stage_ghd_release_assets.py`: retain its validation/staging behavior only if the release asset contract remains the same; its default binary name and expected four-platform matrix are product-specific. (`.github/scripts/stage_ghd_release_assets.py:31-36,125-181`)

### Required additions not supplied by the template

The architect cannot obtain these by copying an existing example because they are absent: product subcommands/profile dispatch; domain/application port interfaces; concrete adapters; `.mockery.yml`; generated per-adapter `mocks/` packages; Testify/testscript/Testcontainers dependencies and corresponding test layers where needed; config-file discovery/parsing if required; and `doc.go` for every Go package. (`internal/cli/root.go:39-80,96-109`; `go.mod:5-25`; `AGENTS.md:15-37,53-70`; [complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1))

## Unknowns and unverified behavior

- **UNVERIFIED by a repository test:** the combined `--message` plus `TEMPLATE_GO_MESSAGE` case. Flag-over-environment precedence follows Viper 1.21.0's documented contract, but the starter tests each source separately. (`internal/cli/root_test.go:38-74`; [Viper v1.21.0 README](https://github.com/spf13/viper/blob/v1.21.0/README.md#why-viper))
- No interface, adapter, mock, subcommand, config-file, `doc.go`, golden, or testscript convention can be inferred beyond “absent”; the architect must follow `AGENTS.md` and introduce those conventions rather than extrapolate from nonexistent examples. (`AGENTS.md:15-37,53-70`; [complete tracked tree](https://api.github.com/repos/meigma/template-go/git/trees/2dc7b019e5689e50f53988ef6cfe91c973b7e16?recursive=1))
