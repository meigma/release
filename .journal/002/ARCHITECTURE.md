# release-cli — Initial Architecture

**Revision 3** — incorporates the adversarial correctness review (`local://architecture-review.md`) and the complexity review (`local://complexity-review.md`), both adjudicated by the owner; dispositions in §13.

Status: proposal to build from. Repo: `meigma/release` (`github.com/meigma/release`, main @ `0c39bed`). One profile-driven Go CLI replaces the bespoke decision logic in the four reusable workflows (`go-pre-publish`, `go-oci-build`, `publish-github-release`, `publish-oci-image`); GitHub Actions stays a thin orchestration shell. Precise scope: **42 of 70 logical steps, an upper bound of ~1,059 of 1,641 lines (~64.5%)**, migrate into the CLI (§11); essentially all embedded bash/github-script *decision* logic moves, but the whole-file "~90%" figure is retired. Deliberately underspecified where prototyping should teach us; every deferral carries an observable trigger.

---

## 1. Decisions up front

1. **One binary, `release-cli`, with stage verbs and a `--profile` flag** — verb-per-language rejected; ecosystems are profiles, not commands. *(user requirement)*
2. **The profile seam is the `goprof` package boundary with direct dispatch on `--profile go`** — no interface, registry, `Name()`, or profile mock while one implementation exists. `--profile` stays the public shape; unknown values are usage errors (exit 2). **Trigger to add the interface/registry: a second real ecosystem profile reaching an implementation PR.** *(inventory §3: only PP-03–PP-08/OB-09–OB-11 are Go-specific)*
3. **Slice 1 persists nothing.** PP-09/PP-10 uploads stay byte-for-byte as today; no manifest, projections, store, or artifact edits. **Trigger for a persisted staged-facts contract: the first downstream CLI command that reads staged facts — that PR serializes only the projection that artifact's consumer validates, never a superset referencing absent files.** *(inventory invariant 4 stays on today's artifacts)*
4. **OCI publication is two-phase: `publish oci prepare` → YAML `actions/attest` → `publish oci finalize`** — attest needs job-level OIDC/attestation permissions only YAML can hold (actions-constraints §2), and invariant 14 requires trust metadata before public tags. The **`OCIPrepareResult`** (digests, tag-plan inputs, attestation subjects) is not a file: it travels as the `--json` envelope `result` that the workflow re-feeds to `finalize` on stdin (`--result -`). The transport is the envelope we already need; there is no persistence port and nothing to clean up. *(inventory OP-13–OP-19, invariant 14)*
5. **Pure core; A1 is satisfied by stdlib interfaces for deterministic local plumbing.** Rule resolution, stated once and binding: **A1 requires a boundary, not a bespoke boundary.** `fs.FS`/`io.Reader`/`io.Writer` are the ports for local file work; R1/R3 win there. Custom consumer-owned ports exist only at remote, subprocess, credential, and irreversible-mutation boundaries, where A1/A2 win (§4).
6. **Narrow use-case ports, one concrete adapter implementing several** — `reg` implements `StateReader`/`ContentPusher`/`TagCommitter`; `ghrel` implements `ReleaseReader`/`Publisher`. *(A2)*
7. **`oras-go` v2 natively for registry ops; goreleaser/melange/apko/cosign stay pinned subprocesses; asset replacement keeps `gh release upload --clobber` behind `AssetReplacer`** — libraries where fakes buy testability, subprocesses where the tool's exact behavior *is* the contract. The scratch-GHCR parity spike is the slice-3 acceptance gate. *(L1, R1; invariant 10)*
8. **GitHub App token minting stays in YAML (`actions/create-github-app-token`)** — the CLI never touches the App private key. *(actions-constraints §3)*
9. **Secrets are structurally unprintable** — opaque `Secret` type redacting in `String`/`MarshalJSON`/error wrapping. *(E1; correctness finding 8)*
10. **Domain types with validating constructors are introduced with the slice that uses them** — I1 applies to terms actually in use, not the eventual vocabulary. Slice 1 needs checksum/selection facts; `Version`/`Digest`/`ChannelState`/`TagPlan` land in slice 3 (§3).
11. **Immutable facts are values; mutable intent is only a purely computed plan applied from fresh remote state** — no stale plan is replayed; `finalize` re-resolves and refuses drift. *(invariants 11–14)*
12. **stdout carries one versioned JSON document under `--json`: `{"schema","command","ok","result"}`; human text on stderr; exit codes 0 (success) / 1 (contract or verification failure, fail-closed) / 2 (usage/config).** No `phase`, no `mutations`, no exit 3. There is **no generic safe-rerun promise**; each mutating command reconciles from fresh remote state (§2). **Trigger for a stable mutation-status report and a retry-exhausted exit code: the first ambiguous remote write plus a caller that programmatically branches on it.** *(correctness finding 6, narrowed — see §13)*
13. **Handoff integrity has three owners:** CLI `verify handoff` validates the API metadata tuple before download; SHA-pinned `actions/download-artifact` (`digest-mismatch: error`) owns transport-digest verification; CLI content commands verify the extracted contract. The CLI never claims to recompute the Actions artifact digest. *(invariant 4; correctness finding 5)*
14. **CI delivery is layered: reusable workflows → one composite `setup-release-cli` → CLI** — job permissions/environments/concurrency/`needs`/timeouts/artifact transport exist only in reusable workflows; the composite is the per-job CLI bridge. *(actions-constraints §1)*
15. **The workflows, the composite action, and the CLI are one release unit with one version and one consumer pin.** `cli-version` does not exist; `cli-path` is an unsupported escape hatch; a stamped-integer guard catches mismatches (§8). *(release-please-config.json:11-16; replaces rev-2 decision 16)*
16. **Bootstrap integrity: sha256 from `checksums.txt` + `gh attestation verify` against the repo derived from `github.action_repository`** — nothing hardcodes `meigma/release`. *(portability)*
17. **Nothing hardcodes org/repo/registry: flags + env + derived defaults only.** Registry defaults to `ghcr.io/<owner>/<repo>` lowercased from `GITHUB_REPOSITORY` (git-remote fallback); signing identity and distribution repo derive from context. **Trigger for `release.toml` (and, with it, `config show`/`config validate`): a real adopter needing a non-derived registry target, or a second durable setting.** The inventory found zero current consumers of either (inventory "negative findings").
18. **Three evidence layers, not five harness modes:** pure unit tests; integration on cheap real substrates (`t.TempDir()`, `fstest.MapFS`, in-memory registry) or mockery mocks for genuine adapters; live cross-repo rehearsal as the T1 layer-3 proof. No testscript/txtar, no `e2e` package. *(T1–T3)*
19. **Strangler migration, one step cluster per PR; slice 1 is four production packages replacing PP-06–PP-08** (§11).
20. **`doc.go` + full godoc + mockery obligations apply to every package and genuine adapter that survives** — fewer packages, not lower standards for the ones that exist. *(D1, D4, T2, T3)*

