# release-cli — Initial Architecture

**Revision 2** — incorporates the adversarial architecture review (`local://architecture-review.md`); dispositions in §13.

Status: proposal for review before code. Repo: `meigma/release` (`github.com/meigma/release`, main @ `0c39bed`). This replaces ~90% of the bespoke logic in the four reusable workflows (`go-pre-publish`, `go-oci-build`, `publish-github-release`, `publish-oci-image`, ~1650 lines of YAML/bash/github-script) with one profile-driven Go CLI, keeping GitHub Actions as a thin orchestration shell. Deliberately thin where prototyping should teach us; flagged in §12.

---

## 1. Decisions up front

1. **One binary, `release-cli`, with stage verbs and a `--profile` flag** — verb-per-language was rejected by the user; ecosystems plug in as profiles, not commands. *(user requirement)*
2. **The profile seam is a Go interface ending at a versioned release manifest; everything downstream is ecosystem-neutral** — the inventory shows only PP-03–PP-08/OB-09–OB-11 are Go-specific. *(A1, inventory §3)*
3. **One logical release manifest with stage-scoped projections, copied into both existing artifacts** — `release-assets` and `oci-build-inputs` are deliberately disjoint (PP-09/PP-10); each downstream stage validates only its projection. Combined-artifact alternative rejected (§13). *(inventory invariant 4)*
4. **OCI publication is an explicit two-phase protocol: `publish oci prepare` → YAML `actions/attest` → `publish oci finalize`** — a single command cannot put attestations before tags while attest stays in YAML; invariant 14 requires prepare/finalize with an immutable publication receipt between. *(inventory OP-13–OP-19, invariant 14)*
5. **Pure core in `internal/rel`; every side effect behind a narrow, use-case-shaped, consumer-declared port — including slice 1's filesystem work (bundle scan, artifact reading, manifest store); no generic `Filesystem` port** *(A1, A2, P2)*
6. **Large services decompose into narrow ports implemented by one concrete adapter each** — `reg` implements `StateReader`/`ContentPusher`/`TagCommitter`; the GitHub adapters implement `DraftFinder`/`AssetReader`/`AssetReplacer`/`Publisher`. *(A2)*
7. **OCI registry ops use `oras-go` v2 natively; goreleaser/melange/apko/cosign stay pinned subprocesses; asset replacement keeps `gh release upload --clobber` behind a port** — libraries where fakes buy testability, subprocesses where the tool's exact behavior *is* the contract. GHCR-parity spike is the slice-3 acceptance gate, not polish. *(L1, R1; user requirement: local testability)*
8. **GitHub App token minting stays in YAML (`actions/create-github-app-token`) by default** — the CLI never touches the App private key; it receives a short-lived token via env. Native minting is a later spike only if it demonstrably shrinks exposure. *(actions-constraints §3)*
9. **Secrets are structurally unprintable** — an opaque `Secret` type redacts in `String`/`MarshalJSON`/error wrapping; the private key never enters the CLI at all under decision 8; `config show` renders only the effective non-secret config. *(E1 hygiene, review finding 8)*
10. **Domain terms get types with validating constructors: `Version`, `Tag`, `Digest`, `Channel`, `AssetName`, `ChecksumSet`…** — parse-time enforcement replaces ~10 hand-rolled regexes. A validated `Bundle` exists only after a scanner reconciles checksum claims against real files. *(I1)*
11. **Immutable facts are values; mutable intent is only ever a purely computed `Plan`/receipt an adapter applies from fresh remote state** — no stale plan is ever replayed; finalize re-resolves and refuses drift. *(A1, invariants 11–14)*
12. **stdout carries one stable versioned JSON envelope (`--json`) including `phase` and a per-mutation status list; human text on stderr; GitHub outputs via an `actenv` adapter; exit codes 0/1/2/3** — exit 3 means "transient failure, retries exhausted", **not** "safe to re-run"; each mutating command defines its own rerun reconciliation (§2). *(user requirement: machine consumption; review finding 6)*
13. **Handoff integrity has three explicit owners:** CLI `verify handoff` validates the API metadata tuple *before* download; SHA-pinned `actions/download-artifact` (`digest-mismatch: error`) owns transport-digest verification; CLI content commands verify the extracted contract. The CLI never claims to recompute the Actions artifact digest. *(inventory invariant 4; review finding 5)*
14. **CI delivery is layered: reusable workflows → one composite `setup-release-cli` action → CLI** — job permissions/environments/concurrency/`needs` only exist in reusable workflows; the composite is the per-job CLI bridge. *(actions-constraints §1, §7)*
15. **Dogfood via explicit `cli-path`:** the owning repo builds the branch binary in a prior step and passes its path; external consumers get the version stamped into the action at their pinned SHA. No magic `version: local`, no owner guard needed. *(actions-constraints §5)*
16. **Skew is prevented by a stamped protocol handshake, not by `$/` alone** — `$/` pins action *source* to the workflow commit; the workflow additionally stamps a required CLI protocol version, and the action rejects any installed/overridden CLI whose `version --json` protocol mismatches, before any side effect. Source-commit binding via provenance predicate is an experiment (§12). *(actions-constraints §4; review finding 7)*
17. **Bootstrap integrity: sha256 from `checksums.txt` + `gh attestation verify` against the distribution repo derived from `github.action_repository`** — nothing hardcodes `meigma/release`. *(user requirement: portability)*
18. **Nothing hardcodes org/repo/registry: defaults derive from `GITHUB_REPOSITORY`/git remote; slice 1 uses flags+env only; `release.toml` is introduced with the first durable knob (expected: slice 3 registry override)** *(user requirement: portability; agile mandate)*
19. **Three test layers: pure-core unit; engine tests with mockery mocks; system tests (testscript/txtar against stateful in-process fakes). Live coverage stays with the cross-repo rehearsal** — fakes prove wiring and decisions, not external contracts; the rehearsal is the T1 layer-3 proof. *(T1, T2, T3)*
20. **Migration is strangler-style, one step cluster per PR; slice 1 replaces PP-06–PP-08 only and scaffolds only the packages it needs** *(user requirement: agile; R1)*
21. **Every package ships `doc.go`; godoc on every function/type/field; 1000-line file cap** — the template currently violates this; we don't copy the omission. *(D1, D4, R2)*

