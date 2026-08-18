# Complexity review: `release-cli` architecture draft v2

## 1. Verdict

**Verdict: substantially simplify.** The central direction is proportionate: one profile-driven CLI, pure release decisions, narrow infrastructure seams, reusable workflows retaining GitHub-owned orchestration, and a two-phase OCI publication barrier. The proposed *initial* architecture is not. It turns 42 explicitly migrated logical operations into 26 non-mock Go packages, 13 mock packages, 21 port contracts/seams, 20 named domain types, 10 runnable command leaves, four configuration sources, and five distinct test modes before the first useful vertical slice has replaced three checks. That is especially costly here because every package adds a required `doc.go`, every adapter normally adds a generated mock package, and every function/type/field adds Godoc under D1/D4/T3. The draft acknowledges that key schemas and package names are expected to be wrong, but still makes those guesses first-PR contracts (Draft §12, line 302; AGENTS.md D1, D4, T3, R1). The single largest source of unnecessary complexity is **speculative cross-stage infrastructure—manifest projections/store, profile registry, stable result/recovery schema, protocol handshake, and fake-backed system harness—multiplied through port/package/mock symmetry before a consumer exists**. The single thing most worth keeping is **the explicit OCI `prepare → actions/attest → finalize` barrier with fresh-state revalidation**, because inventory invariant 14 makes that sequencing irreducible while `actions/attest` remains a YAML step (Draft §1 decision 4, §2, lines 52–59, §9 line 266; inventory OP-13–OP-19, lines 215–221, invariant 14, line 323).

---

## 2. Complexity budget audit

### 2.1 What the proposal actually introduces

| Surface | Actual v2 count | Ground-truth comparison | Assessment |
|---|---:|---|---|
| Runnable CLI leaves in §2 | **10**: `stage`; `image build`; `image verify`; `publish oci prepare`; `publish oci finalize`; `publish github`; `plan tags`; `verify handoff`; `verify bundle`; `version` | Replaces work currently exposed as **4 reusable workflow phases**, containing 70 logical steps | The split is not inherently wrong, but it expands the public executable surface by 2.5×. Two leaves are forced by invariant 14; several others are optional diagnostic projections of behavior already inside mutating commands. |
| Cobra command nodes | **15** including grouping nodes (`image`, `publish`, `oci`, `plan`, `verify`); eventually **18** when `config show` and `config validate` are added | The template currently has **one root command and no subcommands** (template conventions, lines 147–155) | This is all net-new CLI contract, not inherited convention. |
| Named flags in §§2/7 | At least **8**: `--profile`, `--dist`, `--registry`, `--json`, `--dry-run`, `--no-undraft`, `--artifact-id`, `--digest` | Existing workflows use typed workflow inputs and environment context; no CLI flag contract exists | Several are earned by current workflow inputs. `--dry-run` overlaps explicit verification/planning commands. |
| Named domain types in §3 | **20** named types are either declared or required by signatures: `Version`, `Tag`, `Digest`, `Channel`, `AssetName`, `ChecksumSet`, `Bundle`, `BundleScan`, `CanonicalBinary`, `Name`, `Platform`, `RelPath`, `Manifest`, `Release`, `AssetsView`, `ImageView`, `ChannelState`, `TagPlan`, `ExactAction`, `Receipt` | The listed migration covers **42 unique logical steps**; slice 1 covers only **3** | The eventual release/tag types are useful. The slice-1 type budget is not: §11 explicitly promises 6 core types plus projections, and their signatures require more supporting types, for three checks. |
| Port rows in §4 | **20 rows** | 42 uniquely listed migrated logical operations | Globally this is roughly one row per two operations, but the first slice is much worse. |
| Port contracts/seams in §4 | **21**: 18 individually named rows through `ArtifactMeta`, two contracts in `Notifier`/`Outputs`, plus `clock` | The migrated work contains about 40 effectful logical operations, depending on whether residual downloads and native-credential logout are counted | The global ratio can be defended. The slice-1 ratio cannot. |
| Concrete adapter packages | **13**: `goprof`, `bundlefs`, `manfs`, `cosign`, `melange`, `apko`, `layout`, `reg`, `ghrel`, `ghup`, `gitx`, `ghact`, `actenv` | Existing work uses a smaller set of actual external boundaries; `manfs` and `actenv` represent new bookkeeping, not migrated decisions | Several packages are legitimate tool/service boundaries; the local-file and environment wrappers are mostly architecture-created work. |
| Engine packages | **5**: `stage/assets`, `verify`, `stage/puboci`, `stage/pubgh`, `stage/image` | Four current workflow phases | Reasonable eventually, except `verify` is a catch-all for unrelated handoff and bundle concerns and `stage/assets` is excessive for a three-check first slice. |
| Non-mock Go packages in §5 | **26**: 25 production/support namespaces plus `e2e` | Four YAML files, **70** logical steps, **1,641** physical lines | The package count is not larger than the full step count, but it is nearly one package for every migrated logical operation: 26/42 = 0.62 before mocks. |
| Generated mock packages | **13** explicitly marked `+ mocks/` | 13 concrete adapter packages | Required by T3 *if those adapters exist*. It doubles the package namespace count to **39**. D4 then implies up to 39 `doc.go` files, in addition to generated files and Mockery configuration. |
| Slice-1 Go packages | **9** production packages plus **4** mock packages and `e2e`: **14** total | Slice 1 replaces **PP-06–PP-08: 3 logical operations, 32 source lines** (`go-pre-publish.yml:88–119`) | This is the starkest imbalance: 14 namespaces for three checks. Even accepting the draft’s inflated “~80 lines,” it is one package per 5.7 replaced lines. |
| Config source layers | **4**: flags > env > `release.toml` > derived defaults | Inventory found **no current custom registry, namespace, registry secret, alternate architecture, or prerelease path** (inventory “Unknowns and negative findings,” lines 347–349) | Flags/env/derived defaults satisfy current portability. The file layer has no observed consumer. |
| Config inspection commands | **2**: `config show`, `config validate` | Initially **one** proposed durable file knob: `[oci].registry` | The introspection command count exceeds the durable-setting count. Every operational command already validates its own effective inputs. |
| CI execution layers | **3**: reusable workflow → composite setup action → CLI | Existing behavior requires workflow job permissions, `needs`, concurrency, environments, artifact transport, and outputs | This complexity is justified by GitHub’s capability boundaries (actions constraints, lines 9–17, 29–45, 72–76). The setup action should stay thin. |
| Nominal test layers | **3** | T1 explicitly requires three layers | Correct count. |
| Actual test modes in §10 | **5**: pure units; engine tests with generated mocks; adapters against cheap real substrates; binary-level testscript against in-process fakes; live cross-repo rehearsal | T1 asks for unit, mock-adapter integration, and live/end-to-end—not an extra fake-backed system tier (AGENTS.md T1) | The draft calls these three layers by folding two local tiers and live into labels, but the implementation burden is five modes. |
| Named fake/substrate mechanisms | **3 stateful fakes** (`go-containerregistry` registry, GitHub fake, fake clock), **13 generated mock packages**, **5 cheap filesystem/env substrates** (`bundlefs`, `manfs`, `layout`, `gitx`, `actenv`), plus testscript/txtar | The expensive fake behavior is only relevant to later GR-06/14–18 and OP-10–19 | None of it except a real `dist` fixture is needed to learn from PP-06–PP-08. |

