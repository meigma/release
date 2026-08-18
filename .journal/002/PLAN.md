# `release-cli` end-to-end implementation plan

## Binding implementation rules

This plan implements Revision 3 without reopening its decisions. The following rules apply to every PR:

- Keep the four reusable workflows as GitHub-owned orchestration shells. They retain triggers, `workflow_call` contracts, job graphs, permission ceilings, timeouts, concurrency, checkouts, mise installation and tool-path proofs, QEMU, artifact transport, `actions/create-github-app-token`, and `actions/attest`.
- Put deterministic local work behind `fs.FS`, `io.Reader`, and `io.Writer`; place `os.DirFS` at the CLI composition edge. Do not create a filesystem port or adapter (A1 as resolved by architecture §4; R1, R3, P2).
- Introduce only the 13 ports approved in architecture §4, in the PR that first uses each one. Interfaces live in the consumer package (I2); Mockery-generated mocks live under the implementing adapter's `mocks/` package (T2, T3).
- Every package, including every `mocks` package and `package main`, gets `doc.go` (D4). Every function, type, method, and struct field—exported or not—gets Godoc; public command/domain boundaries get fuller contract comments and useful examples (D1, D2). Keep source files below 1,000 lines (R2).
- Keep `release-cli` noninteractive and pipeline-safe: structured data only on stdout under `--json`, diagnostics/warnings on stderr, and exit codes only `0`, `1`, and `2`. The JSON envelope is exactly `{"schema":"release.dev/result/v1","command":...,"ok":...,"result":...}`. There is no `phase`, `mutations`, generic safe-rerun claim, or exit `3`.
- Keep flags and `RELEASE_*` environment variables through an injected, non-global `*viper.Viper`; do not read a config file. Unknown profiles and invalid flags/configuration are usage/config failures (exit `2`). Contract, verification, and fail-closed remote-state failures are exit `1`.
- Merge through GitHub PRs with squash merge. Each PR title below is the Conventional Commit title that must become the squash commit subject. Use one Worktrunk worktree under `.wt/` per PR.
- “Current lines” below refer to the checked-in workflows inventoried on 2026-08-18. When earlier PRs shift physical line numbers, target the named step and inventory ID, not a guessed new line number.

## 1. Plan summary

| PR | Conventional Commit title | Slice | What lands | What it replaces | Dependency | Merge gate |
|---:|---|---|---|---|---|---|
| 1 | `feat(cli): stage Go release artifacts` | 1 | Four-package CLI slice; `stage --profile go`; `version`; JSON/exit contract; composite setup action; atomic `release-mvp` → `release-cli` artifact transition; dogfood `cli-path`; producer cutover | PP-06–PP-08, `.github/workflows/go-pre-publish.yml:88-119`; deletes the `greet` MVP | Self-reference and Release Please stamping spikes pass; fetched `main` | Four required mutation failures, `moon run root:check`, real GoReleaser 2.17.1 rehearsal, composite installed/local-path checks, and first `release-cli` draft release all pass |
| 2 | `feat(cli): verify Actions artifact handoffs` | 2 | `ArtifactMeta` and Actions-runtime ports; `ghact`/`actenv`; `verify handoff`; Mockery infrastructure; composite installed in the other three workflows | OB-06, GR-05, OP-03: the three 36-line metadata scripts at `go-oci-build.yml:106-141`, `publish-github-release.yml:129-164`, `publish-oci-image.yml:88-123` | PR 1 released | Same-run/expiry/digest cases pass locally; one live run proves all three metadata checks followed by `download-artifact` digest verification |
| 3 | `feat(oci): plan immutable release tags` | 3a | `internal/rel`; stable versions/digests/channels/tag planning; `StateReader`; native `reg` read path; `plan tags` | Implements OP-10–OP-12 policy behind the CLI; YAML remains authoritative until PR 5 | PR 2; oras-go/GHCR spike passes | Pure invariant-11/12 tables, in-memory registry reads, and GHCR parity reads pass |
| 4 | `feat(oci): prepare digest publication` | 3b | `ContentPusher`, `Signer`, native digest publication, recursive Cosign signing, `OCIPrepareResult`, `publish oci prepare` and dry-run behavior | Implements OP-09 and OP-13–OP-15; OP-20 becomes obsolete through in-memory credentials; YAML remains authoritative until PR 5 | PR 3 | In-memory partial-push recovery, dry-run zero-write proof, real Cosign argument contract, and scratch GHCR prepare pass |
| 5 | `feat(oci): finalize trusted image tags` | 3c | `TagCommitter`; `publish oci finalize`; fresh-state drift refusal; prepare → three YAML attestations → finalize workflow cutover | Cuts over OP-09–OP-15, OP-19/OP-20 at `publish-oci-image.yml:234-481,511-569`; OP-16–OP-18 remain YAML | PR 4 | Full scratch GHCR rehearsal proves signatures/referrers/attestations precede exact/channel tags and reruns reconcile safely |
| 6 | `feat(release): verify signed release bundles` | 4a | Closed `Bundle`, `BlobVerifier`, Cosign verify adapter, `verify bundle`, Actions outputs for paths/digests | GR-09–GR-12 at `publish-github-release.yml:222-333`; GR-13 remains immediately after it | PR 5 | Grammar/closure/signature-identity failures pass; a live draft attests only checksummed subjects |
| 7 | `feat(release): publish verified GitHub releases` | 4b | `ReleaseReader`, `AssetReplacer`, `Publisher`, `RefResolver`; `ghrel`/`ghup`/`gitx`; `publish github`; bounded polling and reconciliation | GR-06–GR-07 and GR-14–GR-18 at `publish-github-release.yml:166-213,341-482`; GR-08 transport stays YAML | PR 6 | Draft-only, tag/SHA, expected-name-only clobber, 24×5s/12×1s convergence, disabled-undraft, and live draft recovery all pass |
| 8 | `feat(image): build OCI layouts from staged binaries` | 5a | First persisted, artifact-local image-input projection; `APKBuilder`/`Composer`; Melange/apko adapters; `image build` | OB-08–OB-17 at `go-oci-build.yml:150-314`; old YAML verifier remains as an independent oracle | PR 7; engine/adapters may be developed in parallel after PR 2 | CLI output is accepted by the unchanged OB-18–OB-21 verifier; pinned Melange/apko/QEMU rehearsal passes |
| 9 | `feat(image): verify OCI image contracts` | 5b | Exact-byte OCI layout parsing; `image verify`; byte-identity/runtime/SBOM checks; image digest output | OB-18–OB-21 at `go-oci-build.yml:315-436` | PR 8 | Synthetic mutation tables and a real two-platform image prove invariants 5 and 6 and exact `index.json` hashing |
| 10 | `refactor(prepublish): run GoReleaser through release-cli` | 6 | GoReleaser invocation moves into `goprof`; producer becomes a thin shell; PP-04 tool proof remains; full workflow/docs cleanup | PP-05 in `go-pre-publish.yml:74-86` | PR 9 | Full cross-repo live rehearsal passes from tag through public release; both GoReleaser publication-disable controls remain observable |
| 11 | `docs(release): pin the completed release unit` | Program closeout | Replaces temporary `FULL_SHA` markers and the old `fb8c…` example pin with the released slice-6 commit in all copyable docs/examples | Documentation pin only; no behavior | PR 10 release exists and is verified | Every copied workflow reference and signer identity uses the same full SHA; archive and OCI verification commands pass against that release |

The integration order is linear because the PRs touch a shared command tree and workflow files. Development can still be parallel: the slice-4 core can start after PR 2 while slice 3 is underway, and the slice-5 core can start after PR 1. Rebase those worktrees onto the last merged PR before review; do not merge locally.

### Complete 42-step accounting

| Ownership PR | Inventory set | Count | Disposition |
|---|---|---:|---|
| 1 | PP-06–PP-08 | 3 | Replaced by `stage` |
| 2 | OB-06, GR-05, OP-03 | 3 | Replaced by `verify handoff` |
| 3 | OP-10–OP-12 | 3 | Policy and reads land; workflow cutover occurs in PR 5 |
| 4 | OP-09, OP-13–OP-15, OP-20 | 5 | In-memory auth for the CLI's own pushes, digest push/sign; **OP-09 login and OP-20 cleanup survive** (spike B: `actions/attest --push-to-registry` and `cosign` read the docker config, so a login step is required and `cosign` has no `logout`); cutover in PR 5 |
| 5 | OP-19 | 1 | Replaced by `finalize`; OP-16–OP-18 remain residual YAML |
| 6 | GR-09–GR-12 | 4 | Replaced by `verify bundle` |
| 7 | GR-06–GR-08, GR-14–GR-18 | 8 | Decision logic moves; GR-08 itself deliberately remains the SHA-pinned transport step under the three-owner handoff contract |
| 8 | OB-08–OB-17 | 10 | Replaced by `image build` |
| 9 | OB-18–OB-21 | 4 | Replaced by `image verify` |
| 10 | PP-05 | 1 | GoReleaser invocation moves into `goprof` |
| **Total** |  | **42** | No inventory item in the approved migrated set is unaccounted for |

## 2. Preconditions and repository plumbing

### Worktree and base

Before each PR:

```bash
git fetch origin main
wt list --format=json
wt switch --create --base origin/main feat/release-cli-<slice>
git rev-parse --show-toplevel
git branch --show-current
```

The new path must be under `/Users/josh/code/meigma/release/.wt/`, the tree must start from fetched `origin/main`, and no other agent's worktree may be reused. Push normally, open a GitHub PR, wait for required checks, and squash-merge. Never use `wt merge`, `wt step push`, or local `git merge`.

### Current tool and build facts

The executor must preserve these read values unless a PR below explicitly changes them:

- `go.mod`: module `github.com/meigma/release`, Go `1.26.6`, Cobra `v1.10.2`, Testify `v1.11.1`; Viper is currently absent and is added at `v1.21.0` in PR 1 to implement architecture §7's injected `RELEASE_*` resolution.
- `mise.toml`: Go `1.26.6`, Python `3.14.7`, golangci-lint `2.12.2`, GoReleaser `2.17.1`, GitHub CLI `2.97.0`, Syft `1.51.0`, uv `0.12.5`, Moon `2.5.1`, Melange `0.59.1`, apko `1.2.37`, Cosign `3.1.3`; `GOTOOLCHAIN=local`, locked mise lockfile. PR 2 adds Mockery `3.7.3` and refreshes `mise.lock` because the first genuine ports arrive there.
- `moon.yml`: the project is currently titled `release-mvp`; `format` runs `golangci-lint fmt`, `lint` runs `golangci-lint run`, `build` produces `bin/release-mvp` from `./cmd/release-mvp`, `test` runs `go test ./...`, and `check` depends on format/lint/build/test. PR 1 atomically changes only the project/binary paths to `release-cli`; it does not create a second task convention.
- `.golangci.yml`: schema v2 for golangci-lint `2.12.2`; `goimports` uses `github.com/meigma/release`; `golines` is 120 columns; strict complexity, doc, error, security, `slog`, Testify, and staticcheck rules are enabled; `testpackage` is deliberately disabled. Do not weaken this posture. Generated mocks remain generated code; do not add blanket linter suppressions.
- `.release-please-manifest.json` currently maps `"."` to `"0.0.0"`; it remains the one root package history through the rename.
- `.moon/workspace.yml` names `main` as the default branch.

