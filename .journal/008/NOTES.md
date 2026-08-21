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

## 2026-08-21 12:34 — Package publisher composition
Decision: A published composite GitHub Action is a good step-level interface, but it must not own the deployment policy or business logic. `release-cli` should generate, verify, and publish repository state; the action should provide GitHub ergonomics; a workflow in the consumer-owned packages repository should own runner selection, permissions, environment secrets, and global publication concurrency.
Trust-boundary correction: Do not rely on a cross-repository `workflow_call` from each producer to gain access to the packages repository's secrets. Reusable workflows run in the caller context and receive caller tokens and explicitly passed secrets. Trigger a workflow run in the packages repository through a verified dispatch or a merged manifest change instead, so only that repository holds R2 and metadata-signing credentials.
Initialization shape: Scaffold a control-plane packages repository with an allowlisted manifest, a no-secret verification path, and a production publication workflow pinned to the action's full release SHA. Binary packages remain in GitHub Releases and R2, not Git.
Next: If this direction is accepted, spike the action/workflow contract around one existing release before designing the final initializer surface.

## 2026-08-21 12:48 — Refined package repository architecture
Architecture-agent result: Use one consumer-owned packages repository, one R2 bucket/custom hostname, and one serialized writer. Producers dispatch only `{repository, tag}` after publishing a GitHub Release. The packages repository validates an allowlist and signer identity, then invokes a composite action from this release unit; the action acquires pinned tools and the matching CLI, while `release-cli` owns verification, full repository regeneration, signing, ordered upload, idempotence, and live install checks.
State decision: Git stores only `producers.yaml`, public versioned keys, and workflow/configuration. R2 is the published state. On each run, mirror all immutable package objects locally, verify their stored SHA-256 metadata, add the new canonical packages, and regenerate every index. This is deliberately O(total package bytes) because the current pool is tiny and it avoids a database, generated Git manifest, and incremental-state drift.
Signing decision: Producers sign RPM and APK packages with per-producer keys during nFPM staging, before checksums/Cosign/attestations, preserving byte identity between GitHub Releases and R2. The packages repository owns only aggregate APT/RPM metadata and APK index keys. DEB relies on APT's signed metadata chain.
Publication decision: APT uses Acquire-By-Hash from v1 and one inline-signed `InRelease`; RPM uses checksum-named inner metadata plus `repomd.xml`/`.asc`; APK uses one signed `APKINDEX.tar.gz`. Package objects, by-hash metadata, checksum-named RPM metadata, and versioned keys are append-only with long cache TTLs. Mutable roots bypass CDN caching; this is simpler and safer than purge logic or short-TTL signature-pair skew.
Rejected weight: no service, broker, database, Cloudflare Worker, repository manager, per-producer R2 credential, cross-repo reusable-workflow secret dependency, generic storage plugin registry, or package manifest duplicated into Git.
Spike sequence: first generate/sign/install all formats locally from one existing release; then publish and replay two versions against a disposable R2 target; only then add the CLI/action, packages repository, production keys/domain, and producer dispatch job.