---

## 2. Command surface

```
release-cli
  stage        --profile go --dist PATH [--json]   # slice 1: PP-06..08
  verify handoff  --artifact-id --digest           # slice 2: OB-06/GR-05/OP-03 metadata tuple
  publish oci prepare  [--dry-run]                 # slice 3: plan, push by digest, verify, sign
  publish oci finalize --result -                  # slice 3: re-resolve fresh, refuse drift, tag serially
  plan tags                                        # slice 3: pure planning, zero writes
  publish github  [--no-undraft]                   # slice 4: verify bundle, replace, converge, undraft
  verify bundle                                    # slice 4: closed-set + checksum + signature (GR-09..12)
  image build | image verify                       # slice 5: melange+apko layout; OB-18..21 checks
  version                                          # always: version, commit, protocol integer
```

Verbs are pipeline capabilities, ecosystem-neutral; only `stage` takes `--profile`. Each command exists only from its slice onward; slice 1 ships `stage` and `version`.

**Two-phase OCI protocol (invariant 14).** `prepare` re-plans from live registry state, pushes blobs/manifests by digest, verifies resolution, recursively signs index + platform manifests, and emits `OCIPrepareResult` as its `--json` `result`. YAML runs the three `actions/attest` steps against subjects from that result. `finalize` reads the result from stdin, **re-resolves exact and channel state fresh**, refuses any drift from the result's digests, applies tags serially, and verifies every resulting tag plus the exact tag. Trust metadata strictly precedes public tags.

**`--dry-run` across the two phases:** `prepare --dry-run` plans and validates, pushes nothing, and marks its result `"authoritative": false`. `finalize` refuses a non-authoritative result (exit 1). Verification-only publication (`publish-image: false` today) maps to `prepare --dry-run` + skipped finalize (invariant 15).

**Output contract:** with `--json`, stdout emits exactly one `{"schema":"release.dev/result/v1","command":…,"ok":…,"result":{…}}`. Command-specific `result` payloads stabilize as each command ships. Under Actions, today's output names (`image-digest`, `release-url`, …) are written via the Actions runtime seam (§4) so `workflow_call` plumbing survives — these names are **internal to the release unit** (§8), not a consumer API.