Initial setup proof:

```bash
mise install
mise exec -- go version                 # must report go1.26.6
mise exec -- goreleaser --version       # must report 2.17.1
mise exec -- golangci-lint version      # must report 2.12.2
mise exec -- moon run root:check
```

### Mockery decision

`.mockery.yml` must **not** land in PR 1: slice 1 has only stdlib file/stream boundaries and no genuine custom port. It lands in PR 2 with `ArtifactMeta` and the Actions-runtime port. Use Mockery v3's `template: testify` and an explicit `packages`/`interfaces` map. Each approved consumer interface is generated into the corresponding `internal/adapter/<adapter>/mocks/` directory; checked-in `doc.go` is hand-maintained, while mock implementations are generated. Add interfaces to the map only when their slice lands.

The complete, closed port budget is:

1. `pubgh.ArtifactMeta` → `ghact`
2. `puboci.StateReader` → `reg`
3. `puboci.ContentPusher` → `reg`
4. `puboci.TagCommitter` → `reg`
5. `puboci.Signer` → `cosign`
6. `pubgh.ReleaseReader` → `ghrel`
7. `pubgh.AssetReplacer` → `ghup`
8. `pubgh.Publisher` → `ghrel`
9. `pubgh.RefResolver` → `gitx`
10. `pubgh.BlobVerifier` → `cosign`
11. `image.APKBuilder` → `melange`
12. `image.Composer` → `apko`
13. `cli.Actions` → `actenv`

No PR may add a fourteenth port.

## 3. Per-PR specifications

### PR 1 — `feat(cli): stage Go release artifacts`

#### Scope

- Replace the greeting MVP with the four production packages required by architecture §11: `cmd/release-cli`, `internal/cli`, `internal/stage`, and `internal/profile/goprof`.
- Add `stage --profile go --dist PATH [--json]` and `version [--json]`. Directly dispatch only `go`; an unknown profile is exit `2`.
- Implement strict checksum parsing/streaming verification and GoReleaser artifact parsing/selection. Use `os.DirFS(dist)` only in CLI composition.
- Ship `.github/actions/setup-release-cli/action.yml`, cut the public producer workflow over to it, and use `cli-path` for the first dogfood release.
- Atomically rename the repository's released artifact from `release-mvp` to `release-cli`; remove `cmd/release-mvp` and the `greet` command/tests.

#### Files

- Create `cmd/release-cli/doc.go` and `cmd/release-cli/main.go` — thin signal-aware entry point with injected streams and linker variables `version`, `commit`, and `protocol`.
- Delete `cmd/release-mvp/main.go` and the now-empty `cmd/release-mvp/` — clean cutover; no compatibility binary.
- Create `internal/cli/doc.go`, `internal/cli/result.go`, `internal/cli/stage.go`, and `internal/cli/version.go`; modify `internal/cli/root.go` and `internal/cli/root_test.go` — fresh Cobra tree, Viper injection, JSON/error/exit mapping, stage/version commands.
- Create `internal/stage/doc.go`, `internal/stage/checksum.go`, and `internal/stage/checksum_test.go` — checksum grammar and streamed `fs.FS` verification.
- Create `internal/profile/goprof/doc.go`, `internal/profile/goprof/artifacts.go`, and `internal/profile/goprof/artifacts_test.go` — real GoReleaser JSON parser, pure selection, path/executable validation.
- Create `internal/profile/goprof/testdata/goreleaser-2.17.1-artifacts.json` — verbatim parser fixture captured from pinned GoReleaser; document its refresh command in the fixture test, not a new package.
- Create `.github/actions/setup-release-cli/action.yml` — composite acquisition/verification bridge described in §6.
- Modify `.github/workflows/go-pre-publish.yml` — optional `cli-path`, conditional dogfood binary download, composite setup, CLI replacement for PP-06–PP-08; PP-09/PP-10 upload definitions remain byte-for-byte.
- Modify `.github/workflows/release.yml` — add a `build-release-cli` job that builds the tagged checkout, uploads a private `release-cli-dogfood` artifact, and make `release-assets` depend on it and pass the extracted `cli-path`.
- Modify `go.mod`/`go.sum` — add Viper `v1.21.0`; retain Cobra/Testify pins.
- Modify `moon.yml` — title/build/output become `release-cli`, `./cmd/release-cli`, and `bin/release-cli`.
- Modify `.goreleaser.yaml` — `project_name`, build/archive IDs, `main`, and binary name become `release-cli`; add `-X main.protocol=1`; retain platforms, archive/SBOM/checksum/signing, and `release.disable: true`.
- Modify `melange.yaml` — package/install target becomes `release-cli` and `/usr/bin/release-cli`.
- Modify `apko.yaml` — package, entrypoint, and image title become `release-cli`.
- Modify `release-please-config.json` — root `package-name` becomes `release-cli`; add the action version stamp under `extra-files` using the jsonpath proven by the spike.
- Leave `.release-please-manifest.json` at the existing root key/version; package rename must not reset history.
- Modify `README.md`, create `docs/reference/release-cli-contract.md`, and modify `docs/reference/github-release-contract.md`, `docs/how-to/upgrade-github-release-workflows.md`, and `examples/go-release/README.md` — document the CLI/action/one-version contract and label the example's old SHA as the last released revision until PR 11.

Every Go file above receives D1 comments; `NewRootCommand`, result-envelope types, checksum/parser entry points, and `CanonicalBinary` receive D2-level contract comments and examples where useful.

#### Interfaces/signatures

```go
type BuildInfo struct {
    Version  string
    Commit   string
    Protocol int
}

type Options struct {
    In     io.Reader
    Out    io.Writer
    Err    io.Writer
    Viper  *viper.Viper
    Build  BuildInfo
}

func NewRootCommand(options Options) *cobra.Command
func ExitCode(err error) int

func ParseChecksums(r io.Reader) (ChecksumSet, error)
func VerifyBundle(fsys fs.FS, claim ChecksumSet) error

func ParseArtifacts(r io.Reader) ([]Record, error)
func SelectBinaries(recs []Record) ([]CanonicalBinary, error)
func VerifyBinaries(fsys fs.FS, binaries []CanonicalBinary) error
```

`ChecksumSet`, `Record`, and `CanonicalBinary` are validating domain values, not aliases for unvalidated maps/strings (I1). No profile interface, registry, filesystem port, manifest store, or Actions port exists yet.

#### Behavior preserved

- PP-06 (`go-pre-publish.yml:88-93`): strict nonempty checksum claim, every listed payload streamed through SHA-256, and nonempty regular `checksums.txt.sigstore.json`.
- PP-07 (`:95-113`): exactly one Linux `Binary` record for each `amd64` and `arm64`, no extras/duplicates.
- PP-08 (`:114-119`): every selected `dist/`-relative path is confined, regular, and executable.
- Invariants kept enforceable: 4 remains on unchanged PP-09/PP-10 artifacts; 7's producer-side checksum half remains; 17 remains in YAML/GoReleaser until PR 10.

#### Tests

- **Layer 1, pure:** `internal/stage/checksum_test.go` tables for grammar, uppercase normalization, CRLF, empty input, duplicate filename, control self-listing, invalid flat name, and invalid digest; `internal/profile/goprof/artifacts_test.go` tables for malformed JSON, wrong type/OS, duplicate/missing architecture, and lexical path escape.
- **Layer 2, cheap real substrate:** `fstest.MapFS` for missing/empty/nonregular bundle; `t.TempDir()` for real executable mode and symlink/escape behavior; Cobra tests for stdout/stderr separation, JSON envelope, flag-over-env precedence, exit `1` versus `2`, and version protocol output.
- Required deliberate mutations: bad checksum, missing architecture record, escaped path, and cleared execute bit must each fail with the correct observable diagnostic and exit `1`.
- **Layer 3, live:** pinned GoReleaser 2.17.1 produces real `dist/artifacts.json`; the branch-built CLI validates it before the unchanged upload steps.

#### Workflow/YAML changes

Replace only the two current validation steps at `go-pre-publish.yml:88-119` with one `release-cli stage --profile go --dist dist` invocation. Resulting order:

1. tag-ref gate (PP-01);
2. checkout (PP-02);
3. mise install and managed-path proof (PP-03/PP-04; add pinned `gh` for installed-path bootstrap);
4. obtain the CLI through `setup-release-cli` (`cli-path` in dogfood, installed path externally);
5. current GoReleaser invocation (PP-05, unchanged until PR 10);
6. `release-cli stage --profile go --dist dist` (PP-06–PP-08);
7. unchanged `oci-build-inputs` upload (PP-09);
8. unchanged `release-assets` upload (PP-10).

The `build-release-cli` job in `.github/workflows/release.yml` is residual dogfood transport: checkout, setup Go, build with tag version/commit/protocol ldflags, upload `release-cli-dogfood`. It is not a new consumer API.

#### Verification

**Laptop:** 

```bash
mise exec -- go test ./internal/stage ./internal/profile/goprof ./internal/cli
mise exec -- moon run root:check
mise exec -- go build -o bin/release-cli ./cmd/release-cli
./bin/release-cli version --json
mise exec -- goreleaser check
```

The version command must emit one envelope with `version`, `commit`, and integer `protocol`; `stage` must emit no file. `git diff` of the PP-09/PP-10 `with:` blocks must be empty apart from line movement.

**CI-only/live:** the `$/` and stamping spikes must already pass; run the cross-repo rehearsal with publication inputs disabled, and run the owning repo's release from a tag through `cli-path`. Observe a valid real `artifacts.json`, two unchanged handoff artifacts, and a populated draft whose assets are named `release-cli_*`.

#### Must NOT contain

No persisted manifest or artifact projection; no PP-09/PP-10 path change; no profile interface/registry; no `.mockery.yml`; no custom FS port; no `internal/rel`, `verify`, `execx`, `clock`, `manfs`, or `e2e`; no config file; no `phase`/`mutations`; no exit `3`; no `cli-version`; no compatibility `release-mvp` binary or alias.

#### Rollback