---

## 2. Command surface

```
release-cli
  stage        --profile <name>            # stage 1: verify assets, select binaries, emit manifest
  image build                              # stage 2: melange+apko layout from manifest projection
  image verify                             # stage 2 verification only (OB-18..OB-21)
  publish oci prepare                      # stage 4b phase 1: plan, push by digest, verify, sign; emit receipt
  publish oci finalize                     # stage 4b phase 2: consume receipt, re-resolve, tag serially, verify
  publish github  [--no-undraft]           # stage 4a: verify bundle, upload/replace, converge, undraft
  plan tags                                # pure planning from remote state, zero writes
  verify handoff  --artifact-id --digest   # pre-download Actions artifact metadata tuple (OB-06/GR-05/OP-03)
  verify bundle                            # closed-set + checksum + signature validation (GR-09..GR-12)
  version                                  # includes protocol version and source commit
```

- **Verbs are pipeline capabilities**, ecosystem-neutral by construction; adding Rust never adds a verb.
- **A profile names a registered `Profile` implementation**; only `stage` takes it — ecosystem knowledge ends at the release manifest (§6).
- **Flags are run-scoped modifiers** (`--dist`, `--registry`, `--json`, `--dry-run` on mutating commands). Org-durable settings are config (§7).

**Two-phase OCI protocol (invariant 14).** `prepare` re-plans from live state, pushes blobs/manifests by digest, verifies resolution, recursively signs index+platform manifests, and writes an immutable versioned **publication receipt** (digests, tag plan inputs, attestation subjects). YAML runs the three `actions/attest` steps against subjects from the receipt. `finalize` consumes the receipt, **re-resolves exact and channel state fresh**, refuses any drift from the receipt's digests, applies tags serially, and verifies every resulting tag plus the exact tag. Trust metadata therefore strictly precedes public tags.

**Output contract (stable):** with `--json`, stdout emits exactly one document: `{"schema":"release.dev/result/v1","command":…,"ok":…,"phase":…,"mutations":[{"op":…,"target":…,"status":"applied|failed|unknown"}],"result":{…}}`. Envelope and `mutations` are stable from slice 1; command-specific `result` payloads stabilize as each command ships. Under Actions, outputs (`image-digest`, `release-url`, …) are written via `actenv` with today's names, so `workflow_call` plumbing survives. Annotations only when Actions is detected.

**Exit codes:** `0` success; `1` contract/verification failure (fail-closed); `2` usage/config error; `3` transient failure after bounded retries — **no rerun promise**; automation must consult `mutations`. Rerun reconciliation per command: `finalize` reruns always re-plan from fresh state (same version+digest ⇒ complete, ambiguous prior tag write ⇒ resolve-then-decide); `publish github` reruns recognize an already-public release with the exact converged expected asset set as success, otherwise report indeterminate (exit 1) for manual reconciliation — it never creates or re-drafts (invariant 1).

---

## 3. Domain model

Pure package `internal/rel`. No I/O. Key vocabulary (I1):