**Rerun reconciliation (per command, from fresh state — never a blanket promise):** `finalize` reruns always re-plan; same version+digest ⇒ complete; an ambiguous prior tag write is resolved before deciding. `publish github` reruns recognize an already-public release with the exact converged expected asset set as success, otherwise report indeterminate (exit 1) for manual reconciliation — never create or re-draft (invariant 1).

---

## 3. Domain model

Types arrive with the slice that uses them (decision 10). No I/O in any of these.

**Slice 1 (`internal/stage`, `internal/profile/goprof`):**

```go
// stage: ParseChecksums parses the checksums.txt claim — strict grammar,
// duplicates and control self-listing rejected. Verification streams each
// listed payload through SHA-256 against fs.FS and requires a nonempty
// regular checksums.txt.sigstore.json. (PP-06)
func ParseChecksums(r io.Reader) (ChecksumSet, error)
func VerifyBundle(fsys fs.FS, claim ChecksumSet) error

// goprof: pure parse + pure selection over GoReleaser artifacts.json.
// Exactly one Linux Binary per amd64/arm64; dist/-relative confined path;
// regular executable file verified against fs.FS metadata. (PP-07/08)
func ParseArtifacts(r io.Reader) ([]Record, error)
func SelectBinaries(recs []Record) ([]CanonicalBinary, error)
```

**Slice 3 (`internal/rel`, created when types first cross use cases):** `Version` (canonical stable subset, OP-01) with `ParseVersion`/`Compare`/`Tag()`; `Tag`; `Digest` (`ParseDigest` sole constructor); `Channel` + `ChannelsFor`; `ChannelState`; `TagPlan` from pure `PlanTags(v, d, cur)` — absent→create, same→accept, differ→error; monotonic line-scoped channels (invariants 11–12). `OCIPrepareResult`: versioned, immutable; digests, tag-plan inputs, attestation subjects, `authoritative` flag — an OCI-publication handoff, not a general protocol.

**Slice 4/5:** `Bundle` constructed only by reconciling a `ChecksumSet` against a real directory scan (closed set, regular files, no unlisted entries — GR-10, invariant 7); OCI layout values retaining exact `index.json` bytes verbatim (OP-06 hashes original bytes; invariants 5–6).

Facts (`Digest`, `Bundle`, exact index bytes, `OCIPrepareResult`) are values. Channel targets, draft state, and the registry tag map are intent, touched only through plans computed pure and applied from fresh remote reads.

---

## 4. Ports

**Rule resolution (binding, from the adjudicated reviews):** A1/A2 win at remote, subprocess, credential, and irreversible-mutation boundaries — call order, short-circuiting, retry classification, and ambiguous writes are business behavior there; keep narrow consumer-owned ports with mockery mocks (T3). R1/R3 win for local deterministic plumbing — `fs.FS`/`io.Reader`/`io.Writer` are the boundary, tested on `t.TempDir()`/`fstest.MapFS`. A3 applies at use-case boundaries, not per type. T3 stays binding for every adapter that survives; the simplification is fewer adapters, not skipped mocks.

**Local plumbing via stdlib (no custom port, no adapter package, no mock):** dist scan/checksum verification and `artifacts.json` reading (`os.DirFS` at the composition edge, slice 1); OCI layout parsing (slice 5) as package functions over `fs.FS` — exact-byte retention is the load-bearing decision, not the interface; a custom port appears only if the image engine demonstrably needs to mock failure ordering. Poll sleeping is an injected `sleep(ctx, d)` function; no `internal/clock`. **Trigger for shared `internal/execx`: two subprocess adapters repeating meaningful plumbing beyond `exec.CommandContext` setup.**

Genuine ports (declared in the consumer package, I2; mockery mocks in `internal/adapter/<name>/mocks/`; classified errors defined with the slice that needs them):