Do not merge if either spike or rehearsal fails. If a defect appears after merge, revert the squash commit by PR as one atomic unit so configs, command path, action, and workflow return together. Existing consumers remain pinned to the prior full SHA, and any already-started old tag run executes the old workflow from its tag commit. Keep `publish-image:false` and `publish-release:false` in the scratch rehearsal; a failed first dogfood publication leaves the upstream release draft.

---

### PR 2 — `feat(cli): verify Actions artifact handoffs`

#### Scope

- Add the first two genuine ports and Mockery infrastructure.
- Implement `verify handoff --artifact-id --digest`, deriving repository/run/token context from Actions env and failing closed before download.
- Install the composite in the builder and both publishers; pass the dogfood `cli-path` through all four reusable workflows.

#### Files

- Create `.mockery.yml`; modify `mise.toml`/`mise.lock` to pin Mockery `3.7.3`.
- Create `internal/stage/pubgh/doc.go`, `internal/stage/pubgh/handoff.go`, and `internal/stage/pubgh/handoff_test.go` — consumer-owned artifact metadata port and tuple verification.
- Create `internal/adapter/ghact/doc.go`, `client.go`, `client_test.go`, `mocks/doc.go`, and generated `mocks/artifact_meta.go` — go-github adapter and Mockery mock.
- Create `internal/adapter/actenv/doc.go`, `runtime.go`, `runtime_test.go`, `mocks/doc.go`, and generated `mocks/actions.go` — Actions output/annotation adapter.
- Create `internal/cli/handoff.go` and `handoff_test.go`; modify `internal/cli/root.go` — command wiring and Actions output mapping.
- Modify `go.mod`/`go.sum` — add the go-github version selected and recorded by this PR.
- Modify `.github/workflows/go-oci-build.yml`, `publish-github-release.yml`, and `publish-oci-image.yml` — optional `cli-path`, conditional dogfood artifact download, composite setup, and CLI metadata verification.
- Modify `.github/workflows/release.yml` — pass the same `.release-cli/release-cli` path to all four calls.
- Modify `docs/reference/release-cli-contract.md`, `github-release-contract.md`, `oci-image-contract.md`, and `docs/how-to/upgrade-github-release-workflows.md` — command and three-owner handoff contract.

#### Interfaces/signatures

```go
type ArtifactMeta interface {
    Get(ctx context.Context, repository Repository, id ArtifactID) (ArtifactMetadata, error)
}

type Actions interface {
    SetOutput(name OutputName, value string) error
    Warn(message string) error
    AppendSummary(r io.Reader) error
}

func VerifyHandoff(ctx context.Context, meta ArtifactMeta, expected Handoff) (ArtifactMetadata, error)
```

`ArtifactID`, `RunID`, `ArtifactDigest`, `Repository`, `Handoff`, and `ArtifactMetadata` validate positive safe IDs, normalized SHA-256, same-run ownership, and expiry. `ghact.New` accepts an already-authenticated go-github client so raw token text never enters domain output/error formatting.

#### Behavior preserved

- OB-06 (`go-oci-build.yml:106-141`), GR-05 (`publish-github-release.yml:129-164`), and OP-03 (`publish-oci-image.yml:88-123`).
- Invariant 4 remains three-owner: CLI verifies API tuple; pinned `actions/download-artifact` verifies transport digest; later content commands verify extracted data. The CLI never claims to reproduce the Actions ZIP digest.

#### Tests

- **Layer 1:** artifact ID/digest normalization, empty digest, prefix/case normalization, expired artifact, wrong run, nil workflow-run metadata, mismatched digest, context cancellation.
- **Layer 2:** `pubgh` engine with generated `ghact/mocks`; `ghact` against `httptest.Server` exercising go-github JSON and retry-class mapping; `actenv` against `t.TempDir()` `GITHUB_OUTPUT`/summary files, including delimiter-safe multiline output.
- **Layer 3:** all three real artifact metadata calls and their subsequent SHA-pinned download actions in one cross-repo run.

#### Workflow/YAML changes

Replace exactly the three `actions/github-script` metadata blocks cited above. Keep each following download step unchanged: OB-07 `:143-149`, GR-08 `:215-220`, OP-04 `:125-130`. Each affected job's order becomes: existing gates/setup → setup CLI → `verify handoff` → `actions/download-artifact` → existing content/build/publish logic.

All existing `workflow_call` inputs/outputs remain; each workflow gains only optional `cli-path: string`, default `''`. Existing callers that omit it install the stamped released CLI. Dogfood calls conditionally download the fixed same-run `release-cli-dogfood` artifact first.

#### Verification

**Laptop:** 

```bash
mise exec -- mockery
mise exec -- go test ./internal/stage/pubgh ./internal/adapter/ghact ./internal/adapter/actenv ./internal/cli
mise exec -- moon run root:check
```

Generated mocks must be stable after a second Mockery run. **CI-only:** exercise expired/wrong-run/wrong-digest fixtures through focused tests, then observe a live same-run success and a deliberate digest mismatch failing before download in each reusable workflow.

#### Must NOT contain

No content digest recomputation claim; no replacement of `download-artifact`; no registry/release/image ports; no generic retry package; no raw token logging; no new required workflow input/output; no `cli-version`.

#### Rollback

Callers can pin the PR-1 release SHA while PR 2 is reverted. Within the owning workflow, revert all three scripts and `cli-path` plumbing together. Publication inputs remain disabled in rehearsal, so rollback requires no remote cleanup.

---

### PR 3 — `feat(oci): plan immutable release tags`

#### Scope

- Introduce the pure release model only when first used.
- Add native registry state reads, classified not-found/transient/corrupt-state errors, and `plan tags`.
- Land policy ahead of workflow cutover so it can be reviewed and tested independently.

#### Files

- Create `internal/rel/doc.go`, `version.go`, `version_test.go`, `digest.go`, `digest_test.go`, `tag.go`, `tag_test.go`, `secret.go`, and `secret_test.go` — validating immutable release values and redacted `Secret`.
- Create `internal/stage/puboci/doc.go`, `tags.go`, and `tags_test.go` — `StateReader`, fresh-state collection, and pure plan orchestration.
- Create `internal/adapter/reg/doc.go`, `client.go`, `state.go`, `state_test.go`, `mocks/doc.go`, and generated `mocks/state_reader.go` — oras-go v2 read adapter.
- Create `internal/cli/tags.go` and `tags_test.go`; modify `internal/cli/root.go` — `plan tags` and JSON result.
- Modify `.mockery.yml`, `go.mod`, and `go.sum` — add `StateReader` generation and the exact oras-go v2 version proven by the spike.
- Modify `docs/reference/release-cli-contract.md` and `docs/reference/oci-image-contract.md` — stable-version/channel policy and direct-CLI single-writer limitation.

#### Interfaces/signatures

```go
func ParseVersion(value string) (Version, error)
func (v Version) Compare(other Version) int
func (v Version) Tag() Tag
func ParseDigest(value string) (Digest, error)
func ChannelsFor(v Version) []Channel
func PlanTags(v Version, digest Digest, current ChannelState) (TagPlan, error)

type StateReader interface {
    Resolve(ctx context.Context, ref Reference) (rel.Digest, error)
    Version(ctx context.Context, ref Reference) (rel.Version, error)
}
```

`Secret.String`, `Secret.MarshalText`, and `Secret.MarshalJSON` always redact; only adapter composition may unwrap it. Errors must never format the underlying value.

#### Behavior preserved

- OP-10 (`publish-oci-image.yml:251-331`): absent exact tag create, same digest accept, different digest fail.
- OP-11 (`:263-305`): canonical three-component nonnegative stable version comparison without integer overflow.
- OP-12 (`:333-384`): minor/major/latest monotonicity, line scope, newer retention, equal-version/different-digest corruption.
- Invariants 11 and 12. Invariant 13 remains YAML concurrency; document the direct-CLI single-writer limitation.

#### Tests

- **Layer 1:** leading zero, prerelease/build metadata, overflow-sized decimal components, exact absent/same/different, each channel scope, newer/equal/older, corrupt/missing version annotation, deterministic tag order.
- **Layer 2:** engine with `reg/mocks`; `reg` against `go-containerregistry/pkg/registry` via `httptest`, including not found and malformed annotations.
- **Layer 3:** scratch GHCR tag resolution and annotation fetch from the mandatory parity spike.

#### Workflow/YAML changes

None. The existing OP-10–OP-12 script remains authoritative until the complete two-phase path replaces it in PR 5. This avoids a mixed native-plan/ORAS-tag protocol.

#### Verification

**Laptop:** `mise exec -- go test ./internal/rel ./internal/stage/puboci ./internal/adapter/reg ./internal/cli` followed by `mise exec -- moon run root:check`. Observe `plan tags --json` making zero registry writes against the in-memory registry. **CI-only:** rerun the read half of the GHCR parity spike.

#### Must NOT contain

No content push, tag write, prepare/finalize, config file, registry hardcode, persisted plan, `Receipt`, generic retry framework, or additional port.

#### Rollback

No production workflow uses this path yet. Revert the PR without changing release behavior; retain the old YAML while correcting the model.

---

### PR 4 — `feat(oci): prepare digest publication`

#### Scope

- Add digest-addressed OCI layout publication and recursive signing.
- Emit the minimal versioned `OCIPrepareResult`; dry-run validates/plans but performs zero writes and marks the result non-authoritative.
- Keep workflow authority on the old path until finalize exists.

#### Files

- Create `internal/stage/puboci/layout.go`, `prepare.go`, `prepare_test.go`, and `result.go` — exact local descriptor reads over `fs.FS`, preparation engine, and result validation.
- Create `internal/adapter/reg/content.go` and `content_test.go`; add generated `mocks/content_pusher.go` — digest push/resolve implementation.
- Create `internal/adapter/cosign/doc.go`, `signer.go`, `signer_test.go`, `mocks/doc.go`, and generated `mocks/signer.go` — pinned Cosign recursive signer.
- Create `internal/cli/oci.go` and `oci_test.go`; modify `internal/cli/root.go` — `publish oci prepare [--dry-run]`.
- Modify `.mockery.yml`, `docs/reference/release-cli-contract.md`, and `docs/reference/oci-image-contract.md`.

#### Interfaces/signatures

```go
type ContentPusher interface {
    PushBlob(ctx context.Context, repository Repository, descriptor Descriptor, content io.Reader) error
    PushManifest(ctx context.Context, repository Repository, descriptor Descriptor, content io.Reader) error
    Verify(ctx context.Context, ref Reference, expected rel.Digest) error
}

type Signer interface {
    SignRecursive(ctx context.Context, ref Reference) error
}

type OCIPrepareResult struct {
    Schema         string
    Authoritative  bool
    Repository     Repository
    Version        rel.Version
    IndexDigest    rel.Digest
    Platforms      []AttestationSubject
    ObservedState  rel.ChannelState
}

func Prepare(ctx context.Context, input PrepareInput, state StateReader, pusher ContentPusher, signer Signer) (OCIPrepareResult, error)
```

