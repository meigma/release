# Native Package Repository Architecture

Status: proposal to build from. This design adds DEB, RPM, and APK repository delivery to the existing `release-cli` release unit. It keeps GitHub Releases authoritative, uses `pkgs.meigma.dev` as a static public origin, and gives one consumer-owned `meigma/pkgs` repository the only write path to Cloudflare R2.

## 1. Decisions

1. **One packages repository:** `meigma/pkgs` owns aggregate repository configuration, publication workflows, Cloudflare credentials, and aggregate metadata-signing keys.
2. **One public origin:** `pkgs.meigma.dev` serves static APT, RPM/DNF, and APK repository trees from one R2 bucket in the maintainer's existing Cloudflare account.
3. **One writer:** producer repositories never receive R2 credentials or aggregate signing keys. One workflow in `meigma/pkgs` serializes every publication.
4. **GitHub Releases remain authoritative:** producer workflows sign native packages before checksums and attestations, then publish the same package bytes to GitHub Releases and R2.
5. **Two-field request:** after publishing a GitHub Release, a producer dispatches only `{repository, tag}`. `meigma/pkgs` re-derives and verifies every other fact.
6. **Three implementation layers:** `release-cli` owns behavior, a composite Action in `meigma/release` provides GitHub ergonomics, and the `meigma/pkgs` workflow owns runner policy, permissions, secrets, environment protection, and concurrency.
7. **R2 is published state:** Git stores configuration and public keys, not package binaries or a duplicate package inventory. Each publication mirrors the immutable package pool locally and regenerates all metadata.
8. **Full regeneration first:** the package pool is small enough that downloading and re-indexing it on each publication is cheaper than maintaining an incremental database or repository-manager state.
9. **Append-only immutable objects:** packages, APT by-hash files, checksum-named RPM metadata, and versioned public keys are not overwritten or routinely deleted.
10. **No formal key lifecycle system:** this solo-maintained organization uses long-lived keys and rotates them only when compromise or an operational need requires it. Versioned public-key filenames make that manual change possible without a new service or policy engine.
11. **Reuse the existing release GitHub App:** it mints the short-lived token that dispatches to `meigma/pkgs`. A second App would add administration without reducing a meaningful risk in this organization.
12. **No serving-path compute:** R2 serves static files directly. No Cloudflare Worker, API service, broker, or database participates.

## 2. System boundary

```text
Producer repository
    |
    | signed GitHub Release
    | repository_dispatch {repository, tag}
    v
meigma/pkgs
    | protected, serialized local workflow
    v
publish-package-repository composite Action
    | matching release-cli + pinned tools
    v
release-cli
    | verify, mirror, regenerate, sign, reconcile, verify
    v
Cloudflare R2
    |
    v
https://pkgs.meigma.dev
```

### Producer repository

The producer:

- builds DEB, RPM, and APK packages through GoReleaser/nFPM;
- signs RPM and APK packages with per-producer keys before `checksums.txt`, Cosign signing, and GitHub attestations;
- publishes the GitHub Release through the existing release path;
- uses the existing release GitHub App to send a publication request to `meigma/pkgs` after the GitHub Release is public.

The producer does not receive R2 credentials, aggregate repository keys, repository layout inputs, or mutable repository state.

### `meigma/pkgs`

The packages repository owns:

- the producer allowlist;
- permitted package names per producer;
- expected producer workflow identities;
- producer RPM and APK public keys;
- the public repository layout and channel configuration;
- versioned aggregate public keys;
- R2 credentials;
- aggregate APT/RPM metadata and APK index private keys;
- the `packages-production` GitHub Environment;
- one global publication concurrency group;
- manual replay and metadata-rebuild entry points.

The packages repository contains no package binaries.

### Composite Action

A new Action in `meigma/release`, beside `setup-release-cli`, provides the step-level interface:

```yaml
uses: meigma/release/.github/actions/publish-package-repository@<full-sha>
```

The Action:

- acquires the matching `release-cli` and checks the existing version/protocol stamp;
- acquires pinned external tools;
- maps workflow inputs and secrets to `RELEASE_*` environment variables;
- invokes one CLI operation;
- exposes the CLI's `release.dev/result/v1` envelope as outputs and a job summary.

The Action contains no repository-generation or publication policy.

### `release-cli`

The CLI owns:

- request validation against repository configuration;
- GitHub Release download;
- checksum, Cosign identity, and GitHub attestation verification;
- native RPM and APK signature verification;
- R2 listing, download, digest verification, and ordered upload;
- APT, RPM, and APK repository generation;
- aggregate metadata signing;
- idempotent reconciliation;
- public-origin read-back and install verification;
- structured success and failure results.

Final command and Action input names remain open until the local spike proves the behavior.

## 3. Repository request

A producer sends:

```json
{
  "repository": "meigma/release",
  "tag": "v0.1.6"
}
```

The packages workflow and CLI treat the payload as a request, not evidence. They require:

- `repository` to exist in the Git-reviewed producer allowlist;
- `tag` to match the repository's strict `v`-SemVer contract;
- every accepted package name to belong to that producer;
- the checksum bundle to match the configured producer workflow identity;
- package checksums and GitHub attestations to verify;
- RPM and APK package signatures to match the producer public keys configured in `meigma/pkgs`.

The publisher derives asset names, versions, architectures, URLs, and digests from the verified GitHub Release. A replayed request converges to the existing state.

## 4. Package and metadata trust

### APT

DEB files rely on APT's signed metadata chain rather than package-level signatures.

The producer publishes canonical DEB bytes. The packages publisher:

- generates `Packages` and compressed variants;
- enables `Acquire-By-Hash: yes` from the first release;
- publishes immutable `by-hash/SHA256/<digest>` objects;
- signs one `InRelease` file with the aggregate APT key;
- publishes `InRelease` as the mutable commit point.

The first implementation targets modern APT clients and does not publish the detached `Release`/`Release.gpg` pair.

### RPM/DNF

The producer signs each RPM during nFPM staging. The packages publisher verifies that signature, generates checksum-named metadata with `createrepo_c`, and signs `repomd.xml` with the aggregate RPM metadata key.

Consumer configuration enables both controls:

```ini
gpgcheck=1
repo_gpgcheck=1
```

The repository publishes the configured producer public keys at stable, versioned URLs.

### APK

The producer signs each APK during nFPM staging. The packages publisher verifies the package signature, generates `APKINDEX.tar.gz`, and embeds the aggregate index signature. Consumers install the configured producer public keys and aggregate index key in `/etc/apk/keys`.

## 5. Published layout

The initial public tree uses one `stable` channel and the two existing Linux architectures:

```text
pkgs.meigma.dev/
├── keys/
│   ├── apt-repository-001.asc
│   ├── rpm-repository-001.asc
│   ├── apk-index-001.rsa.pub
│   └── <producer>-<serial>.asc|rsa.pub
├── apt/
│   ├── pool/main/...
│   └── dists/stable/
│       ├── InRelease
│       └── main/binary-{amd64,arm64}/
├── rpm/stable/
│   ├── x86_64/
│   │   ├── packages/
│   │   └── repodata/
│   └── aarch64/
│       ├── packages/
│       └── repodata/
└── apk/stable/main/
    ├── x86_64/
    │   └── APKINDEX.tar.gz
    └── aarch64/
        └── APKINDEX.tar.gz
```

Version-qualified package names make package objects immutable. The publisher rejects an existing object path whose digest differs from the requested package.

## 6. Publication transaction

The serialized workflow performs these steps:

1. Validate the producer request against the checked-in configuration.
2. Download the public GitHub Release package assets and verification controls.
3. Verify the producer workflow identity, checksums, attestations, package names, and native signatures.
4. List and download every existing immutable package object from R2 into a temporary local mirror.
5. Verify each downloaded object's stored SHA-256 metadata.
6. Add the new canonical package files locally. Reject same-path, different-digest conflicts.
7. Regenerate every APT, RPM, and APK index from the complete local package tree.
8. Sign aggregate metadata.
9. Run local HTTP installation checks with Debian, Fedora, and Alpine clients.
10. Upload missing immutable package and hash-addressed metadata objects.
11. Upload mutable repository roots last.
12. Install the requested package through `https://pkgs.meigma.dev` with real package-manager signature verification enabled.
13. Return `published`, `unchanged`, or a fail-closed result through the standard JSON envelope.

The full local mirror is deliberate. At the current package volume, it removes the need for a database, incremental index mutation, generated Git state, and reconciliation between multiple state stores.

## 7. Upload and cache rules

Immutable objects receive a long cache lifetime:

```text
Cache-Control: public, max-age=31536000, immutable
```

This applies to:

- package files;
- APT by-hash files;
- checksum-named RPM metadata;
- versioned public keys.

