# Workflow capability inventory

The four reusable workflows implement one release chain: GoReleaser builds and signs archives plus canonical Linux executables; Melange/apko package those exact executables into a verified OCI layout; the OCI publisher pushes, signs, attests, and tags the layout; only then may the GitHub publisher change the matching populated draft to non-draft. The local dogfood caller enforces `release-assets -> oci-image -> oci-publish -> github-release`, with `github-release` also explicitly needing the first two jobs. (`.github/workflows/release.yml:14-73`)

## 1. Pipeline map

### Cross-workflow graph and caller context

```text
push v* tag (deleted == false)
  release-assets: go-pre-publish.yml
    ├─ release-assets artifact ───────────────────────────────┐
    └─ oci-build-inputs artifact                              │
         ↓                                                    │
  oci-image: go-oci-build.yml                                 │
         └─ oci-image artifact + OCI index digest             │
              ↓                                               │
  oci-publish: publish-oci-image.yml                           │
         └─ digest-pinned GHCR reference                      │
              ↓                                               ↓
  github-release: publish-github-release.yml (also needs release-assets and oci-image)
```

The caller triggers on `v*` tag pushes, rejects deletion at the producer job, serializes a workflow/ref pair without cancellation, and enables both publishers. (`.github/workflows/release.yml:3-21`, `.github/workflows/release.yml:32-73`) Release Please creates a `v`-prefixed draft release and tag because `force-tag-creation`, `include-v-in-tag`, and `draft` are true. (`release-please-config.json:3-10`) Release Please itself runs on `main` pushes or manual dispatch using a GitHub App token. (`.github/workflows/release-please.yml:6-39`)

### `go-pre-publish.yml`

**`workflow_call` interface.** It declares no inputs and no secrets. Its six outputs have no explicit type field and therefore are passed as Actions output strings. (`.github/workflows/go-pre-publish.yml:3-23`)

| Output | Meaning/source |
| --- | --- |
| `artifact-id` | `steps.upload.outputs.artifact-id`, ID of `release-assets`. (`.github/workflows/go-pre-publish.yml:5-8`, `.github/workflows/go-pre-publish.yml:32-35`) |
| `artifact-url` | `steps.upload.outputs.artifact-url`, URL of `release-assets`. (`.github/workflows/go-pre-publish.yml:9-11`, `.github/workflows/go-pre-publish.yml:32-35`) |
| `artifact-digest` | `steps.upload.outputs.artifact-digest`, SHA-256 transport digest of `release-assets`. (`.github/workflows/go-pre-publish.yml:12-14`, `.github/workflows/go-pre-publish.yml:32-35`) |
| `oci-input-artifact-id` | `steps.upload-oci-input.outputs.artifact-id`, ID of `oci-build-inputs`. (`.github/workflows/go-pre-publish.yml:15-17`, `.github/workflows/go-pre-publish.yml:36-38`) |
| `oci-input-artifact-url` | `steps.upload-oci-input.outputs.artifact-url`, URL of `oci-build-inputs`. (`.github/workflows/go-pre-publish.yml:18-20`, `.github/workflows/go-pre-publish.yml:36-38`) |
| `oci-input-artifact-digest` | `steps.upload-oci-input.outputs.artifact-digest`, SHA-256 transport digest of `oci-build-inputs`. (`.github/workflows/go-pre-publish.yml:21-23`, `.github/workflows/go-pre-publish.yml:36-38`) |

**Job graph and permissions.** There is one job, `release-assets`, with no `needs`; it runs on `ubuntu-24.04`, times out after 20 minutes, and receives only `contents: read` and `id-token: write`. Workflow-scope permissions are empty. (`.github/workflows/go-pre-publish.yml:25-44`)

**Artifacts produced.**

| Artifact | Exact uploaded paths | Retention/encoding | Integrity coordinate |
| --- | --- | --- | --- |
| `oci-build-inputs` | `dist/artifacts.json`, `dist/*_linux_amd64*/**`, `dist/*_linux_arm64*/**`. (`.github/workflows/go-pre-publish.yml:120-127`) | 7 days, `compression-level: 0`, fail if absent. (`.github/workflows/go-pre-publish.yml:128-131`) | Upload action emits ID, URL, and SHA-256 artifact digest; downstream also validates API metadata and download digest. (`.github/workflows/go-pre-publish.yml:32-38`, `.github/workflows/go-oci-build.yml:106-149`) |
| `release-assets` | `dist/*.tar.gz`, `dist/*.zip`, `dist/*.sbom.json`, `dist/checksums.txt`, `dist/checksums.txt.sigstore.json`. (`.github/workflows/go-pre-publish.yml:132-143`) | 7 days, `compression-level: 0`, fail if absent. (`.github/workflows/go-pre-publish.yml:141-145`) | Producer first verifies every checksum and requires a nonempty Sigstore bundle; publisher separately validates API metadata, download digest, every payload digest, and the closed file set. (`.github/workflows/go-pre-publish.yml:88-93`, `.github/workflows/publish-github-release.yml:129-164`, `.github/workflows/publish-github-release.yml:215-333`) |

Under the checked-in GoReleaser profile, the release artifact is six archives, six archive SBOMs, `checksums.txt`, and its Sigstore bundle: fourteen files. (`.goreleaser.yaml:13-63`, `docs/reference/github-release-contract.md:259-272`)

### `go-oci-build.yml`

**`workflow_call` inputs.** (`.github/workflows/go-oci-build.yml:4-23`)

| Input | Type | Required | Default |
| --- | --- | --- | --- |
| `artifact-id` | string | true | none |
| `artifact-digest` | string | true | none |
| `melange-config` | string | false | `melange.yaml` |
| `apko-config` | string | false | `apko.yaml` |

It declares no `workflow_call` secrets. (`.github/workflows/go-oci-build.yml:4-36`)

**Outputs.** All have no explicit type field and are output strings. (`.github/workflows/go-oci-build.yml:24-36`)

| Output | Meaning/source |
| --- | --- |
| `artifact-id` | `steps.upload.outputs.artifact-id`, authoritative `oci-image` artifact ID. (`.github/workflows/go-oci-build.yml:25-27`, `.github/workflows/go-oci-build.yml:45-49`) |
| `artifact-url` | `steps.upload.outputs.artifact-url`. (`.github/workflows/go-oci-build.yml:28-30`, `.github/workflows/go-oci-build.yml:45-49`) |
| `artifact-digest` | `steps.upload.outputs.artifact-digest`, Actions artifact SHA-256 transport digest. (`.github/workflows/go-oci-build.yml:31-33`, `.github/workflows/go-oci-build.yml:45-49`) |
| `image-digest` | `steps.verify.outputs.image-digest`, `sha256:` plus SHA-256 of the exact `layout/index.json` bytes. (`.github/workflows/go-oci-build.yml:34-36`, `.github/workflows/go-oci-build.yml:426-434`) |

**Job graph and permissions.** One job, `oci-image`, with no `needs`; `ubuntu-24.04`, 20-minute timeout, `actions: read`, `contents: read`, and empty workflow-scope permissions. (`.github/workflows/go-oci-build.yml:38-55`)

