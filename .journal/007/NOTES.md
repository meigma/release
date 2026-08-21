---
id: 007
title: New work session
started: 2026-08-21
---

## 2026-08-21 10:27 — Kickoff
Goal for the session: Start a fresh journal session; the substantive work request has not been provided yet.
Current state of the world: `main` is current, release `v0.1.3` is verified, and sessions 001 through 006 are complete.
Plan: Bind this session to the current task, then capture and execute the developer's next request.

## 2026-08-21 10:33 — Scoop familiarization
Goal: Plan Scoop support by reusing the proven Homebrew delivery method rather than inventing a second publication model.
Current Homebrew state: PRs 28 through 34 landed the managed tap CI, fail-closed `pubbrew` reconciliation engine, `ghtap` GitHub adapter, `publish homebrew` CLI command, reusable producer publisher, optional macOS signing/notarization, and deterministic `init homebrew-tap` scaffold. Release `v0.1.5` completed the production path; `meigma/homebrew-tap` PR 6 passed macOS and Linux cask checks and merged the generated `meigma-release-cli` cask.
Pattern to preserve: GoReleaser renders a package-manager control file with `skip_upload: true`; the authoritative Actions artifact carries it but the signed GitHub Release payload excludes it; publication runs only after the GitHub Release is public; a repository-scoped App token opens a deterministic branch and pull request; secret-free bucket/tap CI is the required merge check; the default branch is the published state; reruns converge or fail on explicit conflicts.
Scoop findings: GoReleaser 2.17.1 supports `scoops`, `skip_upload: true`, Windows ZIP archives, SHA-256 values, and `64bit` plus `arm64` manifest entries. Because this repository sets `release.disable: true`, a Scoop entry must provide an explicit `url_template`. A clean local snapshot probe against the current build produced `dist/scoop/meigma-release-cli.json`; setting `directory: bucket` produced `dist/scoop/bucket/meigma-release-cli.json`. The generated manifest named `release-cli.exe` as the binary and referenced the existing Windows amd64 and arm64 release archives. Probe artifacts were removed and `main` remained clean.
Scoop-specific gaps: choose root manifests versus the official BucketTemplate's `bucket/` layout; prove the exact Windows validation, install, update, uninstall, bad-hash, and unavailable-URL behavior in a disposable bucket; decide whether ARM64 is validated immediately or carried untested; pin Scoop test sources instead of copying the template's moving `ScoopInstaller/GithubActions@main`; and decide whether the second reviewed-file publisher justifies extracting channel-neutral reconciliation or should remain a focused `pubscoop` implementation.
Next: draft the Scoop plan in the same rehearsal → managed bucket CI → publisher → producer integration → initializer slices used for Homebrew, with every unresolved assumption assigned to the rehearsal rather than specified prematurely.

## 2026-08-21 11:48 — Scoop implementation plan
Created `.journal/007/PLAN.md` from a focused planning-agent pass, then reviewed it against the current Homebrew implementation, the GoReleaser 2.17.1 schema and Scoop pipe, and pinned Scoop/BucketTemplate sources.
Decision: keep Scoop as a separate channel-specific `pubscoop`/`ghbucket` path, reuse only existing neutral seams, and explicitly avoid a generic package-manager publisher. Default to root-level bucket manifests, subject to a real disposable-bucket rehearsal before the permanent contract lands.
Delivery order: disposable bucket rehearsal, secret-free managed bucket CI, fail-closed reviewed-PR publisher and CLI, producer/reusable-workflow integration, then deterministic bucket initializer and operator documentation.
Checks: the planned `.goreleaser.yaml` `scoops` entry—including `ids`, explicit `url_template`, and `skip_upload: true`—validates with the pinned local GoReleaser. The pinned Scoop schema, bucket tests, GoReleaser source, and BucketTemplate revision all resolve. No production files changed.
