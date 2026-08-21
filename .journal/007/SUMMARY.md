---
id: 007
title: Scoop delivery and bucket initialization
date: 2026-08-21
status: complete
repos_touched: [release, scoop-bucket, scoop-bucket-rehearsal, scoop-publisher-rehearsal]
related_sessions: [006]
---

## Goal

Design, rehearse, implement, and operate a Scoop delivery channel that follows the proven Homebrew model: generated controls, protected destination-repository pull requests, secret-free validation, production workflow integration, and deterministic local initialization.

## Outcome

The goal was met. Four production PRs landed managed Scoop bucket CI, a fail-closed domain publisher and GitHub adapter, the reusable and production publication workflows, and `release-cli init scoop-bucket` with complete operator documentation. Release `v0.1.6` exercised the full production path and opened independently validated Homebrew and Scoop pull requests. The Scoop publisher's App token, repository scope, manifest integrity, native Windows AMD64 lifecycle, and native Windows ARM64 lifecycle were proven against real repositories and public release assets.

## Key Decisions

- Keep Scoop publication channel-specific -> `pubscoop` and `ghbucket` reuse neutral execution and GitHub value seams without forcing Homebrew and Scoop into a generic package-manager engine.
- Use root-level manifests -> GoReleaser, the publisher write path, and the managed validation workflow now agree on `<manifest>.json`; installation and search work without a `bucket/` directory.
- Validate both Windows architectures immediately -> public `windows-11-arm` runners proved the native ARM64 archive instead of carrying an untested generated manifest entry.
- Pin Scoop source, official tests, reusable workflows, and third-party actions by full commit -> destination validation is reproducible and does not inherit moving upstream behavior.
- Keep generated package-manager controls outside the signed and public release payload -> each publisher isolates its own control, verifies the remaining bundle, restores the control, and opens a reviewed destination-repository PR.
- Mint repository-scoped App tokens only after artifact handoff and signed-bundle verification -> disabled workflows require no credential, and enabled runs cannot reach an unrelated repository.
- Keep initializers local-only -> deterministic scaffolds use atomic directory installation; Git, GitHub, rulesets, App installation, credentials, and the first publication remain explicit operator steps.

## Changes

- `release/.github/workflows/scoop-bucket-ci.yml` - added secret-free root-manifest discovery and pinned schema/lifecycle validation on Windows AMD64 and ARM64.
- `release/internal/stage/pubscoop/` and `release/internal/adapter/ghbucket/` - added fail-closed Scoop publication reconciliation, bounded retry, focused GitHub repository reads and writes, and generated mocks.
- `release/internal/cli/scoop.go` and CLI wiring - added authenticated `publish scoop` with stable JSON output and exit behavior.
- `release/.goreleaser.yaml` - generated the `meigma-release-cli` Scoop manifest with explicit release URLs and `skip_upload: true`.
- `release/.github/workflows/publish-scoop.yml` and `release/.github/workflows/release.yml` - added default-off reusable publication and independent production integration after the public GitHub Release.
- `release/internal/cli/scoop_init.go` and `release/internal/cli/scoop_init_test.go` - added deterministic, full-commit-pinned bucket scaffolding with byte-exact and filesystem-safety coverage.
- `release/docs/how-to/set-up-scoop-bucket.md` and `release/docs/reference/release-cli-contract.md` - documented initialization, App access, credentials, producer wiring, branch protection, failure behavior, and install/update/uninstall verification.
- `scoop-bucket` - bootstrapped the protected production bucket through PR 1; release `v0.1.6` opened manifest PR 2 with exact public archive digests.
- `scoop-bucket-rehearsal` - proved root-manifest schema validation, install, update, uninstall, bad-hash rejection, unavailable-URL rejection, and native ARM64 execution.
- `scoop-publisher-rehearsal` - proved `created` -> `open` -> conflict -> `published` convergence; the undeletable disposable repository was archived.

## Lessons

- Scoop installs and searches root-level manifests, but pinned Scoop's `scoop bucket list` implementation counts only files beneath `bucket/` and therefore reports `Manifests 0` for the selected layout. Treat the required validation workflow, search, and real install as the health proof.
- Pinned Scoop syntax tests require CRLF in the Windows checkout. The bucket scaffold therefore owns `.gitattributes`; relying on runner defaults is insufficient.
- The official schema test limits discovery to Git-changed manifests when `CI` is set. Clear `CI` when the intent is to validate every manifest after a workflow-only change.
- A required manifest-only status check does not report on Dependabot pull requests that change only the caller workflow. Reproduce those updates on a maintainer branch with a semantically neutral root-manifest formatting change so validation runs; do not bypass protection.
- GitHub App installation coverage is distinct from App permissions. The first token rehearsal correctly failed until `meigma/scoop-bucket` was added to the selected repositories.

## Open Threads

- `meigma/scoop-bucket` PR 2 and `meigma/homebrew-tap` PR 7 remain open intentionally for protected human review; their checks pass and their generated release URLs target public `v0.1.6` assets.
- MacPorts and a generalized installer remain deferred.
- `meigma/scoop-publisher-rehearsal` is archived because the available token lacks `delete_repo`.

## References

- [Release PR 36: managed Scoop bucket CI](https://github.com/meigma/release/pull/36)
- [Release PR 37: Scoop publisher and CLI](https://github.com/meigma/release/pull/37)
- [Release PR 39: reusable and production Scoop publication](https://github.com/meigma/release/pull/39)
- [Release PR 41: deterministic Scoop bucket initializer](https://github.com/meigma/release/pull/41)
- [Production release v0.1.6](https://github.com/meigma/release/releases/tag/v0.1.6)
- [Production Scoop manifest PR 2](https://github.com/meigma/scoop-bucket/pull/2)
- [Production Homebrew cask PR 7](https://github.com/meigma/homebrew-tap/pull/7)
- [Session 006 summary](../006/SUMMARY.md)