**Artifacts consumed/produced and digest scheme.** It consumes `oci-build-inputs` by numeric ID. Before download it calls `actions.getArtifact`, rejects non-positive/non-safe-integer IDs, expiry, a different `workflow_run.id`, and a caller digest mismatch after removing an optional `sha256:` prefix and lowercasing. (`.github/workflows/go-oci-build.yml:106-141`) `actions/download-artifact` then downloads by ID with `digest-mismatch: error`. (`.github/workflows/go-oci-build.yml:143-149`) It produces `oci-image` from all of `oci-output/`, retained seven days with compression disabled; that directory includes copied configuration, canonical-binary checksums, signed APK repositories and public key, apko lock, OCI layout/blobs, two SPDX SBOMs, and `image-digest.txt`. (`.github/workflows/go-oci-build.yml:150-245`, `.github/workflows/go-oci-build.yml:247-314`, `.github/workflows/go-oci-build.yml:437-445`; `docs/reference/oci-image-contract.md:136-161`)

### `publish-github-release.yml`

**`workflow_call` inputs and secret.** (`.github/workflows/publish-github-release.yml:4-40`)

| Input/secret | Kind | Type | Required | Default |
| --- | --- | --- | --- | --- |
| `artifact-id` | input | string | true | none |
| `artifact-digest` | input | string | true | none |
| `checksum-signing-workflow-ref` | input | string | true | none |
| `release-app-client-id` | input | string | true | none |
| `publish-release` | input | boolean | false | `true` |
| `require-oci-image` | input | boolean | false | `false` |
| `oci-image-reference` | input | string | false | empty string |
| `release-app-private-key` | secret | secret | true | none |

**Outputs.** Both are output strings. (`.github/workflows/publish-github-release.yml:41-47`)

| Output | Meaning/source |
| --- | --- |
| `attestation-url` | `steps.attest.outputs.attestation-url`, GitHub build-provenance attestation URL. (`.github/workflows/publish-github-release.yml:41-44`, `.github/workflows/publish-github-release.yml:56-58`) |
| `release-url` | `steps.release.outputs.url`, HTML URL of the populated release whether left draft or published. (`.github/workflows/publish-github-release.yml:45-47`, `.github/workflows/publish-github-release.yml:56-58`) |

**Job graph and permissions.** One `publish` job with no `needs`; `ubuntu-24.04`, 10-minute timeout, `actions: read`, `artifact-metadata: write`, `attestations: write`, `contents: read`, and `id-token: write`. Workflow-scope permissions are empty. (`.github/workflows/publish-github-release.yml:49-67`)

**Artifact consumed and digest scheme.** It consumes `release-assets` by ID. It applies the same positive-ID, expiry, current-run, normalized GitHub artifact digest check, requests three action retries, and downloads with `digest-mismatch: error`. (`.github/workflows/publish-github-release.yml:129-164`, `.github/workflows/publish-github-release.yml:215-220`) It then treats `checksums.txt` as the authoritative payload list, computes each payload SHA-256, validates the Cosign bundle and signer identity, uploads the payloads plus two control files, and finally compares GitHub's `sha256:` digest for every uploaded release asset against the locally computed map. (`.github/workflows/publish-github-release.yml:222-333`, `.github/workflows/publish-github-release.yml:341-405`, `.github/workflows/publish-github-release.yml:407-482`)

### `publish-oci-image.yml`

**`workflow_call` inputs.** (`.github/workflows/publish-oci-image.yml:4-22`)

| Input | Type | Required | Default |
| --- | --- | --- | --- |
| `artifact-id` | string | true | none |
| `artifact-digest` | string | true | none |
| `image-digest` | string | true | none |
| `publish-image` | boolean | false | `false` |

It declares no `workflow_call` secrets. (`.github/workflows/publish-oci-image.yml:4-41`)

**Outputs.** All are output strings. (`.github/workflows/publish-oci-image.yml:23-41`)

| Output | Meaning/source |
| --- | --- |
| `image-name` | Canonical lowercase `ghcr.io/<owner>/<repo>` from `steps.stage`. (`.github/workflows/publish-oci-image.yml:24-26`, `.github/workflows/publish-oci-image.yml:218-232`) |
| `image-reference` | Digest-pinned reference from `steps.push`, empty when publication is disabled. (`.github/workflows/publish-oci-image.yml:27-29`, `.github/workflows/publish-oci-image.yml:459-467`) |
| `image-digest` | Verified index digest from `steps.stage`, also returned in verification-only mode. (`.github/workflows/publish-oci-image.yml:30-32`, `.github/workflows/publish-oci-image.yml:218-232`) |
| `provenance-attestation-url` | Index provenance URL, empty when publication is disabled. (`.github/workflows/publish-oci-image.yml:33-35`, `.github/workflows/publish-oci-image.yml:483-490`) |
| `amd64-sbom-attestation-url` | amd64 manifest SBOM attestation URL, empty when disabled. (`.github/workflows/publish-oci-image.yml:36-38`, `.github/workflows/publish-oci-image.yml:492-500`) |
| `arm64-sbom-attestation-url` | arm64 manifest SBOM attestation URL, empty when disabled. (`.github/workflows/publish-oci-image.yml:39-41`, `.github/workflows/publish-oci-image.yml:502-510`) |

**Job graph and permissions.** One `publish` job with no `needs`; `ubuntu-24.04`, 15-minute timeout, repository-wide concurrency group `oci-publish-${{ github.repository_id }}` with no cancellation, and permissions `actions: read`, `artifact-metadata: write`, `attestations: write`, `contents: read`, `id-token: write`, `packages: write`. Unlike the other three reusable workflows, this file has no separate top-level `permissions: {}` declaration; its permissions are job-local. (`.github/workflows/publish-oci-image.yml:43-64`)

**Artifact consumed and digest scheme.** It consumes `oci-image` by ID, validates the same Actions artifact metadata tuple, and downloads with `digest-mismatch: error`. (`.github/workflows/publish-oci-image.yml:88-130`) It then validates a second, content-level coordinate: caller `image-digest`, `image-digest.txt`, and SHA-256 of exact `layout/index.json` bytes must all match `^sha256:[0-9a-f]{64}$`. (`.github/workflows/publish-oci-image.yml:132-181`) The artifact ZIP digest and OCI index digest are distinct coordinates. (`.github/workflows/publish-oci-image.yml:88-181`; `docs/reference/oci-image-contract.md:157-161`)

## 2. Step catalog

Effect codes: **P** = pure computation/validation; **F** = local filesystem work; **X** = external process invocation; **R** = remote API, identity service, artifact service, or registry operation. “Generic” means ecosystem-neutral release logic; qualifications identify baked-in assumptions.

### Producer: `go-pre-publish.yml`