```go
type Version struct{ Major, Minor, Patch uint64 }    // canonical stable subset (OP-01)
func ParseVersion(s string) (Version, error)
func (v Version) Compare(o Version) int
func (v Version) Tag() Tag                            // "v1.2.3" — the only Tag constructor

type Tag string       // exact, immutable once published (invariant 11)
type Digest string    // "sha256:<64 lowercase hex>"; ParseDigest is the only constructor
type Channel string   // "latest" | "1" | "1.2"
func ChannelsFor(v Version) []Channel

type AssetName string // flat, no separators, closed-set member

// ChecksumSet is the *claim* parsed from checksums.txt: names and digests only.
// Strict grammar; duplicates and control self-listing rejected. (GR-10 grammar half)
func ParseChecksums(r io.Reader) (ChecksumSet, error)

// Bundle is the *proven*, closed, signed asset set. It is constructed only by
// reconciling a ChecksumSet against a BundleScan (streamed hashes, regular files,
// no unlisted entries) — never from checksum bytes alone. (invariant 7)
func NewBundle(claim ChecksumSet, scan BundleScan) (Bundle, error)

// CanonicalBinary carries what OB-09..OB-11 need: common program identity,
// platform, confined dist-relative source path, digest, executable-verified.
type CanonicalBinary struct{ Program Name; Platform Platform; Path RelPath; SHA256 Digest }

// Manifest is the profile→engine handoff (§6) with stage-scoped projections.
type Manifest struct{ Release Release; Bundle Bundle; Binaries []CanonicalBinary }
func (m Manifest) AssetsProjection() AssetsView   // validated by publish github
func (m Manifest) ImageProjection()  ImageView    // validated by image build / publish oci

type ChannelState struct{ Exists bool; Digest Digest; Version Version }
type TagPlan struct{ Exact ExactAction; Advance []Channel; Hold []Channel }
func PlanTags(v Version, d Digest, cur map[Channel]ChannelState) (TagPlan, error)

type Receipt struct{ … }   // publication receipt: versioned, immutable, digests + subjects
```

**Immutable facts vs mutable intent.** `Digest`, `Tag`, `Bundle`, `Manifest`, exact index *bytes* (retained verbatim, never re-marshaled — OP-06 hashes original bytes), `Receipt`: facts. Channel targets, draft state, registry tag map: intent/state, touched only through plans computed pure and applied from fresh remote reads.

**Enforced by types:** digest grammar, canonical version form, tag prefix, asset-name flatness, channel derivation, receipt immutability. **Runtime checks:** channel monotonicity/line scope (registry reads), exact-tag immutability (resolve), byte-identity (FS), draft existence and tag↔commit binding (GitHub + git), closed remote asset set (Releases API).

---

## 4. Ports

Ports are narrow, use-case-shaped interfaces declared in the consuming package (I2); one concrete adapter may implement several (A2 is about interface purpose, not package count). Mockery generates every mock into `internal/adapter/<name>/mocks/` (T2/T3). Error contracts (classified errors: absent vs retryable vs auth vs malformed vs ambiguous-write) are defined **with the slice that needs them**, not up front (review finding 9).

| Port (declared in) | Responsibility | Kind | Concrete adapter | Test double | Replaces |
|---|---|---|---|---|---|
| `profile.Profile` | Produce a `Manifest` for a tagged release | X | `goprof` (GoReleaser + selection) | mockery + fixtures | PP-05, PP-07/08 |
| `assets.BundleScanner` | Enumerate a dist dir into regular-file entries with streamed SHA-256; reject symlinks/dirs/unlisted | F | `bundlefs` | real `t.TempDir()` (+ generated mock) | PP-06, GR-10 closure half |
| `goprof.ArtifactsReader` | Read+parse `artifacts.json`, confine `dist/`-relative paths, verify executable bit; selection itself is a pure function over records | F | in `goprof` | temp-dir fixtures | PP-07/08, OB-09 read half |
| `manifest.Store` | Persist/load the versioned release manifest | F | `manfs` | temp dir | new (decision 3) |
| `assets.BlobVerifier` | Sigstore bundle verify over a blob, exact identity+issuer | X | `cosign` exec | mockery | GR-11 |
| `puboci.Signer` | Keyless recursive sign of digest-pinned refs (index + both platforms) | X+R | `cosign` exec | mockery | OP-15 |
| `image.APKBuilder` | melange keygen/compile/build of signed APK repos | X | `melange` exec | mockery | OB-12–OB-15 |
| `image.Composer` | apko lock/build of two-platform layout + SBOMs | X | `apko` exec | mockery | OB-16/17 |
| `image.LayoutSource` | Parse on-disk OCI layout into domain values, retaining exact index bytes | F | `layout` | temp dirs | OB-18–21, OP-05–07 read half |
| `puboci.StateReader` | Resolve tags, fetch version annotations | R | `reg` (oras-go v2) | in-memory registry + mockery | OP-10/12 reads |
| `puboci.ContentPusher` | Push blobs/manifests by digest, verify resolution | R | `reg` | in-memory registry | OP-13/14 |
| `puboci.TagCommitter` | Apply planned tags serially, verify each | R | `reg` | in-memory registry | OP-19 |
| `pubgh.DraftFinder` | Paginated draft-by-tag search with poll budget (24×5s preserved) | R | `ghrel` (go-github) | stateful fake + mockery | GR-06 |
| `pubgh.AssetReader` | List assets, digests, states (convergence poll 12×1s preserved) | R | `ghrel` | stateful fake | GR-14/16/17 |
| `pubgh.AssetReplacer` | Replace/upload *expected* assets only, clobber semantics | X | `ghup` (`gh release upload --clobber`) | mockery + stateful fake | GR-15, invariant 10 |
| `pubgh.Publisher` | Undraft + final-state fetch | R | `ghrel` | stateful fake | GR-18 |
| `pubgh.RefResolver` | Tag → commit in the checkout | X | `gitx` (git subprocess) | mockery | GR-07 |
| `verify.ArtifactMeta` | Pre-download Actions artifact metadata tuple (ID, expiry, run, digest) | R | `ghact` (go-github) | mockery | OB-06/GR-05/OP-03 |
| `cli.Notifier`/`cli.Outputs` | Actions detection, `GITHUB_OUTPUT`, annotations, summary | env | `actenv` | in-memory recorder | scattered output writes |
| `clock` (later slices) | Time/sleep for poll budgets | — | real | fake | GR-06/16 loops |

