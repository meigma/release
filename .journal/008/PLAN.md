# Native Package Repository Implementation Plan

Status: provisional execution plan for `.journal/008/ARCHITECTURE.md`. Revalidate this plan after each spike. Command names, Action inputs, tool versions, container digests, configuration syntax, and internal Go package boundaries remain undecided until evidence fixes them.

## 1. Current baseline

Verified repository facts:

- `.github/workflows/release.yml` publishes authoritative assets, OCI artifacts, and the public GitHub Release before starting the Homebrew lane. Package-repository dispatch belongs after `github-release`.
- `.github/workflows/go-pre-publish.yml` uploads DEB, RPM, APK, SBOM, checksum, and Cosign bundle assets from `release-cli stage`.
- `.goreleaser.yaml` and `examples/go-release/.goreleaser.yaml` use nFPM but do not sign RPM or APK packages.
- `internal/profile/goprof.RunGoReleaser` invokes GoReleaser before `internal/stage.Stage` verifies checksums. Native signing can therefore occur before the canonical checksums without changing the release order.
- `publish-homebrew.yml`, `internal/stage/pubbrew`, and `internal/adapter/ghtap` are the closest existing examples: the publisher independently verifies its input, the engine owns reconciliation, the adapter owns I/O, and the workflow remains thin.
- `setup-release-cli` already owns CLI acquisition and version/protocol pairing.
- The repository has no S3/R2 client or pinned APT/RPM/APK repository toolchain. The spikes must select those tools from observed behavior.

The implementation starts with two disposable proofs. Do not merge production code before both gates pass.

## 2. Spike A: local native repository proof

### Goal

Prove package signing, repository layout, metadata generation, native verification, and replay behavior using one existing release and throwaway keys. Do not modify tracked source files.

### Work

Use a temporary directory outside the repository. Record all commands in disposable scripts so the same path can be repeated, but do not commit those scripts.

1. Download one existing release containing DEB, RPM, and APK assets, `checksums.txt`, its Cosign bundle, and GitHub attestations.
2. Verify the existing release with its checksum bundle and repository-bound GitHub attestations.
3. Rebuild the same source with temporary nFPM RPM and APK signing configuration.
4. Prove native signatures are embedded before checksums by verifying the package signatures and matching the final package SHA-256 values to the generated checksum manifest.
5. Generate static APT, RPM, and APK repository trees with candidate distro-native tools.
6. Prove the exact APK placement required by `apk`, including package objects referenced by `APKINDEX.tar.gz`.
7. Prove whether modern APT installs using `InRelease` plus by-hash metadata alone. If canonical `Packages` paths are required, classify them as mutable roots uploaded before `InRelease`.
8. Serve the tree over local HTTP and install `release-cli` in clean Debian, Fedora, and Alpine containers with native verification enabled and no insecure flags.
9. Generate the tree twice from identical package bytes. Record which metadata is byte-stable and what facts must be fixed to achieve stable replay.
10. Record exact working tool versions, container image digests, layouts, cache classifications, and command lines in a secret-free evidence summary.

### Acceptance gate

Proceed only when:

- nFPM-signed RPM and APK bytes are the bytes checksummed and verified;
- wrong native keys fail verification;
- the GitHub attestation check binds every package to the requested producer repository, tag, and signer workflow;
- APT, DNF, and APK install from the generated tree without insecure options;
- APT by-hash and APK package placement are settled;
- replay is either byte-stable or has a safe, externally verified no-write rule;
- all production tool choices can be pinned.

Stop and revise the architecture if producer identity cannot be established, native clients require a contradictory layout, signing occurs after checksums, or replay cannot avoid unsafe mutation.

### Cleanup

Delete all throwaway private keys, temporary package trees, scripts, worktrees, containers, and networks. Retain only the secret-free evidence summary.

## 3. Spike B: disposable R2 proof

### Goal

Prove the accepted local publication algorithm against real R2 and Cloudflare custom-domain caching. Reuse the Spike A generator; add only the smallest S3 transport wrapper needed for the experiment.