The actual JSON fields are frozen only after the PR's real GHCR experiment, but they may contain only digests, tag-plan inputs/observed state, exact attestation subjects, and `authoritative`. The schema is specific to OCI preparation, not a generic receipt.

#### Behavior preserved

- OP-09: credentials move to an in-memory oras-go client; no login file is written.
- OP-13 (`:386-432`): referenced config/layer deduplication and digest push.
- OP-14 (`:433-467`): platform/index manifest push and digest resolution verification.
- OP-15 (`:469-481`): keyless recursive index/platform signing.
- OP-20 becomes intentionally obsolete because no credential store persists.
- Invariants 8 subject shape, 11/12 pre-write planning, 14's pre-tag half, and 15 zero-write verification mode.

#### Tests

- **Layer 1:** descriptor graph validation, missing blob, digest mismatch, duplicate blob deduplication, result JSON round trip, non-authoritative dry-run.
- **Layer 2:** engine call-order/short-circuit tables with generated mocks; in-memory registry across failure after blob N, after platform manifest, and after index push; real Cosign command construction test without OIDC.
- **Layer 3:** scratch GHCR digest push, resolution, recursive signature/referrer discovery, and retry after injected partial failure.

#### Workflow/YAML changes

None; the command is available for the acceptance spike and manual verification. Old OP-09–OP-19 remains until PR 5 so invariant 14 is never temporarily weakened.

#### Verification

**Laptop:** `mise exec -- go test ./internal/stage/puboci ./internal/adapter/reg ./internal/adapter/cosign ./internal/cli`, `mise exec -- moon run root:check`, and an in-memory-registry CLI smoke showing `--dry-run` leaves zero repositories/tags. **CI-only:** scratch GHCR prepare must show index plus both platform signatures/referrers and no public tag mutation.

#### Must NOT contain

No tag mutation; no prepare-result file; no persistence port; no stale `TagPlan` replay; no YAML attestation replacement; no `mutations`/exit `3`; no ORAS fallback unless the spike selected it behind the same ports.

#### Rollback

No workflow cutover exists. Revert the PR; scratch GHCR content is digest-addressed and untagged, so it is nonpublic garbage eligible for normal registry retention/explicit scratch cleanup.

---

### PR 5 — `feat(oci): finalize trusted image tags`

#### Scope

- Add fresh-state finalize, drift refusal, serial verified tag commits, and full two-phase workflow cutover.
- Re-feed the prepare JSON envelope through stdin; keep all three `actions/attest` calls job-level.

#### Files

- Create `internal/stage/puboci/finalize.go` and `finalize_test.go`.
- Create `internal/adapter/reg/tag.go` and `tag_test.go`; add generated `mocks/tag_committer.go`.
- Modify `internal/cli/oci.go`/`oci_test.go`, `internal/cli/root.go`, and `.mockery.yml` — `publish oci finalize --result -`.
- Modify `.github/workflows/publish-oci-image.yml` — replace OP publication scripts, preserve permissions/concurrency/attest steps and existing outputs.
- Create `docs/explanation/two-phase-oci-publication.md`; modify `docs/reference/release-cli-contract.md`, `docs/reference/oci-image-contract.md`, and `docs/how-to/configure-oci-images.md`.

#### Interfaces/signatures

```go
type TagCommitter interface {
    Commit(ctx context.Context, repository Repository, digest rel.Digest, tags []rel.Tag) error
}

func Finalize(ctx context.Context, prepared OCIPrepareResult, state StateReader, committer TagCommitter) (FinalizeResult, error)
```

`Finalize` rejects a non-authoritative result, schema mismatch, subject/digest inconsistency, or any fresh remote-state drift from preparation. It recomputes `PlanTags` from the fresh state; it never applies a serialized stale plan. `TagCommitter` applies tags serially and verifies each result; the engine independently verifies the exact tag postcondition.

#### Behavior preserved

- Cuts over OP-09–OP-15 and OP-19/OP-20 (`publish-oci-image.yml:234-481,511-569`).
- OP-16–OP-18 (`:483-510`) remain pinned `actions/attest` steps between prepare and finalize.
- Invariants 8, 11, 12, 13, 14, and 15. Workflow repository-wide concurrency and no-cancel policy remain unchanged (invariants 13, 16).

#### Tests

- **Layer 1:** envelope decode from stdin, wrong schema, `authoritative:false`, altered digest/subject/state, tag ordering.
- **Layer 2:** full prepare→finalize over in-memory registry; generated-mock tests for attestation-boundary short-circuit (finalize is never called by workflow when any attest step fails), fresh drift, half-completed tag rerun, ambiguous write resolved by read, and newer channel preservation.
- **Layer 3:** real GHCR run with index provenance, amd64 SBOM, arm64 SBOM, Cosign recursive signatures, then tags; rerun same tag/digest; inject one tag failure and prove fresh reconciliation.

#### Workflow/YAML changes

Replace current login/plan/push/sign/tag/logout blocks. Resulting job order:

1. stable tag gate;
2. mise setup for pinned `gh` and Cosign (ORAS only if fallback selected);
3. setup CLI;
4. `verify handoff`;
5. unchanged `download-artifact`;
6. existing extracted OCI content/digest validation (until slice 5 shares the final parser);
7. `publish oci prepare --json`; `actenv` writes internal outputs including the complete JSON envelope and attestation subjects;
8. index provenance `actions/attest`;
9. amd64 SBOM `actions/attest`;
10. arm64 SBOM `actions/attest`;
11. pass the step output through an environment variable to stdin and run `publish oci finalize --result -` only when publication is enabled.

For `publish-image:false`, call `prepare --dry-run`, skip all attest/finalize steps, and preserve existing `image-name`/`image-digest` outputs with empty publication/attestation outputs. Never write the result to a receipt file.

#### Verification

**Laptop:** `mise exec -- go test ./internal/rel ./internal/stage/puboci ./internal/adapter/reg ./internal/cli` and `mise exec -- moon run root:check`. **CI-only:** the mandatory scratch GHCR rehearsal must inspect registry referrers and tag timestamps/resolutions to prove trust metadata precedes tags; all six existing workflow outputs must retain their current meanings.

#### Must NOT contain

No action-based tag mutation before attestations; no file receipt; no supported result compatibility across independently pinned CLI versions; no parallel tag writes; no persisted credentials; no generic idempotency claim; no extra output names at `workflow_call` level.

#### Rollback

Set `publish-image:false` to retain verification-only behavior while diagnosing. Pin external callers to the PR-4/previous released SHA or revert the workflow/CLI PR together. A failed prepare leaves only digest-addressed content/trust metadata; a failed finalize is reconciled from fresh state before any retry and must not be manually replayed from saved JSON.

---

### PR 6 — `feat(release): verify signed release bundles`

#### Scope

- Upgrade the producer checksum claim into a closed `Bundle` only after scanning the downloaded directory.
- Add exact Sigstore identity/issuer verification and replace GR-09–GR-12 with `verify bundle`.

#### Files

- Create `internal/stage/pubgh/bundle.go` and `bundle_test.go` — closed-set reconciliation over `fs.FS` and `BlobVerifier` port.
- Create `internal/adapter/cosign/verifier.go` and `verifier_test.go`; add generated `mocks/blob_verifier.go`.
- Create `internal/cli/bundle.go` and `bundle_test.go`; modify `internal/cli/root.go` and `.mockery.yml`.
- Modify `.github/workflows/publish-github-release.yml` — CLI bundle verification and output mapping; keep GR-13.
- Modify `docs/reference/release-cli-contract.md`, `docs/reference/github-release-contract.md`, and `docs/how-to/configure-github-releases.md`.

#### Interfaces/signatures

```go
type BlobVerifier interface {
    Verify(ctx context.Context, request BlobVerification) error
}

func BuildBundle(fsys fs.FS, claim stage.ChecksumSet) (Bundle, error)
func VerifyBundle(ctx context.Context, fsys fs.FS, verifier BlobVerifier, trust TrustPolicy) (Bundle, error)
```

`Bundle` is constructible only after a real directory scan proves flat unique names, regular files, all payload digests, both controls, and no extra entry. `TrustPolicy` validates the exact workflow certificate identity and `https://token.actions.githubusercontent.com` issuer.

#### Behavior preserved

- GR-09–GR-12 (`publish-github-release.yml:222-333`) and invariant 7.
- GR-13 stays immediately after successful local verification, preserving invariant 8's `subject-checksums` shape and invariant 9's verify→attest→upload order.

#### Tests

- **Layer 1:** empty/invalid manifest reuse from slice 1; closed-set extra entry, directory, symlink, missing payload/control, duplicate control, changed payload, and deterministic path/digest output.
- **Layer 2:** `t.TempDir()` scans and generated `cosign/mocks`; real Cosign `verify-blob` argument test using a fixed nonsecret bundle where feasible.
- **Layer 3:** live signer identity/issuer verification and GitHub attestation subjects in the consumer repository.

#### Workflow/YAML changes

Replace only `publish-github-release.yml:222-333` with `release-cli verify bundle --dist dist --json`. Through `actenv`, keep the existing internal step-output meanings `assets` and `digests` so the still-YAML upload/convergence scripts continue to work. Keep `actions/attest@1e69…` at current GR-13 directly afterward.

#### Verification

**Laptop:** `mise exec -- go test ./internal/stage ./internal/stage/pubgh ./internal/adapter/cosign ./internal/cli` and `mise exec -- moon run root:check`. **CI-only:** a draft rehearsal shows validation, attestation, then upload; a wrong signer identity fails before GR-13 and before any release asset mutation.

#### Must NOT contain

No GitHub release mutation; no App private key in CLI; no stateful GitHub fake unless its trigger fires; no controls listed as checksum subjects; no file-size field without an invariant; no custom filesystem adapter.

#### Rollback

With `publish-release:false`, the release remains draft. Revert to the prior SHA/YAML verifier if the live Cosign contract fails; do not weaken identity/issuer or closed-set checks to force acceptance.

---

### PR 7 — `feat(release): publish verified GitHub releases`

#### Scope

- Implement the draft release state machine with official YAML App-token minting.
- Preserve exact tag/SHA binding, expected-name-only clobber via `gh release upload --clobber`, convergence, optional undraft, and explicit post-undraft indeterminate behavior.

#### Files