| ID | Logical operation; inputs → outputs | Effect | Scope | Source |
| --- | --- | --- | --- | --- |
| PP-01 | Require `GITHUB_REF_TYPE=tag`; ref metadata → pass/fail. | P | Generic tag gate. | `.github/workflows/go-pre-publish.yml:46-52` |
| PP-02 | Checkout the tagged repository with full history, blob filtering, and no persisted credential; GitHub repository → worktree/git history. | R/F (`actions/checkout`) | Generic. | `.github/workflows/go-pre-publish.yml:54-59` |
| PP-03 | Install mise 2026.8.8 and locked `go`, GoReleaser, Syft, Cosign; repository mise files/network → tool executables/cache. | R/F/X (`mise`) | Go-specific tool selection. | `.github/workflows/go-pre-publish.yml:61-72` |
| PP-04 | Prove active `go`, `goreleaser`, `syft`, `cosign` paths equal `mise which`, then print Go version; PATH/mise state → validation/logs. | X (`bash`, `realpath`, `go`, `mise`) | Go-specific. | `.github/workflows/go-pre-publish.yml:74-86` |
| PP-05 | Run `goreleaser release --clean --skip=publish`; source/tag/config → cross-platform archives, archive SBOMs, `artifacts.json`, checksum manifest, keyless Cosign bundle under `dist`. | X/F/R (`goreleaser`, which invokes Go/Syft/Cosign; Cosign uses GitHub OIDC/Sigstore) | Inherently Go/GoReleaser-specific. | `.github/workflows/go-pre-publish.yml:74-86`, `.goreleaser.yaml:13-69` |
| PP-06 | Verify all `checksums.txt` payload hashes and require nonempty `checksums.txt.sigstore.json`; bundle files → pass/fail. | X/F (`sha256sum`, `test`) | Generic integrity validation over GoReleaser filenames. | `.github/workflows/go-pre-publish.yml:88-93` |
| PP-07 | Parse `dist/artifacts.json` for Linux `Binary` records, require exactly two and architectures exactly `amd64`,`arm64`; GoReleaser JSON → selected records/pass-fail. | P/F/X (`jq`, `cut`, `sort`, `diff`) | GoReleaser-specific schema and GOOS/GOARCH values. | `.github/workflows/go-pre-publish.yml:95-113` |
| PP-08 | Require each selected canonical binary path to be executable; selected records/worktree → pass/fail. | F (`test -x`) | Go-specific handoff shape. | `.github/workflows/go-pre-publish.yml:114-119` |
| PP-09 | Upload canonical manifest and two Linux directory trees as `oci-build-inputs`; files → artifact ID/URL/digest. | R/F (`actions/upload-artifact`) | Generic transport with GoReleaser-specific path globs. | `.github/workflows/go-pre-publish.yml:120-131` |
| PP-10 | Upload archives, ZIPs, SBOM JSON, checksum and Sigstore control files as `release-assets`; files → artifact ID/URL/digest. | R/F (`actions/upload-artifact`) | Generic transport with GoReleaser-specific output patterns. | `.github/workflows/go-pre-publish.yml:132-145` |

### Builder: `go-oci-build.yml`

| ID | Logical operation; inputs → outputs | Effect | Scope | Source |
| --- | --- | --- | --- | --- |
| OB-01 | Require a tag ref and a `v`-prefixed tag name. | P | Generic with baked-in `v` convention. | `.github/workflows/go-oci-build.yml:56-65` |
| OB-02 | Checkout full tagged repository, no persisted credentials. | R/F (`actions/checkout`) | Generic; needed for consumer configs and tag history. | `.github/workflows/go-oci-build.yml:67-73` |
| OB-03 | Install locked GitHub CLI, Melange, and apko with mise 2026.8.8. | R/F/X (`mise`) | Ecosystem-neutral packaging tools; current inputs are Go binaries. | `.github/workflows/go-oci-build.yml:74-85` |
| OB-04 | Install/enable arm64 binfmt through a digest-pinned QEMU image. | R/X (container pull, `docker/setup-qemu-action`) | Generic multi-architecture execution. | `.github/workflows/go-oci-build.yml:86-90` |
| OB-05 | Verify `gh`, `melange`, and `apko` resolve to mise-managed paths and print Melange/apko versions. | X (`bash`, `realpath`, `mise`, `melange`, `apko`) | Generic tool provenance. | `.github/workflows/go-oci-build.yml:92-104` |
| OB-06 | Validate canonical artifact ID, expiry, current run ownership, and normalized GitHub-reported digest through `GET Actions artifact` (`github.rest.actions.getArtifact`). | R/P (GitHub Actions REST) | Generic artifact handoff. | `.github/workflows/go-oci-build.yml:106-141` |
| OB-07 | Download artifact by ID to `oci-input`, failing on download digest mismatch. | R/F (`actions/download-artifact`) | Generic artifact handoff. | `.github/workflows/go-oci-build.yml:143-149` |
| OB-08 | Require both config files and `artifacts.json`; replace temporary/output trees and create per-architecture source/output directories. | F/X (`test`, `rm`, `mkdir`) | Generic staging, fixed two-architecture layout. | `.github/workflows/go-oci-build.yml:150-177` |
| OB-09 | For `amd64:x86_64` and `arm64:aarch64`, select exactly one Linux `Binary` from GoReleaser JSON, require a `dist/`-relative path, resolve it under the downloaded root, reject escape, and require a regular file. | P/F/X (`jq`, `realpath`, `test`) | Inherently GoReleaser-specific schema/path mapping. | `.github/workflows/go-oci-build.yml:178-209` |
| OB-10 | Inspect each file as static 64-bit ELF of the expected machine, require a common artifact name, and install it mode 0755 as `sources/<apkarch>/application`. | F/X (`file`, `install`) | Go profile-specific static Linux binary assumption; packaging itself is generic. | `.github/workflows/go-oci-build.yml:210-233` |
| OB-11 | Derive version by removing `v`, write Melange vars JSON, derive build date from tagged commit, copy configs, hash both canonical inputs, and emit step outputs (`work`, `output`, `version`, `build-date`, `binary-name`). | P/F/X (`jq`, `git`, `install`, `sha256sum`) | Generic release metadata with baked `v` tag and Go binary naming. | `.github/workflows/go-oci-build.yml:234-245` |
| OB-12 | Compile/validate Melange configuration for x86_64. | X/F (`melange compile`) | Ecosystem-neutral APK packaging. | `.github/workflows/go-oci-build.yml:247-262` |
| OB-13 | Generate ephemeral RSA APK signing key and retain only its public key in output. | X/F (`melange keygen`, `install`) | Ecosystem-neutral APK signing. | `.github/workflows/go-oci-build.yml:263-264` |
| OB-14 | Build x86_64 and aarch64 signed APK repositories using Docker, namespace/source/revision/build date, vars, and `--generate-provenance`. | X/F (`melange build`, Docker) | Ecosystem-neutral package build over staged Go executable. | `.github/workflows/go-oci-build.yml:266-279` |
| OB-15 | Require each architecture's `APKINDEX.tar.gz` and exactly one `.apk`. | F/X (`test`, shell glob/`wc`) | Generic, but exactly-one-package invariant. | `.github/workflows/go-oci-build.yml:281-284` |
| OB-16 | Lock apko config for x86_64+aarch64 against the local signed repositories/key. | X/F (`apko lock`) | Ecosystem-neutral OCI composition; fixed platforms. | `.github/workflows/go-oci-build.yml:285-303` |
| OB-17 | Build two-platform OCI layout and SPDX SBOMs with stable build date and version/revision annotations. | X/F (`apko build`) | Ecosystem-neutral, fixed Linux amd64/arm64 and local image reference. | `.github/workflows/go-oci-build.yml:304-314` |
| OB-18 | Validate OCI index schema/media type, exactly two Linux amd64/arm64 manifests, and required source/title/description/license/version/revision annotations. | P/F/X (`jq`) | Generic OCI contract, fixed platform/annotation set. | `.github/workflows/go-oci-build.yml:315-355` |
| OB-19 | For each platform, validate manifest media type, exactly one layer, annotations, config architecture/OS, one entrypoint, numeric user `65532`, and labels. | P/F/X (`jq`) | Generic OCI runtime contract with single-layer/single-command assumptions. | `.github/workflows/go-oci-build.yml:356-405` |
| OB-20 | Inspect layer entry mode and owner, then hash extracted executable bytes and require equality with the canonical staged binary. | F/X (`tar`, `sha256sum`) | Generic no-rebuild byte-identity check; paths originate in Go handoff. | `.github/workflows/go-oci-build.yml:406-423` |
| OB-21 | Require each architecture SPDX SBOM to contain an application package at `${VERSION}-r0`; compute/write index digest and expose it. | P/F/X (`jq`, `sha256sum`) | Generic SPDX/OCI integrity with Melange `-r0` version convention. | `.github/workflows/go-oci-build.yml:424-436` |
| OB-22 | Upload all `oci-output/` as `oci-image`, no compression, seven-day retention. | R/F (`actions/upload-artifact`) | Generic artifact transport. | `.github/workflows/go-oci-build.yml:437-445` |