Mutable repository roots bypass Cloudflare CDN caching:

- `InRelease`;
- `repomd.xml`;
- `repomd.xml.asc`;
- `APKINDEX.tar.gz`.

R2's direct strong consistency then governs these paths. The design does not require cache-purge automation.

Upload order is:

1. package objects;
2. immutable inner metadata;
3. signed mutable roots.

A crash before the mutable roots leaves the previous repository active. RPM's detached `repomd.xml` signature requires two adjacent writes and can produce a brief fail-closed verification window. It cannot produce silently trusted mismatched metadata.

APT by-hash objects remain append-only in the initial implementation. Routine pruning is not part of publication.

## 8. State, idempotence, and recovery

R2 is the only published-state store. `meigma/pkgs` Git history stores configuration and public keys only.

- Same request and same bytes: no-op.
- Existing path with a different digest: hard failure.
- Failed immutable upload: rerun; no root references the incomplete generation.
- Failed mutable-root upload: rerun full regeneration and reconciliation.
- Metadata repair: replay the release or run the manual rebuild entry point.
- Package withdrawal: explicit emergency operation followed by a rebuild; not automatic retention cleanup.

The workflow uses one global concurrency group with `cancel-in-progress: false`. No distributed lock, compare-and-swap layer, or queue service exists.

## 9. Credentials

| Credential | Location | Scope |
| --- | --- | --- |
| R2 access key | `meigma/pkgs` production Environment | One bucket, object read/write |
| Aggregate APT/RPM metadata key | `meigma/pkgs` production Environment | Repository metadata only |
| Aggregate APK index key | `meigma/pkgs` production Environment | APK indexes only |
| Producer RPM key | Producer repository secret | That producer's RPM packages |
| Producer APK key | Producer repository secret | That producer's APK packages |
| Existing release GitHub App key | Existing producer release configuration | Short-lived dispatch token for `meigma/pkgs` |

The maintainer owns `pkgs.meigma.dev`, the Cloudflare account, and R2 billing. No second GitHub App or formal key-rotation system is required.

## 10. Deliberate exclusions

The first implementation does not include:

- a package-repository SaaS;
- a Cloudflare Worker;
- a central API service, broker, or daemon;
- a database or repository-manager state directory;
- a Git manifest duplicating R2 package state;
- per-producer R2 credentials;
- an organization-wide package signing key exposed to every producer;
- a cross-repository reusable-workflow dependency for privileged secrets;
- a generic storage plugin registry;
- automatic retention, pruning, mirroring, or withdrawal;
- prerelease or per-distribution channels.

## 11. Implementation sequence

### Local proof

Using one existing GitHub Release and throwaway keys:

1. Sign RPM and APK packages during staging.
2. Verify the complete release trust chain.
3. Generate and sign all three repository trees.
4. Serve the trees locally.
5. Install with Debian, Fedora, and Alpine clients without insecure flags.

### Disposable R2 proof

Using the same code path and a scratch R2 target:

1. Publish one version.
2. Replay it and prove a no-op.
3. Publish a second version and regenerate all metadata.
4. Interrupt publication before a mutable root and prove the prior repository remains usable.
5. Install through HTTPS and inspect actual cache behavior.

### Production cutover

After both proofs pass:

1. Add native producer signing to the reusable pre-publish path.
2. Implement the verified CLI behavior.
3. Publish the composite Action in the existing release unit.
4. Initialize `meigma/pkgs` with configuration, workflows, Environment, and keys.
5. Configure the R2 bucket and `pkgs.meigma.dev`.
6. Add `meigma/release` as the first producer and dispatch its releases.
7. Document installation, verification, replay, and manual key replacement.

## 12. Acceptance criteria

The architecture is implemented when:

- one approved producer can publish a GitHub Release and trigger `meigma/pkgs` without receiving R2 or aggregate signing credentials;
- the packages repository independently verifies the public release and native package signatures;
- the published DEB, RPM, and APK bytes match the GitHub Release assets;
- APT, DNF, and APK install from `pkgs.meigma.dev` with native verification enabled;
- replaying the same release changes no immutable object and succeeds as `unchanged`;
- a same-path, different-digest package is rejected;
- a failed pre-root upload leaves the previous repository usable;
- all publication behavior is implemented in `release-cli`, with the Action and workflow limited to GitHub-specific orchestration;
- no service, database, Worker, repository manager, or duplicate Git package manifest is required.
