# Scoop Support Plan

## Decision

Add Scoop as a second, channel-specific reviewed-file publisher, deliberately parallel to Homebrew:

1. GoReleaser 2.17.1 renders one Scoop manifest with `skip_upload: true`.
2. The authoritative `release-assets` Actions artifact carries that manifest, while `checksums.txt`, its Sigstore bundle, and the public GitHub Release continue to cover only released payloads.
3. The GitHub Release becomes public before Scoop publication starts.
4. `release-cli publish scoop` uses a repository-scoped GitHub App installation token to reconcile one deterministic branch and pull request in a protected bucket.
5. Secret-free Windows bucket CI validates the manifest; a maintainer merge makes the default branch the published state.

Use root-level manifests in `meigma/scoop-bucket`: GoReleaser writes `dist/scoop/<name>.json`, and the publisher writes `<name>.json`. This is the smallest layout, matches Scoop’s manual bucket example, and follows GoReleaser’s recommendation to leave `directory` empty because `scoop bucket list` reports zero manifests for nested layouts. Slice 0 must still prove this behavior with the pinned Scoop version before any permanent workflow or CLI contract lands; if current Scoop contradicts the documented behavior, resolve the layout there and carry the single decision consistently through every later path and test.

Keep `internal/stage/pubscoop` and `internal/adapter/ghbucket` separate from `pubbrew` and `ghtap`. Reuse existing exact seams—`rel.Version`, `rel.Secret`, `GitHubEndpoint`, result-envelope helpers, and confined filesystem patterns—but do not create a generic package-manager publisher, repository adapter, control-file type, or initializer framework. The duplicated reconciliation shape is intentional: Ruby cask parsing, JSON manifest parsing, remote paths, names, commit messages, CI, and installation invariants are channel-specific.

## Proven assumptions

### Current Homebrew implementation

The implemented Homebrew path confirms the delivery order and trust boundaries to copy:

- `.goreleaser.yaml` renders `dist/homebrew/Casks/meigma-release-cli.rb` with `homebrew_casks[].skip_upload: true` and an explicit release URL template.
- `.github/workflows/go-pre-publish.yml` stages and formats the cask, then includes it in the digest-addressed `release-assets` handoff.
- `.github/workflows/publish-github-release.yml` verifies the handoff, removes the unsigned Homebrew control, verifies the signed bundle, attests/uploads the payloads, and makes the GitHub Release public.
- `.github/workflows/publish-homebrew.yml` runs afterward, verifies the same handoff and signed bundle, mints a tap-repository-scoped App token, and invokes `release-cli publish homebrew`.
- `internal/stage/pubbrew.Publish` owns a fail-closed state machine with `created`, `open`, and `published` outcomes; it accepts only one exact control-file commit based on the observed default-branch head and refuses unknown branches, closed or inconsistent pull requests, same-version/different-content, and newer base content.
- `internal/adapter/ghtap.Client` maps `go-github` state and errors into `pubbrew` ports without owning retry or reconciliation policy.
- `internal/cli/homebrew.go` resolves flags and Actions identity, confines `homebrew/Casks/<token>.rb` under an `os.Root`, and injects narrow read/write ports.
- `.github/workflows/homebrew-tap-ci.yml` is a secret-free required check; `internal/cli/homebrew_init.go` deterministically renders its SHA-pinned caller without network access.

Homebrew-only behavior must not leak into Scoop: cask Ruby/version parsing, `Casks/`, `homebrew-<name>` repository naming, `brew style`/`audit`, macOS/Linux cask installation, quarantine handling, and Apple signing/notarization remain unchanged.

### GoReleaser and Scoop

Repository probes already established that GoReleaser 2.17.1:

- accepts `scoops` with `skip_upload: true` under the current production-compatible staging path;
- writes `dist/scoop/meigma-release-cli.json`, or `dist/scoop/bucket/meigma-release-cli.json` when `directory: bucket` is set;
- renders the existing Windows ZIPs as `architecture.64bit` and `architecture.arm64`, with SHA-256 `hash`, release `url`, and `release-cli.exe` in `bin`;
- requires an explicit `url_template` here because `.goreleaser.yaml` sets `release.disable: true`.

GoReleaser 2.17.1 source confirms that the Scoop pipe filters uploadable Windows archives, maps `amd64` to `64bit` and `arm64` to `arm64`, obtains each archive’s SHA-256, and writes `dist/scoop/<directory>/<name>.json`. Official Scoop documentation defines a bucket as a Git repository of JSON app manifests and defines `version`, architecture-specific `url`, `hash`, and `bin`. The official manual bucket example puts manifests at repository root; BucketTemplate uses `bucket/`, so the real rehearsal—not assumption—settles current client, validator, and listing behavior.

## Unresolved rehearsal questions

Slice 0 must record evidence for these points before Slice 1 fixes a permanent contract:

- Does a root-level manifest appear correctly in `scoop bucket list` and `scoop search`, and install by qualified bucket/app name on the current stable Scoop client?
- Which exact pinned Scoop schema/Pester invocation validates a generated GoReleaser manifest without relying on `ScoopInstaller/GithubActions@main` or an unpinned Scoop checkout?
- Does the official bucket test harness require CRLF checkout normalization for generated JSON; if so, which minimal `.gitattributes` entry is required?
- What exact commands prove install, command execution, update from the preceding real release, and uninstall without depending on ambient developer state?
- Do a bad SHA-256 and an unavailable archive URL both fail before merge through the same managed check?
- Is a public GitHub-hosted Windows ARM64 runner available and sufficient for Scoop installation? If not, keep the proven `arm64` manifest entry but document that Slice 1 validates `64bit` only; do not silently claim ARM64 runtime coverage.
- What stable required-check name and exact full Scoop/BucketTemplate source revisions result from the successful rehearsal?

## Intended release flow

```mermaid
flowchart LR
    A[GoReleaser builds Windows ZIPs and Scoop manifest] --> B[Authoritative release-assets artifact]
    B --> C[Verify handoff and signed bundle]
    C --> D[Public GitHub Release]
    D --> E[Repository-scoped App token]
    E --> F[release-cli publish scoop]
    F --> G[Bucket branch and pull request]
    G --> H[Secret-free Windows CI]
    H --> I[Maintainer merge]
    I --> J[Manifest published on default branch]
```

The GitHub Release remains authoritative and irreversible. Scoop publication is eventually consistent. A failure after release publication leaves the release public and fails the Scoop job; rerunning must converge to the exact open pull request or exact merged manifest, or fail on an explicit conflict. Homebrew and Scoop publishers may run independently in parallel only after `github-release` succeeds.

## Minimal contracts

### Producer configuration

Add this channel-specific entry to `.goreleaser.yaml`; do not set `directory`:

```yaml
scoops:
  - name: meigma-release-cli
    ids:
      - release-cli
    repository:
      owner: meigma
      name: scoop-bucket
    homepage: https://github.com/meigma/release
    description: Release automation for Meigma projects
    license: Proprietary
    url_template: "https://github.com/meigma/release/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
    skip_upload: true
```

The producer continues using the existing Windows `amd64` and `arm64` ZIP archives. The CLI does not render or rewrite JSON.

### CLI

```text
release-cli publish scoop \
  --dist dist \
  --bucket meigma/scoop-bucket \
  --manifest meigma-release-cli \
  --json

release-cli init scoop-bucket \
  --bucket meigma/scoop-bucket \
  --output ./scoop-bucket \
  --json
```

`publish scoop` reads only the nonempty regular file `scoop/<manifest>.json` beneath the confined `--dist` root, bounded at 1 MiB. It takes `RELEASE_APP_TOKEN` only from the environment and derives source identity from `GITHUB_REPOSITORY`, `GITHUB_REF_NAME`, and `GITHUB_SHA`. Its `release.dev/result/v1` payload is:

```json
{
  "bucket": "meigma/scoop-bucket",
  "manifest": "meigma-release-cli",
  "branch": "release/meigma-release-cli/v1.2.3",
  "pull_request_url": "https://github.com/meigma/scoop-bucket/pull/42",
  "state": "created"
}
```

States are exactly `created`, `open`, and `published`. The only writable repository path is `<manifest>.json`; the only owned branch is `release/<manifest>/v<version>`. The manifest must be valid JSON with a string `version` equal to the stable tag version without `v`.

### Reusable workflows

`publish-scoop.yml` mirrors the Homebrew workflow names without sharing channel policy. Inputs are `artifact-id`, `artifact-digest`, `checksum-signing-workflow-ref`, `bucket`, `manifest`, `release-app-client-id`, and default-false `publish-scoop`; `release-app-private-key` is optional so a disabled call needs no credential. Outputs are `branch`, `pull-request-url`, and `state`.

An enabled call verifies the artifact metadata and signed bundle before minting a token scoped to exactly the bucket repository with `contents: write` and `pull_requests: write`. Neither the workflow nor CLI writes the default branch, force-pushes, deletes a ref/path, enables auto-merge, merges a pull request, or accepts a token flag.

`scoop-bucket-ci.yml` takes no inputs or secrets, uses `permissions: {}` at workflow scope, and is called only from a `pull_request` caller with `contents: read`. It rejects deleted manifests, validates changed root `*.json` files, and runs the exact schema/hash/install/uninstall sequence proven in Slice 0 on Windows. The final stable check is `manifests / Scoop manifest validation`.

## Delivery slices

### Slice 0 — Real disposable-bucket rehearsal

**Dependency:** first; no permanent Scoop contract may precede it.

Create a public disposable bucket such as `meigma/scoop-bucket-rehearsal`. Generate root manifests for two existing public `release-cli` releases with GoReleaser 2.17.1, then add the newer one through a protected pull request. Exercise the real current Scoop client on a clean Windows runner:

- add the Git repository as a bucket, inspect `scoop bucket list`, and search the manifest;
- install the preceding release, run `release-cli version`, update to the pull-request version, run it again, and uninstall;
- run the official schema/bucket tests from full commit pins;
- open separate negative pull requests with a wrong hash and an unavailable URL and prove the required validation fails;
- test `arm64` on a hosted ARM64 runner if one is actually available;
- compare root and `bucket/` discovery only far enough to confirm the root-layout decision;
- record commands, runner labels, action/source SHAs, required-check name, line-ending result, URLs, and observed failures in `.journal/007/NOTES.md`.

**Acceptance and behavioral verification:** the exact released Windows archive installs, reports the expected version, updates, and uninstalls from the disposable bucket; both negative mutations fail before merge; the notes resolve every rehearsal question above. This slice changes no production workflow or CLI behavior.

### Slice 1 — Managed bucket CI

**Dependency:** Slice 0. The workflow implementation and external caller are parallel once the rehearsal contract is fixed.

Add `.github/workflows/scoop-bucket-ci.yml`. Follow the proven `homebrew-tap-ci.yml` structure: validate PR base/head SHAs, reject root-manifest deletions, emit a matrix of added/modified manifest names, validate each on the proven Windows runner(s), and finish with an unconditional aggregation job named `Scoop manifest validation`. Pin `actions/checkout`, Scoop source, and every test/cache action to full commits; never copy BucketTemplate’s moving Scoop checkout or `ScoopInstaller/GithubActions@main` usage.

In the disposable bucket, add `.github/workflows/manifests.yml` as a minimal caller triggered by pull requests changing `*.json`, with top-level `permissions: {}`, caller-job `contents: read`, and a full-SHA reusable-workflow reference. Protect the default branch with `manifests / Scoop manifest validation`.