Draft evidence: command tree and output/exit contracts at §2, lines 35–59; domain model at §3, lines 63–102; ports at §4, lines 111–136; package tree at §5, lines 146–175; config at §7, lines 206–220; CI layering at §8, lines 224–247; tests at §10, lines 273–279; slice 1 at §11, line 289. The template baseline has no application ports, adapters, mocks, Mockery configuration, config-file reader, subcommands, testscript harness, or `doc.go` files (template conventions, lines 147–155, 176–201, 331, 361–366), so all counts above are new work.

### 2.2 The proposed burn-down does not reconcile with the inventory

The inventory catalog contains:

| Workflow | Logical steps | Physical lines |
|---|---:|---:|
| PP | 10 | 145 |
| OB | 22 | 445 |
| GR | 18 | 482 |
| OP | 20 | 569 |
| **Total** | **70** | **1,641** |

The six slice estimates in Draft §11, lines 289–294 sum to **1,460 lines** (`80 + 150 + 450 + 380 + 350 + 50`). Adding the claimed **250–300 residual lines** at line 296 produces **1,710–1,760 lines**, more than the entire 1,641-line source. The estimates are therefore not an additive complexity budget.

There are two concrete accounting defects:

1. Slice 2 claims “four `getArtifact` script blocks,” but the inventory has **three** metadata operations: OB-06, GR-05, and OP-03 (inventory lines 158, 184, 205). Their cited spans are 36 lines each, **108 lines total**, not four blocks/~150 lines.
2. GR-05 is counted once in slice 2 and again in slice 4’s `GR-05–GR-12` range (Draft §11, lines 290, 292).

The explicitly named unique migrated operations are:

- PP: PP-05–PP-08 = **4**
- OB: OB-06 and OB-08–OB-21 = **15**
- GR: GR-05–GR-12 and GR-14–GR-18 = **13**
- OP: OP-03, OP-09–OP-15, OP-19–OP-20 = **10**
- **Total: 42/70 logical operations, or 60%.**

Using the inventory’s cited source spans gives an *upper bound* of roughly **1,059/1,641 lines, or 64.5%**, and even that overstates deletion because PP-05 shares its shell block with residual PP-04 and GR-08/other residual Actions steps do not move into the CLI. The claimed “~90%” may still be defensible as a share of *bespoke decision logic*, but §11 does not demonstrate it. The owner should not use the current line estimates to justify the proposed abstraction budget.

### 2.3 Where abstraction count exceeds behavior count

These are the concrete count inversions:

1. **Slice 1:** 14 Go namespaces, at least 8 expressly promised release/projection types, and 7 relevant port contracts (`Profile`, `BundleScanner`, `ArtifactsReader`, `Store`, `Notifier`, `Outputs`, plus the profile dispatch seam) for **3 logical checks / 32 source lines** (Draft §11 line 289; inventory PP-06–PP-08, lines 143–145).
2. **Configuration:** 2 introspection verbs for **1** speculative durable knob, with **0** observed repositories requiring it (Draft §7, lines 208–217; inventory negative findings, lines 347–349).
3. **Test structure:** 5 implementation modes under a stated 3-layer policy. The fake-backed system tier duplicates proof already divided between mock-adapter integration and the live rehearsal (Draft §10, lines 275–279; AGENTS.md T1).
4. **Profile machinery:** 1 implementation (`go`) is supported by an interface, a registry, a `Name` method, a per-profile config section, a package, and a mock package (Draft §§5–6, lines 154–155, 184–199). That is at least five structural elements around one dispatch case.
5. **Manifest persistence:** slice 1 creates two artifact copies, two projection types, a store port, an adapter package, and a mock package for **zero downstream CLI consumers in slice 1**. Both current consumers continue reading the old artifact shapes (Draft §§3, 6, 11, lines 93–96, 196–199, 289; inventory PP-09/10, lines 146–147).

