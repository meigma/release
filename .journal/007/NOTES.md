---
id: 007
title: New work session
started: 2026-08-21
---

## 2026-08-21 10:27 — Kickoff
Goal for the session: Start a fresh journal session; the substantive work request has not been provided yet.
Current state of the world: `main` is current, release `v0.1.3` is verified, and sessions 001 through 006 are complete.
Plan: Bind this session to the current task, then capture and execute the developer's next request.

## 2026-08-21 10:33 — Scoop familiarization
Goal: Plan Scoop support by reusing the proven Homebrew delivery method rather than inventing a second publication model.
Current Homebrew state: PRs 28 through 34 landed the managed tap CI, fail-closed `pubbrew` reconciliation engine, `ghtap` GitHub adapter, `publish homebrew` CLI command, reusable producer publisher, optional macOS signing/notarization, and deterministic `init homebrew-tap` scaffold. Release `v0.1.5` completed the production path; `meigma/homebrew-tap` PR 6 passed macOS and Linux cask checks and merged the generated `meigma-release-cli` cask.
Pattern to preserve: GoReleaser renders a package-manager control file with `skip_upload: true`; the authoritative Actions artifact carries it but the signed GitHub Release payload excludes it; publication runs only after the GitHub Release is public; a repository-scoped App token opens a deterministic branch and pull request; secret-free bucket/tap CI is the required merge check; the default branch is the published state; reruns converge or fail on explicit conflicts.
Scoop findings: GoReleaser 2.17.1 supports `scoops`, `skip_upload: true`, Windows ZIP archives, SHA-256 values, and `64bit` plus `arm64` manifest entries. Because this repository sets `release.disable: true`, a Scoop entry must provide an explicit `url_template`. A clean local snapshot probe against the current build produced `dist/scoop/meigma-release-cli.json`; setting `directory: bucket` produced `dist/scoop/bucket/meigma-release-cli.json`. The generated manifest named `release-cli.exe` as the binary and referenced the existing Windows amd64 and arm64 release archives. Probe artifacts were removed and `main` remained clean.
Scoop-specific gaps: choose root manifests versus the official BucketTemplate's `bucket/` layout; prove the exact Windows validation, install, update, uninstall, bad-hash, and unavailable-URL behavior in a disposable bucket; decide whether ARM64 is validated immediately or carried untested; pin Scoop test sources instead of copying the template's moving `ScoopInstaller/GithubActions@main`; and decide whether the second reviewed-file publisher justifies extracting channel-neutral reconciliation or should remain a focused `pubscoop` implementation.
Next: draft the Scoop plan in the same rehearsal → managed bucket CI → publisher → producer integration → initializer slices used for Homebrew, with every unresolved assumption assigned to the rehearsal rather than specified prematurely.

## 2026-08-21 11:48 — Scoop implementation plan
Created `.journal/007/PLAN.md` from a focused planning-agent pass, then reviewed it against the current Homebrew implementation, the GoReleaser 2.17.1 schema and Scoop pipe, and pinned Scoop/BucketTemplate sources.
Decision: keep Scoop as a separate channel-specific `pubscoop`/`ghbucket` path, reuse only existing neutral seams, and explicitly avoid a generic package-manager publisher. Default to root-level bucket manifests, subject to a real disposable-bucket rehearsal before the permanent contract lands.
Delivery order: disposable bucket rehearsal, secret-free managed bucket CI, fail-closed reviewed-PR publisher and CLI, producer/reusable-workflow integration, then deterministic bucket initializer and operator documentation.
Checks: the planned `.goreleaser.yaml` `scoops` entry—including `ids`, explicit `url_template`, and `skip_upload: true`—validates with the pinned local GoReleaser. The pinned Scoop schema, bucket tests, GoReleaser source, and BucketTemplate revision all resolve. No production files changed.

## 2026-08-21 12:17 — Scoop Slice 0 rehearsal complete
Created the public disposable repository `meigma/scoop-bucket-rehearsal` and completed the plan's real Windows rehearsal without changing this repository's production files.