### GitHub Release publisher: `publish-github-release.yml`

| ID | Logical operation; inputs → outputs | Effect | Scope | Source |
| --- | --- | --- | --- | --- |
| GR-01 | Require tag ref; when publishing and OCI is required, require exactly lowercase caller `ghcr.io/owner/repo@sha256:<64 lowercase hex>`. | P | Generic GitHub release gate with fixed GHCR naming. | `.github/workflows/publish-github-release.yml:68-96` |
| GR-02 | Checkout full tagged repository without persisted credentials. | R/F (`actions/checkout`) | Generic; used to resolve tag commit. | `.github/workflows/publish-github-release.yml:98-104` |
| GR-03 | Install locked GitHub CLI and Cosign, then execute their version commands through mise. | R/F/X (`mise`, `gh`, `cosign`) | Generic release tooling. | `.github/workflows/publish-github-release.yml:105-119` |
| GR-04 | Mint short-lived GitHub App installation token from client ID/private key with `contents: write`. | R (`actions/create-github-app-token`, GitHub App installation-token API) | Generic GitHub release mutation credential. | `.github/workflows/publish-github-release.yml:121-128` |
| GR-05 | Validate artifact ID, expiry, current workflow run, and normalized GitHub-reported digest. | R/P (GitHub Actions REST `actions.getArtifact`) | Generic handoff. | `.github/workflows/publish-github-release.yml:129-164` |
| GR-06 | With App token, paginate repository releases and poll up to 24 times at five seconds for exact tag; require it exists and is draft. | R/P (GitHub Releases REST `repos.listReleases`) | Generic draft discovery/retry. | `.github/workflows/publish-github-release.yml:166-199` |
| GR-07 | Resolve tag commit with `git rev-list`, require `github.sha`, and output selected release ID. | X/P (`git`) | Generic tag/release binding. | `.github/workflows/publish-github-release.yml:200-213` |
| GR-08 | Download authoritative artifact to `dist` with digest mismatch fatal. | R/F (`actions/download-artifact`) | Generic. | `.github/workflows/publish-github-release.yml:215-220` |
| GR-09 | Require checksum manifest and Sigstore bundle to exist as regular files. | F/P (Node `fs.lstatSync`) | Generic. | `.github/workflows/publish-github-release.yml:222-270` |
| GR-10 | Parse nonempty flat checksum manifest with strict filename/digest grammar, reject duplicates/control self-listing, hash every payload, and reject any unlisted/non-file `dist` entry. | P/F (Node crypto/fs) | Generic closed release-bundle validation. | `.github/workflows/publish-github-release.yml:271-308` |
| GR-11 | Verify `checksums.txt` using its Cosign bundle, exact caller-supplied certificate identity, and GitHub Actions OIDC issuer. | X/R (`cosign verify-blob`; Sigstore verification data) | Generic signature verification. | `.github/workflows/publish-github-release.yml:309-322` |
| GR-12 | Emit JSON arrays/maps for upload paths and SHA-256 digests, including both control files. | P/F (Node crypto/fs) | Generic. | `.github/workflows/publish-github-release.yml:324-333` |
| GR-13 | Create GitHub build-provenance attestations using `dist/checksums.txt` as `subject-checksums`; output attestation URL. | R (`actions/attest`, GitHub attestation/OIDC services) | Generic checksummed subjects. | `.github/workflows/publish-github-release.yml:335-339` |
| GR-14 | Re-list releases with App token, require tag uniquely maps to selected ID, list current release assets, and reject names outside the expected closed set. | R/P (GitHub Releases REST `listReleases`, `listReleaseAssets`) | Generic convergent release validation. | `.github/workflows/publish-github-release.yml:341-390` |
| GR-15 | Upload all expected files with `gh release upload <tag> ... --clobber` under App token. | X/R (`gh`; GitHub release asset upload API) | Generic. | `.github/workflows/publish-github-release.yml:392-405` |
| GR-16 | Poll release assets up to 12 times at one second until expected count and every asset has digest and `uploaded` state. | R/P (GitHub Releases REST `listReleaseAssets`) | Generic remote-state convergence. | `.github/workflows/publish-github-release.yml:407-448` |
| GR-17 | Reject duplicate names, wrong count, missing name, or any GitHub `sha256:` digest differing from the local map. | P | Generic closed-set integrity. | `.github/workflows/publish-github-release.yml:449-466` |
| GR-18 | If enabled, set `draft:false`; then fetch release, require final draft state equals requested mode, and output HTML URL. | R/P (GitHub Releases REST `updateRelease`, `getRelease`) | Generic final publication gate. | `.github/workflows/publish-github-release.yml:468-482` |

### OCI publisher: `publish-oci-image.yml`