Shared subprocess plumbing (arg building, capture, mise-path assertion per PP-04/OB-05) lives in unexported `internal/execx` — used *by* exec adapters, not a port (a generic Runner would violate A2).

**Deliberately not ports (residual YAML; CLI verifies results):** checkout, mise install + tool-path proofs, QEMU/binfmt, artifact upload/download transport (decision 13), `actions/attest` execution (subjects from the receipt; success observed by the workflow, `verify` re-checks URL outputs are non-empty — never represented as "attestation succeeded" by the CLI), `actions/create-github-app-token` (decision 8). Native `reg` credentials are passed in memory to the client and never persisted, which intentionally obsoletes OP-09/OP-20 login/logout.

Nothing side-effecting remains in `internal/rel` or the engines.

---

## 5. Package layout

Extends the template's conventions (thin `main`, `internal/cli` wiring, injected streams/Viper). **Only slice-1 packages are created in slice 1**; the rest land with their slice. Every package gets `doc.go` (D4).

```
cmd/release-cli/          entrypoint: signals, streams, linker vars incl. protocol/commit, exit code
internal/cli/             Cobra tree, flag/env binding, --json envelope, actenv wiring        [slice 1]
internal/rel/             PURE CORE: Version/Tag/Digest/Channel/ChecksumSet/Bundle/Manifest/plans [slice 1]
internal/profile/         Profile port + registry + manifest schema                            [slice 1]
internal/profile/goprof/  Go profile: selection (pure) + ArtifactsReader + GoReleaser (slice 6) [slice 1, + mocks/]
internal/stage/assets/    engine: stage-1 orchestration                                        [slice 1]
internal/adapter/bundlefs/ dist-dir scanner                                                    [slice 1, + mocks/]
internal/adapter/manfs/   manifest store                                                       [slice 1, + mocks/]
internal/adapter/actenv/  Actions runtime env                                                  [slice 1, + mocks/]
internal/verify/          handoff/bundle verification engines                                  [slice 2]
internal/adapter/ghact/   Actions artifact metadata                                            [slice 2, + mocks/]
internal/stage/puboci/    prepare/finalize engines                                             [slice 3]
internal/adapter/reg/     oras-go v2 (implements StateReader/ContentPusher/TagCommitter)       [slice 3, + mocks/]
internal/adapter/cosign/  cosign exec (sign, verify-blob)                                      [slice 3, + mocks/]
internal/stage/pubgh/     GitHub publish engine                                                [slice 4]
internal/adapter/ghrel/   go-github reads/undraft                                              [slice 4, + mocks/]
internal/adapter/ghup/    gh release upload --clobber                                          [slice 4, + mocks/]
internal/adapter/gitx/    git subprocess                                                       [slice 4, + mocks/]
internal/stage/image/     image build engine                                                   [slice 5]
internal/adapter/melange/ + internal/adapter/apko/ + internal/adapter/layout/                  [slice 5, + mocks/]
internal/config/          release.toml (first durable knob)                                    [~slice 3]
internal/execx/           shared subprocess plumbing                                           [with first exec adapter]
internal/clock/           poll clock                                                           [slice 2/4]
e2e/                      testscript .txtar system tests + stateful fakes
.mockery.yml              per-interface config emitting into adapter mocks/ (T2/T3)
```

