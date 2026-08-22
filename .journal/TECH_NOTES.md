# Technical Notes

## Current state

- `main` is at `2d524ae`. The current release remains `v0.1.16` at
  `583937edadfbae183e49f16df46b98e0b36807ba`; release PR 64 is open for
  `v0.1.17`, the first released unit containing the cross-organization policy
  cutover.
- `v0.1.16` completed release asset build, OCI publication, public GitHub
  Release publication, Homebrew and Scoop publication, native RPM/APK signing,
  and automatic package-repository dispatch.
- `meigma/pkgs` is the serialized production receiver. `pkgs.meigma.dev` serves
  signed APT, RPM, and APK repositories containing `v0.1.9` and `v0.1.16`.
  Production publication and unchanged replay passed exact public installs in
  pinned Debian, Fedora, and Alpine clients.
- A release contains six platform archives, six native Linux packages, twelve
  SBOMs, `checksums.txt`, and its Sigstore bundle: 26 GitHub Release assets.
  GHCR receives the signed multi-platform image and semantic channel tags.
- Reusable workflows own orchestration, artifact transport, credential
  boundaries, and control-file isolation. Domain engines in `release-cli` own
  repository reconciliation and every publication state transition.
- Self-release jobs build `release-cli` from the exact reusable-workflow source
  SHA with exact-key Go caches. External workflow consumers install the
  checksum- and attestation-verified release.
- Direct users can install with mise's native GitHub backend or build the tagged
  source with the repository Nix flake.
- External organizations can use every supported publisher with adopter-owned
  credentials and destinations. The maintained documentation is one tutorial,
  six how-to guides, two references, and one architecture explanation; the
  repository is licensed under Apache-2.0 OR MIT.

## Where the design lives

- `.journal/002/ARCHITECTURE.md` (revision 3) is the approved CLI architecture;
  `.journal/002/PLAN.md` contains the completed eleven-PR program and its spike
  results.
- `.journal/003/SUMMARY.md` covers slices 3-4b; `.journal/004/SUMMARY.md` covers
  slices 5a-6; `.journal/005/SUMMARY.md` covers shared subprocess execution and
  cached source-build acquisition; `.journal/006/SUMMARY.md` covers the first
  production releases, native packages, mise, and Nix support;
  `.journal/007/SUMMARY.md` covers Homebrew and Scoop delivery;
  `.journal/008/SUMMARY.md` covers native package repository research,
  disposable proofs, and production slices 1 through 4; and
  `.journal/009/SUMMARY.md` covers production provisioning, hardening,
  automatic dispatch, public installation, and cutover.
- `.journal/010/SUMMARY.md` covers the cross-organization adoption review,
  explicit shared-workflow signer policy, dual licensing, and documentation
  consolidation.

## Binding contracts

- **One release unit.** Workflows, composite action, and CLI share one version and
  one consumer pin. Reusable workflows expose neither `cli-version` nor
  `cli-path`. The setup action retains `cli-path` only as an unsupported direct
  escape hatch: the caller owns the pairing and a stamp mismatch warns.
- **Protocol stamp.** `cli.Protocol` is a Go source constant, not an ldflag.
  Release-please stamps only the version, through the annotated `generic`
  extra-file updater. `scripts/check-protocol-stamp.sh` guards both literals.
- **CLI surface.** Exit codes are only `0`, `1`, `2`. Under `--json`, stdout
  carries exactly one `release.dev/result/v1` envelope, including failures; a
  flag-parse failure yields usage on stderr with no envelope. Side-effecting
  commands are silent on stdout without `--json`. Every boolean `RELEASE_*`
  variable must parse with `strconv.ParseBool` or the command exits 2 — and every
  command must check the settings resolver error before doing anything.
- **Commands:** `stage --profile go`, `verify handoff`, `verify bundle`,
  `plan tags`, `publish oci prepare [--dry-run]`, `publish oci finalize --result -`,
  `publish github [--no-undraft]`, `publish homebrew`, `publish scoop`,
  `publish package-repository`, `init homebrew-tap`, `init scoop-bucket`,
  `image build`, `image verify`, `version`.
- **Ports.** Interfaces remain narrow and domain-owned. Homebrew adds
  `pubbrew.RepositoryReader`/`RepositoryWriter` through `ghtap`; Scoop adds
  `pubscoop.RepositoryReader`/`RepositoryWriter` through `ghbucket`. The
  package channels share neutral GitHub value types and execution seams, not a
  generalized package-manager publisher. `cli.Actions` (`actenv`) remains unbuilt.