| ID | Logical operation; inputs → outputs | Effect | Scope | Source |
| --- | --- | --- | --- | --- |
| OP-01 | Require exact `refs/tags/vMAJOR.MINOR.PATCH` with canonical nonnegative decimal components. | P | Generic stable SemVer subset; baked `v`, no prerelease/build metadata. | `.github/workflows/publish-oci-image.yml:66-75` |
| OP-02 | Install ORAS 1.3.3 and Cosign 3.1.3 via inline mise configuration. | R/F/X (`mise`) | Generic OCI publication. | `.github/workflows/publish-oci-image.yml:77-87` |
| OP-03 | Validate artifact ID, expiry, current run ownership, and normalized GitHub digest. | R/P (GitHub Actions REST `actions.getArtifact`) | Generic handoff. | `.github/workflows/publish-oci-image.yml:88-123` |
| OP-04 | Download authoritative OCI artifact with digest mismatch fatal. | R/F (`actions/download-artifact`) | Generic. | `.github/workflows/publish-oci-image.yml:125-130` |
| OP-05 | Require OCI layout/index/digest files and parse both architecture SPDX JSON documents. | F/P (Node fs/JSON) | Generic OCI/SPDX validation; fixed filenames/platforms. | `.github/workflows/publish-oci-image.yml:132-166` |
| OP-06 | Validate caller digest grammar and equality among caller value, `image-digest.txt`, and computed SHA-256 of exact index bytes. | P/F (Node crypto/fs) | Generic OCI content integrity. | `.github/workflows/publish-oci-image.yml:142-181` |
| OP-07 | Require OCI index v1 with exactly Linux amd64+arm64 manifests, valid descriptor digests, and present platform-manifest blobs. | P/F | Generic OCI validation; fixed platforms. | `.github/workflows/publish-oci-image.yml:183-211` |
| OP-08 | Derive lowercase GHCR repository name and exact/minor/major tags from ref; expose image/platform digests and names. | P | Generic version/tag planning with fixed GHCR owner/repo namespace. | `.github/workflows/publish-oci-image.yml:213-232` |
| OP-09 | Mark workflow token secret and authenticate ORAS to `ghcr.io` using actor + token on stdin. | X/R (`oras login`, GHCR token exchange/auth) | Generic GHCR credential use. | `.github/workflows/publish-oci-image.yml:234-250` |
| OP-10 | Resolve exact version tag before any upload; create it only if absent, accept same digest, reject different digest. | X/R/P (`oras resolve`, GHCR) | Generic immutable exact-tag rule. | `.github/workflows/publish-oci-image.yml:251-331` |
| OP-11 | Parse and compare stable versions component-wise with `BigInt`. | P | Generic version ordering, restricted three-component stable form. | `.github/workflows/publish-oci-image.yml:263-305` |
| OP-12 | Plan minor, major, and `latest` channels by resolving tags, fetching current manifest annotation, checking release-line scope, and advancing only to greater versions. | X/R/P (`oras resolve`, `oras manifest fetch`, GHCR) | Generic monotonic channel policy; relies on OCI version annotation. | `.github/workflows/publish-oci-image.yml:333-384` |
| OP-13 | Read local index/platform manifests, deduplicate referenced config/layer digests, and push each blob by digest. | F/X/R (`oras blob push`, GHCR blob upload) | Generic OCI layout publication. | `.github/workflows/publish-oci-image.yml:386-432` |
| OP-14 | Push both platform manifests and the index by expected digest, resolve the digest-addressed image, require equality, and output reference/digest. | F/X/R/P (`oras manifest push`, `oras resolve`, GHCR) | Generic OCI publication. | `.github/workflows/publish-oci-image.yml:433-467` |
| OP-15 | Keylessly sign the digest-pinned index and platform manifests recursively. | X/R (`cosign sign --yes --recursive`; GitHub OIDC/Sigstore/GHCR referrers) | Generic OCI signing. | `.github/workflows/publish-oci-image.yml:469-481` |
| OP-16 | Create provenance attestation for index digest and push it to registry; output URL. | R (`actions/attest`, GitHub attestation/OIDC, GHCR referrer push) | Generic provenance. | `.github/workflows/publish-oci-image.yml:483-490` |
| OP-17 | Create SPDX SBOM attestation for amd64 manifest digest and push it to registry. | R (`actions/attest`, GitHub/GHCR) | Generic per-platform SBOM. | `.github/workflows/publish-oci-image.yml:492-500` |
| OP-18 | Create SPDX SBOM attestation for arm64 manifest digest and push it to registry. | R (`actions/attest`, GitHub/GHCR) | Generic per-platform SBOM. | `.github/workflows/publish-oci-image.yml:502-510` |
| OP-19 | Apply planned tags serially (`--concurrency 1`), verify every resulting tag, and independently require exact tag resolves to expected digest. | X/R/P (`oras tag`, `oras resolve`, GHCR) | Generic verified tag commit; fixed channel policy from OP-12. | `.github/workflows/publish-oci-image.yml:511-563` |
| OP-20 | Always log out of GHCR when publication was enabled, including after earlier failure. | X/R (`oras logout`) | Generic credential cleanup. | `.github/workflows/publish-oci-image.yml:564-569` |

## 3. Language-specific vs generic

The catalog’s Scope column marks every entry. The inherently Go/GoReleaser-specific core is PP-03 through PP-05, PP-07/PP-08, OB-09, and the Go portions of OB-10/OB-11. GoReleaser schema fields (`type=Binary`, `goos`, `goarch`, `path`, `name`), `dist/` path semantics, the two required Linux GOARCH values, and the static ELF handoff are direct dependencies. (`.github/workflows/go-pre-publish.yml:61-119`, `.github/workflows/go-oci-build.yml:178-245`)

Go assumptions are also baked into otherwise generic operations:

- PP-09 selects directory names containing `_linux_amd64` and `_linux_arm64`; PP-10 selects GoReleaser archive/SBOM filename families. (`.github/workflows/go-pre-publish.yml:120-145`)
- OB-10 assumes the canonical program is a single static ELF executable and renames it `application`; OB-20 proves the image contains those exact bytes rather than a rebuild. (`.github/workflows/go-oci-build.yml:210-233`, `.github/workflows/go-oci-build.yml:406-423`)
- The checked-in Melange pipeline performs only `install` of the staged `application`; it does not compile source. (`melange.yaml:1-26`)
- The checked-in apko contract assumes one command, amd64+arm64, UID/GID 65532, and specific OCI annotations. (`apko.yaml:1-33`)
- The release bundle verifier, GitHub release state machine, Actions artifact handoffs, OCI structural validation, registry publication, signatures, attestations, and tag monotonicity are ecosystem-neutral in implementation (GR-01–GR-18 and OP-01–OP-20), but GR-01 fixes GHCR naming and OP-01/OP-08/OP-12 fix the stable `vMAJOR.MINOR.PATCH` and channel policy. (`.github/workflows/publish-github-release.yml:68-482`, `.github/workflows/publish-oci-image.yml:66-569`)
- The current GoReleaser profile itself builds Darwin/Linux/Windows × amd64/arm64 with CGO disabled, tar.gz except Windows ZIP, archive SBOMs, checksum signing, and publication disabled. (`.goreleaser.yaml:13-69`)

## 4. Embedded scripting

Every inline Bash `run: |` block and every `actions/github-script` program is listed, including small programs so the inventory does not hide orchestration code.