### 2.4 One-to-one wrappers and names without decisions

Not every one-implementation port is bad. A port can earn its name by isolating a side effect or a failure contract. The following distinctions matter.

| Mapping | What it adds | Recommendation |
|---|---|---|
| `profile.Profile` → `goprof` | No runtime decision while only `go` exists; `Name()` duplicates the registry key | Remove the interface/registry now; direct-dispatch `--profile=go`. Add the interface when a second implementation exists. |
| `goprof.ArtifactsReader` → parser “in `goprof`” | A name around `json.Decoder` + path metadata; selection is already described as pure | Replace with `ParseArtifacts(io.Reader)` and pure `SelectBinaries`. Use `fs.FS`/`io.Reader` as the port. |
| `manifest.Store` → `manfs` | A package/interface/mock around encode/decode and file open | Delete from slice 1. Later expose `Encode(io.Writer)` / `Decode(io.Reader)` only when a persisted manifest has a reader. |
| `assets.BundleScanner` → `bundlefs` | This one does contain real policy: streaming hashes, regular-file/symlink rejection, closure | Keep the policy, but it does not require a bespoke interface+adapter+mock. Accept `fs.FS`; test with `fstest.MapFS` and `t.TempDir`. |
| `image.LayoutSource` → `layout` | Mostly filesystem traversal/JSON decoding; exact-byte retention is the real decision | Prefer a package function over `fs.FS`/`io.ReaderAt`. Keep a custom port only if the image engine needs to mock failure ordering separately. |
| `cli.Notifier` + `cli.Outputs` → `actenv` | Two interfaces for one Actions environment/output concern | Collapse to one small Actions runtime capability, or inject the output writer and annotation callback directly. |
| `clock` → `time` | A package around `Now/Sleep`; only `Sleep` is presently needed for polling | Inject a context-aware sleep function/poll policy. Do not create `internal/clock` until more than one clock behavior exists. |
| `verify.ArtifactMeta` → `ghact` | One remote read, but it isolates a credible API failure and invariant-4 tuple | Keep; its seam is load-bearing even though it has one implementation. |
| `image.APKBuilder` → `melange` | One subprocess family | Keep the port because A1 and pinned-tool contract testing require it; do not add generic runner layers. |
| `image.Composer` → `apko` | One subprocess family | Keep for the same reason. |
| `pubgh.AssetReplacer` → `ghup` | One exact `gh release upload --clobber` command | Keep; invariant 10 depends on this exact recovery behavior (inventory line 319). |
| `pubgh.RefResolver` → `gitx` | One `git` subprocess | Keep the engine port because invariant 2 requires tag→commit binding and laptop tests must avoid a real checkout; the adapter should remain tiny. |

The other subprocess-only ports are `assets.BlobVerifier` and `puboci.Signer` → `cosign`. They have one implementation but distinct trust directions and distinct consumers; merging them would make the interface more service-shaped, not simpler. The port names buy test seams for irreversible/external behavior, so they are justified under A1/A2 despite being 1:1.

---

## 3. Slice 1 minimality

### 3.1 Honest load-bearing fraction

Draft §11 line 289 calls this a one-PR slice, but it includes:

- 9 production Go packages;
- 4 generated mock packages;
- an `e2e` package/testscript harness;
- at least 8 expressly promised release/projection types, with more supporting types implied;
- a profile interface and registry with one implementation;
- three filesystem/environment adapters;
- a manifest store and two artifact copies that no slice-1 command reads;
- a stable JSON/recovery envelope designed for later remote mutations;
- an installer, bootstrap verification, protocol handshake, and dogfood mode.

Only the following are load-bearing for learning whether Go release staging belongs in the CLI:

1. the actual `stage --profile go` command and Cobra wiring;
2. checksum verification and nonempty Sigstore-control-file check (PP-06);
3. real GoReleaser `artifacts.json` parsing and exact amd64/arm64 selection (PP-07);
4. path confinement plus executable checks (PP-08);
5. one filesystem boundary that is testable without the real runner;
6. the real pinned-GoReleaser rehearsal;
7. an install/path bridge only if this same PR immediately changes the externally reusable workflow.

By namespace, roughly **5 of the proposed 14 slice-1 Go packages** are needed; the remaining behavior can be kept in those packages using standard interfaces. About **60–65% of the named slice-1 structural surface is scaffold**, even though most of the actual validation code remains useful. That is the opposite of the owner’s prototype-first mandate.

### 3.2 Smallest coherent slice 1

A first PR can be concrete and complete with this shape:

```text
cmd/release-cli/          process entrypoint
internal/cli/             Cobra wiring, streams, exit mapping
internal/stage/           stage orchestration + checksum validation over fs.FS
internal/profile/goprof/  ParseArtifacts(io.Reader), pure record selection
.github/actions/setup-release-cli/   only if external workflow cutover occurs now
```

No parent `internal/profile` Go package is required; a directory can contain only the `goprof` child package. No `internal/rel` package is required until a type genuinely crosses use cases.

Required command contract:

```text
release-cli stage --profile go --dist PATH [--json]
```

Required behavior:

1. Accept only `--profile go`; reject other values as usage errors. This preserves the non-negotiable profile-driven CLI without pretending a registry exists.
2. Wrap `PATH` with `os.DirFS` at the composition edge and pass `fs.FS` into stage logic. This is a standard-library port, so A1 remains satisfied without a generic custom `Filesystem` interface; R3 favors reuse.
3. Parse `checksums.txt`, stream each listed payload through SHA-256, compare it, and require `checksums.txt.sigstore.json` to be a nonempty regular file. Preserve PP-06 exactly; do not pull GR-10’s full remote-publication closure into this slice unless the current `dist` layout can satisfy it.
4. Parse the real GoReleaser JSON from an `io.Reader`; require exactly one Linux `Binary` for each of `amd64` and `arm64`; confine paths beneath `dist`; require regular executable files. Preserve PP-07/08 exactly.
5. Produce no persisted manifest. In human mode, success may be silent except for diagnostics on stderr. If `--json` is retained, emit only `schema`, `command`, `ok`, and the selected binary facts in `result`; no `phase` or `mutations` fields.
6. Leave PP-09 and PP-10 upload paths unchanged. No new file is added to either artifact.
7. Test pure parsing/selection with table tests; test the real filesystem path with `t.TempDir` or `fstest.MapFS`; run the existing rehearsal with pinned GoReleaser 2.17.1. This supplies unit, integration, and live evidence without testscript or a custom fake.
8. If the public reusable workflow is changed in this PR, keep `setup-release-cli` to two acquisition modes: exact stamped default release, or `cli-path`. Verify the released archive checksum and attestation. Do **not** expose `cli-version` override yet, so no protocol handshake is needed. If this PR is explicitly a dogfood-only throwaway experiment rather than public cutover, even the installer can wait; use `cli-path` directly.

### 3.3 Exact cuts from the stated slice

| Cut | Why safe | Requirement/risk that remains satisfied |
|---|---|---|
| `Version`, `Tag`, general `Digest`, `Manifest`, projections from slice 1 | PP-06–08 do not plan tags or feed a downstream CLI. Keep a private checksum digest representation if useful. | PP-06–08 behavior remains; I1 still applies to domain terms actually used. Tag invariants remain in residual YAML until slice 3. |
| Persisted manifest and copies in both artifacts | No slice-1 consumer reads them. Existing PP-09/10 artifacts already satisfy current consumers. | Invariant 4 remains exactly as today. Introduce an artifact-local manifest with its first reader. |
| `manifest.Store`, `manfs`, and its mock | They exist only to persist the unused manifest. | No current invariant depends on the new file. |
| Profile interface, registry, `Name()`, per-profile config section, and profile mock | One implementation provides no substitution decision. | `--profile go` remains the public shape; a direct dispatch preserves the one-profile-driven-CLI requirement. |
| Custom `ArtifactsReader` port | Parsing is a pure `io.Reader` operation; filesystem access can use `fs.FS`. | A1 remains satisfied through standard interfaces; PP-07/08 remain laptop-testable. |
| `actenv` and its two ports/mocks | Stage has no workflow output required by PP-06–08; the artifact IDs still come from upload actions. | Existing `workflow_call` outputs remain unchanged. Add Actions output writing with the first CLI-produced output. |
| Full stable JSON `phase`/`mutations` schema | Stage performs no remote mutation and needs no recovery report. | Machine-readable output remains available through a small versioned envelope. |
| Exit code 3 | There is no retrying remote operation in slice 1. | 0 success, 1 operational/contract failure, and optionally 2 usage are enough. |
| Protocol handshake, provided `cli-version` override is also removed | A stamped action default and its pinned workflow/action source cannot select an arbitrary stale CLI. | Correctness-review finding 7 remains covered by eliminating the skew path, not by modeling it. Restore the handshake before allowing override/independent cadence. |
| Testscript/txtar and fake-backed `e2e` package | The CLI can be invoked through Cobra tests against a temp `dist`; the rehearsal is the real T1 layer-3 proof. | T1’s three evidence layers remain. |

This reopens prior correctness findings 2, 3, 6, and 7 only in narrower form:

- **Finding 2, filesystem ports:** use `fs.FS` and `io.Reader`, not unported `os` calls. The correctness concern remains covered.
- **Finding 3, two artifacts/manifest:** remove the unused manifest instead of copying a partially applicable superset. Existing artifacts remain authoritative, so there is no cross-job contract mismatch.
- **Finding 6, unsafe rerun:** retain command-specific reconciliation later and never promise generic safe rerun. A stable `mutations` array is not required to achieve that.
- **Finding 7, skew:** remove the unsupported CLI-version override. If the override remains, the protocol check must remain too.

---

## 4. Premature generality