| Port (consumer) | Responsibility | Adapter | Slice | Replaces |
|---|---|---|---|---|
| `pubgh.ArtifactMeta` | Pre-download Actions artifact metadata tuple (ID, expiry, run, digest) | `ghact` (go-github) | 2 | OB-06/GR-05/OP-03 |
| `puboci.StateReader` | Resolve tags, fetch version annotations | `reg` (oras-go v2) | 3 | OP-10/12 reads |
| `puboci.ContentPusher` | Push blobs/manifests by digest, verify resolution | `reg` | 3 | OP-13/14 |
| `puboci.TagCommitter` | Apply planned tags serially, verify each | `reg` | 3 | OP-19 |
| `puboci.Signer` | Keyless recursive sign of digest-pinned refs | `cosign` exec | 3 | OP-15 |
| `pubgh.ReleaseReader` | Draft-by-tag search (24×5s budget) + asset list/digest/state reads (12×1s) | `ghrel` (go-github) | 4 | GR-06, GR-14/16/17 |
| `pubgh.AssetReplacer` | Replace/upload *expected* assets only, clobber semantics | `ghup` (`gh release upload --clobber`) | 4 | GR-15, invariant 10 |
| `pubgh.Publisher` | Undraft + final-state fetch | `ghrel` | 4 | GR-18 |
| `pubgh.RefResolver` | Tag → commit in the checkout | `gitx` (git subprocess) | 4 | GR-07 |
| `pubgh.BlobVerifier` | Sigstore bundle verify, exact identity+issuer | `cosign` exec | 4 | GR-11 |
| `image.APKBuilder` | melange keygen/compile/build of signed APK repos | `melange` exec | 5 | OB-12–15 |
| `image.Composer` | apko lock/build of two-platform layout + SBOMs | `apko` exec | 5 | OB-16/17 |
| `cli` Actions runtime seam | Actions detection, `GITHUB_OUTPUT`, annotations, summary — one interface | `actenv` | first CLI-produced output (slice 2) | scattered output writes |

`ReleaseReader` merges rev-2's `DraftFinder`+`AssetReader`: both are read-only observations of the candidate release through one adapter. `AssetReplacer` and `Publisher` stay separate — expected-name replacement is recoverable while draft; undrafting is the irreversible visibility transition (invariants 9–10).

**Deliberately not ports (residual YAML; CLI verifies results):** checkout, mise install + tool-path proofs (PP-04/OB-05), QEMU/binfmt, artifact transport (decision 13), `actions/attest` (subjects from `OCIPrepareResult`; success observed by the workflow, never claimed by the CLI), `actions/create-github-app-token` (decision 8), and **registry login** — see below.

**Correction from spike B (2026-08-18):** an earlier revision claimed native `reg` credentials, held in memory and never persisted, obsoleted OP-09/OP-20 login/logout. That is false while signing and attestation are external. `oras-go` does authenticate purely in memory for the CLI's own pushes, but `cosign` is a separate process and `actions/attest --push-to-registry` resolves credentials **only** from the docker config: it fails with `No credentials found for registry ghcr.io` otherwise. So OP-09 survives as one login step that serves cosign and the attest actions (`cosign login ghcr.io --password-stdin` works and replaces the `oras` binary), and OP-20 survives as cleanup — but `cosign` has no `logout` subcommand, so cleanup is a docker-config edit, not a tool call. `cosign` also accepts `--registry-username`/`--registry-password` per invocation, which is sufficient for `sign`/`verify`/`tree` alone but not for `actions/attest`.

---

## 5. Package layout

Extends the template (thin `main`, injected streams/non-global Viper; template-conventions §2). Packages are created with their slice; every surviving package gets `doc.go` (D4); 1000-line file cap (R2).

```
cmd/release-cli/          entrypoint: signals, streams, linker vars (version, commit, protocol) [slice 1]
internal/cli/             Cobra tree, flag/env binding, --json envelope, exit mapping           [slice 1]
internal/stage/           stage orchestration: checksum verification + bundle checks over fs.FS [slice 1]
internal/profile/goprof/  ParseArtifacts + pure selection; GoReleaser invocation in slice 6     [slice 1]
internal/adapter/ghact/   Actions artifact metadata                                  [slice 2, + mocks/]
internal/adapter/actenv/  Actions runtime seam                                       [slice 2, + mocks/]
internal/rel/             pure core: Version/Tag/Digest/Channel/TagPlan/OCIPrepareResult        [slice 3]
internal/stage/puboci/    prepare/finalize engines                                             [slice 3]
internal/adapter/reg/     oras-go v2 (StateReader/ContentPusher/TagCommitter)        [slice 3, + mocks/]
internal/adapter/cosign/  cosign exec (sign, verify-blob)                            [slice 3, + mocks/]
internal/stage/pubgh/     GitHub publish engine (incl. handoff/bundle verification)             [slice 4]
internal/adapter/ghrel/ ghup/ gitx/                                                 [slice 4, + mocks/]
internal/stage/image/     image build engine + layout functions                                 [slice 5]
internal/adapter/melange/ apko/                                                     [slice 5, + mocks/]
.mockery.yml              per-interface config emitting into adapter mocks/ (T2/T3)
```