`cmd/release-mvp` (placeholder greet CLI) is deleted when slice 1 makes `release-cli` the repo's release artifact. Names are prototyping candidates (§12).

---

## 6. Profile abstraction

**Mechanism: compiled-in Go interface + per-profile config section. Not plugins, not declarative step lists.**

```go
type Profile interface {
    Name() rel.ProfileName
    // Stage builds/verifies the ecosystem's release assets and returns the
    // complete Manifest. In slices 1–5 it verifies GoReleaser output already
    // produced by YAML (PP-05 stays residual); slice 6 moves the invocation in.
    Stage(ctx context.Context, r rel.Release, dist Dir) (rel.Manifest, error)
}
```

**The profile owns** exactly the inventory's Go-specific set: toolchain invocation (eventually), archive/SBOM naming, `artifacts.json` schema knowledge, canonical-binary selection with path confinement and executable checks (PP-05/07/08, OB-09, OB-10's input half). **The engine owns** tag gating, checksum/bundle verification, packaging, layout verification, publication, signing subjects, channel policy — all marked ecosystem-neutral by the inventory.

**The seam is the release manifest** (decision 3): one logical `Manifest`, serialized once, **copied into both existing artifacts** (`release-assets` and `oci-build-inputs` keep their current disjoint payloads; PP-09/PP-10 shapes and outputs unchanged). Each downstream stage validates only its projection — `publish github` requires the bundle files it references to exist in its download; `image build`/`publish oci` require the binaries/layout side. Missing files outside the active projection are structurally validated but not resolved. This keeps invariant 4's per-handoff independence and avoids shipping archives to jobs that don't need them.

**Image seam contract v1 is exactly today's:** one common static ELF per {linux/amd64, linux/arm64}, one package per arch, two manifests, one entrypoint, user 65532 (invariant 6, OB-09–21). A profile that cannot meet it may disable the image stages via config; a profile needing a *different* image input shape requires a new manifest projection version and image-engine work — that is future evidence-driven design, not a current promise. A Rust profile that produces two static Linux binaries plugs in with zero engine changes; that is the honest portability claim.

Why not declarative-only profiles: staging needs conditional logic, tool-schema parsing, and error judgment — YAML-encoded steps rebuild a worse bash. Why not plugin binaries: distribution and skew cost with no third-party-profile requirement.

---

## 7. Configuration and org portability

**Slice 1: flags + env only** (`RELEASE_*`, template-style injected non-global Viper). **`release.toml` is introduced with the first durable knob** — expected at slice 3 (registry override) — with precedence flag > env > file > derived default. `config show`/`config validate` land with the file, and `config show` renders only effective non-secret values (decision 9).

**Derived defaults do the portability work:** registry defaults to `ghcr.io/<owner>/<repo>` lowercased from `GITHUB_REPOSITORY` (git remote fallback); signing identity defaults to `<repo>/.github/workflows/go-pre-publish.yml@<ref>` from context; channel policy defaults to today's exact/minor/major/latest monotonic rules; the CLI distribution repo derives from `github.action_repository`. Nothing names `meigma`.

**A second org's full adoption delta:** install the GitHub App on the org; set `RELEASE_APP_CLIENT_ID` (Actions variable) and `RELEASE_APP_PRIVATE_KEY` (Actions secret) — consumed by the *YAML* token action, never the CLI; optionally:

```toml
# release.toml — usually empty
[oci]
registry = "ghcr.io/otherorg/tool"   # only if not the repo's own GHCR path
```

**Validation is fail-fast:** every command validates inputs, env, manifest schema, and (for the selected profile) tool availability before any side effect; sentinel `ErrConfig`, exit 2 (E1).

---

## 8. CI integration shape

**Recommendation: the layered shape — reusable workflows → composite `setup-release-cli` → CLI** — evaluated against, and agreeing with, the constraints doc: job-scoped permissions, environments, concurrency, `needs`, matrices, and workflow outputs exist only in reusable workflows; a composite is the only supported same-job bridge for a caller-built binary and for orchestration-owning consumers.

**Layer ownership.** Reusable workflows own the job graph, `needs` (including the registry-before-release gate, invariant 3), concurrency groups (invariants 13/16), timeouts, environment gates, artifact transport, permissions declarations, the token-minting and `actions/attest` steps, and the prepare→attest→finalize sequencing. The composite owns CLI acquisition/verification and exposes `cli-path`. The CLI owns decisions, verification, and side effects via ports.

