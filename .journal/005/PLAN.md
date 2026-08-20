# Shared subprocess execution plan

Status: executed in PR [#18](https://github.com/meigma/release/pull/18), squash-merged as `0fd8483`

Target PR: `refactor(exec): centralize subprocess execution`

## Goal

Create one internal subprocess mechanism and migrate every production caller to it without changing command arguments, ordering, environment handling, output routing, error text, retry policy, or domain boundaries.

The clean-cutover result is:

- `internal/execx` is the only production package that imports `os/exec`.
- Six consumer packages call `execx.Run`.
- The consumer-owned ports and tool-specific adapters remain separate.
- Test-only process setup may continue to use `os/exec`; routing fixture setup through the code under test would make those tests circular.

## Evidence and trigger

The architecture deferred `internal/execx` until two subprocess adapters repeated meaningful behavior beyond `exec.CommandContext`. That trigger has fired.

| Consumer package | Production calls | Distinct behavior retained by the consumer |
|---|---:|---|
| `internal/adapter/apko` | 1 | `lock` then `build`, working directory, subcommand error text |
| `internal/adapter/cosign` | 2 | recursive signing, blob verification, verification working directory |
| `internal/adapter/ghup` | 1 | App token environment closure and token-safe errors |
| `internal/adapter/gitx` | 1 | stdout parsing and unknown-tag classification |
| `internal/adapter/melange` | 1 | compile/keygen/build order and stop-on-first-failure behavior |
| `internal/profile/goprof` | 1 | live stdout, ANSI removal from the retained error tail, GoReleaser policy |

All six packages currently duplicate:

- deferred `exec.LookPath` resolution;
- `exec.CommandContext` construction;
- nil-environment inheritance;
- stdout discard or forwarding;
- live stderr forwarding plus a bounded 4 KiB tail;
- a five-second `exec.Cmd.WaitDelay` for leaked grandchild pipes;
- context-error precedence;
- exit-code extraction;
- the same `tailBuffer` implementation.

There are seven production `exec.CommandContext` sites because `cosign` has separate sign and verify paths.

## Decision

Add `internal/execx` as shared local plumbing, not as a port or adapter.

`execx` owns process mechanics. Each existing adapter continues to implement its narrow consumer-owned port and continues to own:

- domain validation;
- exact argument construction and invocation order;
- secret handling;
- domain-specific error prefixes and classifications;
- retry decisions;
- parsing of child output;
- tool-specific output transforms such as GoReleaser ANSI removal.

This split avoids the generic service-shaped `Runner` rejected by the architecture review. It also avoids a new interface or mock: stage engines already mock the narrow domain ports, and adapter tests exercise real child processes.

## Shared package contract

Create:

```text
internal/execx/
├── doc.go
├── execx.go
└── execx_test.go
```

Use a stateless package function and a value request:

```go
type Command struct {
    Program string
    Path    string
    Args    []string
    Dir     string
    Env     []string
    Stdout  io.Writer
    Stderr  io.Writer
}

func Run(ctx context.Context, command Command) error

type RunError struct { /* private state */ }

func (e *RunError) Error() string
func (e *RunError) Unwrap() error
func (e *RunError) ExitCode() (int, bool)
func (e *RunError) StderrTail() string
```

Every declaration and struct field receives Godoc per D1, and `doc.go` documents the package per D4.

### `Command` semantics

- `Program` is the default executable name, such as `cosign` or `git`.
- A nonempty `Path` overrides `Program`; `Run` still resolves the selected value through `exec.LookPath`, preserving current behavior for absolute paths, relative paths, and PATH names.
- `Args` excludes `argv[0]` and is passed as an argument slice. `execx` never invokes a shell or accepts a shell command string.
- An empty `Dir` inherits the parent working directory.
- A nil `Env` inherits the parent environment. A nonnil `Env`, including an empty slice, is used as-is.
- A nil `Stdout` discards stdout.
- A nil `Stderr` discards live stderr but still retains the bounded error tail. A nonnil `Stderr` receives the full live stream while the tail is retained.

### Fixed process policy

Keep the current shared values private to `execx`:

- stderr tail: 4 KiB;
- `WaitDelay`: five seconds;
- one process attempt;
- no retry;
- no implicit timeout beyond the caller's context;
- no environment, argument, or executable logging.

Do not add configuration knobs until a real caller needs different values. Melange, apko, and GoReleaser are stateful and non-idempotent, so `execx` must never retry them.

### Error semantics

`Run` must preserve these cases:

1. Reject a nil context and an empty executable selection before starting a process.
2. Resolve `Path` or `Program` at invocation time. Preserve the current `resolve <name>: <cause>` error shape.
3. If the context ends while the command runs, return the context error so each consumer can retain its current domain prefix and `errors.Is` behavior.
4. For a process start or wait failure, return `*RunError` with the underlying cause and retained stderr tail.
5. When the process exits, expose its exit code through `RunError.ExitCode`.
6. Keep `RunError.Error` free of argv and environment data. Consumers format their existing user-facing errors from the typed metadata.

Do not move tool labels, refs, tags, payload names, or the Git unknown-tag classification into `execx`.

## Architecture amendment

Before the implementation PR, add `.journal/005/ARCHITECTURE-AMENDMENT.md` in
the current session folder:

1. Identify the document as a focused amendment to
   `.journal/002/ARCHITECTURE.md` revision 3.
2. State that the deferred `execx` trigger fired: six packages now repeat
   binary resolution, bounded stderr capture, `WaitDelay`, and process
   execution.
3. Define `internal/execx` as local stdlib plumbing. It is not a fourteenth
   port, an adapter, or a mock target.
4. Add `internal/execx` to the effective package layout.
5. Supersede only revision 3's unconditional `no execx` statement; retain every
   unrelated architecture decision.

Keep the revision 3 architecture and its complexity review unchanged as
historical evidence. The code PR must not add `.journal/` files. Journal
branches remain outside the implementation and PR flow.

## Implementation sequence

Implement the cutover in one PR. Intermediate commits may compile partially, but the PR must contain no compatibility wrapper or second execution path.

### 1. Add and prove `internal/execx`

Add `doc.go`, `execx.go`, and `execx_test.go`.

The package tests must exercise observable process behavior:

- explicit path and PATH fallback resolution;
- argument, working-directory, and environment forwarding;
- stdout forwarding and nil discard;
- full live stderr forwarding plus exact retained-tail truncation;
- normal exit and nonzero exit-code extraction;
- start failure and error unwrapping;
- context cancellation;
- cancellation when a grandchild keeps the stderr pipe open, proving `WaitDelay` bounds `Wait`;
- nil-context and empty-executable rejection.

Reuse the repository's established process-fixture pattern: write an executable fixture once before parallel tests so Linux cannot hit `ETXTBSY` through inherited write descriptors.

### 2. Migrate consumers without changing their public options

Keep each existing `Options` type and constructor stable. Replace only its private process mechanics.

| File | `execx.Command` wiring | Behavior that must remain local |
|---|---|---|
| `internal/adapter/apko/composer.go` | `Program: "apko"`, configured path/environment/stderr, request directory, existing lock/build argv | request validation, lock-before-build order, subcommand-specific errors |
| `internal/adapter/cosign/signer.go` | `Program: "cosign"`, configured path/environment/stderr, sign argv | digest-ref validation and sign error text |
| `internal/adapter/cosign/verifier.go` | same program and streams, configured distribution directory, verify argv | verification request validation and payload-specific errors |
| `internal/adapter/ghup/replacer.go` | `Program: "gh"`, configured path/directory/stderr, token-applied environment, upload argv | token closure, inherited-token replacement, secret-safe errors |
| `internal/adapter/gitx/resolver.go` | `Program: "git"`, configured path/directory/environment/stderr, `bytes.Buffer` stdout | SHA parsing and unknown-tag classification |
| `internal/adapter/melange/builder.go` | `Program: "melange"`, configured path/environment/stderr, existing argv | compile/keygen/build order and subcommand errors |
| `internal/profile/goprof/goreleaser.go` | `Program: "goreleaser"`, configured path/environment/stdout/stderr, fixed release argv | dist validation, publication-disable arguments, ANSI removal from `RunError.StderrTail` |

For each consumer:

1. Replace direct `exec.CommandContext` and private `resolveBinary` calls with `execx.Run`.
2. Change the tool-specific error formatter to inspect `*execx.RunError` instead of `*exec.ExitError` and the private tail buffer.
3. Preserve context-error precedence and exact existing error prefixes.
4. Update Godoc links from `exec.CommandContext` and `exec.LookPath` to `execx.Run` semantics where appropriate.
5. Remove `os/exec` and `time` imports when no longer used.

### 3. Delete every duplicate helper

After all seven call sites use `execx.Run`, remove from the six consumer packages:

- `resolveBinary`;
- `tailBuffer`, `newTailBuffer`, and its methods;
- local `bytesPerKiB`, `stderrTailKiB`, `stderrTailLimit`, and `waitDelay` constants;
- repeated comments that explain `WaitDelay` internals.

Do not retain aliases, shims, deprecated helpers, or a fallback direct-exec path.

### 4. Rebalance tests around ownership

Keep consumer tests for consumer behavior:

- exact argv and invocation order;
- working-directory and environment wiring;
- PATH fallback selection for each tool name;
- domain validation before process start;
- stop-on-first-failure behavior;
- tool-specific error prefixes and exit-code presentation;
- GH token replacement and redaction;
- Git stdout parsing and tag classification;
- GoReleaser ANSI removal and live output forwarding.

Move shared-mechanism coverage to `internal/execx` and remove repeated consumer tests whose only contract is:

- 4 KiB tail-buffer implementation;
- orphan-grandchild `WaitDelay` behavior;
- generic start-error handling.

Retain lightweight consumer cancellation tests where they prove that the caller forwards its context and preserves `errors.Is`. Update GoReleaser tail tests to assert ANSI and bounded-output behavior without importing or duplicating a private `execx` limit.

Keep `internal/adapter/gitx/resolver_test.go`'s direct test-only `exec.Command` helper for temporary-repository setup. It is fixture infrastructure, not a production consumer.

### 5. Update code-facing documentation

- Add complete package and API Godoc for `internal/execx`.
- Update the six consumer package comments and method comments to describe execution through `execx.Run` without weakening their tool-specific contracts.
- Do not change user documentation or the changelog: this refactor has no intended user-visible behavior. If implementation reveals observable drift, stop and treat that drift as a separate decision rather than documenting it as part of this refactor.

## Verification

Run in the implementation worktree, in this order:

```bash
mise exec -- go test ./internal/execx -count=1

mise exec -- go test \
  ./internal/adapter/apko \
  ./internal/adapter/cosign \
  ./internal/adapter/ghup \
  ./internal/adapter/gitx \
  ./internal/adapter/melange \
  ./internal/profile/goprof \
  -count=1

mise exec -- moon run root:check

docker run --rm --cpus 4 \
  -v "$PWD:/src" -w /src \
  -e GOFLAGS=-mod=mod \
  golang:1.26 go test ./... -count=1
```

The direct `execx` tests are the behavioral smoke test: they launch a real child process and exercise the changed process boundary. The existing `gitx` temporary-repository tests retain one real-tool integration check.

Run structural checks after the behavioral checks:

- Production `os/exec` imports exist only under `internal/execx`.
- Production `exec.Command` and `exec.CommandContext` calls exist only under `internal/execx`.
- No consumer package defines `resolveBinary`, `tailBuffer`, `stderrTailLimit`, or `waitDelay`.
- No generated mocks change; the domain port interfaces are unchanged.
- `git ls-files .journal` prints nothing in the implementation worktree before PR creation.

## Acceptance criteria

The PR is complete when all of the following are true:

1. One `internal/execx.Run` implementation owns production subprocess execution.
2. All seven production call sites use it.
3. Existing public and internal consumer option shapes remain source-compatible.
4. Exact argv, ordering, environment, directory, stdout, stderr, context, and error behavior remains covered.
5. GH credentials cannot enter argv or formatted errors through the new layer.
6. The wrapper performs no retries and invokes no shell.
7. Duplicate execution helpers are deleted rather than retained behind aliases.
8. Targeted tests, `root:check`, and the Linux suite pass.
9. The architecture records why the earlier `execx` prohibition was rescinded.

## Risks and controls

- **Error text drift:** typed `RunError` exposes metadata; consumer formatters remain authoritative. Existing error assertions must pass unchanged.
- **Credential exposure:** `execx` stores no environment or argv in `RunError` and does not log either. GH token construction stays in `ghup`.
- **Cancellation regression:** one package-level orphan-grandchild test owns the five-second pipe-leak contract; consumer tests retain context forwarding where useful.
- **Hidden retry:** `execx.Run` performs one `cmd.Run` call. Retry remains only in the remote publication engines.
- **Over-broad abstraction:** no `Runner` interface, mock, command registry, shell parser, logger, telemetry, retry policy, or tool-specific branch belongs in `execx`.
- **Cross-platform fixture races:** write executable fixtures once before parallel tests, matching the Linux `ETXTBSY` lesson already applied across adapter suites.

## Integration

Create an isolated implementation worktree from fetched `main`, use the PR title `refactor(exec): centralize subprocess execution`, push normally, and integrate only through a GitHub squash merge. Do not use `wt merge` or commit journal files on the implementation branch.
