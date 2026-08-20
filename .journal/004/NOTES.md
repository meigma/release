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

## 2026-08-19 19:02 — PR 8 merged; PR 9 opened: `feat(image): verify OCI image contracts`

PR #15 was accepted and squash-merged as **`e235a28`**. `main` fast-forwarded,
the worktree and branch are gone. PR 9 is now open for human review:
https://github.com/meigma/release/pull/16 (branch `feat/release-cli-image-verify`,
commit `ddf29e7`, 14 files, +3535/-1154, CI green first run).

`wt remove` note: `gh pr merge --squash --delete-branch` failed its local
branch-checkout step because `main` is checked out in the primary worktree. The
merge itself had already succeeded, so the fix was `git fetch --prune`,
`git pull --ff-only` in the main checkout, `wt remove <branch> --force`, and
`git push origin --delete <branch>`. Expect this every time in a worktree setup.

What PR 9 lands:

- `internal/stage/image/layout.go` — `ReadLayout`: `oci-layout` presence, exact
  `index.json` bytes and their digest, per-platform manifest/config/layer
  descriptors, blob existence at the declared size.
- `internal/stage/image/verify.go` — `VerifyLayout`, `VerifySBOMs`,
  `CanonicalDigests`. Index/manifest/config annotation and label equality, the
  two-platform set, entrypoint, user 65532, exactly one layer, the layer's
  `usr/bin/<binary>` entry streamed and compared byte for byte against the
  canonical staged binary, and an SPDX `APPLICATION` package at `<version>-r0`.
- `internal/cli/image.go` — `image verify`, which also writes
  `<output>/image-digest.txt` (the publisher workflow reads that file) only after
  both verifiers pass. `internal/cli/image_test.go` had crossed the R2 1,000-line
  cap during PR 8's fix rounds and is now split three ways.
- `.github/workflows/go-oci-build.yml` — the `verify` step keeps its id and runs
  the CLI; the 112-line shell verifier is gone. Upload step byte-identical.

Decisions worth inheriting:

- **Two layout readers, on purpose.** `puboci.ReadLayout` (publish path) and
  `image.ReadLayout` (build-time verification) overlap in roughly 60 lines of
  index parsing. Consolidating would have meant touching `puboci.Descriptor`,
  which is a port parameter type, so the blast radius reaches the mocks, the
  `reg` adapter, and the CLI. Consolidation trigger: a third consumer needing
  on-disk layout parsing.
- **The engine gained a second responsibility, not a second package.**
  `internal/stage/image` now both builds and verifies; `doc.go` says so.
- **Verification never retries and never rebuilds.** Deterministic failure,
  fail closed.

Evidence (the plan called this CI-only; it was done locally instead):

- Against the real apko 1.2.37 layout PR 8's CLI produced, `image verify` reports
  the *same* `index_digest` the removed shell verifier had written into
  `image-digest.txt`, and the same independent `shasum -a 256 layout/index.json`.
  That is a direct oracle comparison against the code being deleted.
- A rebuilt amd64 layer with one flipped byte in `usr/bin/release-cli` — with
  manifest and index descriptors regenerated so every digest and size is
  internally consistent — fails on content, proving the tar is actually read.
- Appending a newline to `index.json` changes the reported digest, so the digest
  really is over raw bytes.

Review, and the one finding that mattered:

- **Round 1 caught a privilege-escalation hole.** `checkBinaryHeader` compared
  `header.FileInfo().Mode().Perm()`, which masks setuid, setgid, and sticky, so a
  `04755` entrypoint passed. The shell script being replaced caught it by
  string-comparing `-rwxr-xr-x`. The image runs as user 65532, so a setuid-root
  entrypoint would have been a consumer-visible privesc primitive, and this
  verifier is the only oracle for it. Now the raw tar mode is compared. Proven on
  a rebuilt real layer: `usr/bin/release-cli has mode 04755, want 0755`.
- **Round 2 overturned a conformance finding with evidence.** Conformance had
  demanded `FileInfo().Mode().IsRegular()` so `tar.TypeRegA` would be accepted.
  The reviewer probed `archive/tar` and showed the reader normalizes the historic
  NUL typeflag to `'0'` before `Next()` returns: the path was dead, its test could
  not fail, and the loose check had silently widened the accepted set to
  `TypeCont` and unknown vendor flags. Final form is strict `Typeflag !=
  tar.TypeReg` with `Mode&0o7777` so a producer that writes file-type bits into
  the mode still passes while setuid/setgid/sticky stay rejected. Lesson: a
  conformance rule citation is not evidence; probe the standard library.
- Conformance's other two findings (a stale `doc.go`, and `--binary` shape
  validated only after the roots were opened, surfacing as exit 1 instead of 2)
  were real and are fixed.

Caller-ceiling audit: PR 9 changes no `permissions` block anywhere, and all four
caller jobs still match their callees in this repo, the copyable example, and the
documented skeleton.

Next: PR 10 moves the GoReleaser invocation into `goprof` and thins
`go-pre-publish.yml`; PR 11 is the documentation pin that closes the program.