**Permissions/OIDC:** callee jobs declare least-privilege exactly as today; the external caller's call job must grant the same ceiling (callee can never elevate — documented constraint). Attestations attach to the *consumer* repo (caller context), as desired. Consumer copy-paste surface: one caller stub of four `uses:` jobs, `permissions`, two credential mappings, optional `cli-version`.

**Dogfood and skew, end to end:**

```yaml
# inside the reusable workflows (all consumers execute):
- uses: $/.github/actions/setup-release-cli        # $/ = same-commit action source
  with:
    version:  ${{ inputs.cli-version }}            # optional consumer override
    cli-path: ${{ inputs.cli-path }}               # dogfood: path to a caller-built binary
```

- **Dogfood:** this repo's `release.yml` builds `./cmd/release-cli` from the branch checkout in a prior step and passes `cli-path`. Explicit, visible source; no owner-guard magic. Because the binary is job-local, each callee job that needs it builds or downloads it — the workflow input threads through.
- **External default:** `action.yml` carries a stamped default CLI version (release-please `extra-files`), so an external caller pinning the workflow `@FULL_SHA` transitively pins action source (`$/`) and default CLI version. **This aligns source pins; it does not by itself prove compatibility** — a caller can override `cli-version` with a stale binary.
- **Protocol handshake (the actual skew defense):** the CLI stamps a `protocol` integer and source `commit` via linker vars, reported by `version --json`. The workflows stamp their required protocol; `setup-release-cli` asserts equality **before any stage command runs** and fails closed on mismatch, including explicit handling of a blank `version` input (blank ⇒ stamped default; the action implements this, never relying on metadata defaults after an explicitly-passed empty value).
- **Bootstrap integrity (released path):** download archive + `checksums.txt`; verify sha256; `gh attestation verify --repo <derived from github.action_repository>` against the pinned signer workflow; then the protocol check. Binding the binary to the exact workflow commit via the provenance predicate's source commit is a slice-1-adjacent experiment (§12) — until proven, the claim is "signed, version-pinned, protocol-compatible", not "same-commit".

---

## 9. Preserved invariants

| Invariant (inventory §6) | Enforced by | Local test |
|---|---|---|
| 1 Draft upstream, publisher never creates | `DraftFinder` returns `ErrNoDraft`; engine has no create path; reruns never re-draft | mock absent/non-draft → exit 1 |
| 2 Tag/run binding, tag→`github.sha` | `verify` engine + `RefResolver`; `rel.Release` constructor | unit + gitx temp-repo test |
| 3 Registry-before-release gate | **residual YAML** caller `needs` + `require-oci-image` input (unchanged) | workflow lint only |
| 4 Two-coordinate handoff | split ownership: `verify handoff` (metadata tuple, pre-download) / `download-artifact` (transport digest) / CLI content commands | unit digest normalization; `ghact` mock; content e2e |
| 5 No-rebuild byte identity | `stage/image` engine compares extracted layer bytes' digest to manifest `CanonicalBinary` | temp-dir layout with mutation → fail |
| 6 Canonical platform/runtime shape | `rel` layout validation over `LayoutSource` (exact index bytes retained) | table-driven units on synthetic layouts |
| 7 Closed, signed bundle | `ParseChecksums` (grammar) + `NewBundle` reconciliation over `BundleScanner` + `BlobVerifier` identity/issuer | pure grammar units; scanner temp-dir tests; cosign mocked |
| 8 Attestation subject shape | receipt carries subjects; YAML `actions/attest` executes; workflow sequencing `verify bundle` → attest → upload (GR-13) and prepare → attest → finalize (OP-16–18) | unit on subject derivation; live rehearsal for attest itself |
| 9 Draft-until-verified | `pubgh` engine order verify→replace→converge→publish; `--no-undraft` | engine mock call-order test; stateful fake e2e |
| 10 Converge only over expected names | `AssetReplacer` clobber contract: replace expected, refuse unexpected, never delete unexpected | stateful fake: 422, delete-success/upload-fail, duplicates, unexpected names |
| 11 Exact tags immutable | `rel.PlanTags` pure: absent→create, same→accept, differ→error | pure unit table |
| 12 Channels monotonic, line-scoped | `rel.PlanTags` over fresh `StateReader` reads at both prepare and finalize | pure unit table |
| 13 Channel mutation serialized | **residual YAML** concurrency (reusable-workflow consumers only) + serial `TagCommitter`; direct-CLI users on shared targets are a documented single-writer limitation | e2e serial-order assert; docs |
| 14 Trust metadata before public tags | **two-phase protocol**: prepare (push/verify/sign) → YAML attest → finalize (fresh re-resolve, refuse drift, tag) | engine order tests; e2e across both phases with in-memory registry |
| 15 Verification-only writes nothing | `--dry-run` + `plan tags`/`image verify` short-circuit before mutating ports | e2e asserting zero registry mutations |
| 16 No-cancel concurrency | residual YAML (irreducible) | workflow lint |
| 17 Producer publication disabled twice | residual YAML until slice 6; then `goprof` passes `--skip=publish` and asserts `release.disable: true` in config | goprof unit on invocation args |

