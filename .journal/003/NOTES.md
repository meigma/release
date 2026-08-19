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

## 2026-08-19 11:10 — PR 4 merged, PR 5 open: meigma/release#12

PR #11 merged as `257ac5f`. PR 5 (`feat(oci): finalize trusted image
tags`) is open as #12 on `feat/release-cli-slice3c`, two commits
(`f694aea`, `4dd6a75`), `moon run root:check` green, both PR checks pass.
Not merged — a human accepts.

This is the cutover: the publisher workflow no longer publishes, signs,
or tags. ORAS is gone from the toolchain, the four github-script
publication blocks are deleted, and the job is now prepare → three
`actions/attest` steps → finalize. Inputs, the six outputs, permissions,
concurrency, timeouts, and SHA pins are untouched, so the caller-ceiling
audit is a no-op this time.

### Live GHCR rehearsal (scratch packages, since deleted)

With a recording cosign stub and a real token:

- prepare pushed content and left `tags: null`; finalize applied
  `0.0.1`, `0.0`, `0`, `latest`, and each resolved to the index digest.
- publishing `0.0.2` moved every channel; republishing the older `0.0.1`
  returned `accepted:[0.0.1] retained:[0.0,0,latest]` and left the tag map
  on `0.0.2`. Invariants 11 and 12 proven against real GHCR, not a fake.
- A fixture whose index annotation said `1.4.0` while I published it as
  `0.0.1` was refused: `channel 0.0 points outside its minor release
  line`. My fixture was wrong and the planner caught it. **Operational
  fact worth keeping: the image's `org.opencontainers.image.version`
  annotation must equal the release version, or channel planning refuses.
  apko stamps that annotation; the workflow derives the version from the
  tag. They must agree.**
- Cleanup: both scratch packages deleted via `gh api -X DELETE
  /orgs/meigma/packages/container/<name>` (the token has
  `delete:packages`, unlike session 001/002's).

Still only provable in Actions: keyless Cosign signing and the three
attest steps. Spike B covered them on real GHCR; the first tag after this
merge is the end-to-end proof, and `publish-image: false` is the
documented rollback.

### Review round 1

`Reviewer-3` withheld acceptance on one correctness finding, and it was a
good one: the drift convergence exception accepted ANY tag that now sat
on the candidate digest, justified in the docs as "our own partial
progress". But a channel the plan chose to **retain** (it pointed at a
newer release) can never be our own work — if something else moved it
onto our digest during the attestation window, finalize converged and
reported `accepted`, masking a channel regression. `rel.PlanTags`
short-circuits on the digest match before the monotonicity check, so
nothing downstream caught it either. The exception is now scoped to tags
whose prepared observation implies a create decision.

Also fixed: the finalize stdin decode was unbounded (the envelope's
`result` is a `json.RawMessage`, so the inner 4 MiB limit never applied);
`Commit` undercounted applied tags when a write landed but its
verification read failed; the credential-scrub step could fail the whole
job — and thereby a successful publication — while leaving the credential
in place; and four test gaps the plan explicitly names, including the
ambiguous-write case and a full prepare→finalize pass over the in-memory
registry.

### Harness lesson

`golangci-lint` silently stopped excluding generated files and flagged
every mock, because its cache still referenced a deleted worktree
(`.wt/feat-release-cli-slice3b`) and the exclusion stage failed to read
those paths. `golangci-lint cache clean` fixed it. After removing a
worktree, clear the lint cache before trusting a lint failure.

## 2026-08-19 12:45 — PR 5 merged, PR 6 open: meigma/release#13

PR #12 merged as `3a649f0`. PR 6 (`feat(release): verify signed release
bundles`) is open as #13 on `feat/release-cli-slice4a`, two commits
(`3dfb22d`, plus the round-1 fix commit), `moon run root:check` green,
both PR checks pass. Not merged — a human accepts.

Shipped: closed `Bundle` reconciliation with the `BlobVerifier` port
(10 of 13), the `cosign verify-blob` adapter, `verify bundle`, and the
`publish-github-release.yml` cutover. verify → attest → upload is intact
and permissions are untouched.

### Smoke used the real published release

Downloaded all 14 assets of `v0.1.0` and verified them with the real
pinned cosign against the real Fulcio identity
`…/go-pre-publish.yml@refs/tags/v0.1.0`: 12 payloads plus both controls,
exit 0. Negatives against the same real bundle: extra unlisted file,
tampered payload, payload replaced by a symlink, missing sigstore
bundle, and a wrong signer identity (`no matching CertificateIdentity
found`) all exit 1. A recording cosign wrapper proved zero invocations on
a local failure and exactly one on success.

### Review round 1: two blocking findings, both proven by experiment

1. **The asset-name grammar had been silently dropped.** The replaced JS
   matched `^([0-9A-Fa-f]{64}) [ *]([A-Za-z0-9][A-Za-z0-9._+-]*)$`;
   `stage.ParseAssetName` only rejected empty names and path separators,
   so `.hidden`, `my file.tar.gz`, and a tab-bearing name verified and
   exited 0 — while this same PR's docs claimed the CLI enforced the
   character set. Reviewer demonstrated it with the built binary. Fixed
   in `stage.ParseAssetName` so `stage` and `verify bundle` share one
   definition. **Lesson: when a port replaces a regex, diff the regex,
   not the prose.**
2. **The symlink defence was untested and the table could not reach it.**
   Every hostile row used an *unlisted* name, so all three exited through
   the "unlisted entry" branch and never reached the regular-file check.
   Reviewer mutated the code: deleting that check entirely left the whole
   suite green, and the mutant binary then accepted a symlinked payload.
   `os.Root` follows in-root symlinks and `fs.Stat` resolves them, so the
   directory-scan mode check is the only protection. Now the closed-set
   scan runs before hashing, and both a MapFS case and a real
   `os.Symlink` case pin it.

Also fixed: the issuer is now validated like the identity (a malformed
issuer was reaching cosign and failing at exit 1 instead of exit 2);
`RELEASE_COSIGN_PATH` resolves through the injected env seam like the
signer path, so it is testable; the workflow derives the asset prefix
from the envelope's own `.result.dist` instead of repeating the literal;
`AssertNotCalled` with no matchers can never fail (testify's `Diff` walks
the recorded args) and was replaced; `BundleEntry.Digest` now uses
`stage.Digest`; and three reference statements plus two stale `doc.go`
comments were corrected.

### Carrying forward

- Mockery's testify template emits no Godoc for generated expecter types.
  Third slice in a row this is raised; it is a template-wide change and
  stays out of scope until someone does it deliberately.
- Next slice is PR 7 (`feat(release): publish verified GitHub releases`),
  which brings four ports at once — `ReleaseReader`, `AssetReplacer`,
  `Publisher`, `RefResolver` — plus `ghrel`, `ghup`, and `gitx`. That is
  the largest port batch in the program; consider splitting the wave by
  adapter.