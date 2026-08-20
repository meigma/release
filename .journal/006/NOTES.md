---
id: 006
title: New work session
started: 2026-08-20
---

## 2026-08-20 13:42 — Kickoff
Goal for the session: Start a fresh journal session; the substantive work request has not been provided yet.
Current state of the world: The release CLI program is complete through cached exact-source self-release builds, all reusable workflows are thin shells, and the next release remains the first production run of the complete CLI-owned build and publication path.
Plan: Bind this session to the current task, then await the substantive request.

## 2026-08-20 13:58 — Homebrew support planning
Goal: Plan a managed Homebrew tap workflow, tap initializer, and release publisher that opens tap pull requests.
Findings: Homebrew now recommends casks for supported binary-only distribution, while GoReleaser deprecates its formula publisher. A local GoReleaser spike proved that `homebrew_casks` with `skip_upload: true` still writes `dist/homebrew/Casks/<token>.rb` under the production-compatible `--skip=publish` path, including macOS and Linux amd64/arm64 URLs and checksums. The official `brew tap-new` scaffold is formula-and-bottle oriented; casks become available when their PR is merged, so a cask-only tap needs validation but no `brew pr-pull` publication workflow.
Decision: Keep GoReleaser as the cask renderer, keep cross-repository mutation in a new `publish homebrew` CLI engine, and treat the tap PR plus protected merge as a separate trust boundary. A managed reusable tap workflow should run without secrets and audit, install, and uninstall changed casks. The initializer should first render the proven cask-only scaffold locally rather than acquire broad repository-administration credentials.
Plan: Rehearse one cask manually in a disposable tap; build the reusable tap CI; implement idempotent GitHub branch/PR publication after the GitHub Release is public; integrate an optional producer workflow; then codify the proven scaffold in `init homebrew-tap`. Prefer one organization tap with arbitrary tap targets supported by the publisher. Current `main` advanced during planning to `611195c` (`v0.1.2`) and now includes native DEB/RPM/APK artifacts.

## 2026-08-20 15:04 — Close

Merged PRs [#21](https://github.com/meigma/release/pull/21) through
[#27](https://github.com/meigma/release/pull/27). The full CLI-owned release
path published and verified `v0.1.2` and `v0.1.3`; the GitHub Release contract
now includes native DEB, RPM, and APK packages; direct mise installation and the
source-built Nix flake are documented and tested; and all consumer pins target
`v0.1.3` commit `0fc99489d31d400bc3f69d6636d60e7d3f3d0251`.
`main` is clean and synchronized at `762bf40`, with no open pull requests.
Homebrew implementation, Scoop, MacPorts, and a generalized installer remain
deferred. See `SUMMARY.md` for the postmortem and verification details.