No `internal/profile` parent package (a directory may hold only the `goprof` child), no `internal/verify` catch-all (`verify handoff`/`verify bundle` live with their consuming use cases), no `manfs`, no `clock`, no `execx`, no `e2e`. **Trigger for any additional split: measured size/cohesion pressure, not symmetry.** `cmd/release-mvp` is deleted when slice 1 makes `release-cli` the repo's release artifact.

---

## 6. Profile

**Mechanism: direct dispatch on `--profile go` to the `goprof` package.** Ecosystem knowledge — GoReleaser schema (`type=Binary`, `goos`/`goarch`/`path`), archive/SBOM naming, canonical-binary selection with path confinement and executable checks, eventually the GoReleaser invocation (PP-05, slice 6) — ends at `goprof`'s returned staged facts. Everything downstream (checksum closure, packaging, layout verification, publication, signing, channel policy) is ecosystem-neutral per the inventory. When a second profile lands (decision 2's trigger), the interface is extracted from `goprof`'s existing shape without changing the command surface.

**Image seam contract v1 is exactly today's:** one common static ELF per {linux/amd64, linux/arm64}, one package per arch, two manifests, one entrypoint, user 65532 (invariant 6, OB-09–21). A profile that cannot meet it disables the image stages; a different image input shape is future evidence-driven design, not a current promise. A Rust profile producing two static Linux binaries plugs in with zero engine changes — that is the honest portability claim.

---

## 7. Configuration and org portability

**Flags + env (`RELEASE_*`, template-style injected Viper) + derived defaults. No config file** (decision 17 and its trigger). Derived defaults do the portability work: registry `ghcr.io/<owner>/<repo>` from `GITHUB_REPOSITORY`; signing identity `<repo>/.github/workflows/go-pre-publish.yml@<ref>` from context; channel policy = today's exact/minor/major/latest monotonic rules; CLI distribution repo from `github.action_repository`. Nothing names `meigma`.

**A second org's full adoption delta:** install the GitHub App; set the client-ID variable and private-key secret consumed by the *YAML* token action (actions-constraints §3). That is the whole delta — credential-level changes only, as required.

**Validation is fail-fast:** every command validates inputs, env, and tool availability before any side effect; sentinel `ErrConfig`, exit 2 (E1).

---

## 8. CI integration and the one-version contract

**Layered shape:** reusable workflows own the job graph, `needs` (registry-before-release gate, invariant 3), concurrency (invariants 13/16), timeouts, environment gates, artifact transport, permissions, token minting, the `actions/attest` steps, and prepare→attest→finalize sequencing. The composite `setup-release-cli` owns CLI acquisition/verification and exposes `cli-path`. The CLI owns decisions, verification, and side effects. Callee jobs declare least privilege exactly as today; external callers grant the same ceiling; attestations attach to the consumer repo (actions-constraints §1–2).

**One release unit.** `release-please-config.json:11-16` defines a single root package with `include-component-in-tag: false`: the workflows, the composite action, and the CLI already version together under one tag. The consumer holds exactly one pin — the workflow `@FULL_SHA` — which transitively pins action source (`$/`, same-commit) and the CLI version stamped into `action.yml` (release-please `extra-files`).

**`cli-version` does not exist. Independent CLI pinning is explicitly not supported.** Supporting it would turn every YAML↔binary touchpoint into a versioned public API with a support window — verbs/flags, `GITHUB_OUTPUT` names, the `--json` envelope and per-command `result`, the `OCIPrepareResult`, the prepare→attest→finalize sequencing, exit codes, staged-artifact layout — plus a workflow-ref × CLI-version CI matrix to keep the promise non-decorative. The failure asymmetry decides it: newer workflow + older CLI fails loudly (unknown flag, exit 2, before side effects); **older workflow + newer CLI can fail silently** (a renamed output reads empty and a gating step proceeds). Consequence: outputs and the prepare-result schema are internal to the release unit, and renames are release-unit-atomic. **Revisit trigger: the first adopter needing genuinely different release cadences for binary versus workflows — the honest response then is a supported N−1 window with a real compat matrix, not an unproven input.**

**`cli-path` remains, documented as an unsupported escape hatch: "you supply the binary, you own the pairing."** It is the dogfood mechanism — this repo's `release.yml` builds `./cmd/release-cli` from the branch checkout and passes its path (actions-constraints §5) — and it also covers regression bisection, vendored/mirrored binaries, and air-gapped orgs, which is precisely why no compat promise is needed.