- Create `internal/stage/pubgh/publish.go`, `publish_test.go`, and `errors.go` — ports, bounded poll policies, state machine, reconciliation.
- Create `internal/adapter/ghrel/doc.go`, `reader.go`, `publisher.go`, tests, `mocks/doc.go`, and generated `mocks/release_reader.go`/`publisher.go`.
- Create `internal/adapter/ghup/doc.go`, `replacer.go`, `replacer_test.go`, `mocks/doc.go`, and generated `mocks/asset_replacer.go`.
- Create `internal/adapter/gitx/doc.go`, `resolver.go`, `resolver_test.go`, `mocks/doc.go`, and generated `mocks/ref_resolver.go`.
- Create `internal/cli/github.go` and `github_test.go`; modify `internal/cli/root.go` and `.mockery.yml`.
- Modify `.github/workflows/publish-github-release.yml` — CLI remote publication, unchanged App token/download/attest barriers and outputs.
- Modify `docs/reference/release-cli-contract.md`, `docs/reference/github-release-contract.md`, `docs/how-to/configure-github-releases.md`, and `docs/how-to/rehearse-and-recover-github-releases.md`.

#### Interfaces/signatures

```go
type ReleaseReader interface {
    FindDraft(ctx context.Context, repository Repository, tag rel.Tag, policy PollPolicy) (Release, error)
    WaitAssets(ctx context.Context, repository Repository, release ReleaseID, policy PollPolicy) (AssetsView, error)
    Get(ctx context.Context, repository Repository, release ReleaseID) (Release, error)
}

type AssetReplacer interface {
    Replace(ctx context.Context, repository Repository, tag rel.Tag, expected []AssetPath) error
}

type Publisher interface {
    Publish(ctx context.Context, repository Repository, release ReleaseID) error
}

type RefResolver interface {
    Resolve(ctx context.Context, tag rel.Tag) (CommitSHA, error)
}

func Publish(ctx context.Context, input PublishInput, reader ReleaseReader, replacer AssetReplacer, publisher Publisher, resolver RefResolver, sleep SleepFunc) (PublishResult, error)
```

Use the approved injected `sleep(ctx,d)` function; no clock package. Define slice-specific classified errors for no draft, unexpected asset, transient/rate-limited read, ambiguous mutation, and indeterminate already-public state (E1, E3).

#### Behavior preserved

- GR-06 (`:166-199`): paginated exact-tag discovery, 24×5-second budget, draft-only start (invariant 1).
- GR-07 (`:200-213`): tag resolves to `github.sha` (invariant 2).
- GR-08 (`:215-220`): explicitly remains the SHA-pinned download/digest owner; it is accounted for but not falsely moved.
- GR-14–GR-18 (`:341-482`): unique selected release, unexpected-name refusal, exact `--clobber`, 12×1-second convergence, duplicate/count/digest/state checks, optional undraft and final fetch.
- Invariants 1, 2, 9, and 10; invariant 3 remains caller `needs`/`require-oci-image` YAML.

#### Tests

- **Layer 1:** release state transitions and post-undraft reconciliation, poll exhaustion/cancellation, disabled undraft.
- **Layer 2:** generated mock ordering and short-circuit tests; `ghrel` against `httptest` for pagination/rate/error mapping; `gitx` against a `t.TempDir()` repository; `ghup` command contract. Required cases: 422/name collision, delete-success/upload-fail behavior embodied by `--clobber`, duplicate remote names, unexpected-name refusal, half-upload rerun, final fetch timeout after successful undraft.
- Do not build a reusable stateful GitHub fake unless focused mocks become unreadable across repeated pagination/eventual-consistency/clobber transitions. If that observable trigger fires, stop and revise the PR plan before adding it.
- **Layer 3:** live draft appearance delay, asset digest convergence, clobber recovery, `publish-release:false`, then a controlled public rehearsal.

#### Workflow/YAML changes

Remove the draft/tag script at `:166-213` and upload/finalize scripts at `:341-482`. Keep order:

1. tag/required-OCI gate;
2. checkout;
3. pinned gh/Cosign setup/proof;
4. `actions/create-github-app-token` (private key never enters CLI);
5. `verify handoff`;
6. `download-artifact` GR-08;
7. `verify bundle`;
8. `actions/attest` GR-13;
9. `publish github [--no-undraft]` using the short-lived App token in an environment variable consumed as redacted `Secret`;
10. map CLI `release-url` to the unchanged workflow output.

#### Verification

**Laptop:** `mise exec -- go test ./internal/stage/pubgh ./internal/adapter/ghrel ./internal/adapter/ghup ./internal/adapter/gitx ./internal/cli` and `mise exec -- moon run root:check`. **CI-only:** live rehearsal first with `publish-release:false`, then a synthetic release with publication enabled after OCI success; observe no create/re-draft call and no deletion of unexpected assets.

#### Must NOT contain

No native App-token minting; no release creation; no deletion of unexpected assets; no broad `GitHubService` port; no blanket safe-rerun claim; no clock package; no public workflow input/output change.

#### Rollback

Set `publish-release:false` for rehearsal/containment. Before undraft, repin/revert and rerun from the authoritative artifact. After an ambiguous undraft, do not rerun blindly: inspect the release and use the CLI's exact-converged-public success/indeterminate result. External consumers can pin the previous full SHA.

---

### PR 8 — `feat(image): build OCI layouts from staged binaries`

#### Scope

- Fire architecture decision 3's persisted-facts trigger: the first downstream CLI command now reads staged facts.
- Persist **only** an `oci-build-inputs` projection inside that artifact; add `image build` with Melange/apko adapters.
- Leave the old OB-18–OB-21 verifier in place as an independent migration oracle.

#### Files

- Create `internal/stage/image_inputs.go` and `image_inputs_test.go` — versioned codec over `io.Reader`/`io.Writer` for `dist/oci-build-inputs.json`; contains only profile, two neutral platform/name/confined-path/digest facts needed by image build.
- Modify `internal/profile/goprof/artifacts.go`/tests and `internal/cli/stage.go`/tests — produce the projection after all slice-1 validation succeeds.
- Create `internal/stage/image/doc.go`, `build.go`, and `build_test.go` — image build engine and two approved ports.
- Create `internal/adapter/melange/doc.go`, `builder.go`, `builder_test.go`, `mocks/doc.go`, and generated `mocks/apk_builder.go`.
- Create `internal/adapter/apko/doc.go`, `composer.go`, `composer_test.go`, `mocks/doc.go`, and generated `mocks/composer.go`.
- Create `internal/cli/image.go` and `image_test.go`; modify `internal/cli/root.go` and `.mockery.yml`.
- Modify `.github/workflows/go-pre-publish.yml` — PP-09 additionally uploads `dist/oci-build-inputs.json`; PP-10 remains unchanged.
- Modify `.github/workflows/go-oci-build.yml` — replace OB-08–OB-17 with `image build`, leave OB-18–OB-21 and OB-22.
- Modify `docs/reference/release-cli-contract.md`, `docs/reference/oci-image-contract.md`, and `docs/how-to/configure-oci-images.md`.

#### Interfaces/signatures

```go
type APKBuilder interface {
    Build(ctx context.Context, request APKBuildRequest) (APKRepositories, error)
}

type Composer interface {
    Build(ctx context.Context, request ComposeRequest) error
}

func EncodeImageInputs(w io.Writer, inputs ImageInputs) error
func DecodeImageInputs(r io.Reader) (ImageInputs, error)
func Build(ctx context.Context, input BuildInput, apk APKBuilder, composer Composer) (BuildResult, error)
```

No GoReleaser-specific record leaks into `image`; composition converts validated `goprof.CanonicalBinary` facts into the neutral projection. The projection references only files present in `oci-build-inputs`, never release archives/SBOMs.

#### Behavior preserved

- OB-08 (`go-oci-build.yml:150-177`) workspace/config/required-file setup.
- OB-09–OB-11 (`:178-245`) exact inputs, confinement, static 64-bit ELF and common name, 0755 install, version/build-date/config copies/canonical hashes.
- OB-12–OB-15 (`:247-284`) Melange compile/keygen/build/provenance and exactly-one-APK checks.
- OB-16–OB-17 (`:285-314`) apko lock/build, two platforms, SBOMs/annotations.
- Invariants 4 (artifact-local projection), 5, and 6 remain checked by the unchanged YAML verifier.

#### Tests

- **Layer 1:** projection schema/round trip; exactly two platforms; no absent-file references; ELF class/machine/static checks using small fixtures; deterministic metadata/argument derivation.
- **Layer 2:** build engine with generated mocks; Melange/apko exact argument and working-directory contracts; `t.TempDir()` output layout. No handwritten mocks.
- **Layer 3:** pinned Melange 0.59.1/apko 1.2.37, Docker/QEMU, real signed APK repositories, and the unchanged YAML OB-18–OB-21 verifier.

#### Workflow/YAML changes

`go-pre-publish`: `stage` now writes `dist/oci-build-inputs.json` only because its first reader exists; include it only in PP-09. `go-oci-build` order becomes tag/checkout/mise/QEMU/tool proof → `verify handoff` → download → `image build` → unchanged YAML `Verify authoritative OCI image` → unchanged upload. All workflow inputs/outputs stay unchanged.

#### Verification

**Laptop:** `mise exec -- go test ./internal/stage ./internal/profile/goprof ./internal/stage/image ./internal/adapter/melange ./internal/adapter/apko ./internal/cli` and `mise exec -- moon run root:check`. **CI-only:** real builder output must pass the old YAML verifier without editing that verifier in this PR.

#### Must NOT contain

No release-assets projection; no universal staging manifest; no `manfs`; no custom layout/filesystem port; no `execx`; no image verification replacement; no alternate platform/multiple-binary seam.

#### Rollback

Repin/revert both producer projection and builder consumption together. The prior artifact shape remains supported by prior SHA-pinned workflows; do not make the new builder silently infer missing projection data.

---

### PR 9 — `feat(image): verify OCI image contracts`

#### Scope

- Replace the final deep shell verifier with exact-byte Go parsing and `image verify`.
- Preserve index bytes verbatim and prove no rebuild of canonical executables.

#### Files

- Create `internal/stage/image/layout.go`, `layout_test.go`, `verify.go`, and `verify_test.go` — OCI descriptor/config/layer/SBOM validation over `fs.FS`.
- Add focused synthetic fixtures under `internal/stage/image/testdata/` for a valid two-platform layout and mutation helpers in tests.
- Modify `internal/cli/image.go`/`image_test.go` and `internal/cli/root.go` — `image verify` and `image-digest` Actions output.
- Modify `.github/workflows/go-oci-build.yml` — replace OB-18–OB-21, retain upload.
- Modify `docs/reference/release-cli-contract.md`, `docs/reference/oci-image-contract.md`, and `docs/how-to/configure-oci-images.md`.

#### Interfaces/signatures