---

## 10. Testing strategy

**T1 layer 1 — unit (pure core).** Version ordering, tag/digest grammar, channel derivation, `PlanTags`, checksum grammar, bundle reconciliation logic, layout predicates, subject derivation. Zero I/O, sub-second. Invariants 6, 7, 11, 12 live here.

**T1 layer 2 — engine + adapter integration.** Engines against mockery mocks (T2/T3) for ordering, short-circuiting, dry-run, bounded retry decisions once each slice defines its classified errors. Adapters against real cheap substrates where they exist: `reg` against an in-memory registry (`go-containerregistry pkg/registry` via `httptest`), `bundlefs`/`manfs`/`layout`/`gitx` against `t.TempDir()`, `actenv` against temp output files. The GitHub fake must be **stateful** — pagination, delayed draft visibility, delayed asset digest convergence, expected-name replacement, duplicates, 429/5xx, ambiguous mutations — recorded rehearsal fixtures seed schemas but are not the behavior model.

**T1 layer 3 — system tests + live.** `e2e/` testscript/txtar drives the built binary through stage flows against the in-process fakes — this proves CLI wiring, plans, ordering, and local verification, **not** GHCR, Sigstore, Actions, or pinned-tool contracts. Genuine layer-3 lives in the existing cross-repo rehearsal: keyless signing, `actions/attest`, GHCR referrers/auth, QEMU/Docker melange/apko, GoReleaser 2.17.1's real `artifacts.json`. Known mock-passes-tool-fails gaps (cosign flags/recursion, melange arg order, apko path context, GoReleaser schema drift, clobber semantics) are covered by pinned-tool contract tests where cheap and by rehearsal otherwise.

**The laptop answer:** `moon run test` runs layers 1–2 in seconds; `go test ./e2e` adds full-flow system tests in seconds more. The engineer validating a tag-policy or verification change runs one test instead of a 20-minute cross-repo rehearsal; the rehearsal shrinks from "the only validation" to "the final smoke".

---

## 11. Migration path

Strangler pattern; workflow-input compatibility for consumers throughout. YAML burn-down from ~1650 lines shown per slice.

1. **Slice 1 (one PR).** Contents, exactly: `internal/rel` (Version/Tag/Digest/ChecksumSet/Bundle/Manifest + projections), `profile`+`goprof` (pure selection + `ArtifactsReader`; GoReleaser stays YAML), `stage/assets` engine, `bundlefs`/`manfs`/`actenv` adapters + mocks, the `--json` envelope, `stage --profile go` replacing PP-06–PP-08's bash (~80 lines), manifest emission added to both residual uploads, and a minimal `setup-release-cli` (install+verify+protocol check; `cli-path` passthrough). Dogfood: `release.yml` builds the branch binary, passes `cli-path`. Validated by: unit tests, txtar, and a rehearsal run specifically confirming real pinned-GoReleaser `artifacts.json` shape and uploaded artifact contents. Replaces `cmd/release-mvp` as the release artifact. Everything else untouched.
2. **Handoff verification:** `verify handoff` + `ghact` (+ classified-error model for it) replaces the four `getArtifact` script blocks (~150 lines). Transport digest stays with `download-artifact` (decision 13).
3. **OCI publish:** `plan tags`, `publish oci prepare|finalize`, `reg`+`cosign` adapters, registry error taxonomy, `release.toml` introduction. Replaces OP-09–15, OP-19/20 (~450 lines); YAML keeps attest steps between the phases. **Acceptance gate: the scratch-GHCR parity spike** (digest push, tag resolve, referrers, partial-failure behavior) — not optional polish.
4. **GitHub publish:** `publish github` with `ghrel`/`ghup`/`gitx`; token minting stays YAML. Replaces GR-05–GR-12, GR-14–GR-18 (~380 lines); **GR-13 (`actions/attest`) remains YAML**, sequenced verify → attest → upload. Stateful-fake e2e incl. clobber/rerun cases; rehearsal with `publish-release: false` first.
5. **Image build:** `image build|verify` with `melange`/`apko`/`layout` replaces OB-08–OB-21 (~350 lines). Temp-dir verification e2e; rehearsal for real builds.
6. **Prepublish completion:** `goprof` invokes GoReleaser (PP-05, ~50 lines; invariant 17 moves in); workflows collapse to thin shells; docs/reference contracts updated (D6).

