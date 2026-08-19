---
id: 002
title: Release CLI architecture and first two slices
date: 2026-08-18
status: complete
repos_touched: [meigma/release, meigma/release-selfref-spike, meigma/release-oras-spike, meigma/release-stamp-spike]
related_sessions: ["001"]
---

## Goal

Design a profile-driven Go CLI to replace the bespoke logic in this repo's four
release workflows, then begin implementing it. The CLI had to be portable across
several GitHub organizations, testable on a laptop without CI, and dogfooded by
this repository releasing itself.

## Outcome

Met for this session's scope, and the system now releases itself.

Delivered: a reviewed architecture (`ARCHITECTURE.md`, revision 3), an
eleven-PR implementation plan (`PLAN.md`), three completed gating spikes, and
the first two PRs merged. `v0.1.0` is published, and the release that published
it was orchestrated by the branch-built `release-cli` — the dogfood loop is
closed and proven, not asserted.

Not done, by design: PRs 3 through 11. The OCI publication, GitHub Release
publication, image build, and GoReleaser-invocation slices remain, along with
the documentation pin that closes the program.

Measured scope correction worth inheriting: the migration covers **42 of 70**
logical workflow steps, an upper bound of about **1,059 of 1,641** lines
(~64.5%). The earlier "~90%" figure only holds if scoped to bespoke *decision*
logic and was retired from the plan.

## Key Decisions

- Ecosystem-neutral verbs with a `--profile` flag, direct-dispatched to a `goprof`
  package -> a profile interface and registry would be structure for a second
  ecosystem that does not exist; the trigger is a second real profile.
- Stdlib `fs.FS`/`io.Reader` boundaries for local file work, `os.OpenRoot` at the
  composition edge -> A1 requires a boundary, not a bespoke one; `os.OpenRoot`
  also rejects symlink escapes that lexical checks miss.
- Two-phase OCI publication (`publish oci prepare` -> YAML `actions/attest` ->
  `publish oci finalize`) -> `actions/attest` needs job-level permissions a CLI
  cannot hold, and invariant 14 requires trust metadata before public tags. A
  single command cannot satisfy both.
- The prepare result travels as the `--json` envelope re-fed on stdin, not a file
  -> the transport already exists and there is nothing to clean up.
- One release unit: workflows, composite action, and CLI share one version and
  one consumer pin; no `cli-version` input -> supporting independent pinning would
  turn every YAML/binary touchpoint into a versioned API with a support matrix.
  `cli-path` remains as an unsupported escape hatch, and the failure asymmetry
  decides it: newer-workflow/older-CLI fails loudly, older-workflow/newer-CLI can
  fail silently.
- Viper removed under L1 -> six calls for three environment variables did not
  justify its dependency tree in a supply-chain tool, and the config file it would
  serve is trigger-gated.
- Retry lives in the `pubgh` engine with an injected `SleepFunc`, not in the
  adapter and not in an `internal/clock` package -> keeps policy testable and
  instant; consequence is that a direct `ghact.Client.Get` caller gets no retry.
- Registry login survives as OP-09 -> spike B proved `cosign` and
  `actions/attest --push-to-registry` read the docker config, so in-memory
  `oras-go` credentials cannot replace it. The earlier claim that login was
  obsolete was false and is corrected in the architecture.
- Standing per-PR method recorded in `PLAN.md` -> parallel implementation agents
  with one owner per file, a long-lived reviewer (max 2 rounds), a long-lived
  conformance agent (1 round), a caller-ceiling audit, then a parent gate.

## Changes

- `.journal/002/ARCHITECTURE.md` - approved architecture, revision 3, after a
  correctness review and a complexity review.
- `.journal/002/PLAN.md` - eleven-PR plan, standing execution method, spike
  results recorded inline, and reconciled drift.
- `.journal/002/ARCHITECTURE-REVIEW.md`, `ARCHITECTURE-COMPLEXITY-REVIEW.md`,
  `research/` - review punch lists and the three grounding documents.