| Workflow/lines | Kind | Description | Fragility/concentrated logic |
| --- | --- | --- | --- |
| `go-pre-publish.yml:48-52` | Bash | Tag-ref guard. | Depends on exact Actions environment string and shell error protocol. |
| `go-pre-publish.yml:76-86` | Bash nested in `mise exec` | Verify managed tools; run GoReleaser. | Nested single-quoted shell, PATH/`realpath` comparison, implicit GoReleaser fan-out to Syft/Cosign. |
| `go-pre-publish.yml:91-93` | Bash | Check checksum manifest and bundle presence. | GNU `sha256sum` behavior and relative working directory. |
| `go-pre-publish.yml:97-119` | Bash | Select exactly two canonical Linux binaries. | Multiline jq, process substitution, TSV splitting using literal tab, sort/diff arch equality, GoReleaser JSON schema/path assumptions. |
| `go-oci-build.yml:59-65` | JavaScript | Require tag and `v` prefix. | String-prefix policy rather than full version validation. |
| `go-oci-build.yml:94-104` | Bash nested in `mise exec` | Verify managed build tools. | Nested quoting, PATH/`realpath`, tool output behavior. |
| `go-oci-build.yml:113-141` | JavaScript | Validate artifact metadata. | Numeric coercion/safe-integer check, optional API fields, current-run binding, digest prefix normalization, API retry wrapper. |
| `go-oci-build.yml:156-245` | Bash | Stage configs and canonical binaries; emit metadata. | Destructive temp cleanup, jq error programs, path-prefix confinement, GOARCH↔APK arch mapping, parsing `file` prose, binary-name consistency, `${GITHUB_REF_NAME#v}`, Git timestamp, multiline `$GITHUB_OUTPUT`. |
| `go-oci-build.yml:254-284` | Bash | Compile config, generate key, build and validate signed APK repositories. | CLI argument ordering, Docker runner/QEMU, ephemeral private key lifecycle, globs/`compgen`, exactly-one-package count. |
| `go-oci-build.yml:291-314` | Bash | apko lock/build. | Repository/key path context after `cd`, fixed arch list, annotations encoded as colon-separated CLI arguments. |
| `go-oci-build.yml:323-436` | Bash | Deep OCI/layout/runtime/SBOM verification and digest calculation. | Large jq predicates, descriptor-to-blob path math, digest prefix stripping, architecture mapping, tar listing field parsing, streamed extraction hashing, fixed layer count/user/mode, Melange `-r0` version, exact index-byte digest. |
| `publish-github-release.yml:75-96` | JavaScript | Tag guard and optional digest-pinned GHCR gate. | Boolean inputs arrive as strings; lowercased owner/repo expectation; `@` splitting; strict lowercase digest regex. |
| `publish-github-release.yml:117-119` | JavaScript | Verify gh/Cosign availability. | Delegates through mise argument arrays; checks execution, not parsed version. |
| `publish-github-release.yml:136-164` | JavaScript | Validate release artifact metadata. | Same ID/current-run/digest normalization and API retry concerns as builder. |
| `publish-github-release.yml:175-213` | JavaScript | Poll for draft and bind it to tag SHA. | 24×5-second manual retry loop layered on action retries, paginated tag search, exact draft state, external `git` output trimming, race between release/tag state. |
| `publish-github-release.yml:228-333` | JavaScript | Validate closed release bundle, hashes, and Cosign identity; serialize outputs. | Strict checksum regex/flat filenames, CRLF/newline handling, sync+stream fs mix, symlink rejection, directory closed-set walk, certificate identity interpolation through env, JSON serialization into step outputs. |
| `publish-github-release.yml:352-405` | JavaScript | Revalidate release and upload assets. | JSON-in-env parsing, release-ID coercion, paginated uniqueness check, unexpected-asset rejection, `gh` argument construction, App token injection, `--clobber` convergence. |
| `publish-github-release.yml:417-482` | JavaScript | Poll, verify digests, optionally publish, verify final state. | 12×1-second state polling, remote digest availability, duplicate-name map semantics, expected-vs-actual closed set, inverted draft-state assertion (`release.draft === publish` is failure), final API race. |
| `publish-oci-image.yml:69-75` | JavaScript | Stable tag regex. | Hand-encoded SemVer subset and full-ref matching. |
| `publish-oci-image.yml:95-123` | JavaScript | Validate OCI artifact metadata. | Same API/current-run/digest normalization and retry concerns. |
| `publish-oci-image.yml:138-232` | JavaScript | Validate required files, JSON, triple index digest, platforms; derive outputs. | Raw-byte digest math, recorded-text trimming, optional-property access, descriptor lookup assumptions, fixed filenames/architectures, version split after regex was checked in another step. |
| `publish-oci-image.yml:240-250` | JavaScript | Authenticate ORAS to GHCR. | Secret string handling, stdin newline, actor username, registry-specific login. |
| `publish-oci-image.yml:263-384` | JavaScript | Plan immutable/channel tags. | Interprets ORAS exit code/stderr `: not found`, parses registry JSON, uses `BigInt` ordering, trusts version annotation, checks channel scope, distinguishes equal-version/different-digest corruption, comma-separated output. |
| `publish-oci-image.yml:394-467` | JavaScript | Push local OCI layout by digest and verify resolution. | Local descriptor/blob path math, optional config/layers, dedup set, sequential remote pushes, media types from descriptors, partial-upload recovery boundary, exact digest resolution. |
| `publish-oci-image.yml:475-481` | JavaScript | Recursive keyless Cosign signing. | One CLI flag (`--recursive`) carries index+platform signing semantics; remote identity and referrer behavior is tool-defined. |
| `publish-oci-image.yml:520-563` | JavaScript | Apply planned tags and verify each plus exact tag. | Comma-split output, serial registry mutation, per-tag postcondition, separate exact-tag invariant. |
| `publish-oci-image.yml:568-569` | JavaScript | ORAS logout. | Runs under `always()` but depends on publisher step execution context and ORAS credential-store behavior. |

## 5. External tool versions

### Action and image pins used by the four target workflows

| Action/image | Exact pin | Declared version | Locations |
| --- | --- | --- | --- |
| `actions/checkout` | `9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0` | v7.0.0 | `.github/workflows/go-pre-publish.yml:54-55`; `.github/workflows/publish-github-release.yml:98-99` |
| `actions/checkout` | `3d3c42e5aac5ba805825da76410c181273ba90b1` | v7.0.1 | `.github/workflows/go-oci-build.yml:67-68` |
| `jdx/mise-action` | `3c2e0cf82a5b2e5249f0d3635a4d83d0ae861518` | v4.2.5 | `.github/workflows/go-pre-publish.yml:61-64`; `.github/workflows/go-oci-build.yml:74-77`; `.github/workflows/publish-github-release.yml:105-108`; `.github/workflows/publish-oci-image.yml:77-80` |
| `actions/upload-artifact` | `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` | v7.0.1 | `.github/workflows/go-pre-publish.yml:120-134`; `.github/workflows/go-oci-build.yml:437-439` |
| `actions/download-artifact` | `3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c` | v8.0.1 | `.github/workflows/go-oci-build.yml:143-144`; `.github/workflows/publish-github-release.yml:215-216`; `.github/workflows/publish-oci-image.yml:125-126` |
| `actions/github-script` | `3a2844b7e9c422d3c10d287c895573f7108da1b3` | v9.0.0 | `.github/workflows/go-oci-build.yml:56-57`, `.github/workflows/go-oci-build.yml:106-107`; `.github/workflows/publish-github-release.yml:68-69`, `.github/workflows/publish-github-release.yml:114-115`, `.github/workflows/publish-github-release.yml:129-130`, `.github/workflows/publish-github-release.yml:166-168`, `.github/workflows/publish-github-release.yml:222-224`, `.github/workflows/publish-github-release.yml:341-342`, `.github/workflows/publish-github-release.yml:407-409`; `.github/workflows/publish-oci-image.yml:66-67`, `.github/workflows/publish-oci-image.yml:88-89`, `.github/workflows/publish-oci-image.yml:132-134`, `.github/workflows/publish-oci-image.yml:234-236`, `.github/workflows/publish-oci-image.yml:251-254`, `.github/workflows/publish-oci-image.yml:386-389`, `.github/workflows/publish-oci-image.yml:469-471`, `.github/workflows/publish-oci-image.yml:511-513`, `.github/workflows/publish-oci-image.yml:564-566` |
| `docker/setup-qemu-action` | `29109295f81e9208d7d86ff1c6c12d2833863392` | v3.6.0 | `.github/workflows/go-oci-build.yml:86-89` |
| `tonistiigi/binfmt` image | `sha256:400a4873b838d1b89194d982c45e5fb3cda4593fbfd7e08a02e76b03b21166f0` | no mutable tag/version used | `.github/workflows/go-oci-build.yml:86-90` |
| `actions/create-github-app-token` | `bcd2ba49218906704ab6c1aa796996da409d3eb1` | v3.2.0 | `.github/workflows/publish-github-release.yml:121-127` |
| `actions/attest` | `1e69f48acb82d1966a394da916b4c1698aa569d6` | v4.2.2 | `.github/workflows/publish-github-release.yml:335-339`; `.github/workflows/publish-oci-image.yml:483-510` |

### CLI/tool pins installed by the target workflows