**Irreducible residual YAML (~250–300 lines):** triggers, permissions, job graph/`needs`/concurrency/timeouts, environment gates, checkout, mise install + tool-path proofs, QEMU/binfmt, artifact transport, `actions/create-github-app-token`, `actions/attest` steps, the composite action, caller stubs.

---

## 12. Risks, unknowns, expected wrongness

**Expect prototyping to invalidate:** the manifest projection schema (versioned, internal until slice 5); engine/package names (settle after slice 3); the receipt format for prepare/finalize; whether the stateful GitHub fake earns its keep versus more mock-driven engine tests.

**Load-bearing assumptions → cheap experiments, in order:**
1. `$/` self-reference under external SHA-pinned callers on current runners (≥ 2.336.0) — scratch consumer repo, **before slice 1's action ships**.
2. release-please can stamp `action.yml` (`extra-files`) — slice 1.
3. Provenance predicate exposes a verifiable source commit to bind binary↔workflow commit — slice-1-adjacent spike; until proven, §8's claim stays "signed + protocol-compatible".
4. `oras-go` v2 GHCR parity incl. referrers alongside cosign — slice-3 acceptance gate; fallback is the ORAS binary behind the same three narrow ports with all decisions staying pure.
5. `sigstore-go` offline bundle verification (removes network TUF from GR-11's local path) — after slice 4.
6. Absorbing `actions/attest` into the CLI (would collapse prepare/finalize to one command) — only if a `sigstore-go` + attestations-API spike is small and preserves permission/identity semantics.
7. Native App-token minting — only if a spike shows a smaller exposure surface than the pinned action; the default stands (decision 8).

**Considered and rejected:** verb-per-ecosystem CLI (user-rejected; multiplies contracts); declarative profiles (rebuilds bash in TOML); plugin-binary profiles (skew/distribution cost, no requirement); JavaScript/Docker action (Node bundle or Linux-only, no job-level gain — constraints §1); reusable-workflow-only delivery (loses same-job dogfood bridge); melange/apko via Go APIs (huge surface, zero contract gain); one combined producer artifact (breaks per-handoff independence, ships archives to jobs that don't need them — §13 F3/A6); single-command `publish oci` (cannot preserve invariant 14 — review finding 1).

---

## 13. Review dispositions

| Finding | Disposition | Note |
|---|---|---|
| 1 attestation-before-tags broken | **Accept** | Two-phase `prepare`/`finalize` + immutable receipt (§2, decision 4). |
| 2 missing slice-1 FS ports | **Accept** | `BundleScanner`, `ArtifactsReader`, `manifest.Store`; no generic FS port (§4). |
| 3 staging/two-artifact mismatch; `ParseChecksums` overreach; container-profile claim | **Accept** | One manifest, projection-validated, copied into both artifacts (decision 3); `ChecksumSet` vs scanner-built `Bundle`; `Asset.Size` dropped; image seam v1 = today's exact contract (§6). |
| 4 service-surface ports | **Accept** | Narrow consumer ports, one concrete adapter each (§4, decision 6). |
| 5 transport-digest ownership | **Accept** | Three-owner split (decision 13). |
| 6 "safe to re-run" exit 3 | **Accept** | Removed; `mutations` status + per-command reconciliation (§2). |
| 7 `$/` skew overclaim; hardcoded repo | **Accept** | Protocol handshake + stamped commit; repo derived from `github.action_repository`; same-commit binding demoted to experiment (§8). |
| 8 App token one-shot/exposure | **Accept** | Token action stays YAML by default; `Secret` type; `config show` redaction (decisions 8–9). Note: with jobs ≤ 15-min timeouts, the 1-hour token makes mid-run expiry a residual, documented risk rather than a modeled state. |
| 9 retry/error taxonomy | **Accept as scoped** | Defined per remote slice with the documented poll budgets (24×5, 12×1); no upfront framework — matches the review's own guidance. |
| 10 clobber semantics | **Accept** | `gh release upload --clobber` retained behind `AssetReplacer`; go-github for reads/undraft (§4). |
| Alt: keep ORAS binary | **Reject (with hedge)** | Native `oras-go` stands for testability, but the scratch-GHCR spike is promoted to slice-3 acceptance; the fallback keeps the same ports (§12.4). |
| Alt: combined producer artifact | **Reject** | Projections preserve invariant-4 independence and download minimality (decision 3). |
| Alt: no composite in slice 1 | **Partially reject** | The reusable workflows execute for external consumers too, so slice 1 needs the install path; the composite ships minimal (`cli-path` + install + protocol check). |
| e2e naming overstates coverage | **Accept** | Renamed system tests; rehearsal is the true layer 3 (§10, decision 19). |
| Slice-1 descoping (config file, package scaffold, magic `local`) | **Accept** | Flags/env only; slice-scoped packages; explicit `cli-path` (decisions 15, 18, 20). |