### Disposable resources

- one timestamped R2 bucket;
- one timestamped `*.meigma.dev` custom hostname;
- one bucket-scoped credential;
- one hostname-scoped Cache Rule only if object headers cannot produce the required behavior;
- throwaway aggregate signing keys.

Do not use a Worker.

### Work

1. Publish version A. Store an explicit SHA-256 value in each object's R2 metadata; never treat ETag as a content digest.
2. Install all three formats through the disposable HTTPS hostname.
3. Replay version A. Require `unchanged`, no immutable mutation, and no writes when the accepted replay rule applies.
4. Begin version B, stop after immutable packages and inner metadata upload but before mutable roots, and prove version A remains installable.
5. Resume version B and prove complete local regeneration, local verification before root writes, immutable-first upload, and live installation.
6. Compare every R2 package SHA-256 to the corresponding GitHub Release asset.
7. Confirm long immutable caching for packages, APT by-hash files, checksum-named RPM metadata, and versioned public keys.
8. Confirm mutable roots never serve a stale generation without purge automation.
9. Capture the minimum R2 operations, headers, endpoint settings, and Cache Rule required by production.

### Acceptance gate

Proceed only when first publish, replay, interrupted pre-root publication, resumed update, byte identity, and all three live installs pass. Stop if mutable paths remain stale, a Worker or purge loop becomes necessary, explicit SHA-256 metadata is lost, or a pre-root interruption changes a trusted root.

### Cleanup

Detach and remove the custom hostname, delete every object and the bucket, remove any disposable Cache Rule or DNS residue, revoke credentials, and delete local keys and containers. Retain only the secret-free evidence summary.

## 4. Production slice 1: opt-in producer signing

### Targets

- `.github/workflows/go-pre-publish.yml`
- `.goreleaser.yaml`
- `examples/go-release/.goreleaser.yaml`
- `internal/cli/stage.go` and focused tests
- `internal/profile/goprof/goreleaser.go` and focused tests
- existing release-contract documentation affected by package trust

### Change

Add an opt-in native-signing path that materializes per-producer RPM and APK private keys in the runner's temporary directory and passes only the spike-proven nFPM inputs and passphrase environment. Validate all required signing values before GoReleaser starts. Keep signing disabled by default until each producer provides its keys.

Do not add repository publication here. This slice changes only canonical package creation.

### Verification

- focused CLI and GoReleaser adapter tests for disabled, complete, and partial configuration;
- secret-redaction checks;
- real nFPM signing smoke proving native signatures and checksum ordering;
- unsigned default path remains byte-compatible with the current workflow.

Gate: merge only if signing produces the exact final GitHub Release bytes later accepted by native verification.

## 5. Production slice 2: local repository publisher

### Targets

- one new focused stage package for native repository publication;
- the smallest adapter package or packages required by the spike-selected repository tools;
- `.mockery.yml` only for genuine interfaces introduced at remote or subprocess boundaries;
- focused unit and adapter tests;
- package `doc.go` and Godoc required by repository rules.

Choose exact package and interface names during implementation. Do not create a general repository framework or storage plugin abstraction.

### Change

Implement the behavior that can run without GitHub or R2:

- strict producer configuration and allowlisting;
- repository/tag/package-name validation;
- package metadata inspection rather than filename trust;
- RPM and APK native signature verification;
- full local package-pool regeneration for APT, RPM, and APK;
- aggregate metadata signing;
- immutable versus mutable path classification and cache metadata;
- same-path, different-digest rejection;
- local install verification before any remote write.

Use `execx` for spike-selected external tools. Keep decisions in the engine and fixed argument mapping in narrow adapters.

### Verification

- table tests for configuration, path confinement, allowlists, architectures, and conflicts;
- adapter tests for exact arguments and secret redaction;
- generated mocks for genuine ports only;
- real local repository generation and Debian/Fedora/Alpine install smoke matching Spike A.

Gate: the production local tree and install results must match the accepted Spike A evidence.

## 6. Production slice 3: verified GitHub source and R2 reconciliation