```go
func ReadLayout(fsys fs.FS) (Layout, error)
func VerifyLayout(fsys fs.FS, expected ExpectedImage) (VerifiedImage, error)
func (image VerifiedImage) IndexDigest() rel.Digest
```

`Layout` retains the exact original `index.json` byte slice and parsed view; digest computation hashes the original bytes, never re-marshaled JSON. No new port is introduced.

#### Behavior preserved

- OB-18 (`go-oci-build.yml:315-355`) index schema/media type/platform/annotations.
- OB-19 (`:356-405`) manifest/config/layer/entrypoint/user/labels.
- OB-20 (`:406-423`) mode `0755`, owner `0/0`, extracted executable digest equality.
- OB-21 (`:424-436`) per-arch SPDX application `${VERSION}-r0` and exact index digest/output.
- Invariants 5 and 6; OP-06 can later reuse exact-byte logic without a custom adapter.

#### Tests

- **Layer 1/2 local:** table-driven mutation of platform count/OS/arch, media types, annotations, layer count, entrypoint, user, labels, mode, owner, canonical bytes, SBOM package/version, missing descriptor blob, invalid digest, and whitespace-only change to `index.json` proving exact-byte hash changes. Use `fstest.MapFS` for pure layouts and `t.TempDir()` for tar modes/ownership.
- **Layer 3:** real two-platform output and both SBOMs on the hosted runner.

#### Workflow/YAML changes

Replace only `go-oci-build.yml:315-436` with `release-cli image verify ...`; keep OB-22 `actions/upload-artifact` unchanged. The CLI writes the same `image-digest` step output consumed by the existing job/workflow output mapping.

#### Verification

**Laptop:** `mise exec -- go test ./internal/stage/image ./internal/cli` and `mise exec -- moon run root:check`. **CI-only:** compare CLI digest to `sha256sum oci-output/layout/index.json` and observe a real image pass, then a deliberate layer-binary mutation fail.

#### Must NOT contain

No JSON re-marshaling for digest, no source rebuild, no additional platform, no custom tar/layout port, no image publication change, no trigger-gated generic layout adapter.

#### Rollback

Revert the verifier step to the prior YAML at the previous full SHA. OB-22 uploads only after verification, so a failed rehearsal publishes no OCI artifact.

---

### PR 10 — `refactor(prepublish): run GoReleaser through release-cli`

#### Scope

- Move PP-05 GoReleaser invocation into `goprof` without adding a fourteenth port.
- Leave PP-04's managed-tool provenance proof in YAML and preserve both publication-disable controls.
- Perform final thin-shell cleanup and full program rehearsal.

#### Files

- Create `internal/profile/goprof/goreleaser.go` and `goreleaser_test.go` — pure argument construction plus the minimal `exec.CommandContext` boundary in the profile package.
- Modify `internal/cli/stage.go`/tests — `stage --profile go` invokes GoReleaser, then performs existing validation/projection.
- Modify `.github/workflows/go-pre-publish.yml` — split tool-path proof from CLI invocation; remove direct `goreleaser release` shell line.
- Modify `docs/reference/release-cli-contract.md`, `docs/reference/github-release-contract.md`, `docs/reference/oci-image-contract.md`, all four files under `docs/how-to/`, and `examples/go-release/README.md` — final command ownership and residual-YAML description.
- Create `docs/tutorials/release-a-go-project.md` — guided use of the copyable example.
- Create `docs/explanation/release-trust-boundaries.md` — why workflows, action, and CLI have distinct responsibilities; link rather than mix this explanation into how-to/reference pages.

#### Interfaces/signatures

```go
type GoReleaserOptions struct {
    Dist string
}

func RunGoReleaser(ctx context.Context, options GoReleaserOptions) error
```

The invoked contract is exactly `goreleaser release --clean --skip=publish`. Argument construction is separately testable; use direct `exec.CommandContext` here because the approved port list intentionally contains no GoReleaser port. Do not extract `execx` unless its explicit trigger has fired and architecture is amended first.

#### Behavior preserved

- PP-05 (`go-pre-publish.yml:74-86`) and invariant 17: command has `--skip=publish`, and `.goreleaser.yaml` still has `release.disable:true`; changelog remains disabled because Release Please owns it.
- PP-03/PP-04 tool installation/path proof and PP-09/PP-10 transport remain YAML.

#### Tests

- **Layer 1:** exact argument list, context cancellation, nonzero subprocess error wrapping without environment/token leakage.
- **Layer 2:** pinned local `goreleaser check` and snapshot fixture refresh; stage continues to exercise real generated JSON through the local file substrate.
- **Layer 3:** GoReleaser 2.17.1 with Go/Syft/Cosign/OIDC in the full tag rehearsal; observe no GoReleaser release publication.

#### Workflow/YAML changes

Final `go-pre-publish` order: tag gate → checkout → mise install → exact managed-path/version proof → setup CLI → `release-cli stage --profile go --dist dist` (now includes PP-05–PP-08) → unchanged two uploads. The other three workflows are already at their final thin-shell order described in PRs 5, 7, and 9.

#### Verification

**Laptop:** 

```bash
mise exec -- goreleaser check
mise exec -- go test ./internal/profile/goprof ./internal/cli
mise exec -- moon run root:check
```

**CI-only:** run the complete external consumer rehearsal with first `publish-image:false`/`publish-release:false`, then the controlled publication path. Inspect logs to prove the managed GoReleaser executable ran, `--skip=publish` was present, and no release API call came from GoReleaser.

#### Must NOT contain

No GoReleaser port, generic runner, `execx`, profile registry, second profile, config file, alternate skip policy, or workflow removal of tool-path proofs.

#### Rollback

Revert PP-05 to the prior shell invocation and repin callers while retaining the same `.goreleaser.yaml`. Because both old and new invocations use `--skip=publish`, rollback cannot create a second release owner.

---

### PR 11 — `docs(release): pin the completed release unit`

#### Scope

- After PR 10's release is public and verified, make the documentation/example copy-paste surface concrete and immutable.

#### Files

- Modify `README.md`.
- Modify every Markdown file under `docs/how-to/`, `docs/reference/`, `docs/tutorials/`, and `docs/explanation/` that contains `FULL_SHA` or the previous `fb8c8098…` pin.
- Modify `examples/go-release/.github/workflows/release.yml` and `examples/go-release/README.md` — all four workflow refs and checksum signer identity use the same released full SHA.
- No Go, workflow implementation, action, or release configuration file changes.

#### Interfaces/signatures

None.

#### Behavior preserved

No inventory step migrates here. The documented contracts are exactly the behavior merged in PRs 1–10, and the old published SHA remains valid for existing consumers.

#### Tests

- **Layer 1/docs:** search proves no `FULL_SHA`, `release-mvp`, or stale `fb8c8098…` remains where it claims to be current.
- **Layer 3:** copy the example into the scratch consumer and run its disabled-publication rehearsal at the pinned SHA.

#### Workflow/YAML changes

Only the example caller changes refs. The production workflows do not.

#### Verification

**Laptop:** `mise exec -- moon run root:check` plus a repository search for all reusable `uses:` and `checksum-signing-workflow-ref` values; all must equal the released commit. **CI-only:** `gh release view v<version> --repo meigma/release`, `gh attestation verify` for a `release-cli` archive, and OCI signature/attestation verification against the corresponding image digest.

#### Must NOT contain

No floating tag/branch pin, no second CLI pin, no behavior/code cleanup, and no claim that the unproven provenance predicate binds the source commit beyond “signed + stamp-matched.”

#### Rollback

Revert only the documentation pin if the release is withdrawn; point examples back to the last verified full SHA. No runtime rollback is needed.

## 4. The `release-mvp` → `release-cli` transition

This transition rides **PR 1**, not a separate PR. A rename-only PR would either introduce `cmd/release-cli` before the four-package slice or temporarily keep both commands, contradicting architecture §5/§11's exact slice-1 package set and clean-cutover rule. Atomicity is also safer: old tag commits continue to run the old `release-mvp` configuration, while the first new tag commit contains the CLI, action stamp, workflows, and release configuration together.

PR 1 performs this exact order inside one change set:

1. Implement and verify `cmd/release-cli` plus the three internal packages; delete `cmd/release-mvp` and `greet` tests.
2. Change `moon.yml` build from `bin/release-mvp ./cmd/release-mvp` to `bin/release-cli ./cmd/release-cli`.
3. Change `.goreleaser.yaml` project/build/archive IDs, main path, binary, and archive prefix to `release-cli`; preserve six OS/arch archives, SBOMs, checksum signing, `changelog.disable`, and `release.disable`.
4. Change `melange.yaml` package/install path and `apko.yaml` package/entrypoint/title to `release-cli` so the repository's OCI artifact runs the same binary.
5. Change `release-please-config.json` root package name to `release-cli`; retain one root package, `include-component-in-tag:false`, and the existing manifest key/version. There is no version reset or second component.
6. Add action stamping. The composite contains a Release Please-updated semantic version field at the spike-proven YAML jsonpath. The binary receives the same semantic version from GoReleaser `-X main.version={{ .Version }}`. The protocol starts at integer `1`: the action's expected integer and `main.protocol`/GoReleaser ldflag are compared by CI and at runtime. If Release Please can update both literals directly, configure both under `extra-files`; otherwise use the fallback in §5 (version via `extra-files`, protocol duplicate guarded by a CI equality check). No consumer controls either stamp.
7. Make `.github/workflows/release.yml` build the current tag's CLI and transport it as the internal `release-cli-dogfood` artifact; every reusable job passes the extracted binary to the composite through `cli-path`. This bootstraps the first release without requiring a preexisting `release-cli` release.
8. Run the cross-repo rehearsal with public mutations disabled, then merge PR 1.
9. Inspect the Release Please release PR: action version, changelog, unchanged manifest history, `v` tag/draft behavior. Merge it only when the stamp is correct.
10. The tag run dogfoods the branch-built `release-cli` to build/sign/package/publish `release-cli` itself. Existing in-flight tags are unaffected because Actions uses the workflow/config from their old tag commits.
11. Do not update consumer examples to the not-yet-known squash/tag SHA. Behavioral docs land in PR 1 using `FULL_SHA`; PR 11 pins the verified release commit.

The protocol is a mismatch guard, not negotiation: no range, compatibility matrix, or deprecation policy.

## 5. Spikes as gates

### A. `$/` self-reference under an external SHA-pinned caller

**Proves:** a reusable workflow called as `owner/repo/.github/workflows/...@FULL_SHA` can invoke `uses: $/.github/actions/setup-release-cli` from the same exact commit on Actions Runner ≥2.336.0; `github.action_repository` identifies the release repository while ordinary `github.repository` remains the consumer.

**Blocks:** PR 1.