| Generality | Present evidence | Decision | Trigger to build more |
|---|---|---|---|
| **Compiled profile registry and second-ecosystem story** (Draft §§5–6, lines 153–155, 184–203) | Inventory has one Go/GoReleaser-specific implementation and explicitly fixed two-static-ELF image assumptions (inventory §3, lines 224–239; invariant 6, line 315). No Rust or third-party profile exists. | **Defer.** Keep `--profile go` and a direct dispatch. Keep Go-specific code in `goprof`; that package boundary is sufficient today. | A second profile reaches an implementation PR with a genuinely different staging parser, or a supported external registration mechanism is requested. |
| **Manifest projections and schema versioning** (Draft §1 decision 3, §3 lines 93–96, §6 lines 196–203) | Two independent artifacts exist today, but no slice-1 downstream CLI reads a manifest (inventory PP-09/10, lines 146–147). | **Defer persistence; simplify shape.** Return an internal stage result if useful. With the first reader, serialize only that artifact’s projection rather than copying a superset with references to absent files. A schema discriminator is warranted only at that process/job boundary. | First downstream CLI command reads staged facts—likely image build or GitHub publication. That PR defines the smallest artifact-local contract it consumes. |
| **Publication receipt abstraction** (Draft §1 decision 4, §2 line 55, §3 line 102) | This is not speculative: OP-16–18 must execute between push/sign and OP-19 tags; invariant 14 requires a durable handoff (inventory lines 218–221, 323). | **Keep, but specialize.** Call it `OCIPrepareResult` or equivalent, not a general `Receipt`; include only digest, exact attestation subjects, and state needed to detect drift. Keep one schema discriminator because YAML and a later CLI invocation consume it. | Already triggered by slice 3. Generalize only if a second command needs the same receipt protocol. |
| **Protocol-version handshake** (Draft §1 decision 16, §8 lines 234–243) | It was added for correctness-review finding 7 because a consumer may override `cli-version`; `$/` alone pins only action source (actions constraints, lines 17, 192). | **Defer only if the override is removed.** A stamped exact default is enough while workflow/action/CLI release together. If arbitrary override stays, keep the handshake. | First `cli-version` override, independent workflow/CLI release cadence, or first incompatible on-disk/result schema. |
| **Stable JSON `mutations` list and `phase` field from slice 1** (Draft §2, lines 57–59; disposition line 326) | No existing workflow branches on either. Stage is read-only; later mutating engines are required to reconcile from fresh state regardless of JSON. | **Simplify.** Keep `schema`, `command`, `ok`, `result`. Add command-specific phase/mutation detail without promising global stability when the first ambiguous remote write must be exposed. | A documented automation consumer needs to inspect partial mutation status, or slice 3/4 demonstrates a real state that cannot be communicated adequately by the command result/error. |
| **Four exit codes** (Draft §2 line 59) | Current Actions treat commands as success/failure. No inventory step branches on usage vs verification vs retry-exhausted status. | **Simplify to 0/1/2 initially; defer 3.** `2` is conventional for usage if desired. Exit 3 should not exist until callers are expected to branch on it. | First remote retry engine plus a real caller that handles retry-exhausted differently from other failure. |
| **`config validate` / `config show`** (Draft §7 line 208) | One proposed file knob; operational commands already fail fast. The template has no config-file reader (template conventions, line 155). | **Defer.** Validate effective settings in the command that uses them. Add `show` only when precedence becomes hard to diagnose. | Two or more durable settings, or a support incident where an operator must inspect resolved flag/env/file/default precedence. |
| **`release.toml`** (Draft §7, lines 208–217; slice 3 line 291) | Current implementation has no custom registry or namespace, and portability is intended to require credential changes only (inventory negative findings, lines 347–349). | **Defer with evidence, not by slice number.** Flags/env/derived defaults cover present consumers. | First real adopter needs a non-derived registry target or another durable repository-level setting. One registry override request is enough; “slice 3 arrived” is not. |

---

## 5. Ports and packages

### 5.1 Ports that should collapse or disappear

1. **`ArtifactsReader` should be a pure parser over `io.Reader`.** Path confinement can operate on the parsed relative path plus `fs.FS` metadata. A bespoke interface, an implementation in the same `goprof` package, fixtures, and a generated mock add four concepts without a substitution decision (Draft §4 line 119; inventory PP-07/08, lines 144–145; AGENTS.md R1, R3).
2. **`manifest.Store` should not exist before a manifest reader exists.** Later, `Encode(io.Writer)` and `Decode(io.Reader)` are enough unless atomic persistence/locking becomes a real requirement (Draft §4 line 120; inventory has no corresponding operation; AGENTS.md R1).
3. **`Notifier` and `Outputs` should be one Actions-runtime seam at most.** They are implemented by the same environment adapter and are consumed by the same CLI layer. Two interfaces do not enforce two independent policies (Draft §4 line 135; A2/R1).
4. **`clock` should be a function or polling option, not a package.** The present behavior is bounded sleep in GR-06/16, not a clock domain (Draft §4 line 136, §5 line 173; inventory lines 185, 195). Inject `sleep(ctx, d)`; add `Now` only if code actually uses time.
5. **`DraftFinder` and `AssetReader` can be one read-only `ReleaseReader` port.** Both are observations of the candidate release through the same `ghrel` adapter and the same `pubgh` engine. Keep `AssetReplacer` and `Publisher` separate because expected-name replacement and undrafting have distinct irreversible contracts (Draft §4 lines 129–132; inventory GR-06 and GR-14–18, lines 185, 193–197; A2/R1).
6. **`BundleScanner` and `LayoutSource` should first be tried as package functions over `fs.FS`.** Their validation policy is real, but custom interfaces are not the only way to put I/O behind a port. Standard `fs.FS`, `io.Reader`, and `io.ReaderAt` preserve A1 and testability while avoiding adapter/mock symmetry (Draft §4 lines 118, 125; inventory PP-06/GR-10 and OB-18–21, lines 143, 189, 170–173; AGENTS.md R3/P2).

### 5.2 Ports whose only credible implementation is a subprocess wrapper

- `assets.BlobVerifier` → Cosign `verify-blob`
- `puboci.Signer` → Cosign recursive signing
- `image.APKBuilder` → Melange
- `image.Composer` → apko
- `pubgh.AssetReplacer` → `gh release upload --clobber`
- `pubgh.RefResolver` → Git