### Targets

- extend the local publisher with narrow release-source and repository-store boundaries;
- extend `internal/adapter/ghrel` only through a new consumer-owned contract;
- add the smallest attestation adapter required for repository/tag/workflow-bound verification;
- add one R2 adapter, using the spike-selected S3 client approach;
- add the CLI composition and standard JSON result;
- focused tests in each changed package and `cmd/release-cli` composition.

### Change

Implement one end-to-end CLI operation that:

1. accepts the requested repository and tag plus checked-in packages-repository configuration;
2. downloads the complete public GitHub Release;
3. verifies the checksum bundle, package attestations, native signatures, and producer allowlist;
4. lists and mirrors all immutable R2 package objects locally;
5. verifies stored SHA-256 metadata while streaming;
6. regenerates and verifies the complete repository tree;
7. uploads missing immutable objects before mutable roots;
8. returns `published` or `unchanged` through `release.dev/result/v1`;
9. verifies installation through the public origin.

Reuse existing Cosign verification and JSON envelope behavior. Do not broaden the existing GitHub draft-release publication contracts to fit this reader.

### Verification

- GitHub API tests for exact public tag, drafts, prereleases, duplicate assets, and partial downloads;
- attestation tests proving repository, tag, signer workflow, and no-self-hosted binding;
- R2 tests for pagination, streaming SHA-256 validation, conflicts, ordered writes, interruption before roots, replay, and secret redaction;
- actual CLI smoke against local HTTP and disposable R2 for first publish, replay, conflict, interruption, and public install.

Gate: the production CLI must reproduce both accepted spikes. Delete remaining spike-only scripts after this gate.

## 7. Production slice 4: published composite Action

### Targets

- `.github/actions/publish-package-repository/action.yml`
- shared acquisition/version-guard code extracted from `setup-release-cli` only if needed to prevent duplication
- release-unit version/protocol stamp checks
- tool pins selected by the spikes

### Change

Publish a composite Action that acquires the matching CLI and pinned tools, maps inputs and secret-backed environment variables, invokes the single CLI operation with `--json`, validates its envelope, and exposes `state` plus the complete result. Keep permissions, environment selection, concurrency, and repository policy out of the Action.

### Verification

- existing setup Action acquisition remains unchanged;
- both Actions report the same version/protocol;
- branch/SHA rehearsal with a caller-supplied CLI, then released acquisition;
- end-to-end Action run against disposable R2;
- no secret in logs, outputs, or job summary.

Gate: release the Action and CLI as one unit before any packages repository pins it.

## 8. Production slice 5: initialize `meigma/pkgs`

### Targets in the new repository

- minimal checked-in producer/repository configuration using the spike-proven schema;
- versioned public keys;
- `.github/workflows/publish.yml`;
- `.gitignore`, `CODEOWNERS`, and concise operator documentation.

### Change

Create a workflow that runs on `repository_dispatch` and manual replay, checks out only trusted default-branch configuration, binds `packages-production`, uses one global concurrency group with `cancel-in-progress: false`, and invokes the released Action by full commit SHA.

Provision the production R2 bucket and `pkgs.meigma.dev` in the maintainer's existing Cloudflare account. Reproduce the accepted cache behavior. Add bucket-scoped R2 credentials and aggregate private keys to the protected Environment. Install the existing release GitHub App on `meigma/pkgs`.

Git tracks no private key, generated repository tree, package object, or duplicate publication inventory.

### Verification

- confirm tracked-file exclusions;
- manually run the workflow against disposable infrastructure first;
- then publish one approved signed release to the empty production bucket while producer dispatch remains disabled;
- install all three formats through `pkgs.meigma.dev`;
- replay and require `unchanged`.

Gate: the receiver, Environment, Action pin, concurrency, origin, and native installs must work before any producer can dispatch automatically.

## 9. Production slice 6: producer dispatch, documentation, and cutover

### Targets

