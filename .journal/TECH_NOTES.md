# Technical Notes

## Current state

- `main` is at `df077f9e7688232e7f0d070c232fdb12bc64c0e5`. The released artifact
  is `release-cli`; `v0.1.0` is published with six archives, six SBOMs,
  `checksums.txt` and its Sigstore bundle, plus a GHCR image tagged `latest`,
  `0`, `0.1`, `0.1.0`.
- The eleven-PR program in `.journal/002/PLAN.md` has **PRs 1 through 7 merged**.
  Remaining: PR 8 `image build`, PR 9 `image verify`, PR 10 GoReleaser invocation
  into `goprof`, PR 11 the documentation pin.
- Both publication workflows are now thin: `publish-oci-image.yml` runs
  prepare -> three `actions/attest` steps -> finalize, and
  `publish-github-release.yml` runs `verify handoff` -> download ->
  `verify bundle` -> `attest` -> `publish github`. Neither contains bespoke
  publication logic any more.

## Where the design lives

- `.journal/002/ARCHITECTURE.md` (revision 3) is the approved architecture and
  `.journal/002/PLAN.md` holds the eleven-PR plan, the **standing per-PR execution
  method**, and the three spike results. Read both before continuing the program.
- `.journal/003/SUMMARY.md` records slices 3-4b, including the deviations below.

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
  variable must parse with `strconv.ParseBool` or the command exits 2.
- **Commands:** `stage --profile go`, `verify handoff`, `verify bundle`,
  `plan tags`, `publish oci prepare [--dry-run]`, `publish oci finalize --result -`,
  `publish github [--no-undraft]`, `version`.
- **Ports.** Budget closed at 13; ten exist. Built: `pubgh.ArtifactMeta` (ghact),
  `puboci.StateReader`/`ContentPusher`/`TagCommitter` (reg), `puboci.Signer` and
  `pubgh.BlobVerifier` (cosign), `pubgh.ReleaseReader`/`Publisher` (ghrel),
  `pubgh.AssetReplacer` (ghup), `pubgh.RefResolver` (gitx). Unbuilt:
  `image.APKBuilder` (melange), `image.Composer` (apko), `cli.Actions` (actenv).
- **`actenv` is deliberately deferred.** Workflows read the CLI's `--json` envelope
  from stdout with a `$GITHUB_OUTPUT` heredoc and `jq`. Build the seam only when a
  command needs annotations or a job summary.
- **Two-phase OCI publication.** `publish oci prepare` -> YAML `actions/attest`
  -> `publish oci finalize`, the prepare result carried as the `--json` envelope on
  stdin. Invariant 14 (trust metadata before public tags) depends on the split.
  Finalize re-reads fresh state, refuses drift, and never replays a serialized plan.
- **Release publication.** The CLI never creates a release, never re-drafts one,
  never deletes an asset, and never mints an App token; the workflow mints it and
  passes it as `RELEASE_APP_TOKEN`, held as a redacted `rel.Secret`.
- **Retry.** Bounded retry lives in the engines with an injected `SleepFunc`: four
  attempts at 1s/2s/4s for transient classes. Poll budgets are engine-owned too —
  24 attempts 5s apart for draft discovery, 12 attempts 1s apart for asset
  convergence — so adapters take single snapshots.
- **Escape hatches.** `--plain-http` is flag-only and refused for non-loopback
  hosts. `RELEASE_COSIGN_PATH`, `RELEASE_GH_PATH`, and `RELEASE_GIT_PATH` locate
  pinned binaries; both publication workflows disable mise shims, so they pass
  explicit paths resolved with `mise which`.

## Platform facts learned the hard way

- **oras hands the content reader to `net/http`, which always closes a request
  body.** A caller-owned `*os.File` therefore gets closed twice. `internal/adapter/reg`
  wraps content in a `readerOnly` type; `fstest.MapFS` cannot catch this because its
  `Close` is idempotent.
- **A streamed request body has no `GetBody`,** so oras-go's retry transport cannot
  replay it. Retry must live where the content can be reopened.
- **HTTP 409 on a blob push is not "already exists."** Only
  `errdef.ErrAlreadyExists` means that; treating 409 as success can yield a signed,
  authoritative image with a missing layer.
- **An image's `org.opencontainers.image.version` annotation must equal the release
  version,** or channel planning refuses with an out-of-line error.
- **Registry login is not obsolete.** `cosign` and `actions/attest
  --push-to-registry` read the docker config. `cosign login` replaces the `oras`
  binary; `cosign` has no `logout`, so cleanup is a docker-config edit.
- **`oras.CopyGraph` is concurrent by default.** The adapters avoid it entirely:
  explicit per-descriptor pushes and strictly serial tag commits.
- **A callee can never exceed its caller's permission ceiling,** and PR CI cannot
  catch a missing grant when the caller only runs on tags.
- **`startup_failure` exposes no diagnostics via REST or GraphQL.** Read the run page.
- **Tags created with `GITHUB_TOKEN` do not trigger tag-push workflows,** so
  `release-please.yml` must keep minting an App token.
- **`$/` self-repository references work** for external SHA-pinned callers on runner
  >= 2.336.0; `github.action_repository` is empty for local-path invocations, so keep
  the `${GITHUB_REPOSITORY}` fallback.
- **`actions/attest` validates SBOM payloads** and rejects anything not recognizably
  SPDX or CycloneDX JSON. It also writes durable, unwithdrawable state, so cheap
  gates belong before it.
- **`golangci-lint` caches per path.** After removing a worktree it can stop
  excluding generated files and flag every mock; run `golangci-lint cache clean`.
- **Adding a repo to the release App's installation requires an organization owner**;
  an `admin:org` token gets 403. Same for a consumer's first public GHCR package.

## How to verify locally

- **Registry work:** build `/tmp` harnesses — a `go-containerregistry`
  `registry.New()` server plus a handmade two-platform OCI layout — and drive the
  CLI with `--plain-http` against `127.0.0.1`. A recording `cosign` stub proves
  argv and invocation counts.
- **Real GHCR:** scratch packages under `ghcr.io/meigma/<name>` work end to end and
  the current token has `delete:packages`, so clean up with
  `gh api -X DELETE /orgs/meigma/packages/container/<name>`.
- **Release work:** create a temporary draft release for a throwaway tag, exercise
  `publish github`, then `gh release delete <tag> --yes --cleanup-tag`.
- **Signature work:** the published `v0.1.0` bundle verifies against
  `https://github.com/meigma/release/.github/workflows/go-pre-publish.yml@refs/tags/v0.1.0`.

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
  `fb8c8098ff27968fb3070e928c00e925f38c698e`. PR 11 replaces it.
- Release-please PR #9 (`chore(main): release 0.1.1`) may still be open; the fix it
  describes is already inside the re-pointed `v0.1.0`.
