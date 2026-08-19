---
id: 003
title: Session start
started: 2026-08-18
---

## 2026-08-18 21:28 — Kickoff

Goal for the session: not yet stated. The developer asked to start a new
session; the actual request follows.

Current state of the world:

- `main` is at `2c262ba` ("fix(release): grant attestations read on the
  oci-image call job (#8)"), clean, in sync with `origin/main`.
- `v0.1.0` is published and the dogfood loop is closed: the tag run builds
  `release-cli` from the tagged commit and that binary stages and verifies its
  own release.
- Session 002 delivered the approved architecture
  (`.journal/002/ARCHITECTURE.md`, revision 3) and the eleven-PR plan
  (`.journal/002/PLAN.md`, which also holds the standing per-PR execution
  method and the three spike results). PRs 1 and 2 of that program are merged.
- Next planned slice is PR 3: `plan tags` with `internal/rel`, `StateReader`,
  and the `reg` read path. Spike B already cleared its gate.
- Known open threads carried in: release-please PR #9
  (`chore(main): release 0.1.1`) is open and harmless; archived scratch repos
  and the `spike/self-ref` branch cannot be deleted without `delete_repo`; the
  dead `release-please--branches--main--components--release-mvp` branch is
  removable; consumer docs and `examples/go-release/` still pin
  `fb8c809`, which PR 11 replaces.

Plan: wait for the developer's stated goal. If it is the release program,
re-read `.journal/002/PLAN.md` and `ARCHITECTURE.md` before touching code and
follow the standing per-PR method.


## 2026-08-18 21:55 — PR 3 underway

Goal set: execute PR 3 of `.journal/002/PLAN.md` — `feat(oci): plan immutable release tags`.

Worktree `.wt/feat-release-cli-slice3` created from fetched `origin/main`
(`2c262ba`), branch `feat/release-cli-slice3`.

Contract decided before spawning any agent (this is the cross-agent
interface; wave agents code against it verbatim):

- `internal/rel`: `Version`/`ParseVersion`/`Compare`/`Tag`, `Digest`,
  `Tag`/`ParseTag`, `Scope` (exact/minor/major/latest), `Channel`,
  `ChannelsFor`, `TagState`, `ChannelState`, `Action`
  (create/accept/retain), `Decision`, `TagPlan.Apply`, `PlanTags`,
  sentinels `ErrImmutableTag`/`ErrChannelCorrupt`/`ErrStateIncomplete`,
  and a redacting `Secret`.
- `internal/stage/puboci`: `Image`/`ParseImage`, `Reference`, sentinels
  `ErrTagAbsent`/`ErrRetryable`/`ErrCorruptState`, the `StateReader` port,
  `CollectState`, `PlanTags`.
- `internal/adapter/reg`: oras-go v2 read client (`New`, `Resolve`,
  `Version`) with `Credentials{Username, Password rel.Secret}`.
- CLI: `plan tags [--image] [--version] --digest [--json]`, derived
  defaults from `GITHUB_REPOSITORY`/`GITHUB_REF_NAME`, envelope command
  `plan tags`, result `{image, version, digest, tags, decisions[]}` where
  `tags` is only the `create` decisions.

Deliberate divergences from the YAML being replaced, both fail-closed:
version components must fit `uint64` (the JS used BigInt), and
`CollectState` propagates a precise corrupt-annotation error instead of
handing the planner a `HasVersion=false` state.

Execution: wave 1 = `RelCore` (internal/rel) + `ContractDocs`
(docs/reference). Both complete. `ContractDocs` caught a real
inconsistency in my example payload — I had listed all four tags under
`tags` while two decisions were accept/retain; corrected to create-only.
Wave 2 in flight: `Puboci` (engine + port + mockery), `Reg` (adapter +
go.mod tidy), `CliTags` (command + root wiring + main).

Dependencies pre-added by the parent so agents do not race on `go.mod`:
`oras.land/oras-go/v2 v2.6.2` (spike B's pinned version) and test-only
`github.com/google/go-containerregistry v0.21.9`.

## 2026-08-18 22:49 — PR 3 open: meigma/release#10

Branch `feat/release-cli-slice3`, two commits (`fbfce88` implementation,
`757cda0` review fixes). `moon run root:check` green; `ci / ci` and
`Kusari Inspector` both pass on the PR. Not merged — a human accepts.

Shipped: `internal/rel` (Version/Digest/Tag/Scope/Channel/TagState/
ChannelState/Action/Decision/TagPlan/PlanTags + redacting Secret),
`internal/stage/puboci` (Image/Reference, `StateReader` = port 2 of 13,
`CollectState`, `PlanTags`), `internal/adapter/reg` (oras-go v2.6.2
read-only client + generated mock), `plan tags`, and the two reference
docs. No workflow change; the github-script planner stays authoritative
until PR 5.

Live read-only proof against the real `ghcr.io/meigma/release` package
(`0.1.0` = `sha256:bb696ae3…`): all-accept/zero-tags at the true digest;
`ErrImmutableTag` + exit 1 at a wrong digest; `0.2.0` creates every tag
after reading the `0.1.0` annotation; `0.0.1` creates exact+minor and
retains `0`/`latest`; derived defaults from `GITHUB_REPOSITORY`/
`GITHUB_REF_NAME`; unreachable registry → retryable transport failure with
no URL in the message. Authenticated path exercised with a real token
through the credential closure.

Review round 1 (`Reviewer`): accepted as-is, six non-blocking findings.
Conformance: 20 PASS / 4 FAIL. Fixed: default oras `retry.DefaultClient`
for production reads (E3), transport failures now classified
`ErrRetryable` with a URL-free reason, credentials moved out of the struct
into the auth closure (a `Client` value printed with `%+v` reflected into
`rel.Secret`'s unexported field — fmt cannot call String on it), no-op
`fmt.Errorf("%w", sentinel)` wraps removed, stale `Envelope`/package Godoc
updated for `plan tags` and exit 1, `RELEASE_JSON` documented, dead test
branch and a misnamed allocation test cleaned up, and `--json` +
configuration-failure coverage added.

Deliberately not fixed, recorded as follow-ups:

- The Mockery testify template emits no per-method Godoc. Conformance
  reads that as a D1 violation in generated code; the mock merged in PR 2
  has the identical shape, so changing it is a template-wide change, not a
  PR-3 fix.
- Collect-then-plan issues three channel reads even when the exact tag
  already conflicts, and can report a channel failure where the workflow
  reports the conflict. Documented as a divergence; revisit in PR 5 when
  the planner becomes authoritative.

Other things worth keeping:

- An in-package test cannot import the mocks it needs (`puboci` →
  `reg/mocks` → `puboci`), so `puboci`'s tests are blackbox
  `package puboci_test`, matching `pubgh`.
- `unparam` fires on *test helpers* whose parameters are always the same
  fixture constant. Three separate rounds of it cost real time; write
  helpers against the fixture constants directly.
- Wave-based dispatch worked: `rel` and docs first, then `puboci`/`reg`/
  `cli` against a contract fixed up front, with `Puboci` broadcasting over
  hub when its package compiled and when the mock existed.

## 2026-08-19 07:45 — PR 3 merged, PR 4 started

PR #10 squash-merged as `de75a92`; remote branch and `.wt` worktree
removed. Note for future merges: `gh pr merge --delete-branch` fails in a
Worktrunk worktree (`fatal: 'main' is already used by worktree at ...`)
*after* the merge succeeds — the merge lands, only the local branch
switch/delete fails. Verify with `gh pr view --json state` before
retrying, then clean up with `wt remove` plus `git push origin --delete`.

PR 4 (`feat(oci): prepare digest publication`) started in
`.wt/feat-release-cli-slice3b` from `origin/main` (`de75a92`).

Contract fixed before spawning: `puboci.Descriptor`, `DigestRef`
(`image@sha256:…`, since signing and content pushes must be digest-pinned
while the shipped `Reference` is tag-based), ports `ContentPusher`
(PushBlob/PushManifest/Verify) and `Signer` (SignRecursive) — ports 3 and
5 of the closed 13 — `ReadLayout` over `fs.FS`, and the
`release.dev/oci-prepare/v1` result.

Deviations from PLAN.md's indicative signatures, deliberate:

- `Repository` in the plan is the already-shipped `Image`; no synonym.
- `Verify(ctx, ref DigestRef)` instead of `Verify(ctx, ref, expected)`:
  a digest-pinned reference already carries the expectation.
- `OCIPrepareResult.ObservedState rel.ChannelState` cannot be JSON: its
  map is keyed by a struct. The wire type carries an ordered
  `observed[]` projection (exact, minor, major, latest) instead.
- No SBOM paths in the attestation subjects. Subjects are
  platform+digest; SBOM file names stay the workflow's business until the
  image slices own them.

Also adding `--plain-http` to the CLI: PLAN.md's own PR-4 verification
asks for an in-memory-registry CLI smoke, which is impossible over HTTPS.
Explicit flag, documented as local-registry only, rather than implicitly
downgrading loopback.

## 2026-08-19 09:16 — PR 4 open: meigma/release#11

Branch `feat/release-cli-slice3b`, two commits (`6ea5304` implementation,
`155c93a` hardening). `moon run root:check` green; `ci / ci` and
`Kusari Inspector` pass. Not merged — a human accepts.

Shipped: `puboci` layout reader + `Prepare` + `ContentPusher`/`Signer`
ports + `release.dev/oci-prepare/v1`, oras push half of `reg`, the
`cosign` exec adapter, and `publish oci prepare [--dry-run]`.

### The smoke test earned its keep

Local harness (kept in `/tmp/pr4-smoke`, not committed): a
`go-containerregistry` registry binary, a handmade two-platform layout
(7 blobs, one layer shared so dedup is observable), and a `cosign` stub
that records argv. First authoritative run failed immediately with
`file already closed`.

Cause: oras hands the content reader to `net/http`, and the HTTP
transport **always closes a request body**. A caller-owned `*os.File`
was therefore closed twice. Every mock and `fstest.MapFS` test passed —
MapFS's Close is idempotent, so the whole unit suite was blind to it.
Fix: the adapter wraps content in a `readerOnly` type that hides the
`io.Closer`; `TestPushBlobLeavesReaderOpen` pins the ownership rule.

Lesson worth carrying: mock-level tests cannot see ownership bugs at a
real I/O boundary. Run the binary against a real server.

### Review round 1

`Reviewer-2` returned "would not accept" with three blocking findings;
`Conformance-2` returned 20 PASS / 4 FAIL. All fixed:

- A stray 10.7 MB `release-cli` binary was committed at the repo root
  (an agent ran `go build ./cmd/release-cli` without `-o`). Removed;
  `/release-cli` added to `.gitignore`. This is the second time a stray
  binary reached a commit — check `git status` before every commit.
- HTTP 409 on a blob push was treated as success. Because verification
  only resolves manifests, a refused layer upload could produce a signed,
  `authoritative:true` result for an image with a missing layer. 409 is
  now an error; `errdef.ErrAlreadyExists` alone means "already present".
- `RELEASE_DRY_RUN=yes` parsed as `false` and performed a real
  publication plus signature. `resolveBool` now returns an error and any
  unparsable boolean is exit 2.
- `--plain-http` was environment-activatable and host-agnostic while
  carrying the registry token: any earlier Actions step could append to
  `$GITHUB_ENV` and downgrade the next run to cleartext. Now flag-only
  and refused unless the image host is loopback.
- E3: a streamed body has no `GetBody`, so oras-go's retry transport can
  never replay it — retry must live where the content can be reopened.
  The engine now retries `ErrRetryable` pushes and verifications four
  times (1s/2s/4s) with a reopened stream and an injected sleep.
- Plus `cmd.WaitDelay` so a cancelled `cosign` cannot hang the CLI
  (a grandchild holding the stderr pipe blocks `cmd.Wait` forever),
  a nil-content guard, a required `platform` on every index descriptor
  (an omitted one produced a `"/"` attestation subject), and tests for
  the 4 MiB JSON bound, symlink escape, and pushed media types.

### Notes for the next slices

- `oras.CopyGraph` was never used: explicit per-descriptor pushes mirror
  the YAML and stay deterministic, which sidesteps spike B's concurrency
  correction entirely.
- `PrepareInput.Sleep` (a `pubgh`-style `SleepFunc`) is the injected
  clock seam; finalize should reuse the same shape.
- Finalize consumes `observed[]` from the result to refuse drift; the
  ordering is exact, minor, major, latest.
- `--plain-http` plus the local registry harness is now the way to smoke
  a registry-touching command without GHCR.