- one thin reusable or local dispatch workflow following the existing Homebrew App-token pattern;
- `.github/workflows/release.yml`;
- the example release workflow with dispatch disabled by default;
- existing trust-boundary, GitHub release, CLI contract, tutorial, and configuration docs;
- at most one new package-repository reference or installation guide if the existing pages cannot carry the contract clearly.

### Change

After a public, non-prerelease GitHub Release exists, mint a short-lived token with the existing release App and send exactly `{repository, tag}` to `meigma/pkgs`. Keep the path disabled by default until the receiver and production origin pass Slice 5.

Document observed configuration, installation, verification, replay, manual public-key replacement, and failure recovery. Do not add formal key lifecycle machinery, a second App, automatic pruning, or extra channels.

### Cutover

1. Release the CLI and Action with dispatch disabled.
2. Pin `meigma/pkgs` to that release's full commit.
3. Prove a signed draft asset set independently; do not dispatch a draft.
4. Review package hashes, key fingerprints, App scope, R2 inventory, origin headers, and roll-forward path.
5. Enable native signing and dispatch for `meigma/release` in one small cutover PR.
6. Publish the first new signed release.
7. Observe the public GitHub Release, dispatch, serialized packages run, `published` result, and three public installs.
8. Replay the release and require `unchanged` without immutable mutation.
9. Publish installation documentation only after the live installs succeed.

## 10. Dependency order

```text
Local spike
    -> R2 spike
        -> producer signing ----------------------+
        -> local repository publisher             |
             -> GitHub/R2 CLI                     |
                  -> composite Action release     |
                       -> meigma/pkgs initialization
                            -> producer dispatch and cutover
```

Producer signing and the local publisher may be developed in parallel after both spikes. No caller is enabled before its callee is released and pinned. Keep `main` releasable after every slice.

## 11. Risks and stop triggers

| Risk | Control or stop trigger |
| --- | --- |
| Shared signer workflow permits producer confusion | Bind every attestation to requested repository, tag, signer workflow, and source ref. Stop if the live proof cannot establish all four. |
| Package layout differs from the proposal | Let native clients settle APT canonical paths and APK placement in Spike A; amend the architecture before production. |
| Metadata timestamps defeat replay | Fix spike-proven inputs or use the externally verified no-write path. Never overwrite immutable objects to chase reproducibility. |
| R2 ETag is mistaken for a digest | Store and stream-verify explicit SHA-256 metadata. |
| CDN caches mutable roots | Bypass caching or use the minimal hostname rule proven in Spike B. Stop if correctness requires a Worker or purge loop. |
| RPM root and detached signature update separately | Serialize adjacent writes and accept only fail-closed transient verification; rerun repairs the pair. |
| Full mirrors become slow | Record bytes and duration in results. Growth is a future re-evaluation trigger, not a reason to add a database now. |
| Secrets appear in subprocess or Action output | Keep secrets in masked environment/files, use restrictive permissions, and test redaction. |

## 12. Definition of done

- `meigma/release` signs RPM and APK packages before checksums and attestations.
- GitHub Release and R2 package SHA-256 values are identical.
- Producers dispatch only `{repository, tag}` after a public release and hold no R2 or aggregate signing credential.
- `meigma/pkgs` is the sole serialized writer and pins the released Action by full commit SHA.
- The CLI independently verifies producer identity, checksums, attestations, package signatures, R2 object digests, repository generation, upload order, and public installation.
- APT, DNF, and APK install from `https://pkgs.meigma.dev` with native verification enabled.
- Replay returns `unchanged`; a same-path, different-digest object fails; interruption before mutable roots leaves the prior repository usable.
- Immutable and mutable cache behavior matches the accepted R2 proof without a Worker or purge automation.
- Focused behavior tests, generated mocks for genuine ports, CLI/Action smoke tests, and live package-manager installations pass.
- Both spikes leave no keys, credentials, buckets, DNS records, Cache Rules, temporary files, containers, or networks.
- No service, database, repository manager, duplicate Git inventory, generic storage plugin system, second GitHub App, formal key policy, automatic pruning, or speculative channel exists.
