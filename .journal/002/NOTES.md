---
id: 002
title: Session start (goal pending)
started: 2026-08-18
---

## 2026-08-18 12:35 — Kickoff
Goal for the session: not yet stated. The user asked to start a new session without naming a task; the concrete goal will be appended when they state their actual request.
Current state of the world: `main` is at squash commit `5566640c061c5e36f3715e0a1b57eaf69646a0ba` (PR #2), which carries the rehearsed GitHub Release and OCI delivery MVP. `journal/jmgilman` is at `b452dba` (`docs(journal): close session 001`) and in sync with `origin`. Session 001 is closed and complete.
Open threads inherited from 001: Homebrew, MacPorts, Nix, Scoop, generalized installer, and mise channels are unimplemented; Release Please PR #3 (`chore(main): release 0.1.0`) is unmerged; documented reusable-workflow pins still point at pre-squash revision `fb8c8098ff27968fb3070e928c00e925f38c698e`; disposable rehearsal repo `meigma/release-oci-remediation-e2e` awaits manual deletion.
Plan: wait for the user's request, then work incrementally in an implementation worktree created from fetched `main`, integrating via GitHub PR squash merge.

## 2026-08-18 12:41 — Adopt template-go rules in AGENTS.md
Goal (now stated): pull the rules from `~/code/meigma/template-go/AGENTS.md` into this repo's `AGENTS.md`.
Findings: our `AGENTS.md` held only the `ai-protocol` block, identical to the template's. `CLAUDE.md` is a symlink to `AGENTS.md`, so it inherits any change automatically.
Done: worktree `docs/agents-go-rules` off `main`; appended the template's `# Go Best Practices` section (A/T/R/P/D/I/E/L rule groups) verbatim. `diff AGENTS.md ~/code/meigma/template-go/AGENTS.md` reports identical. Commit `docs(agents): adopt template-go engineering rules`, PR https://github.com/meigma/release/pull/4.
Gotcha: `read` of the template showed 97 lines while the file has 98; the first `sed -n '14,97p'` silently dropped the final rule line. Verified with `diff`, then reappended with `sed -n '14,$p'`.
Next: PR #4 awaits review/squash merge. Existing Go code in `cmd/release-mvp` and `internal/cli` has not been audited against the newly adopted rules (notably D1/D4 Godoc and `doc.go`).

## 2026-08-18 12:45 — PR #4 merged
Merged PR #4 as squash commit `0c39bedb9551031b2d704993397a2c2ce0a3769b` on `main`; `AGENTS.md` on `main` verified byte-identical to the template's.
Gotcha: `gh pr merge --squash --delete-branch` merged server-side but then failed locally with `fatal: 'main' is already used by worktree at ...` (gh tried to check out `main` inside the branch worktree, which Worktrunk already holds in the primary checkout). Consequence: neither the remote nor local branch was cleaned up by gh. Recovered with `git fetch --prune`, `git pull --ff-only` in the main checkout, `wt remove docs/agents-go-rules`, and `git push origin --delete docs/agents-go-rules`.
Takeaway for future merges in this repo: run `gh pr merge` from the main checkout, or pass `--delete-branch` only after leaving the PR worktree; expect to prune the branch manually otherwise.
Next: still open from session 001 — deferred package channels, release PR #3, doc pins on pre-squash SHA, rehearsal repo deletion. Newly adopted A/T/R/P/D/I/E/L rules have not been applied to existing Go code.

## 2026-08-18 12:52 — New goal: release CLI architecture
Stated goal: design a profile-driven Go CLI that replaces ~90% of the bespoke logic in `go-pre-publish.yml` (145 lines), `go-oci-build.yml` (445), `publish-github-release.yml` (482), and `publish-oci-image.yml` (569) — ~1650 lines total.
Requirements captured from the user: profile flag rather than per-language commands (`release-cli prepublish --profile go`); hexagonal architecture with all I/O behind ports and strong domain typing per AGENTS.md; locally testable without CI; this repo dogfoods the CLI (CI builds the branch binary, consumers install a released version, workflows branch between the two); portable across at least four GitHub orgs with only credential-level changes (no hardcoded `meigma`); weigh custom action vs reusable workflow from the consumer's perspective against GitHub's documented limits; base the CLI on `~/code/meigma/template-go`.
Approach: three parallel grounding researchers (workflow capability inventory, Actions packaging constraints, template-go conventions) -> software-architect -> adversarial architecture review -> final doc in `.journal/002/`.

## 2026-08-18 13:32 — Architecture draft + adversarial review landed in journal
Pipeline run: three parallel grounding researchers -> software-architect (rev 1, 313 lines) -> architecture-reviewer (276-line punch list, verdict "approve with changes") -> architect revision 2 (335 lines, §13 dispositions table).
Deliverables in `.journal/002/`: `ARCHITECTURE.md` (revision 2, for review), `ARCHITECTURE-REVIEW.md`, and `research/{workflow-inventory,actions-constraints,template-conventions}.md` as cited ground truth.
Shape settled in rev 2: `release-cli` with ecosystem-neutral verbs (`stage`, `image build|verify`, `publish oci prepare|finalize`, `publish github`, `plan tags`, `verify handoff|bundle`, `config`), `--profile` only on `stage`; pure core `internal/rel`; narrow consumer-declared ports with one concrete adapter each; profiles as compiled-in Go interfaces emitting one logical release manifest with stage-scoped projections copied into today's two disjoint artifacts.
Key review-driven corrections: OCI publication split into `prepare` -> YAML `actions/attest` -> `finalize` so invariant 14 (trust metadata before public tags) stays enforceable; narrow FS ports (`BundleScanner`, `ArtifactsReader`, `manifest.Store`) instead of I/O in engines; `ChecksumSet` (claim) vs scanner-reconciled `Bundle` (fact); service-surface ports decomposed; "safe to re-run" dropped from exit 3 in favor of a `mutations` status plus per-command postcondition reconciliation; `$/` skew claim demoted to a stamped protocol-integer handshake with the distribution repo derived from `github.action_repository`; `actions/create-github-app-token` stays in YAML; `gh release upload --clobber` retained behind an `AssetReplacer` port.
CI delivery recommendation: layered reusable workflows -> composite `setup-release-cli` -> CLI, because job-scoped permissions/environments/concurrency/needs/matrices/outputs exist only in reusable workflows while a composite is the only same-job CLI bridge. Dogfood via explicit `cli-path` from a branch build; consumers get a stamped default version plus protocol check.
Slice 1 (one PR): `internal/rel`, `profile`+`goprof` selection, `stage/assets` engine, `bundlefs`/`manfs`/`actenv` adapters + mocks, `--json` envelope, `stage --profile go` replacing PP-06–PP-08 bash (~80 lines), manifest emission into both uploads, minimal `setup-release-cli`. GoReleaser invocation stays in YAML until slice 6.
Open decisions for the user: approve rev 2 or push back on the projection-vs-combined-artifact call, native `oras-go` vs ORAS binary (scratch-GHCR spike is a slice-3 acceptance gate), and whether attestation creation eventually moves into the CLI.
Next: user reviews `.journal/002/ARCHITECTURE.md`; no code written yet.