These are **not** automatically overengineering. The external executable is the contract, A1 requires the business logic to test without executing it, and the inventory contains tool-specific credible failures (GR-11, GR-15, GR-07, OP-15, OB-12–17). Keep these ports narrow and avoid adding a generic exported Runner; the draft is right that a service-shaped `execx.Runner` would violate A2/R1 (Draft §4, lines 121–124, 131–133, 138).

### 5.3 Ports whose separation is justified

Keep `StateReader`, `ContentPusher`, and `TagCommitter` separate. They correspond to three different authority boundaries:

- observe state and plan;
- publish immutable content by digest;
- mutate public tags only after attestations.

Combining them into `Registry` would reopen correctness-review finding 4 and hide invariant 14’s barrier. The same concrete `reg` adapter may implement all three without requiring three adapter packages (Draft §4, lines 126–128; inventory OP-10/12, OP-13/14, OP-19, lines 212–216, 221; invariants 11–14, lines 320–323).

Keep `AssetReplacer` separate from `Publisher`. Replacing expected assets is recoverable while the release remains a draft; undrafting is the final visibility transition. Inventory invariants 9–10 make that separation meaningful (inventory lines 318–319).

### 5.4 Packages that exist mainly for symmetry

The following should not be created on the current schedule:

- `internal/profile` as a registry/schema package with one child implementation;
- `internal/adapter/manfs` before a manifest consumer;
- `internal/adapter/actenv` before a CLI-produced Actions output exists;
- `internal/clock` for one injected sleep behavior;
- `internal/execx` before at least two adapters demonstrate identical nontrivial plumbing;
- `internal/verify` as a home for unrelated artifact-metadata and bundle verification—place those behaviors with their consuming use cases until a coherent shared contract emerges;
- `e2e`/testscript before more than one command participates in a flow;
- generated mock subpackages for local deterministic file adapters that can be eliminated in favor of standard `fs.FS`.

Each package carries D4 and D1 costs; under this repository’s rules, a package is not a free folder (Draft §5, lines 146–175; AGENTS.md D1/D4; template conventions, lines 331, 361–366).

### 5.5 Where A1/A2/A3 and R1 genuinely conflict

The rules conflict when “all I/O behind ports” is interpreted as “invent one project interface, adapter package, mock package, and doc package for each call to the filesystem or environment.” A1 requires a boundary; it does **not** require a custom boundary when Go already supplies `fs.FS`, `io.Reader`, and `io.Writer`. A2’s one-purpose rule also does not require one-method interfaces when two read methods form one coherent release-observation capability. A3 asks for one obvious package purpose, not one package per interface.

The decision rule should be:

- **R1/R3 win for deterministic local plumbing** when a standard interface supplies the seam and no alternate failure policy exists. Use direct composition and real cheap substrates.
- **A1/A2 win at remote, subprocess, credential, and irreversible-mutation boundaries** where call order, short-circuiting, retry classification, or ambiguous writes are business behavior. Keep narrow consumer-owned ports and T3 mocks there.
- **A3 wins at cohesive use-case/tool boundaries**, not at every type. A `pubgh` package may own release publication decisions; `ghrel` may implement the read side. Splitting each method into a package would violate R1 even if it superficially satisfies A2.
- **T3 remains binding for genuine project adapters.** The simplification is to avoid unnecessary adapters, not to create adapters and then skip their required mocks.

---

## 6. Simplification proposals

### Immediate simplifications, ranked

| Rank | Cost / confidence | Change | What is lost | Requirement/invariant preserved | Approximate surface removed |
|---:|---|---|---|---|---:|
| 1 | **Very high / high** | Delete manifest/projections/store/emission from slice 1; introduce the persisted artifact contract with its first reader | Early schema speculation and a claimed universal handoff | Profile-driven direction remains; invariant 4 remains on current artifact ID/digest/download/content checks | 1 port, 1 adapter, 1 mock package, 1 production package, 3+ domain types, 2 upload edits, schema tests |
| 2 | **Very high / high** | Replace profile interface/registry with direct `go` dispatch while retaining `--profile` and `goprof` package separation | Runtime registration with one implementation | Non-negotiable one profile-driven CLI remains; a second profile can add the interface without changing the command | 1 package, 1 port, 1 mock package, registry/config/name APIs |
| 3 | **High / high** | Use `fs.FS`/`io.Reader`/`io.Writer` for slice-1 local I/O; make parsers/functions direct | Bespoke mocks for cheap local I/O | A1, P2, R3, PP-06–08, laptop testability | 2–3 custom ports, up to 3 adapter/mock packages |
| 4 | **High / high** | Reduce slice-1 JSON to `schema`, `command`, `ok`, `result`; remove `phase`/`mutations`; start with 0/1/2 exits | Programmatic partial-mutation reporting before mutations exist | Machine-readable mode remains; unsafe rerun promise stays removed; later commands still reconcile fresh state | 2 stable fields, mutation model/types/tests, exit-3 contract |
| 5 | **High / medium-high** | Remove `cli-version` override initially and therefore defer the protocol handshake | Consumer-selected CLI version | Portability is unchanged; stamped default remains pinned and bootstrap-verified | protocol linker var, version JSON contract, composite comparison branch/tests/input |
| 6 | **High / high** | Make `release.toml`, `config show`, and `config validate` evidence-triggered rather than slice-3 deliverables | Optional file override before any adopter requests it | Credential-only four-org portability and derived GHCR defaults remain | 1 package, 3 Cobra nodes, file discovery/schema/precedence/docs/tests |
| 7 | **Medium-high / high** | Collapse test modes to pure unit, adapter/CLI integration on cheap real substrates or mocks, and live rehearsal | A separate fake-backed binary-system tier | T1’s required three layers remain; real external contracts remain in rehearsal | `e2e` package, testscript dependency, txtar fixtures, duplicate fake wiring |
| 8 | **Medium / high** | Collapse `Notifier`/`Outputs`; use direct writers/callbacks until Actions behavior grows | Separate naming of two environment effects | Existing workflow outputs remain; A1 still has an injected output boundary | 1 port and possibly `actenv` package/mock in early slices |
| 9 | **Medium / high** | Do not create `clock`, `execx`, or generic `verify` until duplication/cohesion is observed | Symmetric package tree | Poll determinism remains through injected sleep; subprocess adapters stay narrow; A3/R1 improve | 3 packages, associated docs/tests; possibly 1 fake |
| 10 | **Medium / high** | Correct §11 burn-down numbers before using them for planning | A persuasive but unsupported 90% line story | No functional requirement changes | Removes planning ambiguity and prevents scope justified by double-counted lines |

