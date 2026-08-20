---
id: 004
title: Release CLI slices 5a through 6
date: 2026-08-19
status: complete
repos_touched: [meigma/release]
related_sessions: ["003", "002", "001"]
---

## Goal

Continue the eleven-PR `release-cli` program from `.journal/002/PLAN.md` at PR 8,
using the standing per-PR method: parallel implementation agents with one owner
per file, a long-lived reviewer (max two rounds), a long-lived conformance agent
(one round), a caller-ceiling audit, a parent gate, then a human-accepted PR.

## Outcome

Met. Three PRs implemented, reviewed, hardened, and squash-merged, which
completes every behavioral slice in the program:

| PR | Commit | Slice |
|---|---|---|
| [#15](https://github.com/meigma/release/pull/15) | `e235a28` | 5a: `oci-build-inputs` projection, `image build`, `melange`/`apko` adapters (ports 11 and 12) |
| [#16](https://github.com/meigma/release/pull/16) | `8a5e0a7` | 5b: `image verify`, exact-byte layout parsing |
| [#17](https://github.com/meigma/release/pull/17) | `7197ca2` | 6: GoReleaser invocation moved into `goprof` |

`main` is at `7197ca2`. Twelve of the thirteen approved ports exist; only
`cli.Actions`/`actenv` remains unbuilt. All four reusable workflows are now thin
shells: tag gate, checkout, mise install, managed-tool-path proof, setup action,
`release-cli` invocations, artifact transport. `go-oci-build.yml` lost 271 lines
of shell across the two image PRs.

Not done, by design: PR 11, the documentation pin. It is gated on a released and
verified PR-10 build, and the user explicitly deferred it.

## Key Decisions

- **The build date is a CLI flag, not a git read** -> reproducible Melange and
  apko builds need the tagged commit date, and reading it inside the CLI would
  have required a git port while the budget is closed at 13. The workflow keeps
  `git show -s --format=%cI` and passes `RELEASE_BUILD_DATE`.
- **`*os.Root` is the write seam for image builds** -> the plan forbids a
  filesystem port, so the engine takes confined roots and `os.OpenRoot` stays at
  the CLI composition edge. Layer-2 tests write into `t.TempDir()`.
- **Adapters stay thin; the engine owns layout** -> the engine chooses every path
  and only cross-checks what `APKRepositories` reports, so an adapter cannot
  redirect the signing key or the repository root unnoticed.
- **Two independent OCI layout readers** -> `puboci.ReadLayout` (publish path) and
  `image.ReadLayout` (build-time verification) overlap in roughly 60 lines.
  Consolidating would change `puboci.Descriptor`, a port parameter type, so the
  blast radius reaches the mocks, the `reg` adapter, and the CLI. Consolidation
  trigger: a third consumer needing on-disk layout parsing.
- **No retry around Melange, apko, or GoReleaser** -> local, stateful,
  non-idempotent writes. Retrying could reuse a partial build or a generated
  signing key and would hide deterministic configuration failures.
- **`stage` became a build command** -> with `--clean`, GoReleaser deletes and
  rebuilds the distribution directory, so `--dist` must be a basename (GoReleaser
  writes relative to the working directory and has no `--dist` flag) and every
  configuration failure must be rejected before the build starts.
- **The producer runs the CLI under `mise exec`** -> GoReleaser shells out to
  `go`, `syft`, and `cosign`. Passing only the pinned GoReleaser path, the pattern
  used for `cosign`/`gh`/`git`, would have silently handed the build a different
  Go toolchain.
- **Verification evidence comes from real tools, not mocks** -> every slice was
  proven against a genuine apko layout built on the laptop with pinned Melange
  0.59.1, apko 1.2.37, and Docker, and `image verify` was diffed against the shell
  verifier it replaces.

## Changes

- `internal/stage/image_inputs.go` - the `release.dev/oci-build-inputs/v1`
  projection codec; `stage --profile go` now writes `dist/oci-build-inputs.json`.
- `internal/profile/goprof/artifacts.go` - `BinaryName`, the cross-architecture
  name-agreement check, and a `ParseRootName` that rejects `..` and both path
  separators.
- `internal/profile/goprof/goreleaser.go` - `RunGoReleaser`, invoking exactly
  `goreleaser release --clean --skip=publish`, with ANSI stripped from the
  retained error tail.
- `internal/stage/image/` - `Platform`/`APKArch` values, the `APKBuilder` and
  `Composer` ports, ELF verification, the `Build` engine, plus `ReadLayout`,
  `VerifyLayout`, `VerifySBOMs`, and `CanonicalDigests`.
- `internal/adapter/melange`, `internal/adapter/apko` - thin exec adapters with
  argv byte-identical to the YAML they replaced, and generated mocks.
- `internal/cli/` - `image build`, `image verify`, the rebuilt `stage`, and the
  `RunGoReleaser` seam; `image_test.go` split three ways to respect R2.
- `.github/workflows/go-oci-build.yml` - OB-08 through OB-21 replaced by two CLI
  invocations; `.github/workflows/go-pre-publish.yml` - PP-05 removed, PP-04 kept.
- `examples/go-release/.github/workflows/release.yml` - `release-assets` now grants
  `actions: read` and `attestations: read`, which the callee has required since
  PR 2.
- `docs/` - `image build` and `image verify` contracts, GoReleaser ownership moved
  to the CLI, plus two new pages: `docs/tutorials/release-a-go-project.md` and
  `docs/explanation/release-trust-boundaries.md`.
- `internal/adapter/*/[a-z]*_test.go` - every exec fake is now written once in
  `TestMain`; `.golangci.yml` excludes `gochecknoglobals` in tests.

## Open Threads

- **PR 11 is the only remaining program work.** It replaces the `FULL_SHA`
  placeholders and the stale `fb8c8098…` pin across `README.md`, `docs/`, and
  `examples/go-release/`. It is gated on a released, verified build of `7197ca2`.
- **The next tag is still the end-to-end proof.** Keyless signing, the three
  `actions/attest` steps, and now the full GoReleaser build only run inside
  Actions. `gomod.proxy: true` makes a complete local build impossible because the
  proxy cannot resolve an unpublished tag. `publish-image: false` and
  `publish-release: false` remain the documented rollbacks.
- **The `execx` prohibition should be revisited.** The bounded stderr-tail plus
  `WaitDelay` plus `LookPath` exec helper now exists four times (`cosign`,
  `melange`, `apko`, `goprof`). Conformance recommends amending the architecture
  and extracting it; that is a follow-up with `ARCHITECTURE.md` updated, not a
  slice PR.
- **`Options.RunGoReleaser` is a function-shaped outbound seam**, which
  conformance calls "port 14 in substance". The plan forbade an interface plus
  adapter package, which is what was avoided; read the closed budget that way.
- **Port 13 (`cli.Actions`/`actenv`) is still unbuilt** and still deliberately
  deferred: workflows read the `--json` envelope with `jq` and a `$GITHUB_OUTPUT`
  heredoc.
- **Mockery's testify template emits no Godoc** for generated expecter types.
  Raised in five consecutive slices, still deferred as a template-wide change.
- **Release-please PR #9** (`chore(main): release 0.1.1`) is still open and now
  describes a stale version; the merged features should move it to a minor bump.
- Housekeeping debt from sessions 001 and 002 is unchanged: archived but
  undeletable scratch repositories, the `spike/self-ref` branch, the
  `ghcr.io/meigma/release-oras-spike` package, and the dead
  `release-please--branches--main--components--release-mvp` branch.

## Lessons

- **A test that fails only on the other operating system is failing, not flaky.**
  Three sessions of "load-sensitive" adapter cancellation failures were really
  `ETXTBSY`: each test wrote its shell fake into `t.TempDir()` and exec'd it while
  a parallel sibling's `fork` still held the write descriptor. A containerized
  `go test ./...` reproduced it on the first attempt. Fix the harness, not the
  timeout.
- **When a read-only command becomes side-effecting, re-audit every error it used
  to ignore.** `runStage` had never checked its settings resolver error. That was
  harmless while `stage` only validated, and became a fail-open the moment `stage`
  started building.
- **When you replace a check with an existing domain constructor, diff the two
  rules first.** Collapsing the dist rule onto `ParseRootName` silently narrowed
  separator rejection to `/`, turning a pre-build exit 2 into a post-build exit 1
  and inverting the rule on Windows. Same class as session 003's "when a port
  replaces a regex, diff the regex".
- **Masked comparisons hide privilege bits.** The image verifier compared
  `Mode().Perm()`, so a setuid entrypoint passed while the image runs as user
  65532; the shell script it replaced caught it by comparing `-rwxr-xr-x`. Compare
  the raw mode, or mask deliberately with `&0o7777`.
- **A conformance rule citation is not evidence.** Conformance demanded
  `FileInfo().Mode().IsRegular()` so `tar.TypeRegA` would be accepted; a reviewer
  probe showed `archive/tar` normalizes that flag before `Next()` returns, so the
  path was dead code that had widened the accepted set. Probe the standard
  library.
- **The caller-ceiling audit earns its place in the method.** It found that the
  copyable example granted less than `go-pre-publish.yml` declares, which would
  have failed a new adopter's first run at `startup_failure` with no API-visible
  diagnostic.
- **Diff the oracle before deleting it.** `image verify` was accepted only after
  its digest matched what the shell verifier had written for the same real layout,
  and after a rebuilt layer with one flipped byte failed.

## References

- Plan and architecture: `.journal/002/PLAN.md`, `.journal/002/ARCHITECTURE.md`
- Merged PRs: [#15](https://github.com/meigma/release/pull/15),
  [#16](https://github.com/meigma/release/pull/16),
  [#17](https://github.com/meigma/release/pull/17)
- Prior sessions: `.journal/003/SUMMARY.md`, `.journal/002/SUMMARY.md`,
  `.journal/001/SUMMARY.md`
- New docs: `docs/tutorials/release-a-go-project.md`,
  `docs/explanation/release-trust-boundaries.md`
