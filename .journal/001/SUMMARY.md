---
id: 001
title: Meigma delivery infrastructure
date: 2026-08-18
status: complete
repos_touched: [meigma/release]
related_sessions: []
---

## Goal

Establish one opinionated Go delivery process for Meigma covering GitHub Release artifacts, OCI images, Homebrew, MacPorts, Nix, Scoop, a generalized installer, and mise. Work incrementally, prove each channel through rehearsals, and capture reusable automation and documentation.

## Outcome

Partially met, then intentionally paused. The GitHub Release and OCI portions reached a rehearsed MVP and landed on `main` through PR #2. The other delivery channels remain future work.

The landed system builds canonical Go artifacts once, transfers them through digest-checked Actions artifacts, publishes signed and attested multi-platform GHCR images without rebuilding, and keeps GitHub Release publication gated on required OCI success. It includes thin consumer workflows, a copyable example, adoption and recovery guides, and reference contracts.

## Key Decisions

- Build canonical binaries and archives once with GoReleaser; downstream publishers verify and consume those bytes instead of recompiling.
- Separate unprivileged producers from privileged publishers so registry and release credentials never enter build jobs.
- Keep a GitHub Release in draft state until uploaded assets, attestations, and any required OCI publication are complete.
- Use a narrowly scoped GitHub App installation token for release mutation; do not grant the build workflow release write access.
- Compose OCI images through Melange-signed APK repositories and locked apko configuration; avoid Dockerfiles and preserve canonical binary bytes.
- Treat exact version tags as immutable and channel tags as monotonic; publish public tags only after signatures and attestations succeed.
- Sign the OCI index and both platform manifests, then attach index provenance and per-platform SPDX attestations.
- Use pinned `actions/github-script` programs with explicit `@actions/exec` argument arrays in the privileged OCI publisher rather than large interpolated Bash programs.
- Pin reusable workflows and third-party actions to immutable commit SHAs; upgrades require a reviewed full-contract migration.
- Use faithful cross-repository rehearsals and failed-job reruns as the recovery model instead of nominal dry runs.

## Changes

- `.github/workflows/go-pre-publish.yml` - builds, signs, verifies, and uploads the authoritative Go release-assets artifact.
- `.github/workflows/go-oci-build.yml` - packages canonical Linux binaries with Melange and apko into a verified multi-platform OCI layout.
- `.github/workflows/publish-oci-image.yml` - verifies, publishes, signs, attests, and tags immutable GHCR images.
- `.github/workflows/publish-github-release.yml` - validates the authoritative artifact and optional OCI output before publishing the draft GitHub Release.
- `.github/workflows/release-please.yml` and release configuration - create draft release PRs and tags through the organization GitHub App.
- `.goreleaser.yaml`, `melange.yaml`, and `apko.yaml` - define canonical release artifacts and the byte-preserving OCI packaging path.
- `docs/how-to/` and `docs/reference/` - document consumer configuration, rehearsal, recovery, upgrades, and the GitHub Release and OCI contracts.
- `examples/go-release/` - provides a copyable, publication-disabled consumer example with pinned tools and workflows.
- `cmd/release-mvp/` and `internal/cli/` - provide the small Go command used to exercise the complete release path.

## Open Threads

- Complete the deferred Homebrew, MacPorts, Nix, Scoop, generalized installer, and mise delivery channels.
- Review and decide whether to merge release PR #3 (`chore(main): release 0.1.0`); session close does not publish it.
- Before external consumer adoption, move documented reusable-workflow pins from the pre-squash implementation revision `fb8c8098ff27968fb3070e928c00e925f38c698e` to a reviewed revision reachable from `main`.
- Each consumer's first GHCR package may require an organization owner to make it public after inspecting the signed and attested image.
- Delete the disposable public rehearsal repository `meigma/release-oci-remediation-e2e` manually when its retained evidence is no longer needed; the current token lacks `delete_repo`.

## Lessons

- Cross-repository reusable workflows expose permission, artifact-ownership, and exact identity constraints that local validation cannot prove.
- Digest-first publication plus delayed tag assignment prevents a partially trusted image from becoming consumer-visible.
- Failed-job-only reruns preserve authoritative producer artifacts; complete reruns are appropriate only when those artifacts must be replaced.
- A squash merge changes the `main` commit identity, so consumer pins must be treated as a deliberate post-merge release step.

## References

- MVP merge: https://github.com/meigma/release/pull/2
- Deferred release PR: https://github.com/meigma/release/pull/3
- GitHub Release contract: `docs/reference/github-release-contract.md`
- OCI image contract: `docs/reference/oci-image-contract.md`
- Consumer setup: `docs/how-to/configure-github-releases.md`
- OCI setup: `docs/how-to/configure-oci-images.md`
- Rehearsal and recovery: `docs/how-to/rehearse-and-recover-github-releases.md`
- Final GitHub Script OCI rehearsal: https://github.com/meigma/release-oci-remediation-e2e/actions/runs/32175517384