- `AGENTS.md` - adopted `meigma/template-go`'s numbered rule set (PR #4).
- `cmd/release-cli/`, `internal/cli/`, `internal/stage/`, `internal/profile/goprof/`
  - the CLI and `stage --profile go`, replacing PP-06/07/08 (PR #5).
- `internal/stage/pubgh/`, `internal/adapter/ghact/` - first port, adapter, and
  generated mock; `verify handoff` replacing OB-06/GR-05/OP-03 (PR #7).
- `.github/actions/setup-release-cli/` - composite acquisition with a fail-closed
  installed path and a warn-only `cli-path` path.
- `.github/workflows/` - producer and three consumers cut over; `release.yml`
  dogfood plumbing; new dispatchable `verify-setup-installed.yml`.
- `scripts/check-protocol-stamp.sh`, `scripts/check-mocks.sh`, `.mockery.yml`,
  `moon.yml`, `mise.toml` - protocol-stamp and mock-freshness gates.
- `.goreleaser.yaml`, `melange.yaml`, `apko.yaml`, `release-please-config.json`
  - `release-mvp` -> `release-cli` artifact rename.
- `docs/reference/release-cli-contract.md` plus contract and how-to updates.

## Open Threads

- PRs 3 through 11 remain. PR 3 is `plan tags` with `internal/rel`, `StateReader`,
  and the `reg` read path; spike B already cleared its gate.
- Release-please PR #9 (`chore(main): release 0.1.1`) is open for the hotfix
  commit. Harmless either way: the fix is already inside the re-pointed `v0.1.0`,
  so only that changelog entry is missing.
- Three scratch repos are archived but not deleted, plus the `spike/self-ref`
  branch and the `ghcr.io/meigma/release-oras-spike` package: the token lacks
  `delete_repo`. `meigma/release-oci-remediation-e2e` from session 001 is still
  pending too.
- The dead `release-please--branches--main--components--release-mvp` branch can be
  deleted; it belongs to the retired component name.
- `cmd/release-cli/main.go` has no CLI-level testscript coverage; the plan defers
  testscript.
- A nested or absolute `--dist` value fails closed and is documented as the
  basename rule rather than supported.

## Lessons

- `startup_failure` runs expose no diagnostics through REST or GraphQL — no jobs,
  no logs, empty `checkRuns`. The run page in a browser is the only source of the
  validation message.
- A called workflow can never exceed its caller's permission ceiling, and PR CI
  cannot catch a missing grant when the caller only runs on tags. Audit both
  sides whenever a callee's `permissions` change.
- Tags created with `GITHUB_TOKEN` do not trigger tag-push workflows, which is
  why release-please must keep minting an App token. A user-pushed tag does
  trigger them, which is what made release recovery possible.
- Release-please's `yaml`+`jsonpath` updater re-serializes and strips comments;
  the annotated `generic` updater edits surgically. Use `generic` for
  `action.yml`.
- Twice an apparent regression was a harness error of mine — a tampered smoke
  fixture reused across checks, then a stub server that silently served a valid
  response for cases it did not implement. Verify the harness before believing a
  regression.
- Reviews earned their keep on things tests could not see: a trust root that
  collapsed onto the consumer's own repository, a token never passed to the steps
  that needed it, and a silently dropped retry budget.

## References

- Architecture: `.journal/002/ARCHITECTURE.md`; plan: `.journal/002/PLAN.md`
- PRs merged: https://github.com/meigma/release/pull/4,
  https://github.com/meigma/release/pull/5,
  https://github.com/meigma/release/pull/7,
  https://github.com/meigma/release/pull/8, and release
  https://github.com/meigma/release/pull/6
- Release: https://github.com/meigma/release/releases/tag/v0.1.0
- Dogfood run: https://github.com/meigma/release/actions/runs/32214805914
- Installed-path proof: https://github.com/meigma/release/actions/runs/32215196650
- Spike evidence: `PLAN.md` sections 5A, 5B, 5C
- Prior session: `.journal/001/SUMMARY.md`
