# Technical Notes

## Current state

- `main` is at `7197ca2`. The released artifact is `release-cli`; `v0.1.0` is the
  only published release (six archives, six SBOMs, `checksums.txt` plus its
  Sigstore bundle, and a GHCR image tagged `latest`, `0`, `0.1`, `0.1.0`).
- The eleven-PR program in `.journal/002/PLAN.md` has **PRs 1 through 10 merged**.
  Only **PR 11**, the documentation pin, remains, and it is gated on a released
  and verified build of `7197ca2`.
- **Every reusable workflow is now a thin shell:** tag gate, checkout, mise
  install, managed-tool-path proof, setup action, `release-cli` invocations,
  artifact transport. No workflow contains staging, build, verification, or
  publication logic. `go-oci-build.yml` shed 271 lines of shell in PRs 8 and 9.
- **Nothing has been released since the CLI took over the build.** The next tag is
  the first run of the complete path: GoReleaser inside `stage`, both image
  commands, and the two publication workflows.

## Where the design lives

- `.journal/002/ARCHITECTURE.md` (revision 3) is the approved architecture;
  `.journal/002/PLAN.md` holds the eleven-PR plan, the **standing per-PR execution
  method**, and the three spike results. Read both before continuing.
- `.journal/003/SUMMARY.md` covers slices 3-4b; `.journal/004/SUMMARY.md` covers
  slices 5a-6 and the lessons from them.

## Binding contracts

- **One release unit.** Workflows, composite action, and CLI share one version and
  one consumer pin. No `cli-version` input. `cli-path` is an unsupported escape
  hatch: the installed path fails closed on a stamp mismatch, `cli-path` warns.
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
  `publish github [--no-undraft]`, `image build`, `image verify`, `version`.
- **Ports.** Budget closed at 13; twelve exist. `pubgh.ArtifactMeta` (ghact),
  `puboci.StateReader`/`ContentPusher`/`TagCommitter` (reg), `puboci.Signer` and
  `pubgh.BlobVerifier` (cosign), `pubgh.ReleaseReader`/`Publisher` (ghrel),
  `pubgh.AssetReplacer` (ghup), `pubgh.RefResolver` (gitx), `image.APKBuilder`
  (melange), `image.Composer` (apko). Unbuilt: `cli.Actions` (actenv).
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

## Credentials

- Release mutation uses org variable `MEIGMA_RELEASE_APP_CLIENT_ID` and org secret
  `MEIGMA_RELEASE_APP_PRIVATE_KEY`, both scoped to selected repositories. Adopting
  organizations change only these plus the App installation.

## Housekeeping debt

- Archived but undeletable without `delete_repo`: `meigma/release-selfref-spike`,
  `meigma/release-oras-spike`, `meigma/release-stamp-spike`, and
  `meigma/release-oci-remediation-e2e`. Also pending: the `spike/self-ref` branch,
  the `ghcr.io/meigma/release-oras-spike` package, and the dead
  `release-please--branches--main--components--release-mvp` branch.
- Consumer docs and `examples/go-release/` still pin the pre-squash revision
  `fb8c8098ff27968fb3070e928c00e925f38c698e` and carry `FULL_SHA` placeholders.
  PR 11 replaces both.
- Release-please PR #9 (`chore(main): release 0.1.1`) is open and its version is
  stale now that PRs 8-10 added features.
- The bounded stderr-tail exec helper is duplicated in `cosign`, `melange`, `apko`,
  and `goprof`. The plan forbids `execx`; conformance recommends amending the
  architecture and extracting it.
- Mockery's testify template emits no Godoc for generated expecter types.
