# Shared subprocess execution amendment

Status: approved for implementation on 2026-08-19

This document amends only the `internal/execx` deferral in
`.journal/002/ARCHITECTURE.md` revision 3. Every unrelated decision and review
disposition remains binding. The revision 3 document remains unchanged as the
historical architecture baseline.

## Trigger

Revision 3 deferred `internal/execx` until two subprocess adapters repeated
meaningful process plumbing beyond `exec.CommandContext` setup. The repository
now has six packages and seven production call sites that independently repeat:

- deferred `exec.LookPath` resolution;
- context-bound child-process construction;
- stdout discard or forwarding;
- live stderr forwarding plus a bounded 4 KiB tail;
- a five-second `exec.Cmd.WaitDelay` for leaked grandchild pipes;
- context-error precedence and exit-code extraction;
- the same tail-buffer implementation.

The six packages are `apko`, `cosign`, `ghup`, `gitx`, `melange`, and `goprof`.
The trigger has fired.

## Decision

Add `internal/execx` as shared local stdlib plumbing. It is not a domain port,
an adapter, a mock target, or a fourteenth entry in the port budget.

`execx` owns only process mechanics:

- select an explicit path or default program name and resolve it with
  `exec.LookPath` at invocation time;
- invoke one explicit argument slice through `exec.CommandContext`;
- apply the caller's directory, environment, stdout, and stderr writers;
- retain the trailing 4 KiB of stderr while forwarding the full live stream;
- apply a five-second `exec.Cmd.WaitDelay`;
- return typed run failure metadata without logging or retaining argv or the
  environment.

The package exposes a stateless `Run` function, a value `Command`, and a typed
`RunError`. It does not expose a stateful `Runner`, interface, retry policy,
shell command string, logger, telemetry hook, or configurable process-policy
framework.

## Responsibility boundary

The existing consumer-owned ports and concrete tool adapters remain unchanged.
Each consumer continues to own:

- domain validation;
- exact argv construction and multi-command ordering;
- tool-specific working-directory and environment decisions;
- credential handling and redaction;
- domain error prefixes and classifications;
- child-output parsing and tool-specific transforms;
- retry decisions.

`execx` performs exactly one process attempt. In particular, it never retries
Melange, apko, or GoReleaser because their local writes are stateful and
non-idempotent.

## Effective package layout

The revision 3 package layout gains:

```text
internal/execx/            shared subprocess mechanics; no port, adapter, or mock
```

This addition supersedes revision 3's unconditional `no execx` statement. The
original deferral was correct when written; the measured duplication now
satisfies its named extraction trigger.

## Verification boundary

`internal/execx` tests own path resolution, stream routing, bounded stderr,
exit metadata, cancellation, and leaked-grandchild `WaitDelay` behavior.
Consumer tests continue to own argv, order, environment, credentials, output
parsing, and domain error behavior. Production `os/exec` imports and command
construction must exist only in `internal/execx` after the clean cutover.