- **Package-manager controls.** GoReleaser renders the Homebrew cask and Scoop
  manifest with `skip_upload: true`. They travel in the authoritative Actions
  artifact but are removed from signed-bundle verification, attestations, and
  public GitHub Release assets. Each publisher isolates and restores only its
  own control file before invoking the CLI.
- **Reviewed package publication.** Homebrew and Scoop run independently only
  after the public GitHub Release succeeds. Each mints a short-lived Release App
  token scoped to one destination repository, then converges a deterministic
  branch and pull request without merging or enabling auto-merge.
- **Bucket and tap CI.** Destination repositories call secret-free reusable
  workflows pinned by full source commit. Homebrew validates casks on macOS and
  Linux. Scoop keeps manifests at the root, enforces CRLF checkouts, and runs
  pinned official schema and lifecycle checks on native Windows AMD64 and ARM64.
- **Local initializers.** `init homebrew-tap` and `init scoop-bucket` write
  deterministic scaffolds into absent or empty directories through an atomic
  directory install. They perform no Git, GitHub, ruleset, App, or credential
  operation; operator guides cover those boundaries.
- **Subprocess execution.** `internal/execx` is the only production `os/exec`
  boundary. It owns deferred binary lookup, process construction, output routing,
  a 4 KiB stderr tail, five-second `WaitDelay`, and typed exit metadata. Tool
  adapters retain argv, environment policy, secret handling, parsing, domain
  errors, and retry decisions. `execx` itself runs exactly one attempt.
- **CLI acquisition.** `setup-release-cli` accepts `local-build: auto|always|never`.
  `auto` builds source only for a matching self-release; `always` still requires
  the action and reusable workflow to come from the same repository. Source builds
  use the runner-provided workflow SHA for checkout, stamps, and exact OS,
  architecture, Go-version, and source-SHA `GOCACHE`/`GOMODCACHE` keys. `never`
  installs the stamped release and verifies its checksum and GitHub attestation.
- **Native packages.** GoReleaser nFPM repackages each canonical Linux binary as
  DEB, RPM, and APK without rebuilding it. Opt-in producer RPM and APK signing
  runs before checksums and attestations, so the release trust chain and native
  package managers authenticate the same bytes.
- **Static package repository.** `publish package-repository` verifies the exact
  public GitHub Release, checksum identity, attestations, package metadata,
  producer signatures, and existing R2 object digests. It regenerates complete
  APT, RPM, and APK trees, verifies local installs, uploads immutable objects
  before signed mutable roots, then verifies public installs.
- **Package-repository signer policy.** Each producer declares
  `checksum_identity` as the exact GitHub certificate identity including one
  full workflow commit SHA, and `attestation_signer` as the exact reusable
  publisher workflow without a revision. These signer repositories may differ
  from the producer repository; the removed producer-relative fields are
  rejected by strict YAML decoding.
- **Repository ownership.** `meigma/pkgs` is the sole serialized writer and
  holds aggregate signing and R2 credentials in its protected
  `packages-production` environment. Producers mint a short-lived release App
  token scoped to the receiver and dispatch only `{repository, tag}` after
  public release. Git stores reviewed policy and versioned public keys, not
  packages or generated repository state.
- **Direct mise installation.** `mise use github:meigma/release@<version>` uses
  mise's built-in GitHub backend. `release-cli` has no shared mise registry
  entry; a short name requires a consumer-local `[tool_alias]`.
- **Nix distribution.** The root flake builds the exact source revision with
  CGO disabled and exposes package, app, and check outputs for Darwin and Linux
  on `arm64` and `amd64`. Nixpkgs 26.05 is pinned because it is the last release
  supporting `x86_64-darwin`; Go 1.26.6 and the module closure use fixed hashes.
  The Nix path does not install or attest the prebuilt GitHub Release archive.
- **`actenv` is deliberately deferred.** Workflows read the `--json` envelope from
  stdout with a `$GITHUB_OUTPUT` heredoc and `jq`. Build the seam only when a
  command needs annotations or a job summary.
- **`stage --profile go` builds.** It runs `goreleaser release --clean
  --skip=publish`, then validates checksums, bundle, artifacts, and binaries, then
  writes `dist/oci-build-inputs.json`. `--clean` deletes the distribution
  directory, and `--dist` must be a basename because GoReleaser has no `--dist`
  flag and writes relative to the working directory.
- **The staged projection is the build handoff.** `release.dev/oci-build-inputs/v1`
  carries profile plus, per Linux platform, binary name, artifact-relative path,
  and canonical SHA-256. `image build` verifies those digests against the
  downloaded bytes before packaging.