**Procedure:**

1. Create a scratch branch/commit in the release repository containing a minimal composite that prints nonsecret action identity and a reusable workflow using `$/`.
2. In an unrelated scratch consumer, call that workflow at the full commit SHA with least-privilege permissions.
3. Use `ubuntu-24.04`; record the runner version from the job setup log and require ≥2.336.0.
4. Have the composite output its embedded marker, `github.action_repository`, and action path. Assert the marker is from the pinned commit, not `main`, and no checkout is required merely to load the action.
5. Rerun failed job and whole workflow; both must resolve the intended code under GitHub's documented rerun rules.

**Pass:** exact-commit marker, correct action repository, external caller context preserved, no syntax/runner failure.

**Result — PASSED 2026-08-18; PR 1 is unblocked.** Callee `meigma/release@a47ed6648850c73b91d31f794fbd443e6beffff6` (branch `spike/self-ref`), external caller `meigma/release-selfref-spike` (archived), runner `2.336.0` on `ubuntu-24.04`, `permissions: {}` at workflow and job level. Evidence: `Download action repository 'meigma/release@a47ed66…'`; `github.action_repository=meigma/release` while `github.repository=meigma/release-selfref-spike`; `github.action_ref` equals the pinned full SHA; `GITHUB_WORKSPACE` empty (0 entries — no checkout needed); callee workflow outputs propagated to the caller. Pin immunity: after the branch tip moved to `c47415f6480fd9101f222b0f717031991601acd2` (marker `MARKER-B`), the caller still pinned at `a47ed66…` resolved `MARKER-A` (run 32191526616), and a caller pinned at `c47415f…` resolved `MARKER-B` (run 32191534462). Rerun-failed-job on attempt 2 re-resolved the same pinned commit and `MARKER-A`. Runs: 32191413160, 32191526616, 32191534462 in `meigma/release-selfref-spike`.

**Fallback (corrected — the original is not implementable):** `github.job_workflow_ref` and `github.job_workflow_sha` were observed to evaluate to `null` inside the called reusable workflow's expression context, and `github.workflow_ref`/`github.workflow_sha` report the *caller's* workflow and commit (`meigma/release-selfref-spike/.github/workflows/call-spike-b.yml@refs/heads/main`, `477325db…`), not the callee's. Those two values exist as OIDC token claims, not as usable `github` context properties, so a checkout-the-callee fallback cannot derive the callee commit from them without minting an OIDC token (which would require `id-token: write` purely for plumbing). If `$/` ever regresses, the fallback is instead: add an explicit callee-ref input that the caller sets to the same full SHA it pins, check the callee out at that SHA, and invoke the composite by local path — still no hardcoded `meigma/release`, same action, same CLI, same ports. This fallback is unused while `$/` passes.

### B. oras-go v2 GHCR parity

**Proves:** the `reg` adapter can preserve the current ORAS 1.3.3 contract on GHCR: credentialed resolve, annotation fetch, digest blob/manifest push, digest resolution, exact/channel tags, Cosign recursive referrers, coexistence with three `actions/attest` referrers, and recovery after partial failure.

**Blocks:** PR 3 and all slice-3 merges.

**Procedure:**

1. In a scratch repository/package, generate the same two-platform OCI layout and SBOM subjects used by the workflow.
2. Use a short-lived Actions token with `packages:write`, `id-token:write`, and `attestations:write`; pass credentials only in memory.
3. Against a unique synthetic stable version, verify absent resolve; push referenced blobs, platform manifests, and index by digest; resolve each digest.
4. Run pinned Cosign recursive signing and inspect referrers for the index and both platform manifests.
5. Run the three pinned `actions/attest` calls and confirm all referrers coexist and verify.
6. Apply exact/minor/major/latest serially; fetch version annotations and verify line/monotonic behavior.
7. Repeat with failpoints after one blob, one platform manifest, index push, and one tag; rerun from fresh reads and verify no wrong tag or duplicate corruption.
8. Record response/error shapes needed for not-found, auth, 429/5xx, and ambiguous writes; select and pin the exact oras-go v2 module version in `go.mod`.

**Pass:** byte/digest parity, referrer coexistence, all postconditions, and recoverable partial failures without persisted credentials.

**Result — PASSED 2026-08-18; PR 3 and PR 4 are unblocked, with two contract corrections.** `oras-go` v2 **v2.6.2** pinned. Scratch repo `meigma/release-oras-spike` (archived), package `ghcr.io/meigma/release-oras-spike`, run 32193386872 green end to end after three corrected attempts.

Proven against real GHCR, in this order: exact tag absent before publication; a deliberately aborted graph copy leaves content addressable but **no tag** (so nothing is consumer-visible); re-push converges over already-present blobs; the index resolves by digest to exactly the expected digest; both platform manifests resolve by digest; the index version annotation is readable for channel planning; publication completed with **no tag applied** (`digest-published-without-public-tag`, class `absent`); then `cosign sign --recursive` plus three `actions/attest` calls (one provenance on the index, one SPDX SBOM per platform, all `push-to-registry: true`); `cosign tree` then showed **coexisting referrers** — index carries `sigstore.dev/cosign/sign/v1` + `slsa.dev/provenance/v1`, each platform manifest carries `sigstore.dev/cosign/sign/v1` + `spdx.dev/Document/v2.3`; only then were `v9.9.12`, `9.9`, `9`, `latest` applied serially and verified; re-tagging the same digest was accepted; `cosign verify` and `gh attestation verify oci://…` both passed **after** tagging. Invariant 14 is therefore enforceable exactly as the two-phase protocol assumes, and the local (`-mode=local`, in-process registry) run produced an identical index digest to the GHCR run, so layout digests are deterministic across substrates.

**Corrections this spike forces:**

1. **Registry login is not obsolete.** `actions/attest --push-to-registry` fails with `No credentials found for registry ghcr.io` unless a docker config exists, and `cosign` is a separate process from the CLI. PR 4 keeps one login step — `cosign login ghcr.io --username … --password-stdin` replaces the `oras` binary — and PR 4's cleanup is a docker-config edit because **`cosign` has no `logout` subcommand** (`unknown command "logout" for "cosign"`). `cosign --registry-username/--registry-password` works per invocation for `sign`/`verify`/`tree`, but does not help the attest actions.
2. **`oras.CopyGraph` is concurrent by default** (`DefaultCopyGraphOptions` sets `Concurrency: 3`), so partial-failure abort points are not strictly ordered. The `ContentPusher` adapter must either set `Concurrency: 1` where deterministic recovery reporting matters or treat the pushed-set as unordered; `TagCommitter` must remain strictly serial regardless.
3. **`actions/attest` validates SBOM payloads.** A loose JSON blob is rejected with `Unsupported SBOM format. Must be valid SPDX or CycloneDX JSON`, so PR 8/9 fixtures must be schema-valid SPDX, not plausible-looking JSON.

Also observed, benign: `actions/attest` logs `Failed to create storage record: no artifacts found` in a scratch repo with no build artifacts; attestations were still created and verifiable (`attestations/41481491`, `41481498`, `41481504`).

**Fallback (unused):** retain the pinned ORAS binary behind the **same** `StateReader`, `ContentPusher`, and `TagCommitter` ports. Planner/prepare/finalize behavior and tests remain unchanged; only `internal/adapter/reg` shells out. No fourth registry port or workflow decision script is permitted.

### C. Release Please action/version/protocol stamping

**Proves:** the one root Release Please package updates the semantic version embedded in `action.yml`, preserves/updates the exact protocol integer mechanism, and the resulting tag's binary reports matching stamps.

**Blocks:** PR 1 release cutover.

**Procedure:**

1. In a scratch branch/fork, add the proposed `action.yml` version/protocol locations and `release-please-config.json` `extra-files` entries.
2. Create a Conventional Commit, run the pinned Release Please action, and inspect the generated release PR rather than assuming generic YAML behavior.
3. Merge the scratch release PR; confirm the tag/draft, action stamp, archive version, and `release-cli version --json` values.
4. Run the composite installed path: version and protocol must compare equal before a stage command.
5. Change only the candidate action protocol and show installed-path setup fails closed; pass the mismatched binary by `cli-path` and show a warning plus reported outputs.

**Pass:** no manual semantic-version edit, exact installed-path equality, deliberate mismatch behavior, and one root tag/version.

**Fallback:** use Release Please `extra-files` for the semantic version only; keep protocol `1` as an explicit literal in the action step and linker default/config, and add a PR/CI script that parses `action.yml` plus `release-cli version --json` and fails if they differ. Protocol bumps change both literals in one release-unit PR. This retains the exact guard and all ports; it does not introduce `cli-version` or negotiation.

The later provenance-predicate, `sigstore-go`, and native App-token experiments are trigger-gated research, not implementation gates and not scheduled by this plan.

## 6. Composite action and consumer contract

### `setup-release-cli/action.yml`

Public metadata:

```yaml
inputs:
  cli-path:
    description: Unsupported path to a caller-supplied release-cli binary; the caller owns version pairing.
    required: false
    default: ''
outputs:
  cli-path:
    description: Absolute executable path selected by the action.
  version:
    description: Version reported by release-cli.
  commit:
    description: Source commit reported by release-cli.
  protocol:
    description: Integer protocol stamp reported by release-cli.
```

There is no `cli-version`, command, mutation, token, registry, or profile input. The action sets only acquisition outputs; reusable workflow steps invoke commands directly.

### Installed path (`cli-path` empty)

1. Derive distribution repository from `github.action_repository`; never hardcode an owner/repo.
2. Use the action's Release Please-stamped semantic version to download the matching platform archive and `checksums.txt` from tag `v<version>`.
3. Parse the exact archive entry from `checksums.txt` and verify SHA-256 before extraction.
4. Run `gh attestation verify` against the archive with `--repo <github.action_repository>` and the derived publisher workflow identity. Until the later predicate spike passes, claim only signed + repository/workflow verified + stamp matched—not same-source-commit provenance.
5. Extract to `RUNNER_TEMP`, require a regular executable, run `version --json`, and require both semantic version and protocol integer equal the action stamps. Any mismatch fails before a stage/publish command.
6. Write absolute path/version/commit/protocol outputs.

The supported reusable workflows run on `ubuntu-24.04`; do not promise an untested Windows/macOS composite contract merely because GoReleaser produces those direct-download archives.

### `cli-path` path

Require the supplied path to exist, be regular, and executable; run `version --json`; expose all four outputs. Do not enforce semantic version. If protocol differs, emit a GitHub warning annotation and continue: “you supply the binary, you own the pairing.” A same-commit dogfood binary matches protocol and stays quiet.

### Dogfood transport