**Mismatch guard: a stamped integer, not a protocol system.** The release unit stamps one integer in `action.yml` and (via linker var) in the binary, reported by `version --json`. On the **installed path** the composite asserts equality at setup and fails closed before any stage command runs. No ranges, no negotiation, no deprecation policy. On the **`cli-path` path** it does not hard-fail: it reports the binary's version and protocol integer in step outputs and emits a warning annotation on mismatch, so debugging stays possible but visibly off-contract. (A dogfood binary built from the same commit matches the stamp and stays quiet.)

**Bootstrap integrity (installed path):** download archive + `checksums.txt`; verify sha256; `gh attestation verify --repo <derived from github.action_repository>` against the pinned signer workflow; then the integer check. The claim is "signed, release-unit-pinned, stamp-matched"; same-commit binding via the provenance predicate stays an experiment (§12).

---

## 9. Preserved invariants

| Invariant (inventory §6) | Enforced by | Local test |
|---|---|---|
| 1 Draft upstream, publisher never creates | `ReleaseReader` returns `ErrNoDraft`; no create path; reruns never re-draft | mock absent/non-draft → exit 1 |
| 2 Tag/run binding | `pubgh` engine + `RefResolver` | unit + gitx temp-repo test |
| 3 Registry-before-release gate | **residual YAML** caller `needs` + `require-oci-image` (unchanged) | workflow lint |
| 4 Two-coordinate handoff | three-owner split (decision 13) | digest-normalization unit; `ghact` mock |
| 5 No-rebuild byte identity | image engine compares extracted layer bytes to canonical binary digest | temp-dir layout mutation → fail |
| 6 Canonical platform/runtime shape | layout validation functions (exact index bytes retained) | table-driven units on synthetic layouts |
| 7 Closed, signed bundle | `ParseChecksums` grammar + directory-scan reconciliation + `BlobVerifier` identity/issuer | grammar units; `t.TempDir()` scans; cosign mocked |
| 8 Attestation subject shape | `OCIPrepareResult` carries subjects; YAML attest executes; sequencing verify→attest→upload (GR-13) and prepare→attest→finalize (OP-16–18) | subject-derivation unit; live rehearsal |
| 9 Draft-until-verified | `pubgh` order verify→replace→converge→publish; `--no-undraft` | engine call-order test |
| 10 Converge only over expected names | `AssetReplacer` clobber contract: replace expected, refuse unexpected, never delete | mock: 422, delete-success/upload-fail, duplicates |
| 11 Exact tags immutable | pure `PlanTags`: absent→create, same→accept, differ→error | pure unit table |
| 12 Channels monotonic, line-scoped | `PlanTags` over fresh `StateReader` reads at prepare and finalize | pure unit table |
| 13 Channel mutation serialized | **residual YAML** concurrency + serial `TagCommitter`; direct-CLI users on shared targets: documented single-writer limitation | serial-order assert; docs |
| 14 Trust metadata before public tags | **two-phase protocol**: prepare (push/verify/sign) → YAML attest → finalize (fresh re-resolve, drift refusal, tag); result transported in the envelope, re-fed on stdin | engine order tests; in-memory registry across both phases |
| 15 Verification-only writes nothing | `prepare --dry-run` non-authoritative result + `plan tags`/`image verify` | zero-mutation assertion |
| 16 No-cancel concurrency | residual YAML (irreducible) | workflow lint |
| 17 Producer publication disabled twice | residual YAML until slice 6; then `goprof` passes `--skip=publish` and asserts `release.disable: true` | goprof unit on invocation args |

---

## 10. Testing strategy

**Layer 1 — pure unit.** Checksum grammar, artifact selection, path confinement predicates, version/tag/digest grammar, `PlanTags`, layout predicates, subject derivation. Zero I/O. Invariants 6, 7, 11, 12 live here.

**Layer 2 — integration on cheap real substrates or mockery mocks.** Local file logic against `t.TempDir()`/`fstest.MapFS` (no custom fakes needed); `reg` against an in-memory registry (`go-containerregistry pkg/registry` via `httptest`); engines against mockery mocks for ordering, short-circuiting, dry-run, and classified-error decisions once each slice defines them. **Trigger for a reusable stateful GitHub fake: slice-4 tests demonstrating repeated pagination/eventual-consistency/clobber transitions beyond what focused mocks express.**

**Layer 3 — live.** The existing cross-repo rehearsal: keyless signing, `actions/attest`, GHCR referrers/auth, QEMU/Docker melange/apko, pinned GoReleaser 2.17.1's real `artifacts.json`. Known mock-passes-tool-fails gaps (cosign flags/recursion, melange arg order, apko path context, GoReleaser schema drift, clobber semantics) are covered by pinned-tool contract tests where cheap and by rehearsal otherwise.

