---
id: 009
title: Native package repository production cutover
date: 2026-08-21
status: complete
repos_touched: [meigma/release, meigma/pkgs]
related_sessions: ["008"]
---

## Goal
Finish production slices 5 and 6 of the native package repository program: provision the central receiver and credentials, publish a signed release, prove public APT/DNF/APK installation and convergent replay, enable automatic producer dispatch, and document consumer and operator flows.

## Outcome
The goal was met. `meigma/pkgs` is the serialized production receiver, `pkgs.meigma.dev` serves signed APT, RPM, and APK repositories, and `meigma/release` `v0.1.16` completed the automatic producer-to-repository path. Production run `32551330487` published 42 repository artifacts with 31 uploads; independent pinned Debian, Fedora, and Alpine clients installed exact version `0.1.16`; replay run `32551525329` returned `state: unchanged` with zero uploads. All package-repository slices are complete.

## Key Decisions
- Keep aggregate R2 and repository-signing credentials only in the protected `meigma/pkgs` environment -> producers can request publication but cannot mutate repository state.
- Reuse the release App through a short-lived token scoped to `meigma/pkgs` -> dispatch carries only the exact `{repository, tag}` identity and no publication credentials.
- Keep producer RPM/APK signing before checksums and attestations, with distinct aggregate repository keys -> native package managers and the release trust chain authenticate the same package bytes without sharing private keys.
- Require local and public APT, DNF, and APK installs plus an unchanged replay before cutover -> the production origin, package-manager trust paths, and reconciliation behavior are release gates rather than operator assumptions.
- Replace obsolete R2 content instead of migrating it -> the existing July candidate repository was unrelated to the reviewed policy and production trust roots.

## Changes
- `meigma/pkgs/.config/package-repository.yaml`, `.config/keys/`, and `.github/workflows/publish.yml` - established reviewed producer policy, versioned public keys, serialized publication, protected-environment use, and the final `v0.1.16` publisher pin.
- `meigma/release/.github/workflows/release.yml` and `.github/workflows/request-package-repository.yml` - enabled native producer signing and exact post-release dispatch through the release App.
- `meigma/release/internal/adapter/{pkgverify,pkginstall,repogen}` and `internal/stage/pkgrepo` - hardened exact APK key/version checks, root-owned metadata handoff, public APT trust bootstrapping, output traversal, and canonical lowercase RPM path recognition.
- `meigma/release/docs/how-to/install-release-cli-from-package-repositories.md`, `docs/how-to/set-up-package-repository.md`, and `examples/go-release/` - documented tested install/update commands, failure recovery, key replacement, producer setup, and a disabled example integration.
- Production infrastructure - installed the release App on `meigma/pkgs`, provisioned aggregate signing and bucket-scoped R2 credentials in `packages-production`, purged 54 obsolete R2 objects, and verified hostname-scoped immutable-cache and mutable-bypass rules.

## Open Threads
- None for native package repository delivery. Homebrew and Scoop review PRs recorded in prior sessions remain independent of this completed cutover.

## Lessons
- Disposable macOS proofs did not expose Linux container traversal, ownership, and CA-bundle boundaries; public-package validation must run in the same pinned distributions used by consumers.
- A same-version replay can mask an incomplete historical-package classifier when incoming release assets supply every format; a new-version publication is required to prove historical mirroring.
- Repository-generation paths are protocol data. The lowercase RPM `packages/` tree must be recognized exactly or the completeness mask fails closed before upload.

## References
- Prior architecture and slices 1-4: `.journal/008/SUMMARY.md`
- Central repository foundation: https://github.com/meigma/pkgs/pull/1
- Native producer signing: https://github.com/meigma/release/pull/44
- APK key-name hardening: https://github.com/meigma/release/pull/46
- Generated-output ownership and traversal: https://github.com/meigma/release/pull/48 and https://github.com/meigma/release/pull/50
- Exact APK version and public APT trust fixes: https://github.com/meigma/release/pull/54 and https://github.com/meigma/release/pull/56
- Automatic package dispatch: https://github.com/meigma/release/pull/58
- Canonical RPM mirroring: https://github.com/meigma/release/pull/60
- Consumer and producer documentation: https://github.com/meigma/release/pull/62
- Final receiver pin: https://github.com/meigma/pkgs/pull/11
- Release: https://github.com/meigma/release/releases/tag/v0.1.16
- Automatic production run: https://github.com/meigma/pkgs/actions/runs/32551330487
- Unchanged replay: https://github.com/meigma/pkgs/actions/runs/32551525329
