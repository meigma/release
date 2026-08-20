---
id: 005
title: Shared execution and cached source builds
date: 2026-08-20
status: complete
repos_touched: [meigma/release]
related_sessions: ["002", "004"]
---

## Goal

Remove duplicated subprocess plumbing without collapsing tool-specific adapters, then assess and simplify the release workflows' CLI acquisition path.

## Outcome

Met. PR #18 introduced one internal subprocess runner and migrated all seven production call sites without changing their domain contracts. PR #19 replaced the release workflow's build-once CLI artifact with exact-source local builds backed by exact-key Go caches, while consumer repositories retained the released, checksum- and attestation-verified acquisition path. Both PRs were squash-merged.

## Key Decisions

- Treat `internal/execx` as local standard-library plumbing, not a port or adapter; command policy, retries, parsing, and errors remain consumer-owned.
- Keep the OCI workflow's Bash orchestration and the composite setup action; neither a `github-script` rewrite nor a bundled JavaScript action removes the external process boundary.
- Build from source automatically only for a matching self-release; require action and reusable workflow repository identity plus the exact runner-provided workflow SHA.
- Cache exact `GOCACHE` and `GOMODCACHE` entries, never the executable; a cache miss remains a complete build.
- Preserve the installed release path for consumers and force it in installed-path verification, retaining checksum and GitHub attestation verification outside self-release jobs.

## Changes

- `internal/execx/` — added the sole production `os/exec` boundary with deferred binary lookup, context cancellation, bounded stderr capture, output forwarding, `WaitDelay`, and typed exit metadata.
- `internal/adapter/{apko,cosign,ghup,gitx,melange}/` and `internal/profile/goprof/` — migrated seven command paths to `execx.Run` and removed six private helper implementations.
- `.github/actions/setup-release-cli/action.yml` — added `local-build` acquisition policy, exact reusable-workflow source selection, pinned Go setup, exact-key cache restoration, source build stamping, and version/protocol verification.
- `.github/workflows/{release,go-pre-publish,go-oci-build,publish-github-release,publish-oci-image}.yml` — removed the dedicated CLI build job, same-run artifact transport, workflow-level `cli-path` inputs, and artifact-placement steps.
- `.github/workflows/verify-setup-installed.yml` — forces `local-build: never` so the released acquisition path remains exercised.
- `docs/explanation/release-trust-boundaries.md`, `docs/how-to/`, and `docs/reference/` — documented source-build acquisition, permissions, cache behavior, and the resulting trust boundaries.
- `.journal/005/{PLAN.md,ARCHITECTURE-AMENDMENT.md}` — recorded and executed the shared execution design.

## Open Threads

- No tag was created during this session. The next release remains the first production run of the complete CLI-owned build and publication path.
- The documentation-pin slice from `.journal/002/PLAN.md` remains gated on a released and verified current CLI build.
- `cli.Actions`/`actenv` remains deliberately deferred until a CLI command needs native Actions annotations or a job summary.
- Release Please PR #9 must be refreshed after the next intended release version is chosen.

## Lessons

- Exact-key Go build and module caches remove repeated compilation cost across sequential jobs without treating the output binary as a cache artifact.
- Four independent Ubuntu 24.04/amd64 builds from the same source SHA were byte-identical, supporting per-job source builds without a cross-job binary artifact.
- `gh pr merge --delete-branch` can merge remotely and then fail during local branch cleanup when another worktree owns `main`; inspect PR state before retrying the merge.

## References

- PR #18: https://github.com/meigma/release/pull/18
- PR #19: https://github.com/meigma/release/pull/19
- Production PR #19 CI: https://github.com/meigma/release/actions/runs/32383263681
- Clean-cache proof: https://github.com/meigma/release/actions/runs/32335377611
- Shared execution plan: `.journal/005/PLAN.md`
- Architecture amendment: `.journal/005/ARCHITECTURE-AMENDMENT.md`
- Baseline architecture: `.journal/002/ARCHITECTURE.md`
