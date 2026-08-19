---
id: 004
title: Goal pending at kickoff
started: 2026-08-19
---

## 2026-08-19 14:14 — Kickoff

Goal for the session: not yet stated. The user asked only to start a new
session; the actual request follows. Update the title and this log once the goal
arrives.

Current state of the world:

- `main` is at `df077f9` ("feat(release): publish verified GitHub releases
  (#14)"). Working tree clean; `journal/jmgilman` is in sync with its remote.
- The eleven-PR `release-cli` program in `.journal/002/PLAN.md` has PRs 1-7
  merged. Remaining: PR 8 `image build`, PR 9 `image verify`, PR 10 moving the
  GoReleaser invocation into `goprof`, PR 11 the documentation pin that replaces
  `fb8c809`.
- Ten of thirteen approved ports exist. Unbuilt: `image.APKBuilder` (melange),
  `image.Composer` (apko), `cli.Actions` (actenv, deliberately deferred).
- `publish-oci-image.yml` and `publish-github-release.yml` are both thin and
  carry no bespoke publication logic.
- Open threads inherited from 003: the next real tag is the first end-to-end CI
  proof of the two-phase OCI path plus the release path together; mockery's
  testify template emits no Godoc for expecter types; housekeeping debt from
  sessions 001-002 (archived-but-undeletable scratch repos, `spike/self-ref`
  branch, `ghcr.io/meigma/release-oras-spike` package, dead
  `release-please--branches--main--components--release-mvp` branch); possibly
  still-open release-please PR #9.

Plan: wait for the stated goal, then follow the standing per-PR execution method
in `.journal/002/PLAN.md` if the work continues the program.

## 2026-08-19 15:54 — PR 8 opened: `feat(image): build OCI layouts from staged binaries`

Goal (stated after kickoff): execute PR 8 of `.journal/002/PLAN.md` with the
standing per-PR method. Done; PR is open for human acceptance:
https://github.com/meigma/release/pull/15 (branch `feat/release-cli-image-build`,
commit `a9aa400`, 42 files, +6990/-240).

What landed:

- `stage --profile go` now writes `dist/oci-build-inputs.json`
  (`release.dev/oci-build-inputs/v1`): profile plus, per Linux platform, binary
  name, artifact-root-relative path, and canonical SHA-256 digest. The
  `stage --json` envelope is unchanged. `goprof.CanonicalBinary` gained
  `BinaryName`, and `SelectBinaries` now enforces the shared-name check the YAML
  used to do.
- `release-cli image build` replaces OB-08 to OB-17. `internal/stage/image` holds
  the engine, its `Platform`/`APKArch` value types, ELF verification, and ports 11
  and 12; `internal/adapter/melange` and `internal/adapter/apko` are thin exec
  adapters with argv byte-identical to the removed YAML.
- `go-oci-build.yml` runs one `image build --json` step (`id: build`) that resolves
  the pinned tools with `mise which` and writes five step outputs with `jq`. The
  old verifier stays as an independent oracle: its `run:` script is byte-identical
  to `main`, only the `env:` step-output references moved.

Deliberate deviations from the plan's file list, both recorded here rather than
chased: `internal/stage/image` was split into `doc.go`/`values.go`/`ports.go`/
`elf.go`/`build.go` instead of one `build.go` (matches `puboci`'s `ports.go`
convention and R2), and the engine defines its own `BuildBinary` instead of
importing the `internal/stage` projection type, so the wire type never reaches the
engine and the two agents could work in parallel.

Design decisions worth inheriting:

- **The build date is a flag, not a git read.** Reproducible Melange/apko builds
  need the tagged commit date; reading it in the CLI would have needed a git port,
  and the budget is closed at 13. The workflow keeps `git show -s --format=%cI` and
  passes `RELEASE_BUILD_DATE`, validated as RFC 3339.
- **`*os.Root` is the write seam.** The plan forbids a filesystem port, so the
  engine takes confined roots and the CLI owns `os.OpenRoot`. Layer-2 tests write
  into `t.TempDir()`.
- **Adapters stay thin; the engine owns layout.** The engine chooses every path and
  only cross-checks what `APKRepositories` reports, so an adapter cannot redirect
  the signing key or the repository root unnoticed.
- **No retry around Melange or apko.** They are local, stateful, non-idempotent
  writes; retrying could reuse a partial build or a generated key. Conformance
  agreed explicitly (E3).

Verification (real, not mock-only, per session 003's lesson):

- `moon run root:check` green: format, lint, build, test, protocol stamp, mocks.
- Real rehearsal with pinned Melange 0.59.1, apko 1.2.37, and Docker on this
  laptop: two real static Go Linux binaries -> `stage` projection -> `image build`
  -> genuine two-platform layout. The **unchanged YAML verifier**, run in
  `ubuntu:24.04` with GNU tar 1.35, printed `VERIFIER_OK` and an `image-digest`
  equal to an independent `sha256sum` of `layout/index.json`. Layer inspection:
  mode 0755, uid/gid 0/0, bytes identical to both canonical inputs. This is PR 8's
  documented merge gate, met locally instead of waiting for CI.
- Fail-closed exercises against the built binary: tampered byte, `ET_DYN` binary,
  swapped-architecture bytes, populated roots, trailing JSON in the projection,
  `--work` nested in `--output`, and four exit-2 configuration failures that create
  no directories.

Review outcome:

- Reviewer round 1: four findings, all fixed. The important one was a real
  fail-open — with `--work` nested under `--output`, the ephemeral APK **private**
  signing key was written inside the directory `upload-artifact` publishes, which
  contradicted two doc sentences shipped in the same patch. Unreachable through the
  shipped workflow (paths are hard-coded), reachable through the documented flags.
  Now rejected as exit 2 before any directory is created, with a separator-aware
  prefix test so `/a/out` and `/a/outx` stay valid.
- Reviewer round 2: nothing blocking, 15 of 15 mutations caught. Three
  non-blocking findings were fixed anyway (a doc exit-code error, a missing
  ordering regression test, an over-strong codec Godoc).
- Conformance: 22 PASS, 2 FAIL, both fixed rather than documented away — the
  projection decoder documented a 4 MiB bound it did not enforce (`io.LimitReader`
  only fakes EOF, so a valid document plus a second value passed; now
  `*io.LimitedReader` at limit+1 plus `decoder.More()`, matching
  `parsePrepareEnvelope`), and three documented-versus-actual ordering claims were
  wrong. `runImageBuild` now decodes the projection, opens the configs, and builds
  the ports **before** creating any directory.

Two bugs found outside the slice and fixed here:

- **Caller-ceiling audit caught a latent adopter break.** `examples/go-release`'s
  `release-assets` job granted only `contents: read` and `id-token: write`, while
  `go-pre-publish.yml` declares `actions: read` and `attestations: read`. A
  consumer copying the example would have died at `startup_failure` with no
  API-visible diagnostic. Fixed in the example plus the caller skeleton and
  permission table in `docs/reference/github-release-contract.md` and the builder
  grant sentence in `docs/reference/oci-image-contract.md`. All four caller jobs
  now match their callees exactly in this repo, the example, and the docs.
- **A shared test-scaffolding flake.** Every exec adapter's cancellation test used
  one budget for both fixture startup (load-dependent) and the post-cancel return
  (the actual contract), so `root:check` went red under load in `cosign`, `ghup`,
  and `gitx`. Split into `startWait` (30s) and the unchanged tight `cancelWait`.

Open follow-ups: mockery's testify template still emits no Godoc for generated
expecter types (fourth slice in a row, still deferred as a template-wide change).
Housekeeping debt from sessions 001-002 is untouched. PR 9 (`image verify`) is
next; it deletes the YAML oracle this PR deliberately kept.

CI on PR #15 was queued when this entry was written; the parent gate had already
run the same checks locally.

## 2026-08-19 16:25 — Correction: the adapter flake was ETXTBSY, not load

PR #15's first CI run failed: `TestBuildCanceledContextReturnsPromptly` in
`internal/adapter/melange` burned the full 30s start wait because the fake
never started. The earlier entry's "load-sensitive" diagnosis was a
symptom-level read. Correcting it here.

Real cause: every exec-adapter test wrote its POSIX shell fake into its own
`t.TempDir()` with `os.WriteFile(..., 0o755)` and exec'd it immediately. On
Linux a parallel sibling's `fork/exec` inherits the still-open write descriptor,
so exec'ing that file fails with `ETXTBSY` ("text file busy"). macOS never
reproduces it, which is why the program has been mislabeling it as flakiness
since the first exec adapter landed.

How it was proven, after `go test ./internal/adapter/melange` passed in
isolation twice: run the whole suite on Linux.
`docker run --rm --cpus 4 -v <worktree>:/src -w /src -e GOFLAGS=-mod=mod
golang:1.26 go test ./... -count=1` reproduced it on the first try —
`melange compile: fork/exec /tmp/TestBuildStopsAfterCompileFailure.../melange:
text file busy` — and a second run hit `cosign` the same way. Different test,
same file-write-then-exec race.

Fix (`6ad3935`): `TestMain` writes each package's fake exactly once into an
`os.MkdirTemp` directory before `m.Run()`; helpers return that shared path.
Per-test `t.TempDir()` still holds argv records, start markers, stderr fixtures,
and working directories. `gochecknoglobals` joined the existing `_test.go`
exclusion list because the shared fixture path must be package scoped; the rule
still covers production code. The `startWait` split from the earlier round stays
as defense in depth, but it was never the fix.

Verified: four independent Linux full-suite container runs with zero failures
and zero `text file busy`, `mise exec -- go test ./internal/adapter/... -count=2`
green, `moon run root:check` green, and **PR #15 CI now passes** (run
32312135030, 31s).

Lesson for the summary: a test that only fails on the other operating system is
not flaky, it is failing. Reproduce it in that OS before believing a timing
story. A containerized `go test ./...` is cheap and would have caught this three
slices ago.

PR #15 is awaiting human review; nothing is merged.