- **`image build` is fail-closed on its workspace.** Both `--work` and `--output`
  must be empty or absent, and they must be disjoint: the ephemeral APK signing
  key lives in `--work` and `--output` is uploaded.
- **`image verify` is the independent oracle.** The index digest is SHA-256 over
  the exact `index.json` bytes, never re-marshaled JSON. The entrypoint's tar entry
  must be a regular file whose low twelve mode bits are exactly `0755` (so no
  setuid, setgid, or sticky) owned `0:0`, with content byte-identical to the
  canonical staged binary. It writes `image-digest.txt`, which the publisher reads.
- **Two-phase OCI publication.** `publish oci prepare` -> YAML `actions/attest` ->
  `publish oci finalize`, the prepare result carried as the `--json` envelope on
  stdin. Invariant 14 (trust metadata before public tags) depends on the split.
- **Release publication.** The CLI never creates a release, never re-drafts one,
  never deletes an asset, and never mints an App token; the workflow mints it and
  passes `RELEASE_APP_TOKEN`, held as a redacted `rel.Secret`.
- **Retry.** Bounded retry lives in the `pubgh`/`puboci` engines with an injected
  `SleepFunc`: four attempts at 1s/2s/4s. Poll budgets are engine-owned: 24
  attempts 5s apart for draft discovery, 12 attempts 1s apart for asset
  convergence. **Local build tools are never retried** — Melange, apko, and
  GoReleaser are stateful and non-idempotent.
- **Escape hatches.** `--plain-http` is flag-only and refused for non-loopback
  hosts. `RELEASE_COSIGN_PATH`, `RELEASE_GH_PATH`, `RELEASE_GIT_PATH`,
  `RELEASE_MELANGE_PATH`, `RELEASE_APKO_PATH`, and `RELEASE_GORELEASER_PATH`
  locate pinned binaries; the workflows disable mise shims, so they resolve paths
  with `mise which`.
- **The producer runs the CLI under `mise exec`.** GoReleaser shells out to `go`,
  `syft`, and `cosign`; an explicit binary path alone would hand the build a
  different Go toolchain.

## Platform facts learned the hard way

- **Writing an executable fixture inside a parallel test and exec'ing it races on
  Linux.** A sibling's `fork` inherits the open write descriptor and the exec fails
  with `ETXTBSY`. Every exec-adapter test now writes its fake once in `TestMain`.
  macOS never reproduces it; a containerized `go test ./...` reproduces it
  immediately.
- **`archive/tar` normalizes the historic NUL typeflag** to `'0'` before `Next()`
  returns, so `tar.TypeRegA` never reaches a header check. Comparing
  `FileInfo().Mode().Perm()` silently accepts setuid, setgid, and sticky bits.
- **`goreleaser release` has no `--dist` flag** and `gomod.proxy: true` resolves
  `module@current-tag` through the module proxy, so a full local build with an
  unpublished tag is impossible. GoReleaser also colorizes into a pipe, so any
  captured tail needs ANSI stripping before it enters a JSON envelope.
- **A callee can never exceed its caller's permission ceiling**, the run dies at
  `startup_failure` with no API-visible diagnostic, and PR CI cannot catch it when
  the caller only runs on tags. Audit this repo's `release.yml`, the copyable
  example, and the documented skeleton together.
- **Linux package tools traverse and write across user boundaries.** Restore the
  host owner after root-owned metadata generation, and make only the generated
  public output tree traversable to unprivileged verification helpers. Keep
  source and work roots private.
- **Minimal Debian images need CA certificates before an HTTPS APT update.**
  Bootstrap `ca-certificates` before replacing sources with
  `https://pkgs.meigma.dev`; otherwise TLS failure masks repository behavior.
- **A same-version replay can mask an incomplete historical-package
  classifier.** Canonical RPM objects live under lowercase `packages/`. Prove
  historical mirroring with a later release whose incoming assets cannot fill
  an omitted old format.
- **oras hands the content reader to `net/http`, which always closes a request
  body**, so a caller-owned `*os.File` gets closed twice; `fstest.MapFS` cannot
  catch it because its `Close` is idempotent. A streamed body has no `GetBody`, so
  retry must live where content can be reopened.
- **HTTP 409 on a blob push is not "already exists"**; only
  `errdef.ErrAlreadyExists` is.
- **An image's `org.opencontainers.image.version` annotation must equal the release
  version**, or channel planning refuses.
