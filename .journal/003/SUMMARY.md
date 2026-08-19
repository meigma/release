---
id: 003
title: Release CLI slices 3 through 4b
date: 2026-08-19
status: complete
repos_touched: [meigma/release]
related_sessions: ["002", "001"]
---

## Goal

Continue the eleven-PR `release-cli` program from `.journal/002/PLAN.md`, starting
at PR 3. Each slice replaces bespoke `actions/github-script` logic in the four
reusable workflows with a tested Go command, following the standing per-PR method:
parallel implementation agents with one owner per file, a reviewer round, a
conformance audit, a parent gate, then a human-accepted PR.

## Outcome

Met, and further than planned for one session. Five PRs (3 through 7) were
implemented, reviewed, hardened, and squash-merged:

| PR | Commit | Slice |
|---|---|---|
| [#10](https://github.com/meigma/release/pull/10) | `de75a92` | `plan tags`, `internal/rel`, `StateReader`, `reg` read path |
| [#11](https://github.com/meigma/release/pull/11) | `257ac5f` | `publish oci prepare`, `ContentPusher`/`Signer`, cosign adapter |
| [#12](https://github.com/meigma/release/pull/12) | `3a649f0` | `publish oci finalize`, `TagCommitter`, two-phase workflow cutover |
| [#13](https://github.com/meigma/release/pull/13) | `6930882` | `verify bundle`, `BlobVerifier`, cosign verify adapter |
| [#14](https://github.com/meigma/release/pull/14) | `df077f9` | `publish github`, four ports, `ghrel`/`ghup`/`gitx`, release cutover |

`main` is at `df077f9`. Ten of the thirteen approved ports exist; the OCI and
GitHub Release workflows no longer contain bespoke publication logic.

Not done, by design: PRs 8 through 11 (`image build`, `image verify`, moving the
GoReleaser invocation into `goprof`, and the documentation pin that closes the
program).

## Key Decisions

- **Defer the `actenv` Actions-runtime seam** (twice: PRs 5 and 6) -> the workflow
  captures the CLI's `--json` envelope with a `$GITHUB_OUTPUT` heredoc and `jq`,
  which needs no new port. The seam stays deferred until a command genuinely
  needs annotations or a job summary. Port 13 remains unbuilt.
- **Engine-owned polling** (PR 7) -> `FindDraft` and `WaitAssets` are single
  snapshots and the 24x5s / 12x1s budgets live on `PublishInput` with an injected
  sleep. One place decides retry policy and the state machine tests instantly.
- **Retry belongs where the content can be reopened** (PR 4) -> a streamed request
  body has no `GetBody`, so oras-go's retry transport can never replay it. The
  prepare engine retries `ErrRetryable` with a reopened stream instead.
- **`--plain-http` is flag-only and loopback-only** -> an environment-activatable
  transport downgrade could be set by any earlier Actions step through
  `$GITHUB_ENV` and would leak the token in cleartext.
- **Unparsable `RELEASE_*` booleans are exit 2** -> `RELEASE_DRY_RUN=yes` parsed as
  `false` and performed a real publication plus signature. Failing closed beats a
  silent unsafe default.
- **Wire types are strings, domain types are not** -> `OCIPrepareResult` carries an
  ordered `observed[]` projection because `rel.ChannelState` is a struct-keyed map
  and cannot be JSON.
- **Verify against real services wherever a laptop can reach them** -> live GHCR
  publication and tag monotonicity, verification of the real published `v0.1.0`
  Sigstore bundle, and two temporary draft releases in this repository. Mock-only
  evidence was treated as insufficient for every slice that mutates remote state.

## Changes

- `internal/rel` - the pure release model: `Version`, `Digest`, `Tag`, `Scope`,
  `Channel`/`ChannelsFor`, `TagState`, `ChannelState`, `Action`, `Decision`,
  `TagPlan`, `PlanTags`, and a redacting `Secret`.
- `internal/stage/puboci` - `Image`/`Reference`/`DigestRef`/`Descriptor`, the
  `StateReader`, `ContentPusher`, `Signer`, and `TagCommitter` ports, `ReadLayout`,
  `CollectState`, `PlanTags`, `Prepare`, `Finalize`, and the
  `release.dev/oci-prepare/v1` and `release.dev/oci-finalize/v1` results.
- `internal/stage/pubgh` - `BuildBundle`/`VerifyBundle` with `TrustPolicy` and the
  `BlobVerifier` port; the `ReleaseReader`, `AssetReplacer`, `Publisher`, and
  `RefResolver` ports with `Publish`; shared bounded retry in `retry.go`.
- `internal/adapter/reg` - oras-go v2 read, digest push, and serial tag commit.
- `internal/adapter/cosign` - `cosign sign --yes --recursive` and
  `cosign verify-blob`, sharing one hardened exec path.
- `internal/adapter/ghrel`, `ghup`, `gitx` - release reads and the single
  `draft:false` mutation, `gh release upload --clobber`, and `git rev-list -n 1`.
- `internal/cli` - `plan tags`, `publish oci prepare`, `publish oci finalize`,
  `verify bundle`, `publish github`, plus `RegistryConfig` and the port factories.
- `.github/workflows/publish-oci-image.yml` - ORAS and four github-script blocks
  replaced by prepare -> three `actions/attest` steps -> finalize.
- `.github/workflows/publish-github-release.yml` - draft discovery, upload, and
  finalize scripts replaced by `verify bundle`, `attest`, and `publish github`,
  with an early tag-to-commit gate.
- `docs/` - `explanation/two-phase-oci-publication.md` plus reference and how-to
  updates for every new command.

## Open Threads

- **PRs 8 through 11 remain.** PR 8 (`image build`) fires the plan's deferred
  persisted-projection trigger; PR 9 is `image verify`; PR 10 moves the GoReleaser
  invocation into `goprof`; PR 11 replaces the documentation pin `fb8c809` with the
  final program revision.
- **The next tag is the end-to-end proof.** Keyless Cosign signing and the three
  `actions/attest` steps only run inside Actions. The two-phase OCI path and the
  release path have never executed together in CI. `publish-image: false` and
  `publish-release: false` remain the documented rollbacks.
- **Ports 11-13 unbuilt:** `image.APKBuilder`, `image.Composer`, and the
  `cli.Actions`/`actenv` seam. The budget is still closed at 13.
- **Mockery's testify template emits no Godoc** for generated expecter types.
  Raised by conformance in three consecutive slices and deliberately deferred; it
  is a template-wide change, not a slice fix.
- Housekeeping debt inherited from sessions 001 and 002 is unchanged: archived but
  undeletable scratch repositories, the `spike/self-ref` branch, the
  `ghcr.io/meigma/release-oras-spike` package, and the dead
  `release-please--branches--main--components--release-mvp` branch.

## Lessons

- **Mock suites cannot see I/O ownership bugs.** The first live prepare run died
  with `file already closed`: oras hands the content reader to `net/http`, and the
  HTTP transport always closes a request body, so a caller-owned `*os.File` was
  closed twice. Every `fstest.MapFS` test passed because MapFS's `Close` is
  idempotent. Run the binary against a real server.
- **When a port replaces a regex, diff the regex.** The bundle verifier silently
  dropped the `[A-Za-z0-9][A-Za-z0-9._+-]*` asset-name grammar while the same PR's
  docs claimed it was enforced.
- **Ask what a test cannot fail.** Reviewers twice proved a check was unprotected
  by mutation: deleting the closed-set regular-file check left the suite green,
  and `AssertNotCalled` without argument matchers can never fail.
- **Fail-open defaults hide in boolean parsing and in "success" branches.**
  `RELEASE_DRY_RUN=yes` published for real; `--no-undraft` was ignored on the
  already-public branch; a 409 on a blob push counted as success.
- **Order irreversible steps first.** Folding the tag/SHA binding into the CLI
  moved it after `actions/attest`, so a mis-bound tag would have left an
  unwithdrawable provenance attestation before failing.
- **Re-broadcast a contract the moment it changes.** An agent improved the polling
  design mid-wave; a sibling had already built the old shape, and a dead
  `PollPolicy` parameter survived a round.
- **`golangci-lint` caches per path.** After removing a worktree it stopped
  excluding generated files and flagged every mock; `golangci-lint cache clean`
  fixes it.
- **The image's `org.opencontainers.image.version` annotation must equal the
  release version**, or channel planning refuses with an out-of-line error. apko
  stamps it; the workflow derives the version from the tag.

## References

- Plan and architecture: `.journal/002/PLAN.md`, `.journal/002/ARCHITECTURE.md`
- Merged PRs: [#10](https://github.com/meigma/release/pull/10),
  [#11](https://github.com/meigma/release/pull/11),
  [#12](https://github.com/meigma/release/pull/12),
  [#13](https://github.com/meigma/release/pull/13),
  [#14](https://github.com/meigma/release/pull/14)
- Prior sessions: `.journal/002/SUMMARY.md`, `.journal/001/SUMMARY.md`
- Two-phase rationale: `docs/explanation/two-phase-oci-publication.md`
