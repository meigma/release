---
id: 008
title: Native package repository delivery slices 1 through 4
date: 2026-08-21
status: complete
repos_touched: [meigma/release]
related_sessions: ["006", "007"]
---

## Goal
Research a fully automated DEB, RPM, and APK delivery path without a package-repository service, prove the critical trust and publication behavior, and implement the safe reusable foundation.

## Outcome
The goal was met through production slice 4. Two disposable proofs established native signing, deterministic repository generation, package-manager installation, R2 reconciliation, interruption recovery, byte identity, and cache behavior. PR 43 delivered producer signing, the local publisher, the verified GitHub/R2 CLI, reusable automation, and documentation; release `v0.1.7` published them as one versioned unit. Production provisioning and cutover remain intentionally deferred to slices 5 and 6 in a new session.

## Key Decisions
- Keep GitHub Releases canonical and use R2 only as a static repository tree -> avoids a stateful repository service and duplicate inventory.
- Sign RPM and APK packages in the producer before checksums and attestations -> native package managers verify the same bytes authenticated by the release trust chain.
- Give only the central `meigma/pkgs` workflow R2 and aggregate metadata-signing credentials -> producers dispatch `{repository, tag}` and cannot mutate repository state.
- Regenerate the complete repository from verified immutable packages -> current scale makes this simpler and safer than a database or incremental manifest.
- Upload packages and inner metadata before signed mutable roots -> an interrupted publication leaves the previous repository generation usable.
- Use long-lived caching only for immutable objects and bypass caching for mutable roots -> no purge loop or Worker is required.

## Changes
- `.goreleaser.yaml` and reusable Go release workflow - added opt-in producer RPM and APK signing before checksums.
- `internal/stage/pkgrepo` - added reviewed policy, package planning, deterministic repository generation, trust verification, and ordered reconciliation.
- `internal/adapter/{pkgmeta,pkgverify,repogen,gpg,ghattest,ghrel,r2,pkginstall}` - added narrow inspection, signing, GitHub, R2, and native installation adapters.
- `internal/cli` - added `release-cli publish package-repository` and JSON result handling.
- `.github/actions/setup-package-repository` and `.github/workflows/publish-package-repository.yml` - added ephemeral aggregate-key setup and reusable publication orchestration.
- `docs/how-to/set-up-package-repository.md`, `docs/reference/package-repository-contract.md`, and trust/CLI documentation - recorded configuration, trust, replay, and recovery contracts.
- `flake.nix` - updated the fixed Go module closure hash after adding publisher dependencies.

## Open Threads
- Slice 5: create `meigma/pkgs`, production producer and aggregate keys, the protected environment, R2 bucket, `pkgs.meigma.dev`, and the hostname cache rule; rehearse against disposable infrastructure before production.
- Slice 6: enable producer native signing and post-release dispatch, publish the first signed production release, verify APT/DNF/APK installs, and require an unchanged replay.
- Native signing and package-repository dispatch remain disabled until those gates pass.
- Homebrew PRs 7/8 and Scoop PRs 2/3 remain open for human review and are independent of package-repository cutover.

## Lessons
- GoReleaser nFPM passphrases use `NFPM_<ID>_<FORMAT>_PASSPHRASE`; its RPM integration does not expose nFPM's `key_id`.
- Encrypted APK signing requires traditional `RSA PRIVATE KEY` PEM input rather than encrypted PKCS#8.
- APT clients request every strong by-hash family advertised in `Release`; SHA-256-only publication is insufficient when SHA-512 is advertised.
- Stable replay requires fixed metadata and signature times, including `SOURCE_DATE_EPOCH` for APK index generation and signing.
- DNF and APK installation must trust both the aggregate repository key and the producer package key.
- R2 ETags are not content identities; store and stream-verify explicit SHA-256 metadata.

## References
- Architecture: `.journal/008/ARCHITECTURE.md`
- Execution plan: `.journal/008/PLAN.md`
- Implementation PR: https://github.com/meigma/release/pull/43
- Release PR: https://github.com/meigma/release/pull/42
- Release: https://github.com/meigma/release/releases/tag/v0.1.7
- Production release run: https://github.com/meigma/release/actions/runs/32539849827
- Native package baseline: `.journal/006/SUMMARY.md`
- Package-channel precedent: `.journal/007/SUMMARY.md`