### Reasonable later refinements—not initial architecture

1. Add a profile registry when the second real profile lands.
2. Promote an artifact-local schema to a versioned manifest when a downstream CLI first reads it.
3. Add a protocol handshake when independent version selection or release cadence exists.
4. Extract shared `execx` only after two subprocess adapters contain materially identical code beyond `exec.CommandContext` boilerplate.
5. Promote the GitHub state machine into a reusable fake only after slice 4 tests demonstrate repeated stateful behavior. GR-06/14–18 justify such a fake then, not now.
6. Split packages only after actual size/cohesion evidence. Do not pre-split `rel` into type buckets, but also do not let `rel` become a home for every release term merely because I1 requires types.

---

## 7. Where the proposal is correctly complex

The following complexity should **not** be deleted.

1. **OCI prepare/finalize and a minimal durable handoff.** `actions/attest` must run with job-level OIDC/attestation/package permissions that a CLI or composite cannot grant. Public tags must follow signatures and all three attestations. A phase boundary plus fresh final-state reads is the smallest design that enforces invariant 14 (Draft §§1–2, lines 14, 55; §9 line 266; inventory OP-13–19, lines 215–221, invariant 14 line 323; actions constraints, lines 72–76).
2. **Reusable workflow ownership of job policy.** `needs`, concurrency, permissions ceilings, environment gates, timeouts, and artifact transfer are job/workflow constructs. Keeping them in residual YAML is not distrust of the CLI; it is a documented platform constraint (Draft §8, lines 224–231; actions constraints, lines 9–17, 29–45, 373–382). The composite is also justified as the eventual same-job install/`cli-path` bridge; it should simply avoid becoming a second orchestrator.
3. **Bootstrap checksum and attestation verification for released binaries.** This CLI controls release publication. Verifying the downloaded tool against its checksum and provenance is proportionate to the threat model. The protocol check is conditional; the integrity checks are not (Draft §1 line 27, §8 lines 240–243).
4. **Three-owner artifact handoff verification.** API metadata tuple, Actions transport-digest verification, and post-extraction content validation are distinct coordinates. Collapsing them would weaken invariant 4 or make a false claim about recomputing the Actions ZIP digest (Draft §1 decision 13, §9 line 256; inventory OB-06/07, GR-05/08, OP-03/04, lines 158–159, 184/187, 205–206; invariant 4 line 313).
5. **Strict bundle verification.** Flat unique names, no self-listed controls, streamed hashes, closed set, and exact Sigstore identity/issuer are current security invariants, not future flexibility (Draft §3 lines 78–87, §9 line 259; inventory GR-09–11, lines 188–190, invariant 7 line 316). The timing may move to slice 4, but the behavior must remain.
6. **Exact OCI bytes and no-rebuild verification.** Exact `index.json` bytes, two fixed platforms, runtime metadata, and extracted executable digest equality enforce current invariants 5–6. These checks deserve real domain types and careful layout parsing (Draft §3 line 105, §9 lines 257–258; inventory OB-18–21, lines 170–173, invariants 5–6 lines 314–315).
7. **Pure version/tag planning.** `Version`, `Digest`, `ChannelState`, and `TagPlan` turn current exact-tag immutability and channel monotonicity into deterministic policy. They replace a large embedded script and directly enforce invariants 11–12 (Draft §3 lines 68–100; inventory OP-10–12, lines 212–214, invariants 11–12 lines 320–321). They belong in slice 3, not slice 1.
8. **Registry capability separation.** Reads, immutable content push, and tag commit have materially different replay and authority properties. Keep the three ports and one adapter (Draft §4 lines 126–128; inventory OP-10/12/13/14/19).
9. **GitHub release state-machine safeguards.** Draft-only start, tag/SHA binding, expected-name-only clobber, digest convergence, and final undraft are a credible partial-failure domain, not excess defense (Draft §9 lines 253–262; inventory GR-06/07/14–18, lines 185–197, invariants 1, 9, 10 lines 310, 318–319). Stateful testing is justified when slice 4 arrives.
10. **Official App-token action remaining in YAML.** It keeps the private key out of the CLI and avoids inventing token-refresh behavior. That is a simplification and a security improvement, not residual mess (Draft §1 decisions 8–9; inventory GR-04 line 183 and credential inventory; correctness-review finding 8 disposition at Draft §13 line 328).
11. **Mockery for real adapters and D1/D4 documentation.** T2/T3/D1/D4 are binding repository rules. The correct response is to reduce unnecessary ports/packages, not ignore those obligations for the ones that remain.

---

## 8. Cheapest experiment that would collapse the design