| Tool | Pin and pin location | Installed/used by |
| --- | --- | --- |
| mise | `2026.8.8` is the action input in each target workflow. (`.github/workflows/go-pre-publish.yml:61-72`, `.github/workflows/go-oci-build.yml:74-85`, `.github/workflows/publish-github-release.yml:105-112`, `.github/workflows/publish-oci-image.yml:77-87`) | All four. |
| Go | `1.26.6` in repository `mise.toml`; installed by producer `install_args: go`. (`mise.toml:1-2`, `.github/workflows/go-pre-publish.yml:61-72`) | Producer. |
| GoReleaser | `2.17.1` in `mise.toml`. (`mise.toml:5`, `.github/workflows/go-pre-publish.yml:61-72`) | Producer. |
| Syft | `1.51.0` in `mise.toml`. (`mise.toml:7`, `.github/workflows/go-pre-publish.yml:61-72`) | Producer/GoReleaser SBOM generation. |
| Cosign | `3.1.3` in `mise.toml`; OCI publisher independently repeats `3.1.3` in inline `mise_toml`. (`mise.toml:12`, `.github/workflows/go-pre-publish.yml:61-72`, `.github/workflows/publish-github-release.yml:105-112`, `.github/workflows/publish-oci-image.yml:77-87`) | Producer, GitHub publisher, OCI publisher. |
| GitHub CLI (`gh`) | `2.97.0` in `mise.toml`. (`mise.toml:6`, `.github/workflows/go-oci-build.yml:74-85`, `.github/workflows/publish-github-release.yml:105-112`) | Builder installs/verifies it; GitHub publisher uploads release assets with it. |
| Melange | `0.59.1` in `mise.toml`. (`mise.toml:10`, `.github/workflows/go-oci-build.yml:74-85`) | OCI builder. |
| apko | `1.2.37` in `mise.toml`. (`mise.toml:11`, `.github/workflows/go-oci-build.yml:74-85`) | OCI builder. |
| ORAS | `1.3.3` in OCI publisher inline `mise_toml`. (`.github/workflows/publish-oci-image.yml:77-87`) | OCI publisher. |

The four workflows also invoke runner-provided `bash`, `git`, `jq`, GNU `sha256sum`, `realpath`, `file`, `install`, `tar`, `cut`, `sort`, `diff`, and Docker without installing or version-pinning them in these workflows. (`.github/workflows/go-pre-publish.yml:46-119`, `.github/workflows/go-oci-build.yml:92-436`) This is a negative finding: no exact versions for those binaries were found in the four target YAML files or `mise.toml`. (`mise.toml:1-18`)

### Caller/context action pins

The requested context workflows additionally pin `googleapis/release-please-action` v5.0.0 at `45996ed1f6d02564a971a2fa1b5860e934307cf7`, `actions/cache` v6.0.0 at `2c8a9bd7457de244a408f35966fab2fb45fda9c8`, and reuse the same App-token, checkout, and mise-action pins above. (`.github/workflows/release-please.yml:26-39`, `.github/workflows/go-ci.yml:17-58`) `ci.yml` and `release.yml` call local reusable workflows rather than external reusable workflows. (`.github/workflows/ci.yml:16-19`, `.github/workflows/release.yml:14-73`)

## 6. Invariants and contracts

1. **Draft is upstream state, not created by the publisher.** Release Please is configured to force-create a `v` tag and matching draft; the publisher only searches for that exact draft and fails if absent or non-draft. (`release-please-config.json:3-10`, `.github/workflows/publish-github-release.yml:166-213`)
2. **Tag/run binding.** Producer and builders require tag refs; OCI publication requires stable `vMAJOR.MINOR.PATCH`; GitHub publication requires a tag, and the selected tag must resolve to `github.sha`. (`.github/workflows/go-pre-publish.yml:46-52`, `.github/workflows/go-oci-build.yml:56-65`, `.github/workflows/publish-oci-image.yml:66-75`, `.github/workflows/publish-github-release.yml:68-96`, `.github/workflows/publish-github-release.yml:200-213`)
3. **Job gate is registry-before-release.** `github-release` cannot run until `oci-publish` succeeds and passes its digest-pinned `image-reference`; when `require-oci-image=true`, public release requires the caller repository’s exact GHCR digest reference. (`.github/workflows/release.yml:32-73`, `.github/workflows/publish-github-release.yml:68-96`)
4. **Each handoff is same-run and two-coordinate where applicable.** IDs must be positive, artifacts unexpired, `workflow_run.id=context.runId`, caller transport digest equal GitHub metadata, and download digest verification must pass. OCI additionally requires recorded, recomputed, and caller index digests to agree. (`.github/workflows/go-oci-build.yml:106-149`, `.github/workflows/publish-github-release.yml:129-220`, `.github/workflows/publish-oci-image.yml:88-181`)
5. **Build once/no rebuild.** The OCI path consumes the exact GoReleaser Linux binaries, packages them without source compilation, then extracts the final image executable and requires byte-for-byte SHA-256 equality with each canonical input. (`.github/workflows/go-oci-build.yml:178-233`, `.github/workflows/go-oci-build.yml:247-314`, `.github/workflows/go-oci-build.yml:406-423`; `melange.yaml:24-26`)
6. **Canonical platform/runtime shape.** Exactly Linux amd64 and arm64; static 64-bit ELF input; one package per APK arch; two OCI manifests; one layer each; configured single entrypoint; user 65532; executable owner `0/0`, mode 0755; required OCI labels/annotations; per-arch SPDX includes application `${VERSION}-r0`. (`.github/workflows/go-oci-build.yml:178-233`, `.github/workflows/go-oci-build.yml:281-284`, `.github/workflows/go-oci-build.yml:315-436`)
7. **Release bundle is signed and closed.** `checksums.txt` is nonempty, has strict flat unique filenames, excludes its two control files, hashes every payload, and permits no other directory entry; its Sigstore bundle must verify the exact reusable-workflow identity and GitHub OIDC issuer. (`.github/workflows/publish-github-release.yml:222-333`)
8. **Attestation subject shape.** GitHub release provenance uses `subject-checksums: dist/checksums.txt`, so checksummed archives/SBOMs are subjects while the checksum and bundle are controls. OCI creates one provenance attestation for the index and one SPDX SBOM attestation per platform digest, all with `push-to-registry:true`. (`.github/workflows/publish-github-release.yml:335-339`; `.github/workflows/publish-oci-image.yml:483-510`; `docs/reference/github-release-contract.md:289-291`)
9. **Draft-until-verified.** Release asset upload occurs only after local checksum/signature validation and attestation; publication occurs only after the remote asset set has exact count, unique names, uploaded states, and exact GitHub SHA-256 digests. A disabled publication input must leave the release draft. (`.github/workflows/publish-github-release.yml:222-339`, `.github/workflows/publish-github-release.yml:341-482`)
10. **Recovery converges only over expected names.** Existing expected assets may be replaced with `--clobber`; any unexpected asset blocks upload, and the workflow does not delete it. (`.github/workflows/publish-github-release.yml:341-405`; `docs/reference/github-release-contract.md:329-331`)
11. **Exact OCI tags are immutable before write.** Before the first candidate blob upload, absent exact tag is planned, same-digest exact tag is accepted, and different-digest exact tag is fatal. (`.github/workflows/publish-oci-image.yml:251-331`)
12. **Channel tags are monotonic.** `MAJOR.MINOR`, `MAJOR`, and `latest` advance only when the candidate version is greater; minor/major tags must already point within their release line; a newer current version is retained; equal version with different digest is fatal. (`.github/workflows/publish-oci-image.yml:263-384`)
13. **Channel mutation is serialized.** Repository-wide concurrency prevents competing release runs from planning/updating shared channels concurrently; ORAS applies a run’s tag set with concurrency one. (`.github/workflows/publish-oci-image.yml:43-50`, `.github/workflows/publish-oci-image.yml:511-563`)
14. **Trust metadata precedes public tags.** The publisher plans tags, pushes all content by digest, verifies digest resolution, recursively signs index+platform manifests, creates provenance and both SBOM attestations, and only then mutates public tags. (`.github/workflows/publish-oci-image.yml:251-510`, `.github/workflows/publish-oci-image.yml:511-563`; `docs/reference/oci-image-contract.md:232-240`)
15. **Verification-only mode writes neither registry nor OCI trust metadata.** Login, planning, push, signing, attestations, tags, and logout are all gated by `inputs.publish-image`; artifact/content validation still runs and returns image name/digest. (`.github/workflows/publish-oci-image.yml:132-232`, `.github/workflows/publish-oci-image.yml:234-569`)
16. **Release and registry concurrency do not cancel in-flight publication.** Caller serialization is workflow+ref with `cancel-in-progress:false`; OCI publisher is repository-wide with the same no-cancel policy. (`.github/workflows/release.yml:8-12`, `.github/workflows/publish-oci-image.yml:48-50`)
17. **Producer publication is disabled twice.** GoReleaser config has `release.disable:true`, while command invocation also passes `--skip=publish`; changelog generation is disabled because Release Please owns release notes/draft. (`.goreleaser.yaml:54-69`, `.github/workflows/go-pre-publish.yml:74-86`)