**The laptop answer:** `moon run test` runs layers 1–2 in seconds; the engineer validating a tag-policy or verification change runs one test instead of a 20-minute rehearsal, which shrinks to the final smoke.

---

## 11. Migration path

Strangler pattern; workflow-input compatibility throughout. The four workflows total **145 + 445 + 482 + 569 = 1,641 lines** (inventory §2). Replaced-line figures below are the inventory's cited spans and are **upper bounds**: a few counted lines survive or are obsoleted rather than moved (GR-08's download step stays YAML per decision 13; OP-09/OP-20 login/logout are obsoleted by in-memory credentials; PP-05 shares its block with residual PP-04).

| Slice | Contents | Steps replaced | Cited spans | Lines |
|---|---|---|---|---:|
| 1 | `stage --profile go` | PP-06/07/08 — **3 logical checks** | `go-pre-publish.yml:88-119` | 32 |
| 2 | `verify handoff` + `ghact` + `actenv` — the **three** metadata blocks | OB-06, GR-05, OP-03 | `go-oci-build.yml:106-141`, `publish-github-release.yml:129-164`, `publish-oci-image.yml:88-123` (36 each) | 108 |
| 3 | `plan tags`, `publish oci prepare/finalize`, `reg`+`cosign`, registry error taxonomy. **Acceptance gate: scratch-GHCR parity spike** | OP-09–15, OP-19/20 | `publish-oci-image.yml:234-481`, `:511-569` | 307 |
| 4 | `publish github`, `verify bundle` with `ghrel`/`ghup`/`gitx`; GR-13 attest stays YAML, sequenced verify→attest→upload | GR-06–12, GR-14–18 | `publish-github-release.yml:166-333`, `:341-482` | 310 |
| 5 | `image build`/`image verify` with `melange`/`apko` | OB-08–21 | `go-oci-build.yml:150-436` | 287 |
| 6 | `goprof` invokes GoReleaser; invariant 17 moves in; workflows collapse to thin shells; docs updated (D6) | PP-05 | `go-pre-publish.yml:74-86` | 13 |
| — | **Residual** (28 steps: triggers, `workflow_call` interfaces, permissions, job graph, checkout, mise+tool proofs, QEMU, transport, token action, attest steps) | 28 of 70 | everything not cited above | 584 |

**Check: 32 + 108 + 307 + 310 + 287 + 13 = 1,057; 1,057 + 584 = 1,641.** Taking the named ranges contiguously (PP-05–08, GR-05–12 as single spans) gives 1,059 — the two-line delta is separator lines booked to residual. Headline, precisely: **42 of 70 logical steps migrate; upper bound ~1,059 of 1,641 lines (~64.5%).** The 584 residual is today's non-replaced line count, not the end-state size — the thin-shell workflows will be smaller after cutover.

**Slice 1, concretely (one PR):** four production packages with co-located tests — `cmd/release-cli`, `internal/cli`, `internal/stage`, `internal/profile/goprof`. `os.DirFS(dist)` at the composition edge; `fs.FS`/`io.Reader` inward. Command: `stage --profile go --dist PATH [--json]`. Preserves PP-06 exactly (every `checksums.txt` payload hash + nonempty regular `checksums.txt.sigstore.json`) and PP-07/08 exactly (exactly one Linux `Binary` per amd64/arm64 from real GoReleaser JSON, `dist/`-relative path confinement, regular executable file). No persisted output; PP-09/PP-10 uploads untouched. The composite `setup-release-cli` ships in this PR **only if** the public reusable workflow cuts over in it; otherwise `release.yml` passes `cli-path` from a branch build. **Proof obligation:** four deliberate-mutation cases — bad checksum, missing architecture record, escaped path, cleared exec bit — each failing for the right observable reason, plus one rehearsal against pinned GoReleaser 2.17.1 confirming the real `artifacts.json` shape. Replaces `cmd/release-mvp` as the repo's release artifact.

---

## 12. Risks, unknowns, expected wrongness

**Expect prototyping to invalidate:** package/engine names (settle after slice 3); the `OCIPrepareResult` field set; whether slice-4 mocks suffice or the stateful GitHub fake's trigger fires.

**Load-bearing assumptions → cheap experiments, in order:** (1) `$/` self-reference under external SHA-pinned callers on runners ≥ 2.336.0 — scratch consumer repo before slice 1's action ships; (2) release-please stamps `action.yml` version + protocol integer via `extra-files` — slice 1; (3) `oras-go` v2 GHCR parity incl. referrers — slice-3 acceptance gate; fallback is the ORAS binary behind the same three ports; (4) provenance predicate exposes a verifiable source commit — later spike; until proven, §8's claim stays "signed + stamp-matched"; (5) `sigstore-go` offline bundle verification — after slice 4; (6) native App-token minting — only if a spike shows smaller exposure than the pinned action.

