---
id: 001
title: Meigma delivery infrastructure
started: 2026-08-16
---

## 2026-08-16 21:51 — Kickoff
Goal for the session: Establish one opinionated Go delivery process for the Meigma organization covering GitHub Release artifacts, OCI images, Homebrew, MacPorts, Nix, Scoop, a generalized installer script, and mise.
Current state of the world: The public `meigma/release` repository exists, its session protocol is installed on `main`, and the personal `journal/jmgilman` worktree is ready.
Plan: Capture lightweight research and decisions per delivery channel, starting with the shared release contract and GitHub Release artifacts, then formalize the proven process into reusable automation and documentation.

## 2026-08-16 21:59 — Initial automation structure
The working direction is one centrally governed release DAG per producer profile, not independent release-type workflows that each rebuild. A language-specific build job should produce one immutable release bundle containing distributable files, `checksums.txt`, SBOMs, and a machine-readable manifest tied to the tag and commit. Optional publisher jobs then consume and verify that bundle without compiling source.
Reusable workflows should own runner boundaries, matrices, permissions, secrets, publication, and attestations. Composite actions should be reserved for repeated step-level mechanics inside jobs. Repositories should use a thin caller workflow and select supported publishers through explicit inputs.
Likely producer profiles are Go, Python, and later specific IaC deliverable types; IaC is not one coherent build profile. GitHub Releases, OCI packaging, and package-manager metadata can share the same release bundle, but distributed publication cannot be truly atomic. Publishers must therefore be idempotent and the GitHub Release should remain a draft until required publication and attestation stages succeed.
Open issues for channel research: whether MacPorts and Nix should intentionally rebuild from source or consume authoritative binaries; which channels are required versus optional for finalization; and the exact release-bundle manifest schema.
