---
id: 003
title: Session start
started: 2026-08-18
---

## 2026-08-18 21:28 — Kickoff

Goal for the session: not yet stated. The developer asked to start a new
session; the actual request follows.

Current state of the world:

- `main` is at `2c262ba` ("fix(release): grant attestations read on the
  oci-image call job (#8)"), clean, in sync with `origin/main`.
- `v0.1.0` is published and the dogfood loop is closed: the tag run builds
  `release-cli` from the tagged commit and that binary stages and verifies its
  own release.
- Session 002 delivered the approved architecture
  (`.journal/002/ARCHITECTURE.md`, revision 3) and the eleven-PR plan
  (`.journal/002/PLAN.md`, which also holds the standing per-PR execution
  method and the three spike results). PRs 1 and 2 of that program are merged.
- Next planned slice is PR 3: `plan tags` with `internal/rel`, `StateReader`,
  and the `reg` read path. Spike B already cleared its gate.
- Known open threads carried in: release-please PR #9
  (`chore(main): release 0.1.1`) is open and harmless; archived scratch repos
  and the `spike/self-ref` branch cannot be deleted without `delete_repo`; the
  dead `release-please--branches--main--components--release-mvp` branch is
  removable; consumer docs and `examples/go-release/` still pin
  `fb8c809`, which PR 11 replaces.

Plan: wait for the developer's stated goal. If it is the release program,
re-read `.journal/002/PLAN.md` and `ARCHITECTURE.md` before touching code and
follow the standing per-PR method.


## 2026-08-18 21:55 — PR 3 underway

Goal set: execute PR 3 of `.journal/002/PLAN.md` — `feat(oci): plan immutable release tags`.

Worktree `.wt/feat-release-cli-slice3` created from fetched `origin/main`
(`2c262ba`), branch `feat/release-cli-slice3`.

Contract decided before spawning any agent (this is the cross-agent
interface; wave agents code against it verbatim):

- `internal/rel`: `Version`/`ParseVersion`/`Compare`/`Tag`, `Digest`,
  `Tag`/`ParseTag`, `Scope` (exact/minor/major/latest), `Channel`,
  `ChannelsFor`, `TagState`, `ChannelState`, `Action`
  (create/accept/retain), `Decision`, `TagPlan.Apply`, `PlanTags`,
  sentinels `ErrImmutableTag`/`ErrChannelCorrupt`/`ErrStateIncomplete`,
  and a redacting `Secret`.
- `internal/stage/puboci`: `Image`/`ParseImage`, `Reference`, sentinels
  `ErrTagAbsent`/`ErrRetryable`/`ErrCorruptState`, the `StateReader` port,
  `CollectState`, `PlanTags`.
- `internal/adapter/reg`: oras-go v2 read client (`New`, `Resolve`,
  `Version`) with `Credentials{Username, Password rel.Secret}`.
- CLI: `plan tags [--image] [--version] --digest [--json]`, derived
  defaults from `GITHUB_REPOSITORY`/`GITHUB_REF_NAME`, envelope command
  `plan tags`, result `{image, version, digest, tags, decisions[]}` where
  `tags` is only the `create` decisions.

Deliberate divergences from the YAML being replaced, both fail-closed:
version components must fit `uint64` (the JS used BigInt), and
`CollectState` propagates a precise corrupt-annotation error instead of
handing the planner a `HasVersion=false` state.

Execution: wave 1 = `RelCore` (internal/rel) + `ContractDocs`
(docs/reference). Both complete. `ContractDocs` caught a real
inconsistency in my example payload — I had listed all four tags under
`tags` while two decisions were accept/retain; corrected to create-only.
Wave 2 in flight: `Puboci` (engine + port + mockery), `Reg` (adapter +
go.mod tidy), `CliTags` (command + root wiring + main).

Dependencies pre-added by the parent so agents do not race on `go.mod`:
`oras.land/oras-go/v2 v2.6.2` (spike B's pinned version) and test-only
`github.com/google/go-containerregistry v0.21.9`.