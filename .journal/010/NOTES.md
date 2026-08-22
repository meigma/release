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
