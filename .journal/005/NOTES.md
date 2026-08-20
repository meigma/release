---
id: 005
title: New work session
started: 2026-08-19
---

## 2026-08-19 20:33 — Kickoff
Goal for the session: Start a new journal session and wait for the user's actual request.
Current state of the world: `main` is at `7197ca2`; release CLI behavioral slices 1 through 10 are merged, `v0.1.0` is published, and the final documentation-pin slice remains gated on a released and verified build of the current CLI.
Plan: Receive the request, scope the work from current repository context, execute it, and checkpoint meaningful progress here.

## 2026-08-19 20:35 — Exec wrapper inventory
Counted six production packages with private `os/exec` infrastructure: `apko`, `cosign`, `ghup`, `gitx`, `melange`, and `goprof`. They contain seven `exec.CommandContext` call sites because `cosign` has separate signing and verification paths. Each package owns its own `resolveBinary` and bounded `tailBuffer` implementation; the lone additional `exec.Command` site is test-only in `gitx`.

## 2026-08-19 20:43 — Shared exec migration plan
Created `PLAN.md` for an atomic clean cutover to `internal/execx`. The plan keeps tool-specific ports, arguments, secret handling, parsing, and errors in their current packages; centralizes only binary resolution, child process setup, output routing, bounded stderr capture, `WaitDelay`, and typed exit metadata; migrates all seven production call sites; deletes all six private helper copies; and verifies the boundary with direct process tests, targeted consumer suites, `root:check`, and the Linux suite. A focused architecture amendment will live in this session folder rather than rewriting session 002's closed architecture.

## 2026-08-19 21:05 — Exec architecture amendment approved
Added `ARCHITECTURE-AMENDMENT.md`. It records that revision 3's deferred extraction trigger has fired, defines `internal/execx` as local stdlib plumbing rather than a port or adapter, fixes the process policy at one attempt with a 4 KiB stderr tail and five-second `WaitDelay`, and leaves tool-specific validation, argv, secrets, parsing, errors, and retry policy in the existing consumers.

## 2026-08-19 21:16 — Shared exec cutover merged
Implemented the approved plan in PR [#18](https://github.com/meigma/release/pull/18), squash-merged as `0fd8483`. `internal/execx` now owns every production `os/exec` call and all seven production consumers use `execx.Run`; the six copies of `resolveBinary`, `tailBuffer`, stderr-limit constants, and `WaitDelay` were deleted. Tool-specific ports, options, argv, ordering, credentials, parsing, retries, and error presentation stayed local. Verification passed: direct `internal/execx` tests including the five-second orphan-grandchild case; all six targeted consumer packages; `mise exec -- moon run root:check`; the full Linux `golang:1.26 go test ./... -count=1` suite; CI; Kusari Inspector; and structural searches showing only `internal/execx` plus the deliberate test-only Git fixture import `os/exec`. The implementation worktree and local/remote feature branches were removed after merge.

## 2026-08-19 21:19 — OCI build workflow glue explained
Reviewed `.github/workflows/go-oci-build.yml:155-208`. The Bash block is orchestration glue: enable strict shell failure, resolve and validate pinned Melange/apko executables through mise, derive the tagged commit's committer date, invoke `release-cli image build --json` with the step's input/work/output/config environment, then parse the result envelope and publish `work`, `output`, `version`, `build-date`, and `binary-name` as step outputs. The CLI owns canonical-binary digest verification, signed APK repository construction, ephemeral key isolation, and apko OCI layout composition; the following workflow step independently runs `image verify`.

## 2026-08-19 21:21 — Keep OCI glue out of github-script
Recommendation: do not replace the OCI build block with `actions/github-script` or a first-party JavaScript action. It performs no GitHub API work and would still spawn mise, git, and release-cli, adding a Node/action boundary without removing process orchestration. Bash is the direct adapter for this one build-specific sequence. If output-bridge duplication becomes the target, the stronger follow-up is the already-budgeted optional `cli.Actions`/`actenv` seam: three workflows currently parse CLI JSON and write `GITHUB_OUTPUT`. Keep `--json` for local portability and make Actions output opt-in rather than hiding this one block in a composite action.