**Considered and rejected:** verb-per-ecosystem CLI (user-rejected); declarative/plugin profiles (rebuilds bash / skew cost); JS/Docker action (no job-level gain — constraints §1); single-command `publish oci` (cannot preserve invariant 14); supported `cli-version` input (silent-failure asymmetry, unbounded compat surface — §8); combined producer artifact (breaks per-handoff independence); prepare-result file (a persistence port with no reader beyond the same workflow run — the envelope already crosses that gap).

---

## 13. Review dispositions

**Correctness review** (all ten findings remain covered; narrowed fixes noted):

| Finding | Disposition | Rev-3 status |
|---|---|---|
| 1 attestation-before-tags | Accept | prepare/attest/finalize + `OCIPrepareResult` in the envelope (§2, decision 4). |
| 2 slice-1 FS ports | **Accept, narrowed** | The concern was raw `os` calls inside business logic. Covered by stdlib boundaries: `fs.FS`/`io.Reader` at the composition edge, pure functions inward — a boundary exists and laptop testability holds; it is simply not bespoke (decision 5). |
| 3 staging/two-artifact mismatch | **Accept, narrowed** | The concern was a cross-job contract whose referenced files are absent per artifact. Covered by not creating the contract: slice 1 persists nothing, artifacts stay byte-for-byte; the first reader serializes only its own projection (decision 3). `ChecksumSet` vs scanned `Bundle` split retained; image seam v1 = today's exact contract. |
| 4 service-surface ports | Accept | Narrow consumer ports; `reg`/`ghrel` each implement several (§4). |
| 5 transport-digest ownership | Accept | Three-owner split (decision 13). |
| 6 "safe to re-run" exit 3 | **Accept, narrowed** | Substance kept: no generic safe-rerun promise anywhere; each mutating command reconciles from fresh state (§2). The stable `mutations` array and exit 3 are deferred to their trigger (decision 12) — they were reporting machinery, not the safety property. |
| 7 skew | **Satisfied structurally** | `cli-version` is removed, so there is no supported skew path to defend (§8). The stamped-integer guard fails closed on the installed path; `cli-path` is explicitly off-contract with visible reporting. Replaces rev-2's protocol handshake and the consumer-selected-version ergonomics story. |
| 8 App token/secrets | Accept | Token action stays YAML; `Secret` type (decisions 8–9). |
| 9 retry/error taxonomy | Accept as scoped | Defined per remote slice with the 24×5/12×1 budgets; no upfront framework. |
| 10 clobber semantics | Accept | `gh release upload --clobber` behind `AssetReplacer` (§4). |

**Complexity review:**

| Item | Disposition | Reason |
|---|---|---|
| Cut 1: manifest/projections/store from slice 1 | Accept | No slice-1 reader; trigger recorded (decision 3). |
| Cut 2: profile interface/registry | Accept | One implementation; `goprof` boundary suffices (decision 2). |
| Cut 3: JSON `phase`/`mutations`, exit 3 | Accept | No slice-1 mutation; finding-6 substance preserved (decision 12). |
| Cut 4: `release.toml` + `config show`/`validate` | Accept | Zero observed consumers; evidence triggers (decision 17). |
| Cut 5: `cli-version` + handshake | **Decided differently** | Stronger than the review's conditional defer: the input is removed permanently as unsupported, grounded in the one-release-unit fact, *and* a cheap stamped-integer guard is kept because the silent-failure asymmetry warrants a fail-closed check even without an override path (§8). |
| Cut 6: testscript/`e2e` | Accept | Cobra tests + rehearsal cover the layers (decision 18). |
| Cut 7: §11 arithmetic | Accept | Reconciled to exactly 1,641 with the check shown (§11). |
| Collapses 1–7 (stdlib FS, four-package slice 1, `actenv` merge, `ReleaseReader`, injected sleep, deferred `execx`/`verify`, three layers) | Accept | Implemented throughout §§4–5, 10–11; §5.5 rule resolution carried as binding (§4). |
| Keep-as-is list 1–11 | Accept | All retained: two-phase OCI, YAML job policy, bootstrap verification, three-owner handoff, strict bundle, exact-byte OCI, pure planner, registry port separation, replacer/publisher split, YAML token action, mockery/doc obligations. |