- **Registry login is not obsolete:** `cosign` and `actions/attest
  --push-to-registry` read the docker config. `cosign` has no `logout`, so cleanup
  is a docker-config edit.
- **Tags created with `GITHUB_TOKEN` do not trigger tag-push workflows**, so
  `release-please.yml` must keep minting an App token.
- **`$/` self-repository references work** for external SHA-pinned callers on
  runner >= 2.336.0; `github.action_repository` is empty for local-path
  invocations, so keep the `${GITHUB_REPOSITORY}` fallback.
- **`actions/attest` validates SBOM payloads** and writes durable, unwithdrawable
  state, so cheap gates belong before it.
- **`golangci-lint` caches per path.** After removing a worktree it stops excluding
  generated files and flags every mock; run `golangci-lint cache clean`.
- **`gh pr merge --delete-branch` fails its local step** when `main` is checked out
  in another worktree. The merge still succeeds; finish with `git fetch --prune`,
  `git pull --ff-only`, `wt remove <branch> --force`, and
  `git push origin --delete <branch>`.
- **Adding a repo to the release App's installation requires an organization
  owner**; an `admin:org` token gets 403. Same for a consumer's first public GHCR
  package.

## How to verify locally

- **Image work:** build real static Go Linux binaries, hand-write the
  `oci-build-inputs` projection or produce it with `stage`, then run `image build`
  with pinned Melange/apko and Docker; `image verify` then checks the result. Both
  ran end to end on an M4 laptop in seconds.
- **Cross-checking a verifier being deleted:** run the old shell script inside
  `ubuntu:24.04` (GNU tar 1.35 matters — BSD `tar -tvf` prints ownership
  differently) and require the same `image-digest`.
- **Negative image evidence:** rebuild a layer with one flipped byte, or with a
  `04755` entrypoint, regenerating manifest and index descriptors so every digest
  and size stays consistent. Anything less proves only the size check.
- **Registry work:** a `go-containerregistry` `registry.New()` server plus a
  handmade two-platform layout, driven with `--plain-http` against `127.0.0.1`.
- **Real GHCR:** scratch packages under `ghcr.io/meigma/<name>`; the token has
  `delete:packages`, so clean up with
  `gh api -X DELETE /orgs/meigma/packages/container/<name>`.
- **Release work:** a temporary draft release for a throwaway tag, then
  `gh release delete <tag> --yes --cleanup-tag`.
- **Suite stability:** run `docker run --rm --cpus 4 -v <worktree>:/src -w /src -e
  GOFLAGS=-mod=mod golang:1.26 go test ./... -count=1` before trusting a green
  macOS suite.
- **Package repository work:** generate signed packages with distinct producer
  and aggregate keys, then exercise local and public APT, DNF, and APK installs.
  R2 proofs must cover first publication, unchanged replay, immutable conflict,
  interruption before roots, resume, GitHub/R2 byte identity, and cache
  behavior. `.journal/008/SUMMARY.md` records the disposable evidence;
  `.journal/009/SUMMARY.md` records the production proof.

## Credentials

- Release mutation uses org variable `MEIGMA_RELEASE_APP_CLIENT_ID` and org secret
  `MEIGMA_RELEASE_APP_PRIVATE_KEY`, both scoped to selected repositories. Adopting
  organizations change only these plus the App installation.
- Package repository production requires a protected `packages-production`
  environment in `meigma/pkgs` containing bucket-scoped R2 credentials and
  aggregate OpenPGP/APK private keys. Producer signing keys remain in producer
  release environments; producers never receive R2 or aggregate keys.

## Open work and housekeeping

- Homebrew and Scoop are implemented through protected repository pull
  requests. Production `v0.1.6` and `v0.1.7` opened `meigma/homebrew-tap` PRs
  7/8 and `meigma/scoop-bucket` PRs 2/3; they remain open intentionally for
  human review.
- MacPorts and a generalized installer remain deferred.
- A future Nixpkgs update must deliberately retain the final
  `x86_64-darwin`-capable pin or drop that system.
- Archived but undeletable without `delete_repo`: `meigma/release-selfref-spike`,
  `meigma/release-oras-spike`, `meigma/release-stamp-spike`,
  `meigma/release-oci-remediation-e2e`, and
  `meigma/scoop-publisher-rehearsal`. Also pending: the `spike/self-ref` branch,
  the `ghcr.io/meigma/release-oras-spike` package, and the dead
  `release-please--branches--main--components--release-mvp` branch.
- Mockery's testify template emits no Godoc for generated expecter types.
