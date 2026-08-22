---
id: 010
title: New repository session
started: 2026-08-21
---

## 2026-08-21 21:42 — Kickoff
Goal for the session: Create a fresh journal session and wait for the user's actual repository request.
Current state of the world: `main` is at `ca0370f`, release `v0.1.16` completed the native package repository production cutover, and session 009 is closed.
Plan: Capture the user's next request, then perform the work in an isolated implementation worktree when needed.

## 2026-08-21 21:50 — Release channel inventory
Goal: Identify every release and publishing method officially supported by the current repository.
Findings: The CLI exposes five publishers: GitHub Release, OCI/GHCR, static native package repositories, Homebrew cask pull requests, and Scoop manifest pull requests. The public GitHub bundle contains six Darwin/Linux/Windows archives, six Linux DEB/RPM/APK packages, twelve SBOMs, `checksums.txt`, and its Sigstore bundle. The repository also documents direct archive acquisition through mise and exact-source builds through its Nix flake.
Verification: Cross-checked `README.md`, `.goreleaser.yaml`, the release/OCI/package contracts, `.github/workflows/release.yml`, current `release-cli publish --help`, and the live 26-asset `v0.1.16` GitHub Release.
Next: Present the methods by publication destination, distinguish installation front ends from publishers, and state the current support boundaries.

## 2026-08-21 22:07 — Cross-organization adoption review
Review standpoint: An external GitHub organization wants to use `meigma/release` for one or more Go applications and publish to every supported destination.
Conclusion: Partially possible. GitHub Releases and GHCR are adopter-configurable; Homebrew and Scoop are parameterized but absent from the documented `v0.1.3` pin and copyable example. Static APT/RPM/APK publication is blocked for shared-workflow consumers because the receiver derives checksum and attestation identities from the producer repository instead of the reusable workflow repository and pinned revision.
Application topology: The current Go profile supports one application per repository. Staging requires one shared binary name across exactly one Linux `amd64` and one Linux `arm64` binary, OCI supports one entrypoint, and package releases move in lockstep under one stable `vMAJOR.MINOR.PATCH` tag.
Additional findings: The example mixes release-unit revisions, adopter-owned GitHub App creation is undocumented, and stable non-component tag constraints are not explicit. `meigma/release` visibility was verified as public, so cross-organization workflow and release reads are available.
Evidence: Reviewer output is preserved at `agent://CrossOrgReleaseReview`; repository visibility was verified with `gh repo view meigma/release`.

## 2026-08-21 22:36 — Cross-org fix and documentation design
Implementation: Programmer committed `23131c4b` on isolated branch `fix/cross-org-signer-policy`. The package policy now requires explicit `checksum_identity` and `attestation_signer` values, accepts signer repositories distinct from producers, and rejects non-immutable or malformed identities. The repository now carries full Apache-2.0 and MIT license texts and dual-license metadata.
Verification: All tests passed in `internal/stage/pkgrepo`, `internal/adapter/ghattest`, and `internal/cli`. `goreleaser check`, `nix flake check --no-build`, and LSP diagnostics passed. The implementation worktree is clean and tracks no journal files.
Documentation proposal: Replace the current README plus 17 documents with a concise cross-org set: one README, one first-release tutorial, six operator/how-to guides, two references, and one architecture/trust explanation. Consolidate organization setup, app adoption, Homebrew/Scoop operation, Cloudflare R2 native repositories, release recovery/upgrades, and CLI installation around external-organization workflows.
Constraint: No README or `docs/**` file was modified.