Evidence:
- GoReleaser-generated root manifests for public releases `v0.1.3` and `v0.1.5` were bound to the exact release checksums before publication to the rehearsal bucket.
- Root-level discovery works with pinned Scoop commit `b588a06e41d920d2123ec70aee682bae14935939`: `scoop bucket list` reported one manifest and `scoop search meigma-release-cli` found it.
- The pinned official Scoop bucket tests pass against the root manifest.
- Rehearsal PR 1 proved install `v0.1.3`, execute `release-cli version`, update to `v0.1.5`, execute the updated binary, and uninstall with no application directory left behind. The required workflow passed before merge.
- Rehearsal PR 2 replaced the x64 SHA-256 with zeros. The required workflow rejected it during forced update with `Hash check failed`, reporting the expected zero digest and the actual release digest. The intentionally failing PR was closed and its branch deleted.
- Rehearsal PR 3 pointed the x64 manifest at a nonexistent release archive. The required workflow rejected it with HTTP 404 before installation. The intentionally failing PR was closed and its branch deleted.
- GitHub's standard public `windows-11-arm` runner is available. Rehearsal PR 4 asserted an ARM64 host, installed the manifest's native ARM64 archive, executed `release-cli version`, and uninstalled it. Run `32517568028` passed both `manifests / Scoop manifest validation` and `manifests / Windows ARM64 lifecycle`.
- `main` is protected with strict required checks for both workflow jobs, enforced for administrators, PR-only changes, linear history, conversation resolution, and no force pushes or deletion.

Decisions and learned constraints:
- Keep root-level manifests. Scoop discovers them correctly, and this avoids a needless `bucket/` directory.
- Validate ARM64 immediately rather than carrying its generated entry untested.
- Pin the Scoop checkout and official tests by full commit. Do not copy the moving `ScoopInstaller/GithubActions@main` template workflow.
- Set `.gitattributes` to CRLF for text because Scoop's pinned syntax tests enforce Windows line endings.
- Isolated Scoop bootstrap requires the `apps`, `buckets`, `cache`, `persist`, and `shims` directories plus an `apps/scoop/current` junction to the pinned checkout. Setting `LAST_UPDATE` prevents a network self-update from replacing the pinned core during lifecycle tests.
- The official schema test discovers only Git-changed manifests when `CI` is true. The rehearsal clears `CI` for that invocation so workflow-only changes still validate every manifest instead of failing Pester 6's empty parameterized test.
- Keep the permanent Slice 1 workflow behaviorally equivalent to the rehearsal workflow. The disposable repository is the executable contract.

Remote state: rehearsal `main` is `2d13801`; PRs 1 and 4 merged, PRs 2 and 3 closed, and no rehearsal branch remains open.
Next: implement Slice 1 managed bucket CI in the production repository, preserving the two required check names and the proven root-manifest lifecycle.

## 2026-08-21 13:14 — Scoop Slice 1 managed bucket CI complete
Merged production PR 36 as `d14fb9af3ad1e43cf9aafe9e69342187af0ef0f1`, adding `.github/workflows/scoop-bucket-ci.yml`.

Implemented contract:
- The reusable workflow accepts only `workflow_call`, has empty top-level permissions, and grants only `contents: read` to discovery and validation jobs.
- Discovery validates pull-request base/head SHA shape, rejects root-manifest deletion or rename, accepts only safe added/modified root `*.json` names, and emits a compact manifest matrix.
- Each manifest runs on `windows-2025`/AMD64 and `windows-11-arm`/ARM64. AMD64 runs the pinned official Scoop bucket tests against an isolated single-manifest bucket; both architectures install or force-update the candidate with pinned Scoop and then uninstall it.
- Every action and source checkout uses a full commit SHA. The final `always()` aggregation job is named `Scoop manifest validation`.