The owning `.github/workflows/release.yml` adds `build-release-cli`, uploads the tagged binary as `release-cli-dogfood`, and makes release jobs depend on it. Each reusable workflow conditionally downloads this same-run artifact when its optional `cli-path` is nonempty, then passes the extracted path to the composite. This honors job-local filesystems without introducing an independent CLI version or consumer output.

### Reusable workflow interface evolution

- PR 1: `go-pre-publish.yml` gains only optional `cli-path:string=''`; its six outputs remain identical.
- PR 2: `go-oci-build.yml`, `publish-github-release.yml`, and `publish-oci-image.yml` gain the same optional input. All existing required/default inputs, secret names, and outputs remain byte-for-byte compatible.
- PRs 3–10: no `workflow_call` interface change. CLI/action result fields and prepare output names are internal to the single release unit.

External consumers continue to repeat one reviewed `FULL_SHA` across four `uses:` fields and the checksum signer identity. They do not select a CLI version. Existing SHA-pinned callers remain permanently on their old workflow/behavior.

## 7. Documentation work

Documentation follows Diátaxis and D5/D6; behavior docs land in the same PR, while PR 11 merely replaces the not-yet-knowable final SHA.

| PR | Tutorial | How-to | Reference | Explanation | Example |
|---:|---|---|---|---|---|
| 1 | — | Modify `upgrade-github-release-workflows.md` | Create `release-cli-contract.md`; modify `github-release-contract.md` | — | Modify `examples/go-release/README.md`; leave workflow pinned to last released SHA, clearly labeled |
| 2 | — | Modify upgrade guide for optional `cli-path` and three-owner handoff | Modify all three contract files | — | No caller change; old inputs remain valid |
| 3 | — | Modify OCI configuration guide for tag planning diagnostics | Modify CLI/OCI contracts | — | None |
| 4 | — | Modify OCI guide for prepare/dry-run | Modify CLI/OCI contracts | — | None |
| 5 | — | Modify OCI guide for prepare→attest→finalize | Modify CLI/OCI contracts | Create `two-phase-oci-publication.md` | None |
| 6 | — | Modify GitHub configuration guide for closed bundle verification | Modify CLI/GitHub contracts | — | None |
| 7 | — | Modify GitHub configuration and recovery guides | Modify CLI/GitHub contracts | — | None |
| 8 | — | Modify OCI guide for artifact-local image-input projection | Modify CLI/OCI contracts | — | None |
| 9 | — | Modify OCI guide for exact-byte verification | Modify CLI/OCI contracts | — | None |
| 10 | Create `docs/tutorials/release-a-go-project.md` | Modify all four existing how-to files | Final ownership/residual-YAML updates to all three contracts | Create `release-trust-boundaries.md` | Update README narrative/config-copy guidance |
| 11 | Replace final SHA markers | Replace final SHA markers | Replace final SHA markers | Update links only if needed | Pin `examples/go-release/.github/workflows/release.yml` and README to the verified full SHA |

Do not mix explanatory trust-boundary prose into contract tables or task instructions; link across document types. `README.md` remains a restrained index into these docs.

## 8. Test infrastructure plan

1. **Slice 1 fixtures:** capture real GoReleaser 2.17.1 `artifacts.json` from `mise exec -- goreleaser release --snapshot --clean --skip=publish,sign` (or the exact accepted v2.17.1 syntax discovered locally). Check in only the small JSON parser fixture; build checksum/filesystem fixtures in `t.TempDir()`/`fstest.MapFS`. The live OIDC rehearsal remains the proof for real checksum signing.
2. **Local filesystem throughout:** use `fstest.MapFS` for pure readable trees and `t.TempDir()` when permissions, symlinks, executable modes, git, tar metadata, or path confinement matter. Stream payload/layer hashing; do not buffer release archives (P2).
3. **Mockery from slice 2:** pin v3.7.3, explicit per-interface `.mockery.yml`, generated Testify mocks under each adapter's `mocks/`; run Mockery and ensure a second run is clean. Never handwrite a mock (T2).
4. **OCI from slice 3:** use `go-containerregistry/pkg/registry` over `httptest` for `reg` integration and full prepare/finalize tests. This is not GHCR proof; retain the live parity/rehearsal gate.
5. **GitHub from slice 4:** use generated engine mocks and focused adapter `httptest` servers. Do **not** prebuild a reusable stateful fake. Trigger: slice-4 tests actually require repeated pagination/eventual-consistency/clobber state transitions that focused mocks cannot express without duplicated/brittle setup. If triggered, stop and amend the plan rather than silently adding scope.
6. **Pinned subprocesses:** unit-test argument/environment construction for Cosign, gh, Melange, apko, git, and GoReleaser; never place secrets in expected argument dumps. Real tool semantics remain live rehearsal evidence.
7. **CI shape:** `moon run root:test`/`root:check` runs pure and cheap integration layers in seconds. Live cross-repo rehearsal is explicit CI-only evidence and is not hidden inside ordinary `go test`.

## 9. Risk register and abort criteria

| Slice | Stop/re-plan condition | Evidence that triggers abort |
|---|---|---|
| 1 | The parser cannot accept real GoReleaser output without weakening PP-06–PP-08, or the action cannot maintain the one-unit stamp/bootstrap contract | Real 2.17.1 fixture/rehearsal fails; `$/`/stamping spike has no same-port fallback; PP-09/PP-10 would need an unapproved artifact change |
| 2 | go-github metadata cannot reproduce ID/expiry/run/digest checks before download | Live API omits/changes a coordinate or a proposed fix conflates extracted content with transport digest |
| 3 | Neither oras-go nor ORAS fallback can enforce digest push, referrers, fresh-state planning, and serial verified tags | Scratch GHCR shows incompatible resolve/referrer/tag semantics or ambiguous writes cannot be reconciled without new authority/ports |
| 4 | Draft ownership or expected-name-only convergence cannot be preserved, or post-undraft ambiguity is silently treated as success | Live rehearsal creates/re-drafts, deletes unexpected assets, accepts wrong digest, or cannot distinguish exact complete public state from indeterminate state |
| 5 | CLI-built output cannot pass the unchanged YAML oracle or exact-byte checks cannot be expressed without rebuilding/re-marshaling | Real Melange/apko output differs in contract; canonical executable hash, platform shape, or index-byte digest diverges |
| 6 | CLI invocation changes GoReleaser's tool/OIDC/publish behavior | Missing `--skip=publish`, `release.disable` removed, wrong managed binary, or real assets/signature differ |

### Deferred triggers—do not build early

- Add a profile interface/registry only when **a second real ecosystem profile reaches an implementation PR**.
- Persist staged facts only when **the first downstream CLI command reads them**. This trigger fires deliberately in PR 8; serialize only `oci-build-inputs`' validated projection.
- Add stable mutation-status reporting and exit `3` only on **the first ambiguous remote write plus a caller that programmatically branches on it**.
- Add `release.toml`, `config show`, and `config validate` only when **a real adopter needs a non-derived registry target, or a second durable setting exists**.
- Extract `internal/execx` only after **two subprocess adapters repeat meaningful plumbing beyond `exec.CommandContext` setup**; architecture amendment is required because it is currently prohibited.
- Add a custom image-layout port only if **the image engine demonstrably needs to mock failure ordering**.
- Add a reusable stateful GitHub fake only when **slice-4 tests demonstrate repeated pagination/eventual-consistency/clobber transitions beyond focused mocks**.
- Split another package only for **measured size/cohesion pressure**, not symmetry.
- Spike provenance source-commit predicates later; until proven, the bootstrap claim stays **signed + stamp-matched**.
- Evaluate `sigstore-go` offline bundle verification only **after slice 4** and only if network/TUF behavior creates measured pain.
- Replace the pinned App-token action only if a spike proves **smaller secret exposure and correct expiry/refresh behavior**.
- Add cross-caller/shared-registry locking only when a real consumer targets a shared registry; until then document the direct-CLI **single-writer limitation**.
- Do not design non-Go, alternate-architecture, multiple-executable, prerelease, or container-only profiles before their corresponding real adopter exists.

## 10. Definition of done

The program is done only when all of the following are observable:

### Final workflows

- **`go-pre-publish.yml`:** `workflow_call` and six outputs; tag gate; checkout; pinned mise and exact tool-path proof; setup action; one `release-cli stage --profile go --dist dist` invocation that runs GoReleaser and PP-06–PP-08; unchanged release/OCI-input uploads (plus the triggered OCI-only projection in PP-09).
- **`go-oci-build.yml`:** existing inputs/four outputs; tag gate; checkout; mise; QEMU; tool proof; setup action; `verify handoff`; pinned `download-artifact`; `image build`; `image verify`; unchanged OCI artifact upload.
- **`publish-oci-image.yml`:** existing inputs/six outputs; stable-tag gate; permissions/concurrency; mise/setup action; `verify handoff`; pinned download; content validation; `prepare` or dry-run; three unchanged job-level `actions/attest`; stdin `finalize`; no persisted registry credential and no pre-attestation public tag.
- **`publish-github-release.yml`:** existing inputs/secret/two outputs; tag/required-OCI gate; checkout; pinned tools; YAML App-token action; setup action; `verify handoff`; pinned download; `verify bundle`; unchanged `actions/attest`; `publish github`; no release creation path.

Residual YAML consists only of platform-owned orchestration: triggers/interfaces, permissions, environments, `needs`, concurrency/no-cancel, timeouts, checkout, mise/tool provenance, QEMU, dogfood CLI artifact transport, upload/download actions, App-token minting, attestation actions, and simple command invocations/output plumbing. Bespoke publication/checksum/tag/layout decision scripts are gone.

### Consumer surface

The copyable consumer uses the four existing reusable workflow files at one repeated full SHA, the same SHA in `checksum-signing-workflow-ref`, and the same existing inputs/outputs/secrets. It has no CLI version pin. `cli-path` is documented as unsupported and omitted by normal consumers. A second organization changes only App installation and its client-ID variable/private-key secret.

### Observable dogfood proof

A stable tag in `meigma/release` must:

1. build `release-cli` from that tag and feed it to all reusable jobs through `cli-path`;
2. report the expected protocol without warning;
3. use that binary to produce `release-cli` archives, signed checksum bundle, SPDX SBOMs, APKs, and a two-platform OCI image containing the exact canonical binary bytes;
4. push/sign/attest the image by digest before serial exact/channel tags;
5. verify/attest/converge the matching upstream draft before undrafting it;
6. publish a GitHub Release whose `release-cli` archive passes checksum and `gh attestation verify`;
7. allow the released `setup-release-cli` installed path to download that archive and pass exact version/protocol checks; and
8. leave all PR CI (`moon run root:check`) and the external live rehearsal green.

Only that end-to-end observation proves the repository is releasing and consuming its own approved release unit.