**Acceptance and behavioral verification:** rerun the successful, bad-hash, unavailable-URL, malformed-schema, and deletion pull requests through the managed workflow. The success case passes; every negative case fails the stable required check; no job receives secrets or write permission. ARM64 appears in the matrix only if Slice 0 proved it.

### Slice 2 — Reviewed-PR publisher and CLI

**Dependency:** Slice 1 for the protected merge boundary. Domain engine and adapter tests are parallelizable; CLI wiring follows their contracts.

Add the Scoop-specific implementation:

- `internal/stage/pubscoop/{doc.go,errors.go,retry.go,values.go,publish.go,publish_test.go}` with documented exported `PublicationState`, `PullRequestState`, snapshots, ports, `PublishInput`, `PublishResult`, `Repository`, `ManifestName`, commit/blob/path values, `ParseRepository`, `ParseManifestName`, and `Publish`;
- `internal/adapter/ghbucket/{doc.go,client.go,errors.go,reader.go,reader_test.go,writer.go,writer_test.go}` plus `mocks/{doc.go,repository_reader.go,repository_writer.go}`;
- `.mockery.yml` entries for `pubscoop.RepositoryReader` and `RepositoryWriter`, generating both mocks into `internal/adapter/ghbucket/mocks`—never handwrite them;
- `internal/cli/{scoop.go,scoop_test.go,oci.go,root.go}` to add `publish scoop`, Scoop reader/writer options and factories, exact flag/environment resolution, confined manifest loading, and the existing JSON envelope;
- `cmd/release-cli/main.go` to wire `ghbucket.NewAuthenticated` factories;
- `docs/reference/release-cli-contract.md` to document command syntax, fields, paths, state transitions, retry limits, errors, silent non-JSON success, and exit codes.

Copy the `pubbrew.Publish` ordering, but parse a top-level JSON `version`, use `<manifest>.json`, and use Scoop-specific diagnostics and commit/PR text. Preserve exact reconciliation: return `published` for identical base bytes; refuse malformed, same-version/different, or newer base manifests; accept only one exact one-parent/one-file publication commit based on the observed base; reconcile lost write responses; return the exact open PR; reject closed/inconsistent PRs and unknown branch content. All exported declarations and fields receive identifier-leading Godoc; both new packages receive `doc.go`.

Do not refactor `pubbrew`, `ghtap`, or their mocks into generic packages. The only shared code is already-valid repository-neutral infrastructure such as `rel.Version`, secrets, endpoint resolution, result writing, and the existing required-flag helper.

**Acceptance and behavioral verification:** behavior tests prove create/rerun-open, exact existing PR, already-merged/published, same-version/different-content, newer base, unexpected branch/parent/path, malformed or mismatched JSON version, retry reconciliation, nil ports, missing token before factory construction, JSON envelope shape, regular/nonempty/size bounds, and escaping-symlink rejection. Against the disposable bucket, run first publish, rerun while open, merge, rerun after merge, and manually seeded conflict cases; every run either converges without duplicate writes or fails without modifying the default branch.

### Slice 3 — Producer and reusable publisher integration

**Dependency:** Slice 2. GoReleaser generation/handoff changes and the new reusable publisher can be implemented in parallel; orchestration follows both.

Change these exact surfaces:

- `.goreleaser.yaml`: add the root-layout `scoops` entry shown above.
- `.github/workflows/go-pre-publish.yml`: include `dist/scoop/*.json` in `release-assets`; add only the formatting/normalization step proven necessary in Slice 0.
- `.github/workflows/publish-scoop.yml`: add the default-off reusable publisher contract described above. After artifact-digest verification, require exactly `scoop/<manifest>.json`, isolate it, remove unrelated `dist/homebrew`, verify the signed bundle, restore the Scoop control, mint the bucket-scoped App token, invoke the CLI, and validate the three output states.
- `.github/workflows/publish-github-release.yml`: after handoff verification, explicitly remove both `dist/homebrew` and `dist/scoop` before signed-bundle verification, attestation, and upload.
- `.github/workflows/publish-homebrew.yml`: after handoff verification, remove the unrelated `dist/scoop` control before its existing signed-bundle verification, while retaining its existing cask isolation/restoration and all Homebrew behavior.
- `.github/workflows/release.yml`: add `scoop-publish`, needing `release-assets` and `github-release`, targeting `meigma/scoop-bucket` and `meigma-release-cli`, using the same Release App client ID/private key, and setting `publish-scoop: true`. It is independent of `homebrew-publish` after the public release.
- `docs/reference/{github-release-contract.md,release-cli-contract.md}`: distinguish the digest-protected Actions controls from signed/public release payloads; document default-off Scoop inputs, outputs, dependency order, and App scope.

The Scoop control is integrity-bound by the Actions artifact digest but deliberately absent from `checksums.txt`, attestations, and public GitHub Release assets, exactly like the cask. This workflow interaction is the only required Homebrew change; verify that adding the second unsigned control does not alter the Homebrew result.

**Acceptance and behavioral verification:** a staged run contains exactly one `dist/scoop/meigma-release-cli.json` with `64bit` and `arm64` entries referencing the existing ZIPs, hashes matching those archives, and `release-cli.exe` as the binary. A disabled reusable call requires no App values, mints no token, and performs no bucket request. An enabled real tag run first publishes the GitHub Release, then opens one valid bucket PR; handoff or bundle failure prevents token creation. In the same run, the Homebrew job still isolates its cask, verifies the bundle, and reaches its previous state. After review and merge, install and uninstall the published Scoop app on the rehearsed Windows platform(s).

### Slice 4 — Deterministic bucket initializer and operator documentation

**Dependency:** Slice 1’s proven scaffold. Renderer/tests and the how-to guide are parallelizable; command wiring follows the result type.

Add:

- `internal/cli/scoop_init.go` with `newScoopBucketInitCommand`, `runScoopBucketInit`, configuration resolution, deterministic renderer, and atomic writer;
- `internal/cli/scoop_init_test.go` for byte-exact output and filesystem behavior;
- `internal/cli/homebrew_init.go` only to register the sibling initializer and reuse the existing private `scaffoldFile`, confined write, empty-output, and atomic-install helpers without changing Homebrew rendering;
- `internal/cli/result.go` with fully documented `ScoopBucketInitResult { Bucket, Output, Files }`;
- `docs/how-to/set-up-scoop-bucket.md` and the corresponding initializer section in `docs/reference/release-cli-contract.md`.

The command performs no network, Git, GitHub, ruleset, App-installation, or credential operation. It accepts any validated `owner/repository` bucket name, requires a full source commit stamped into the CLI, and refuses files, symlinks, and nonempty output directories without overlaying user content. The baseline root-layout scaffold contains exactly `.github/workflows/manifests.yml`, `.github/dependabot.yml`, and `README.md`; include a minimal `.gitattributes` only if Slice 0 proved checkout normalization is required. The caller workflow uses `*.json`, `contents: read`, no secrets, and pins `scoop-bucket-ci.yml` to the CLI’s full source commit. The README records `scoop bucket add <repository-name> https://github.com/<owner>/<repository>` and qualified install syntax.

The how-to guide covers scaffold inspection, `gh repo create`, selected-repository App installation, selected producer Actions variable/secret access, the exact GoReleaser entry, `publish-scoop.yml` caller job, required `manifests / Scoop manifest validation` ruleset, and first publish/install/update/uninstall verification. Record final production URLs and verification evidence in `.journal/007/NOTES.md`, and close the session in `.journal/007/SUMMARY.md`.