## 7. Identity and credentials

| Credential/identity | Job(s) | Source and allowed use |
| --- | --- | --- |
| Job `GITHUB_TOKEN`, `contents: read` | `go-pre-publish/release-assets` | Checkout repository/tag history only; no release/package write. (`.github/workflows/go-pre-publish.yml:25-44`, `.github/workflows/go-pre-publish.yml:54-59`) |
| GitHub OIDC job token capability, `id-token: write` | `go-pre-publish/release-assets` | Enables keyless Cosign signing invoked by GoReleaser; checksum signature is later constrained to the reusable producer identity and issuer `https://token.actions.githubusercontent.com`. (`.github/workflows/go-pre-publish.yml:39-44`, `.goreleaser.yaml:54-63`, `.github/workflows/publish-github-release.yml:309-322`) |
| Job `GITHUB_TOKEN`, `actions: read`, `contents: read` | `go-oci-build/oci-image` | Read artifact metadata/download and checkout consumer config/history. It has no `packages`, `attestations`, `id-token`, or release App credential. (`.github/workflows/go-oci-build.yml:38-55`, `.github/workflows/go-oci-build.yml:67-73`, `.github/workflows/go-oci-build.yml:106-149`; `docs/reference/oci-image-contract.md:58`) |
| `release-app-client-id` input + `release-app-private-key` secret | `publish-github-release/publish` | Passed to `actions/create-github-app-token` to mint a short-lived installation token requesting only `permission-contents: write`. (`.github/workflows/publish-github-release.yml:18-21`, `.github/workflows/publish-github-release.yml:37-40`, `.github/workflows/publish-github-release.yml:121-128`) |
| Release App installation token (`steps.release-app.outputs.token`) | `publish-github-release/publish` | Used through a dedicated Octokit client to list/select/update the draft and list assets; injected as `GH_TOKEN` only for `gh release upload`. It mutates release contents/state, while the job token remains `contents: read`. (`.github/workflows/publish-github-release.yml:166-213`, `.github/workflows/publish-github-release.yml:341-405`, `.github/workflows/publish-github-release.yml:407-482`) |
| Publisher job `GITHUB_TOKEN`: `actions: read`, `artifact-metadata: write`, `attestations: write`, `contents: read`, `id-token: write` | `publish-github-release/publish` | Reads/downloads Actions artifact and creates GitHub build-provenance attestations with OIDC; it does not receive `contents: write`. (`.github/workflows/publish-github-release.yml:49-64`, `.github/workflows/publish-github-release.yml:129-164`, `.github/workflows/publish-github-release.yml:335-339`) |
| `checksum-signing-workflow-ref` trust identity | `publish-github-release/publish` verifier | Converted to `https://github.com/<input>` and supplied to `cosign verify-blob` together with issuer `https://token.actions.githubusercontent.com`; the dogfood caller sets `<repository>/.github/workflows/go-pre-publish.yml@<tag ref>`. (`.github/workflows/publish-github-release.yml:14-17`, `.github/workflows/publish-github-release.yml:222-322`, `.github/workflows/release.yml:64-70`) |
| OCI publisher `GITHUB_TOKEN` with `packages: write` | `publish-oci-image/publish` | Explicitly passed as `GHCR_TOKEN=${{ github.token }}` to `oras login ghcr.io`; subsequent ORAS pushes/tags and Cosign/registry attestation referrers use the authenticated GHCR session. No long-lived registry secret is declared. (`.github/workflows/publish-oci-image.yml:43-57`, `.github/workflows/publish-oci-image.yml:234-250`, `.github/workflows/publish-oci-image.yml:386-569`) |
| OCI publisher `id-token: write`, `attestations: write`, `artifact-metadata: write` | `publish-oci-image/publish` | Enables keyless recursive Cosign signature and GitHub provenance/SBOM attestations, including registry pushes. (`.github/workflows/publish-oci-image.yml:51-57`, `.github/workflows/publish-oci-image.yml:469-510`) |
| OCI publisher `actions: read`, `contents: read` | `publish-oci-image/publish` | `actions: read` supports artifact metadata/download. `contents: read` is granted, but the workflow does not check out source or explicitly pass the token to a contents REST mutation. (`.github/workflows/publish-oci-image.yml:51-57`, `.github/workflows/publish-oci-image.yml:88-130`) |
| Organization variable `MEIGMA_RELEASE_APP_CLIENT_ID` and secret `MEIGMA_RELEASE_APP_PRIVATE_KEY` | Dogfood `release.yml` GitHub publisher and `release-please.yml` | Caller maps them into the reusable publisher’s input/secret; Release Please uses the same pair to mint the App token that manages release PR/tag/draft state. (`.github/workflows/release.yml:64-73`, `.github/workflows/release-please.yml:1-4`, `.github/workflows/release-please.yml:26-39`) |

## Unknowns and negative findings

- Exact Sigstore Fulcio/Rekor service URLs are not configured in the requested files; only the GitHub OIDC issuer and expected certificate identity are explicit. Cosign therefore uses its pinned tool defaults; concrete service endpoints are **UNVERIFIED** from repository configuration. (`.goreleaser.yaml:54-63`, `.github/workflows/publish-github-release.yml:309-322`, `.github/workflows/publish-oci-image.yml:469-481`)
- The exact GitHub-hosted runner image revision and versions of runner-provided shell/core utilities and Docker are not pinned in the requested files; only `runs-on: ubuntu-24.04` and the listed action/tool pins are present. (`.github/workflows/go-pre-publish.yml:28-44`, `.github/workflows/go-oci-build.yml:41-55`, `.github/workflows/publish-github-release.yml:52-67`, `.github/workflows/publish-oci-image.yml:44-65`)
- No custom registry host, namespace, registry credential secret, alternate architecture, prerelease tag, multiple executable, or source rebuild path was found. GHCR name, Linux amd64/arm64, one staged executable, stable tags, and GITHUB_TOKEN registry authentication are fixed in the current implementation. (`.github/workflows/go-oci-build.yml:178-233`, `.github/workflows/go-oci-build.yml:285-436`, `.github/workflows/publish-oci-image.yml:66-75`, `.github/workflows/publish-oci-image.yml:218-250`)

