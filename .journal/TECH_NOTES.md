# Technical Notes

## Current state

- `main` is at `2c262ba3d807c587480e0006efe3d2475ec7121e`. The repository's
  released artifact is `release-cli`; the old `release-mvp` exercise binary is
  gone.
- `v0.1.0` is published with six archives, six SBOMs, `checksums.txt`, and its
  Sigstore bundle, plus a GHCR image tagged `latest`, `0`, `0.1`, `0.1.0`.
- The dogfood loop is closed and proven: the tag run builds `release-cli` from
  the tagged commit, passes it to all four reusable workflows through `cli-path`,
  and that binary stages and verifies its own release.

## Where the design lives

- `.journal/002/ARCHITECTURE.md` (revision 3) is the approved architecture;
  `.journal/002/PLAN.md` is the eleven-PR plan and also holds the **standing
  per-PR execution method** and the three spike results. Read both before
  continuing the program. PRs 1 and 2 are merged; PR 3 (`plan tags`,
  `internal/rel`, `StateReader`, `reg` read path) is next.

## Binding contracts

- **One release unit.** Workflows, the composite action, and the CLI share one
  version and one consumer pin. There is no `cli-version` input and independent
  pinning is explicitly unsupported. `cli-path` is an unsupported escape hatch:
  the installed path fails closed on a stamp mismatch, `cli-path` only warns.
- **Protocol stamp.** `cli.Protocol` is a Go source constant, deliberately not an
  ldflag. Release-please stamps only the version, through the **annotated
  `generic`** extra-file updater — the `yaml`+`jsonpath` updater re-serializes and
  strips comments. `scripts/check-protocol-stamp.sh` guards the two literals and
  the stamping markers.
- **CLI surface.** Exit codes are only `0`, `1`, `2`. Under `--json`, stdout
  carries exactly one `release.dev/result/v1` envelope, including failures, but a
  flag-parse failure yields usage on stderr and no envelope. `version` prints its
  data to stdout; side-effecting commands stay quiet on stdout without `--json`.
- **Ports.** Budget closed at 13, introduced with the slice that needs each one.
  Interfaces live in the consuming package; Mockery generates into the
  implementing adapter's `mocks/`. Local file work uses stdlib `fs.FS`/`io.Reader`
  with `os.OpenRoot` at the composition edge — no bespoke filesystem port. No
  Viper, no `internal/clock`, no `execx`, no generic retry package.
- **Three-owner handoff.** The CLI verifies the API metadata tuple; the pinned
  `actions/download-artifact` owns the ZIP transport digest; later commands verify
  extracted content. The CLI never claims to recompute the transport digest.
- **Retry.** Bounded retry lives in the `pubgh` engine with an injected
  `SleepFunc`: four attempts, 1s/2s/4s, transient classes only. It matches the
  `retries: 3` octokit semantics it replaced. A direct `ghact.Client.Get` caller
  gets no retry.
- **OCI publication is two-phase.** `publish oci prepare` -> YAML
  `actions/attest` -> `publish oci finalize`, with the prepare result carried as
  the `--json` envelope on stdin. Invariant 14 (trust metadata before public
  tags) depends on this split.

## Platform facts learned the hard way

- **Registry login is not obsolete.** `cosign` and `actions/attest
  --push-to-registry` read the docker config, so in-memory `oras-go` credentials
  cannot replace OP-09. `cosign login` replaces the `oras` binary; `cosign` has no
  `logout`, so cleanup is a docker-config edit.
- **`oras.CopyGraph` is concurrent by default** (`Concurrency: 3`). Pin
  concurrency where deterministic partial-failure reporting matters; tag commits
  must stay strictly serial.
- **A callee can never exceed its caller's permission ceiling**, and PR CI cannot
  catch a missing grant when the caller only runs on tags. Audit both sides.
- **`startup_failure` exposes no diagnostics via REST or GraphQL** — no jobs, no
  logs, empty `checkRuns`. Read the run page in a browser.
- **Tags created with `GITHUB_TOKEN` do not trigger tag-push workflows**, so
  `release-please.yml` must keep minting an App token. User-pushed tags do
  trigger them.
- **`$/` self-repository references work** for external SHA-pinned callers on
  runner >= 2.336.0 and are pin-immune, but `github.job_workflow_ref`/`_sha` are
  OIDC claims and evaluate to `null` in workflow expressions.
  `github.action_repository` is empty for local-path action invocations, so keep a
  `${GITHUB_REPOSITORY}` fallback there and never hardcode a repository.
- **`actions/attest` validates SBOM payloads** and rejects anything that is not
  recognizably SPDX or CycloneDX JSON.
- **Adding a repo to the release App's installation requires an organization
  owner**; an `admin:org` token gets 403. Same for making a consumer's first GHCR
  package public.

## Credentials

- Release mutation uses org variable `MEIGMA_RELEASE_APP_CLIENT_ID` and org secret
  `MEIGMA_RELEASE_APP_PRIVATE_KEY`, both scoped to selected repositories. Adopting
  organizations change only these plus the App installation.

## Housekeeping debt

- Archived but undeletable without `delete_repo`: `meigma/release-selfref-spike`,
  `meigma/release-oras-spike`, `meigma/release-stamp-spike`, and session 001's
  `meigma/release-oci-remediation-e2e`. Also pending: the `spike/self-ref` branch
  (kept as the evidence anchor for spike A's cited SHAs), the
  `ghcr.io/meigma/release-oras-spike` package, and the dead
  `release-please--branches--main--components--release-mvp` branch.
- Consumer docs and `examples/go-release/` still pin the pre-squash revision
  `fb8c8098ff27968fb3070e928c00e925f38c698e`, labelled as the last released
  revision. PR 11 replaces it with the final program revision.