Build a **throwaway, single-command vertical prototype in one day**:

```text
release-cli stage --profile go --dist ./dist
```

Implementation constraints for the experiment:

- only `cmd/release-cli`, `internal/cli`, `internal/stage`, and `internal/profile/goprof`;
- `os.DirFS`/`fs.FS` and `io.Reader`, no custom filesystem ports;
- direct `go` dispatch, no profile interface/registry;
- no manifest, projections, store, Actions-output adapter, stable mutation schema, protocol integer, Mockery, or testscript;
- run pinned GoReleaser 2.17.1 in the existing rehearsal fixture, then compare the prototype’s pass/fail and selected paths with PP-06–PP-08;
- mutate one checksum, one architecture record, one escaped path, and one executable bit to prove it fails for the right observable reasons.

This experiment answers the highest-uncertainty first-slice questions: the real `artifacts.json` shape, whether standard filesystem interfaces are sufficient, which facts actually need types, whether a separate stage engine is useful, and what output the workflow genuinely consumes. It does not weaken any publication invariant because it performs no publication and leaves PP-09/10 unchanged.

If it works, the owner can **delete rather than further specify**:

- slice-1 `Manifest`/projection/receipt-adjacent material from §3;
- the first four custom-port rows in §4 except the scanner policy itself;
- `internal/profile`, `manfs`, `actenv`, their mocks, and `e2e` from the slice-1 portion of §5;
- the registry/config-section mechanism from the current part of §6;
- the fake-backed system-harness requirement for slice 1 in §10;
- most of the “Contents, exactly” list in §11 item 1.

The second-cheapest experiment is the already-listed `$/` external consumer test, but it validates delivery syntax, not the large architecture. It should not block learning whether the core stage can remain four packages.

---

## 9. Punch list

### Cut now

1. Remove manifest/projections/store/emission from slice 1. Reopen correctness-review finding 3 as described above; existing artifacts remain authoritative and invariant 4 remains intact.
2. Remove the profile registry/interface/`Name()`/profile mock until a second implementation exists. Keep `--profile go`.
3. Remove stable `phase` and `mutations` from the slice-1 JSON contract and remove exit code 3 until a remote retrying command exists.
4. Remove `release.toml`, `config show`, and `config validate` from all calendar-based slices; restore them only on the named adopter/config triggers.
5. Remove `cli-version` override initially; then remove the protocol handshake until independent selection/cadence exists. If override remains, keep the handshake.
6. Remove testscript/txtar and the fake-backed `e2e` namespace from slice 1.
7. Correct the §11 line/step budget: three metadata blocks, GR-05 counted once, and additive totals reconciled to 1,641 lines.

### Collapse now

1. Use `fs.FS`, `io.Reader`, and `io.Writer` instead of custom `ArtifactsReader`/`Store` and most slice-1 file adapters.
2. Collapse slice 1 to about four production Go packages plus co-located tests; add the minimal setup action only if the public workflow cuts over in that PR.
3. Collapse `Notifier` and `Outputs` into one Actions runtime seam when one becomes necessary.
4. Collapse `DraftFinder` and `AssetReader` into one read-only release observation port in slice 4; keep replacement and publication ports separate.
5. Inject a context-aware sleep function instead of creating `internal/clock`.
6. Delay `internal/execx` and generic `internal/verify` until repeated code/cohesion proves those packages.
7. Treat the test strategy as three evidence layers, not five harness modes.

### Defer with named trigger

1. **Persisted manifest/schema:** first downstream CLI reader of staged facts.
2. **Profile registry/interface:** second implemented ecosystem profile.
3. **Protocol handshake:** supported CLI-version override, independent workflow/CLI cadence, or first incompatible protocol.
4. **`release.toml`:** first real non-derived registry target or second durable setting.
5. **`config show`/`validate`:** multiple settings or a demonstrated precedence-debugging need.
6. **Stable mutation report and exit 3:** first ambiguous remote mutation plus a caller that programmatically branches on it.
7. **Reusable stateful GitHub fake:** slice 4 tests show repeated pagination/eventual-consistency/clobber transitions beyond focused mocks.
8. **Shared `execx`:** two subprocess adapters repeat meaningful behavior beyond ordinary `exec.CommandContext` setup.
9. **Extra package splits:** measured size/cohesion pressure, not symmetry with the final tree.

### Keep as-is

1. `publish oci prepare → actions/attest → publish oci finalize`, with minimal versioned OCI prepare result and fresh-state drift refusal.
2. Reusable workflow ownership of permissions, `needs`, concurrency, environments, timeouts, artifact transport, and attestation steps.
3. Thin composite as eventual acquisition/`cli-path` bridge; released-binary checksum and attestation verification.
4. Three-way artifact-integrity ownership.
5. Strict closed/signed bundle validation.
6. Exact OCI-byte/no-rebuild/platform/runtime verification.
7. Pure `Version`/tag/channel planner in slice 3.
8. Separate registry read/content-push/tag-commit ports.
9. Separate GitHub expected-asset replacement and final publication capabilities.
10. Official GitHub App token action in YAML and secret-redaction discipline.
11. Mockery and package/Godoc requirements for the smaller set of genuine adapters/packages that survives simplification.

**Smallest coherent change set:** implement the four-package `stage --profile go` vertical slice over standard filesystem/reader interfaces, verify PP-06–PP-08 against real pinned GoReleaser output, keep current artifact uploads unchanged, and add only the minimum acquisition bridge required for immediate external reusable-workflow cutover. Everything else should enter on the explicit triggers above.
