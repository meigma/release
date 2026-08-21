---
id: 008
title: New work session
started: 2026-08-21
---

## 2026-08-21 11:40 — Kickoff
Goal for the session: Start a new work session; the substantive work request has not been provided yet.
Current state of the world: The personal journal is configured on `journal/jmgilman`, and the repository's established release infrastructure remains available for the next task.
Plan: Bind this session to the current task, then capture meaningful checkpoints after the user provides the work request.

## 2026-08-21 11:53 — Native package delivery research
Goal: Evaluate fully automated DEB, RPM, and APK repository delivery without a package-repository SaaS.
Current evidence: `v0.1.3` carries six native packages totaling 20,342,950 bytes. The existing release bundle publishes them as unsigned standalone GitHub assets protected by checksums, Cosign, and GitHub attestations; native package managers do not consume that trust metadata.
Findings: APT needs a signed `InRelease`/`Release` chain, RPM should use signed packages plus signed `repomd.xml`, and APK needs RSA-signed packages and `APKINDEX.tar.gz`. GoReleaser/nFPM can sign RPM and APK packages during the authoritative build. Static repositories can be generated with `apt-ftparchive`, `createrepo_c`, and `apk index`/`abuild-sign`.
Decision direction: Use a dumb static origin rather than a stateful repository service. Cloudflare R2 behind a custom domain is the leading managed-storage option: current Standard pricing includes 10 GB-month, one million Class A requests, ten million Class B requests, and free egress. Its main liabilities are persistent bucket credentials, non-atomic multi-object publication, and stale custom-domain cache entries on overwritten metadata.
Recommended shape: Keep GitHub Release assets canonical, add native RPM/APK signing before checksums and attestations, and run an idempotent post-release repository publisher. Upload immutable package and hash-addressed metadata first, mutable signed indexes last; bypass CDN cache for mutable metadata and cache immutable packages aggressively. Start with one `stable` channel and rehearse local installs plus a disposable R2 prefix before production.
Alternative: If “no SaaS” is literal, serve the same static tree from Caddy on a small host and atomically switch a generation symlink; accept host patching, backups, availability, bandwidth, and SSH-key operations.
Next: Present the tradeoffs and proposed spike sequence; implementation remains unstarted.
