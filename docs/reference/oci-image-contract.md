# OCI image contract

This page defines the cross-repository contract for the reusable Go OCI builder and GHCR publisher at revision `72945990eda349f83c0f7628e85521fb30071fc6`.

For adoption steps, see [Configure OCI image publication](../how-to/configure-oci-images.md). The [GitHub Release contract](github-release-contract.md) defines the upstream GoReleaser producer and GitHub Release publisher. A complete consumer is available in the [Go release example](../../examples/go-release/).

## Pipeline boundary

```text
GoReleaser producer
  -> canonical linux/amd64 and linux/arm64 binaries
  -> verified oci-input artifact
  -> Melange signed APK repository
  -> locked apko multi-architecture OCI layout
  -> verified oci-image artifact
  -> ORAS GHCR publication
  -> recursive keyless Cosign signatures
  -> GitHub and registry provenance/SBOM attestations
  -> public GitHub Release
```

The image builder consumes prebuilt GoReleaser binaries. Melange packages them without compiling, stripping, or otherwise replacing them. The publisher consumes the builder's OCI layout and does not check out or execute consumer repository code.

## Reusable workflows

Consumers call both workflows at the same immutable revision:

```yaml
uses: meigma/release/.github/workflows/go-oci-build.yml@72945990eda349f83c0f7628e85521fb30071fc6
```

```yaml
uses: meigma/release/.github/workflows/publish-oci-image.yml@72945990eda349f83c0f7628e85521fb30071fc6
```

Moving branches and tags are not supported workflow references.

## Builder interface

`go-oci-build.yml` accepts these inputs:

| Input | Required | Default | Meaning |
| --- | --- | --- | --- |
| `artifact-id` | yes | none | Numeric ID of the canonical Linux binary artifact from `go-pre-publish.yml`. |
| `artifact-digest` | yes | none | GitHub artifact SHA-256 digest for that artifact. |
| `melange-config` | no | `melange.yaml` | Consumer-relative Melange configuration path. |
| `apko-config` | no | `apko.yaml` | Consumer-relative apko configuration path. |

It returns:

| Output | Meaning |
| --- | --- |
| `artifact-id` | Numeric ID of the authoritative `oci-image` artifact. |
| `artifact-url` | GitHub URL for the authoritative artifact. |
| `artifact-digest` | GitHub artifact SHA-256 digest. This covers the uploaded ZIP transport, not the OCI index. |
| `image-digest` | `sha256:` digest of the exact bytes in `layout/index.json`. |

The caller grants `actions: read` and `contents: read`. The builder has no registry credentials, package write permission, attestation permission, or release credential.

## Publisher interface

`publish-oci-image.yml` accepts these inputs:

| Input | Required | Default | Meaning |
| --- | --- | --- | --- |
| `artifact-id` | yes | none | Numeric ID of the `oci-image` artifact from `go-oci-build.yml`. |
| `artifact-digest` | yes | none | Expected GitHub artifact SHA-256 digest. |
| `image-digest` | yes | none | Expected OCI index digest. |
| `publish-image` | no | `false` | When `true`, push, sign, and attest the verified image. When `false`, stop after verification. |

It returns:

| Output | Meaning |
| --- | --- |
| `image-name` | Canonical `ghcr.io/owner/repository` name. |
| `image-reference` | Digest-pinned published reference, or empty when publication is disabled. |
| `image-digest` | Verified OCI index digest, regardless of publication mode. |
| `provenance-attestation-url` | GitHub provenance attestation URL, or empty when publication is disabled. |
| `amd64-sbom-attestation-url` | GitHub amd64 SBOM attestation URL, or empty when publication is disabled. |
| `arm64-sbom-attestation-url` | GitHub arm64 SBOM attestation URL, or empty when publication is disabled. |

The caller grants:

```yaml
permissions:
  actions: read
  artifact-metadata: write
  attestations: write
  contents: read
  id-token: write
  packages: write
```

`packages: write` authenticates ORAS, Cosign, and registry-backed GitHub attestations to GHCR. `id-token: write` supplies short-lived Sigstore identity. The publisher receives no GitHub App key and cannot mutate repository contents or releases.

## Consumer configuration

### GoReleaser

The upstream producer must emit exactly one canonical Linux binary for each pair:

| GOOS | GOARCH | GOAMD64 |
| --- | --- | --- |
| `linux` | `amd64` | `v1` |
| `linux` | `arm64` | unset |

The binaries must be static and executable. The OCI builder rejects a missing target, duplicate target, wrong architecture, symlink, non-executable file, or checksum mismatch.

### Melange

`melange.yaml` must:

- declare `x86_64` and `aarch64` target architectures;
- use `${{vars.version}}` as the package version;
- install the staged file named `application` as the intended command;
- preserve mode `0755` and ownership `0:0`; and
- name the package consumed by `apko.yaml`.

The workflow injects the stable tag version and writes an ephemeral APK signing key. It retains the public key, signed APKs, signed repository indexes, package SBOMs, and Melange provenance in the workflow artifact. The private signing key is never uploaded.

### apko

`apko.yaml` must:

- consume the Melange package;
- define `amd64` and `arm64` only;
- set exactly one executable entrypoint;
- run as numeric user and group `65532` through the `nonroot` account;
- include source, title, description, and SPDX license annotations; and
- include the runtime files the command requires.

The current example includes Alpine's CA certificate bundle. A command that does not make TLS connections may deliberately omit it after testing; a command that needs other runtime data must declare the corresponding package explicitly.

## Authoritative artifact

The builder uploads `oci-image` with seven-day retention and no additional ZIP compression. Its contract includes:

```text
apko-lock.json
apk-signing.rsa.pub
configuration/apko.yaml
configuration/melange.yaml
image-digest.txt
layout/index.json
layout/oci-layout
layout/blobs/sha256/*
packages/aarch64/*
packages/x86_64/*
sboms/sbom-aarch64.spdx.json
sboms/sbom-x86_64.spdx.json
```

The package directories also contain signed APK repository indexes, Melange provenance, embedded package SBOMs, and the signed APKs. Files not listed above may be diagnostic outputs from the pinned tools; consumers must not infer a stable API from undocumented filenames.

The publisher verifies all three handoff coordinates before a registry write:

1. artifact ID belongs to the current workflow run;
2. GitHub artifact digest equals the caller-supplied digest; and
3. recorded, recomputed, and caller-supplied OCI index digests are identical.

It also requires one Linux manifest for `amd64`, one for `arm64`, both referenced blobs, and parseable SPDX JSON for each architecture.

## Published image

### Name

The image name is derived from the caller repository:

```text
ghcr.io/<lowercase-owner>/<lowercase-repository>
```

Custom registry hosts, namespaces, and image names are outside the current contract.

### Tags

A stable release tag `vMAJOR.MINOR.PATCH` publishes:

| Image tag | Behavior |
| --- | --- |
| `MAJOR.MINOR.PATCH` | Exact release version. Must never be reassigned to different content. |
| `MAJOR.MINOR` | Moves to the latest stable patch in that minor line. |
| `MAJOR` | Moves to the latest stable release in that major line. |
| `latest` | Moves to the latest stable release. |

All four tags must resolve to the builder's expected OCI index digest immediately after publication. The publisher rejects prerelease, build-metadata, malformed, branch, and untagged refs.

Digest-pinned references are the durable consumer interface:

```text
ghcr.io/owner/repository@sha256:<index-digest>
```

### Runtime invariants

For both platforms, the builder verifies:

- Linux operating system and expected architecture;
- one apko index containing exactly two platform manifests;
- package and image executable bytes equal the canonical GoReleaser binary;
- executable ownership `0:0` inside the image layer;
- executable mode `0755` from package configuration;
- configured entrypoint;
- runtime user `65532`;
- source, version, revision, title, description, and license annotations; and
- SPDX SBOM inclusion of the application package version.

## Signatures and attestations

The publisher signs the index and both platform manifests recursively with Cosign keyless signing. Verification must require:

| Field | Value |
| --- | --- |
| Certificate identity | `https://github.com/meigma/release/.github/workflows/publish-oci-image.yml@72945990eda349f83c0f7628e85521fb30071fc6` |
| Certificate OIDC issuer | `https://token.actions.githubusercontent.com` |
| Subject | Digest-pinned image or platform manifest. |

The publisher also creates:

- one SLSA provenance attestation for the multi-architecture index;
- one SPDX SBOM attestation for the amd64 platform manifest; and
- one SPDX SBOM attestation for the arm64 platform manifest.

Each attestation is written to the consumer repository's GitHub attestation store and pushed to GHCR as an OCI referrer. Verification should constrain the reusable signer workflow, signer revision, consumer source tag, and GitHub-hosted runner as shown in the configuration guide.

Cosign signatures and GitHub attestations are distinct. A passing check for one does not prove the other exists.

## Publication states

| State | Registry effect | Expected next action |
| --- | --- | --- |
| `publish-image: false` | None. Artifact and index verification only. | Inspect the rehearsal and enable publication in a reviewed commit. |
| Push incomplete | Some blobs or tags may exist. GitHub Release remains draft. | Fix the cause and rerun the same unpublished tag. |
| Push complete, signing incomplete | Tags resolve correctly but trust metadata is incomplete. GitHub Release remains draft. | Rerun; do not advertise or make the package public. |
| Signed, attestation incomplete | Image is signed but does not satisfy the complete contract. GitHub Release remains draft. | Rerun and verify every attestation. |
| Complete, package private | Authenticated pulls work. | Inspect, then perform the one-time public visibility change. |
| Complete, package public | Anonymous digest and tag pulls work. | Monitor and publish only additive corrective releases. |

A rerun is content-addressed and rechecks every published tag. It may add duplicate valid signatures or attestations after a partial success. It must not publish the GitHub Release until the image publisher job succeeds.

## GHCR visibility

GHCR visibility is independent of source repository visibility. The organization's package-creation setting determines the initial visibility. The workflow links the package to its source repository through `GITHUB_TOKEN` publication and the `org.opencontainers.image.source` annotation; inherited repository access permissions do not make the package public.

The completed delivery state is public. Inspect `visibility` through the Packages REST API after the first complete publication. If it is private, an organization owner must inspect the signed and attested image, then make the package public through its settings page. GitHub does not expose a supported Packages REST operation for this visibility change.

## Security boundary

The current boundary is deliberately split:

- `go-pre-publish.yml` compiles and signs release inputs without release or package write access;
- `go-oci-build.yml` packages and composes the image without registry credentials;
- `publish-oci-image.yml` does not check out consumer source and writes only to the caller's GHCR package and attestation store; and
- `publish-github-release.yml` waits for image publication but uses a separate short-lived Release App token for release mutation.

The workflow artifact is temporary transport, not a public distribution channel. The OCI digest, registry content, Cosign identity, and attestation identities form the public verification boundary.

## Unsupported cases

The current contract does not support:

- CGO-dependent or dynamically linked commands;
- architectures other than Linux amd64 and arm64;
- prerelease tags;
- multiple commands or entrypoints in one image;
- custom registries or image names;
- private package visibility automation;
- long-lived registry credentials;
- mutable exact-version tags; or
- publication from branch or manual-dispatch refs.
