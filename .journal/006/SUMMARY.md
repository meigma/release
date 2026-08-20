---
id: 006
title: Native packages, mise, and Nix support
date: 2026-08-20
status: complete
repos_touched: [meigma/release]
related_sessions: ["001", "005"]
---

## Goal

Plan the deferred Homebrew lane, then complete the first two delivery-backlog items: prove the full CLI-owned production release path and finish the consumer documentation pin. The session later expanded to native DEB/RPM/APK packages, direct mise installation, and first-class Nix support.

## Outcome

Met. The full release path ran successfully twice, publishing `v0.1.2` and `v0.1.3`. GoReleaser now emits native DEB, RPM, and APK packages from the canonical Linux binaries; direct mise installation and a source-built Nix flake are documented and verified; all consumer workflow, signer, contract, and example pins now target the verified `v0.1.3` revision. Homebrew remains designed but intentionally unimplemented.

## Key Decisions

- Generate DEB, RPM, and APK packages with GoReleaser nFPM -> this reuses the canonical Linux binaries, avoids a second build path, and adds one SBOM per native package.
- Keep native package publication inside the authoritative GitHub Release bundle -> checksums, Cosign signing, GitHub attestations, and draft convergence cover the packages without a separate repository or signer.
- Use mise's built-in GitHub backend rather than a custom plugin or installer -> release archive naming already satisfies mise asset selection, checksum, and attestation verification.
- Build the Nix package from the exact flake revision -> fixed hashes for same-version release archives cannot be known before publication, while a source build remains valid through a Release Please version bump.
- Pin Nixpkgs 26.05 and override Go to fixed-source 1.26.6 -> this preserves all four Darwin/Linux `arm64`/`amd64` outputs while matching `go.mod` exactly.
- Land release-producing changes before documentation pins -> examples and guides only name a tag and commit after the complete production release has passed.
- Keep Homebrew as a cask-rendering and tap-PR workflow -> GoReleaser can render the cask locally, but cross-repository mutation and protected tap merge remain a separate trust boundary.

## Changes

- `.goreleaser.yaml` and Go profile contract tests - added nFPM DEB/RPM/APK outputs and package metadata without changing canonical binary selection.
- `.github/workflows/go-pre-publish.yml` and release contracts - expanded the authoritative bundle to six archives, six native packages, twelve SBOMs, and 26 total release assets.
- `docs/` and `examples/go-release/` - documented package generation, installation, recovery, the 26-file contract, and advanced every release-unit pin to `v0.1.3` commit `0fc99489d31d400bc3f69d6636d60e7d3f3d0251`.
- `docs/how-to/install-release-cli-with-mise.md` - documented project, temporary, global, and local-alias installation through mise's native GitHub backend.
- `flake.nix`, `flake.lock`, and `.github/workflows/ci.yml` - added locked source-built Nix package/app/check outputs and a pinned Determinate Nix CI job.
- `examples/nix-release-cli/` and `docs/how-to/install-release-cli-with-nix.md` - added a locked consumer flake plus direct-run, profile, project-flake, update, and trust-boundary guidance.
- Published `v0.1.2` and `v0.1.3`; each completed the release asset build, OCI image build, OCI publication, and GitHub Release publication jobs.

## Open Threads

- Implement the planned Homebrew cask/tap path: disposable-tap rehearsal, tap CI, idempotent branch/PR publisher, optional producer integration, then `init homebrew-tap`.
- Scoop, MacPorts, and a generalized installer remain deferred package channels.
- Nixpkgs 26.05 is the last release supporting `x86_64-darwin`; a future Nix update must either retain that pin deliberately or drop the system explicitly.
- The successful release run still emitted non-blocking annotations for the pinned QEMU action's Node.js 20 runtime and artifact-metadata storage during OCI publication.

## Lessons

- A flake that fetches same-release archives cannot derive its version from the release manifest: Release Please changes the version before the assets and their fixed hashes exist. Build from source or update binary hashes only after publication.
- Current mise can install `release-cli` directly with `github:meigma/release@<version>` because the release archive names match its native GitHub backend; no registry entry is required.
- `nix profile add` is the current command; `nix profile install` remains only as a deprecated alias.

## References

- Native packages: https://github.com/meigma/release/pull/21
- `v0.1.2` release: https://github.com/meigma/release/pull/22 and https://github.com/meigma/release/releases/tag/v0.1.2
- Native-package docs and pins: https://github.com/meigma/release/pull/23
- mise installation docs: https://github.com/meigma/release/pull/24
- Repository Nix flake: https://github.com/meigma/release/pull/25
- `v0.1.3` release: https://github.com/meigma/release/pull/26 and https://github.com/meigma/release/releases/tag/v0.1.3
- Nix consumer example and final pins: https://github.com/meigma/release/pull/27
- Prior execution context: `.journal/005/SUMMARY.md`