## 2026-08-19 20:24 — PR 9 merged; PR 10 opened: `refactor(prepublish): run GoReleaser through release-cli`

PR #16 was accepted and squash-merged as **`8a5e0a7`**. PR 10, the last
behavioral slice, is open: https://github.com/meigma/release/pull/17 (branch
`feat/release-cli-goreleaser`, commits `08be6e3` + `42f823d` + `ddc2b88`, CI green
first run). All four reusable workflows are now thin shells; only PR 11's
documentation pin remains.

What landed:

- `internal/profile/goprof/goreleaser.go` — `RunGoReleaser` invoking exactly
  `release --clean --skip=publish`. No port, no adapter package, no `execx`; the
  exec boundary lives in the profile package because the approved port list has
  no GoReleaser port.
- `internal/cli/stage.go` — `stage --profile go` now builds, then validates, then
  writes the projection. `RELEASE_GORELEASER_PATH` is env-only. Both GoReleaser
  streams go to the CLI's stderr so `--json` stdout stays one envelope.
- `.github/workflows/go-pre-publish.yml` — PP-04's tool proof is its own step; the
  staging step resolves `mise which goreleaser` and runs the CLI **under
  `mise exec`**, because GoReleaser shells out to `go`, `syft`, and `cosign`.
  That detail is the one thing a naive "pass the pinned binary path" port of this
  step would have broken.
- New docs required by the plan: `docs/tutorials/release-a-go-project.md` and
  `docs/explanation/release-trust-boundaries.md`.

Facts discovered while implementing:

- **`goreleaser release` has no `--dist` flag.** Dist comes from configuration
  only. So `GoReleaserOptions.Dist` is validated, never forwarded, and `--dist`
  must be a basename because GoReleaser writes relative to the working directory.
  The consumer obligation — `.goreleaser.yaml`'s dist must equal the workflow's
  `--dist` — is documented, not cross-checked by the CLI.
- **GoReleaser colorizes even into a pipe**, so the retained stderr tail carried
  raw ANSI into the `--json` envelope error. Escapes are now stripped from the
  tail only; the live stream keeps its color for the workflow log.
- **A laptop cannot run the full build.** `gomod.proxy: true` resolves
  `module@current-tag` through the module proxy, so an unpublished local tag fails
  at proxying. The real invocation was still proven end to end: it cleaned dist,
  printed `skipping announce and publish`, reported `release is disabled`, and
  died at the proxy. The complete build stays a tag-time event, as the plan says.

Three defects the review rounds caught, all fixed:

1. **A fail-open older than this slice.** `runStage` never checked
   `settings.err`, so a malformed `RELEASE_*` boolean was silently ignored. That
   was harmless while `stage` only validated; once it builds, `RELEASE_JSON=yes`
   would have run GoReleaser and could exit 0 against the documented exit-2
   contract. Lesson: when a read-only command becomes a side-effecting one,
   re-audit every error it used to be able to ignore.
2. **A nil-seam panic.** The injected `RunGoReleaser` had no not-configured
   guard, so deleting the `withDefaults` assignment made the binary panic, emit
   no envelope, and exit 2 — colliding with the reserved usage code. Every other
   injected collaborator already had that guard.
3. **A regression introduced by one of my own fix instructions.** Collapsing the
   dist rule onto `ParseRootName` (the right I1 call) quietly narrowed separator
   rejection to `/` only, because `ParseRootName` used
   `ContainsRune(raw, filepath.Separator)` while the ad-hoc CLI check had used
   `ContainsAny(dist, "/\\")`. `--dist 'a\b'` went from a pre-build exit 2 to a
   post-build exit 1, and would have inverted on Windows. Lesson: when you tell an
   agent to delete a check in favor of an existing domain constructor, diff the two
   rules first — this is the same class as "when a port replaces a regex, diff the
   regex" from session 003.

Judgment calls recorded for later, deliberately not acted on:

- **The `execx` prohibition should probably be rescinded.** The bounded
  stderr-tail plus `WaitDelay` plus `LookPath` exec helper now exists four times:
  `cosign`, `melange`, `apko`, `goprof`. Conformance recommends amending the
  architecture and extracting it. That is an architecture change, so it is out of
  scope for a slice PR; it belongs in a follow-up with `ARCHITECTURE.md` updated.
- **`Options.RunGoReleaser` is a function-shaped outbound seam, not just a test
  hook.** Conformance calls it port 14 in substance. The plan's prohibition was on
  an interface plus adapter package, which is what we avoided; the budget claim
  should be read that way rather than as "no outbound seam at all".

Caller-ceiling audit: no `permissions` block changed anywhere; all four caller
jobs still match their callees in this repo, the example, and the docs skeleton.

Next: PR 11 replaces the `FULL_SHA` placeholders and the stale
`fb8c8098…` pin with the released slice-6 commit across README, docs, and the
example. It depends on PR 10 being released and verified first, so the release
that PR 10's merge produces is the gate.