Managed-caller proof:
- Rehearsal PR 5 replaced 195 lines of repository-owned workflow logic with a 15-line caller pinned to production commit `d14fb9af3ad1e43cf9aafe9e69342187af0ef0f1`. Run `32522006997` passed discovery, pinned AMD64 validation, native ARM64 validation, and `manifests / Scoop manifest validation`; PR 5 merged as rehearsal commit `f5f1c12`.
- Rehearsal PR 9 set the x64 digest to zeros. Run `32522273015` reported `Hash check failed`; the stable aggregation check failed.
- Rehearsal PR 7 used an absent x64 asset. Run `32522273388` reported HTTP 404; the stable aggregation check failed.
- Rehearsal PR 6 added an unsupported top-level property. Run `32522272152` reported that `unexpected` is not defined and additional properties are forbidden; the stable aggregation check failed.
- Rehearsal PR 8 deleted the root manifest. Run `32522272893` failed discovery with `Scoop bucket CI does not publish manifest deletions or renames`; validation skipped and the stable aggregation check failed.
- All four intentional failure PRs were closed and their remote branches deleted.

Protection now requires only the stable strict check `manifests / Scoop manifest validation`; PR enforcement, administrator enforcement, linear history, conversation resolution, and force-push/deletion prohibitions remain enabled. This corrects the Slice 0 note's temporary two-check protection: the managed workflow deliberately aggregates discovery and both architecture jobs behind one durable branch-protection context.

Verification: `actionlint` passes for the merged reusable workflow and caller. Production and rehearsal `main` trees are clean, no journal path is tracked on production `main`, the rehearsal has no open PR or branch beyond `main`, and the integrated implementation worktree was removed.
Next: Slice 2, the fail-closed reviewed-PR Scoop publisher and CLI.

## 2026-08-21 13:53 — Scoop Slice 2 publisher and CLI complete
Merged production PR 37 as `66f43c7906b12a201b7530fad043ac9a77974076`.

Implemented contract:
- `internal/stage/pubscoop` is a separate fail-closed reconciliation engine. It owns only root `<manifest>.json` paths and deterministic `release/<manifest>/v<version>` branches, validates the generated JSON `version`, retries ambiguous writes after fresh reads, and returns only `created`, `open`, or `published`.
- `internal/adapter/ghbucket` is a focused go-github adapter with generated reader and writer mocks. No generic package-manager publisher or Homebrew domain refactor was introduced.
- `release-cli publish scoop --dist --bucket --manifest` reads only the nonempty regular `scoop/<manifest>.json` file beneath a confined distribution root, accepts the Release App token only through `RELEASE_APP_TOKEN`, uses the existing result envelope, and wires authenticated bucket reader and writer factories in the binary.
- `.goreleaser.yaml` now declares the `meigma-release-cli` Scoop manifest for archive ID `release-cli`, repository `meigma/scoop-bucket`, the release asset URL template, and `skip_upload: true`.

Verification:
- Repository format, lint, build, tests, protocol stamp, and generated-mock checks passed. LSP reported no diagnostics in the new domain, adapter, CLI, options, or binary wiring.
- `goreleaser check` accepted the configuration. A pinned GoReleaser 2.17.1 snapshot generated `dist/scoop/meigma-release-cli.json` with `64bit` and `arm64` Windows archives, `release-cli.exe`, the expected URL template, homepage, proprietary license, and description.
- PR 37 checks passed: repository CI, Nix flake, and Kusari Inspector.
- A public disposable repository rehearsal opened PR 1 and observed `created` on the first invocation and `open` on an exact rerun. Changing only the desired manifest content failed with `scoop publication conflict: publication branch release/meigma-release-cli/v0.1.5 has unexpected content`. After restoring the bytes and squash-merging the review, the next invocation returned `published`.
- The disposable repository could not be deleted because the local GitHub token lacks `delete_repo`; it was archived instead. It is public at `meigma/scoop-publisher-rehearsal`, has no open pull request, and has only `main`.

Impact note: GitNexus rated the additive `cli.Options` surface critical because it has 47 direct dependents and rated `newPublishCommand` high transitively. LSP found 87 `Options` references; the complete repository suite and live CLI rehearsal covered the changed command path.

Next: Slice 3, deterministic `init scoop-bucket` scaffolding and operator documentation. Do not start it without an explicit request.