**Acceptance and behavioral verification:** repeated runs with identical inputs produce byte-identical files and a lexically ordered JSON file list; absent and pre-created empty outputs succeed; nonempty, symlink, invalid repository, and unstamped/abbreviated commit cases fail before mutation; a generated scaffold matches the successfully protected bucket and its workflow passes there without edits.

## Risks and impact points

- `NewRootCommand` is a high-impact seam: the stale GitNexus index reports 23 direct and 125 total dependents, and exact text search confirms production wiring plus the shared command test harnesses. Keep additions to `Options`, `newPublishCommand`, and `newInitCommand` mechanical and run every changed command contract.
- Adding a second unsigned control affects three bundle-verification workflows. Missing one removal/isolation step would make GitHub, Homebrew, or Scoop publication fail despite valid payloads; Slice 3 must exercise all three paths together.
- Publication happens after an irreversible public release. Preserve bounded fresh-read reconciliation and explicit conflict failure rather than rollback or direct-branch remediation.
- A Scoop manifest can contain install scripts. PR validation therefore executes only on ephemeral GitHub-hosted Windows runners under `pull_request`, with no secrets and read-only repository permission; never use `pull_request_target`.
- The Release App must be installed on the bucket and producer, but its minted token must name only the bucket repository and request only contents/PR writes. The App cannot bypass the protected default branch.
- Official Scoop test behavior and Windows ARM64 availability are external and version-sensitive. Pin the exact proven sources and report untested ARM64 honestly rather than weakening validation or adding secret-bearing self-hosted runners.
- Root filenames and JSON versions are security boundaries, not generic strings: confine local reads, validate manifest names, parse JSON, and authorize exactly one remote path.

## Explicit non-goals

- No changes to Homebrew publication semantics, cask layout, tap initializer output, or macOS signing/notarization.
- No generic publisher/state-machine/SCM/package-manager abstraction and no refactor of `pubbrew` or `ghtap` to reduce duplication.
- No GoReleaser built-in Scoop uploader or pull-request publisher; `skip_upload: true` remains mandatory.
- No direct default-branch writes, force pushes, ref/path deletion, auto-merge, automatic merge, or ruleset bypass.
- No repository creation, ruleset administration, App installation, or organization-secret management in `release-cli`.
- No Scoop Excavator, scheduled autoupdates, issue/PR bots, known-bucket registration, Scoop Directory submission, or copied BucketTemplate automation beyond the validation behavior actually rehearsed.
- No MSI/NSIS publishing, source builds, alternate manifest generation, Windows signing program, or changes to the existing Windows ZIP payloads.
- No MacPorts, Nix, mise, installer-script, or other package-channel work.

## Primary sources

- [GoReleaser: Scoop Manifests](https://goreleaser.com/customization/publish/scoop/)
- [GoReleaser 2.17.1 Scoop pipe source](https://github.com/goreleaser/goreleaser/blob/v2.17.1/internal/pipe/scoop/scoop.go)
- [GoReleaser: Scoop requires a Windows ZIP archive](https://goreleaser.com/resources/errors/scoop-archive/)
- [Scoop: Buckets](https://github.com/ScoopInstaller/Scoop/wiki/Buckets)
- [Scoop: App Manifests](https://github.com/ScoopInstaller/Scoop/wiki/App-Manifests)
- [Scoop manifest schema at pinned core revision](https://github.com/ScoopInstaller/Scoop/blob/b588a06e41d920d2123ec70aee682bae14935939/schema.json)
- [Scoop bucket tests at pinned core revision](https://github.com/ScoopInstaller/Scoop/blob/b588a06e41d920d2123ec70aee682bae14935939/test/Import-Bucket-Tests.ps1)
- [Scoop BucketTemplate CI at reviewed revision](https://github.com/ScoopInstaller/BucketTemplate/blob/ef6a51bc217629322ef9f53cefe8c8462ca4841b/.github/workflows/ci.yml)
- [GitHub: Generating an installation access token for a GitHub App](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)
