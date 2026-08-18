# Adversarial architecture review: `release-cli`

## Verdict

**Approve with changes for prototyping slice 1 only.** The central direction—profile-scoped staging, a pure release model, native API adapters where useful, subprocess adapters where the executable is the contract, and reusable-workflow → composite-action → CLI layering—is sound enough to learn from. It is **not** sound enough to implement unchanged: slice 1 has no explicit filesystem boundary for the work it moves, and its “one `staging.json`” contract does not fit the two artifacts that actually cross jobs. Later, the proposed single `publish oci` command cannot preserve the most important publication invariant while `actions/attest` remains a YAML step. The single change that would improve the design most is to make every residual-YAML boundary an explicit phase contract—most urgently, split OCI publication into a prepare result and a post-attestation finalize/commit step rather than pretending one CLI invocation owns both sides of an external YAML barrier.

## What the design got right

1. **The decomposition starts from observed behavior rather than a generic release framework.** Draft §1 lines 9–28 and §11 lines 280–289 map work to PP/OB/GR/OP clusters and preserve transport, permissions, concurrency, and attestations in YAML where Actions owns the capability. That matches `workflow-inventory.md` §2 and `actions-constraints.md` §§1, 7.
2. **The ecosystem seam is mostly in the right place.** Draft §1 line 10 and §6 lines 173–187 isolate GoReleaser schema and canonical-binary selection from registry and release publication. Inventory §3 lines 226–235 confirms PP-03–05, PP-07/08, OB-09, and part of OB-10/11 are the actual Go-specific areas.
3. **The immutable-value/plan model is a strong fit for tag policy.** Draft §1 line 16 and §3 lines 94–101 turn exact-tag immutability and channel monotonicity into pure decisions. That directly addresses inventory invariants 11–12 (`workflow-inventory.md` lines 320–321) and removes the current BigInt JavaScript from the decision path.
4. **The Actions layering is correct.** Draft §1 lines 19–23 and §8 lines 216–239 correctly retain reusable workflows for job permissions, `needs`, environments, concurrency, artifacts, and outputs, while using a composite only as a job-local bridge. This agrees with `actions-constraints.md` lines 8–12, 29–42, and 362–407. The draft also correctly states that the caller must grant the permission ceiling and that attestations attach to the caller repository.
5. **The draft preserves several critical release postures explicitly.** Draft §9 lines 247–260 retains upstream draft ownership, tag/run binding, no-rebuild checks, closed bundles, exact tags, monotonic channels, verification-only behavior, and no-cancel concurrency. These correspond to inventory invariants 1–2, 5–7, 11–13, and 15–16 (`workflow-inventory.md` lines 310–325).
6. **The migration strategy is appropriately agile.** Draft §11 lines 280–289 is a real strangler plan, not a speculative rewrite. Draft §12 lines 295–303 also labels assumptions that deserve experiments instead of trying to specify every future adapter up front.
7. **The subprocess/library split is defensible.** Draft §1 lines 13–14 keeps melange/apko/cosign/GoReleaser as pinned tool contracts while proposing native clients for stateful APIs. That is a reasonable application of AGENTS.md L1 and R1, provided the adapter-contract holes below are fixed.

## Blocking defects, ranked

### 1. Critical — **Wrong:** the proposed OCI command cannot put attestations before tags

**Draft location:** §2 line 39 defines one `publish oci` command as “plan, push, sign, tag”; §4 line 127 leaves `actions/attest` in YAML; §9 line 258 nevertheless claims engine order `push→verify→sign→attest-subjects→tags`; §11 line 284 again says the single CLI command replaces OP-09–15 and OP-19 while YAML keeps attest steps.

**Evidence:** Inventory OP-13–19 (`workflow-inventory.md` lines 215–221) and invariant 14 (line 323) require this real order: push digest-addressed content, verify it, recursively sign index and platforms, create the index provenance attestation and both platform SBOM attestations with `push-to-registry:true`, **then** mutate public tags. The actual workflow implements exactly that order at `.github/workflows/publish-oci-image.yml:386-563`. Computing attestation subjects is not creating trust metadata.

**Failure:** If `publish oci` applies tags before returning, YAML attestations necessarily happen too late. If it returns before tags, no listed command resumes tag application. The design currently has no component that can enforce invariant 14.

**What to change:** Make this an explicit two-phase protocol before slice 3:

- a prepare command that re-plans, pushes by digest, verifies resolution, signs index and both platform manifests, and emits a versioned immutable publication plan/subject set;
- YAML `actions/attest` steps;
- a finalize/commit command that consumes that plan, re-resolves exact and channel state, refuses drift, applies tags serially, and verifies every resulting tag.

Absorbing attestations into the CLI later is a valid alternative, but “emit subjects” must not be represented as equivalent to “attestation succeeded.”

### 2. High — **Wrong for slice 1:** required filesystem effects have no narrow ports

**Draft location:** §1 line 11 and §4 line 129 promise that every side effect is behind a consumer-declared port and that engines only orchestrate ports. Yet the port table at lines 111–123 has no boundary for bundle directory enumeration, regular-file/symlink checks, streaming file hashes, executable metadata, `artifacts.json` reading, or staging-manifest persistence. `LayoutSource` at line 120 only covers OCI-layout reads. Slice 1 at line 282 moves PP-06–08 into `stage/assets` and `goprof`, and §9 line 253 says the assets engine performs the closed-set walk.

**Evidence:** PP-06–08 are filesystem/process work (`workflow-inventory.md` lines 143–145); GR-09/10 and OB-08–10 perform regular-file checks, directory closure, path confinement, hashing, and executable inspection (lines 160–162, 188–190). AGENTS.md A1 requires all I/O behind ports, and A2 forbids solving this with a generic whole-filesystem wrapper.

**Failure:** An implementation following the package plan must either put `os.ReadDir`, `os.Open`, `os.Stat`, path resolution, and file writes in business engines—violating A1—or hide all of them behind the already-broad `Profile.Stage`, which gives the core no independently testable contract for PP-06 and release-bundle closure.

**What to change:** For slice 1 only, introduce the smallest use-case-shaped file boundaries actually needed, for example a bundle scanner that returns regular-file metadata/readers, a Go artifact source that returns selected canonical records after path confinement, and a staging-manifest store/codec boundary. Do not create a general `Filesystem` interface. Stream payloads through `io.Reader` as required by AGENTS.md P2.

### 3. High — **Wrong/incomplete domain contract:** one `Staging` value does not fit the two artifact handoffs

**Draft location:** §3 lines 88–91 defines one `Staging` containing both the full release bundle and canonical binaries; §6 line 187 says one `staging.json` is the only downstream handoff and every later stage reads it; slice 1 line 282 emits it while leaving artifact transport in YAML. §12 line 297 permits schema churn but does not address transport shape.

**Evidence:** PP-09 and PP-10 deliberately upload two disjoint artifacts (`workflow-inventory.md` lines 146–147; actual `.github/workflows/go-pre-publish.yml:120-145`). `oci-build-inputs` contains `artifacts.json` plus Linux trees; `release-assets` contains archives, SBOMs, checksums, and the Sigstore bundle. Downstream jobs download only the artifact they need. Inventory invariant 4 (line 313) treats each handoff independently.

**Failure:** If the same manifest is copied into both artifacts, half of its referenced paths are absent in each download. If it is uploaded only once, one downstream path cannot read the promised sole handoff. If validation accepts silently missing sections, `Staging` is not the complete validated contract claimed by §6.

There is a second modeling defect at §3 lines 80–85: `ParseChecksums(io.Reader) (Bundle, error)` cannot produce a complete `Bundle` containing `Asset.Size` or prove directory closure from checksum bytes alone. It should parse a checksum set; an I/O boundary should reconcile that set with the directory and only then construct a validated bundle.

**What to change:** Decide in slice 1—because this becomes a cross-job artifact contract—between (a) one combined authoritative artifact, or (b) one logical release manifest with explicit stage-specific projections that is copied into both existing artifacts and validated by projection. Rename the pure parse result to something like `ChecksumSet`; construct `Bundle` only after a scanner proves files, sizes, digests, controls, and closure. This is an expensive boundary to get wrong, so deciding it before code is justified despite the agile mandate.

Also delete the unsupported claim at §2 line 54/§6 line 189 that a zero-binary or N-binary container profile leaves `image build` untouched. Inventory invariant 6 (line 315) and OB-09–21 (lines 161–173) require exactly one common static ELF for each of amd64/arm64, one package per architecture, two manifests, and one entrypoint. A future profile needs either that exact contract or a different image input seam.

### 4. High — **Wrong against A2:** the two largest ports are service-surface wrappers

**Draft location:** §1 line 12 says “never a service-surface wrapper.” §4 line 116 then gives `puboci.Registry` resolve, annotation fetch, blob push, manifest push, and tag mutation; line 117 gives `pubgh.ReleaseStore` draft search, asset listing, upload/replacement, publish, and final-state fetch.

**Evidence:** AGENTS.md A2 says one-purpose adapters must not wrap an entire service surface. Inventory separates OCI read/planning (OP-10/12), content publication (OP-13/14), and tag commit/verification (OP-19), and separates GitHub draft discovery (GR-06), asset convergence (GR-14–17), and release-state mutation (GR-18).

**Failure:** The interfaces permit an engine to depend on unrelated mutation capabilities and make mocks assert a service-shaped API instead of a use-case contract. They also obscure the crucial prepare/finalize boundary from finding 1.

**What to change:** Keep one concrete `reg` package and one concrete `ghrel` package if desired, but have those concrete types implement several narrow consumer-owned ports: registry state reader, content publisher, tag committer; draft finder, asset reader/replacer, release publisher. This fixes A2 without multiplying adapter packages.

### 5. High — **Wrong ownership claim:** the CLI cannot recompute the Actions transport digest from extracted files

**Draft location:** §4 line 127 says the CLI verifies upload/download results; §9 line 250 credits `verify handoff` with `ArtifactMeta + on-disk digest recompute in rel` for the two-coordinate handoff.

**Evidence:** Inventory invariant 4 (line 313), OB-06/07 (lines 158–159), GR-05/08 (lines 184, 187), and OP-03/04 (lines 205–206) distinguish three operations: API metadata/run/digest validation, `actions/download-artifact` verifying the artifact ZIP transport digest with `digest-mismatch:error`, and content-specific post-download validation. The extracted directory is not the original transport ZIP byte stream.

**Failure:** Hashing extracted files cannot reproduce the upload action’s artifact digest. Claiming it does weakens the handoff and risks removal of the only actual download-integrity check.

**What to change:** Make ownership explicit: `verify handoff` performs metadata tuple validation before download; SHA-pinned `actions/download-artifact` remains responsible for transport digest verification; a later CLI command verifies the extracted content contract (bundle hashes or OCI index triple). Never describe the latter as recomputing the Actions artifact digest.

### 6. High — **Wrong:** exit code 3 is not generically “safe to re-run”

**Draft location:** §2 line 61 calls every exhausted transient failure safe to rerun; §1 line 18 and §10 line 268 leave retry/idempotency to an unspecified engine policy.

**Evidence:** GR-14–18 (`workflow-inventory.md` lines 193–197) can partially replace assets and can successfully set `draft:false` before the final read fails. Inventory invariant 1 (line 310) says a normal run starts from a draft and rejects a non-draft. OP-19 (line 221) can apply some tags before a later tag call or verification fails. `actions-constraints.md` lines 166–169 only guarantee code-pin behavior on job reruns; they do not roll back remote state.

**Failure:** A post-undraft read timeout could return exit 3, yet the rerun immediately fails `DraftByTag` because the release is now public. A registry 5xx can be an ambiguous success, so blindly replaying a stale tag plan can overwrite state. The blanket promise is actively dangerous for automation.

**What to change:** Remove “safe to re-run” from the exit-code contract. Define exit 3 only as “transient failure exhausted,” and include a structured phase/mutation-status field in JSON. Each mutating command must reconcile postconditions: OCI reruns must re-plan from fresh state and resolve an ambiguously failed tag; GitHub reruns must either safely recognize an already-complete matching public release or report an explicit indeterminate/manual-reconcile state. Do not weaken the upstream-draft invariant accidentally to gain idempotency.

### 7. High — **Unproven trust/skew claim:** `$/` pins action source, not binary provenance or protocol compatibility

**Draft location:** §1 line 20 says `$/` “kills version skew by construction”; §8 line 237 calls workflow SHA → action → CLI version an atomic chain; line 239 hardcodes `--repo meigma/release`; the consumer may still supply `cli-version` (§8 line 226).

**Evidence:** `actions-constraints.md` lines 181–192 says a workflow SHA and CLI semantic version are different pins and must not be equated without a release guarantee; lines 406–407 explicitly retain drift as a layering cost. `$/` only guarantees sibling action source at the same commit (lines 194–202). A caller can pin an arbitrary non-release SHA or override `cli-version` to an incompatible old binary. Draft §12 line 303 only proposes verifying that Release Please can stamp the file; it does not prove the release exists or was built from that source commit.

**Failure:** The installer verifies checksum, signer workflow, and requested version, but does not state that it checks the attestation’s source repository/commit against the action/workflow commit or that the CLI supports the workflow’s manifest/result protocol. A correctly signed stale binary can therefore be accepted. The literal `meigma/release` also contradicts §1 line 24 and §7 line 197’s portability claim.

**What to change:** Weaken the claim unless the release pipeline actually proves it. Embed/stamp a required CLI protocol version and source commit in the action/workflow, have `release-cli version --json` report them, and reject an incompatible override before work starts. Derive the distribution repository from the action identity rather than hardcoding `meigma/release`. If “same source commit” remains a security claim, verify it from the build-provenance predicate, not just the signer workflow and semantic version. Make blank `cli-version` → embedded default explicit in the composite implementation; do not rely on an action metadata default after the workflow explicitly passes an empty value.

### 8. High — **Under-specified recovery/security:** App authentication is reduced to a one-shot token

**Draft location:** §4 line 118 defines `Mint(...) (Token, error)`; §7 line 195 loads the private key from environment; §12 line 299 postpones deciding whether native minting is safer.

**Evidence:** `actions-constraints.md` line 113 states installation tokens expire after exactly one hour. Inventory credentials lines 335–337 show the current pinned action mints a contents-write token while the job token stays contents-read, and that the App token is scoped only to release operations. GR-06 and GR-14–18 include polling and multiple mutations.

**Failure:** The domain cannot represent expiry or refresh. An expired token midway through asset replacement can leave a partial draft, and the engine cannot distinguish auth expiry from a permanent 403. Pulling the private key into the general CLI config also creates a future `config show`/JSON/error-redaction hazard; draft §2 lists `config show` but never says secrets are excluded.

**What to change:** Default to keeping `actions/create-github-app-token` in YAML for slice 4 unless a spike proves a smaller exposure surface. If native auth wins, give the GitHub adapter a refreshing authenticated transport or a credential source that exposes expiry/classified refresh errors—not a one-shot token value. Keep private-key resolution out of the printable config object and make `config show`, JSON, logs, and wrapped errors structurally incapable of emitting secrets.

### 9. Medium — **Under-specified:** retry policy lacks an error taxonomy and the current poll contracts

**Draft location:** §1 line 18 says retries are engine policy; §4 line 123 supplies only `Now/Sleep`; §10 line 268 says mocks test retry policy. No port error contract distinguishes absent state, retryable 5xx/429, auth expiry, malformed remote state, or ambiguous write success.

**Evidence:** GR-05 uses three API retries; GR-06 polls 24×5 seconds; GR-16 polls 12×1 second (`workflow-inventory.md` lines 184–196). OP-10/12 distinguish not-found from corruption, while E3 requires retrying acceptable transient failures. The actual workflows also paginate release and asset listings.

**Failure:** A fake clock proves only that sleeps occurred. It cannot tell an engine whether to retry a 404 while waiting for a draft, fail a 404 for a vanished selected release, honor `Retry-After`, refresh a token, or resolve an ambiguous registry write. A mock can therefore pass while production retries the wrong class indefinitely or fails immediately.

**What to change:** When each remote slice is implemented—not in slice 1—define the minimal classified adapter errors and bounded policy needed by that slice, including context cancellation, pagination, existing 24×5 and 12×1 convergence budgets (or an explicitly justified change), retry hints, and read-after-write reconciliation. Do not build a generic retry framework now.

### 10. Medium — **Unproven adapter semantics:** native GitHub upload does not yet replace `--clobber`

**Draft location:** §1 line 14 chooses `go-github`; §4 line 117 gives the port only `Upload`; §9 line 247 promises convergent upload, and §11 line 285 says it replaces GR-14–18.

**Evidence:** GR-14/15 and inventory invariant 10 (`workflow-inventory.md` lines 193–195, 319) require this exact recovery behavior: reject unexpected names, replace expected existing assets with `gh release upload --clobber`, and never delete unexpected assets. The actual implementation is `.github/workflows/publish-github-release.yml:341-405`.

**Failure:** A generic upload call does not express replacement, which existing asset may be deleted, or how a delete-success/upload-failure rerun converges. A fake that merely accepts duplicate upload names would hide the production API difference.

**What to change:** Either keep the pinned `gh release upload --clobber` subprocess behind a narrow adapter or specify a `ReplaceExpectedAsset` contract implemented as list/check/delete-only-expected/upload/poll. Exercise 422/name collision, delete success plus upload failure, duplicate remote names, and unexpected-name refusal against a stateful fake and the rehearsal repository.

## Coverage audit

### Step catalog

| Inventory work | Draft owner | Audit result |
|---|---|---|
| **PP-01** tag gate | Engine per §6 line 185 | Covered in principle. Keep the current residual gate until the CLI replacement is active. |
| **PP-02** checkout | Residual YAML, §1 line 23 | Correctly retained. |
| **PP-03/04** locked tools and mise-path proof | Residual mise plus exec adapter plumbing, §1 line 23 and §4 lines 124–125 | Installation is covered; the path/provenance assertion must remain explicit. Inventory PP-04 (line 141) is more than “tool available.” |
| **PP-05** GoReleaser, including `--skip=publish` | `goprof`, deferred to slice 6 (§11 line 287) | Covered, but invariant 17 is missing from §9. Preserve both command-level `--skip=publish` and config-level `release.disable:true` (`workflow-inventory.md` line 326). |
| **PP-06–08** checksums, bundle presence, exact Linux records, executable files | Slice 1, §11 line 282 | Intended coverage is correct; blocked by missing file boundaries and incomplete bundle type (findings 2–3). |
| **PP-09/10** two uploads | Residual YAML, §1 line 23 | Correct ownership, but the staging manifest has no defined placement/projection (finding 3). |
| **OB-01/02** tag gate and checkout | Engine + residual YAML | Covered. |
| **OB-03–05** mise, QEMU, tool provenance | Residual YAML, §1 line 23 | Correct. Preserve exact path/version assertions, not only install steps. |
| **OB-06/07** metadata tuple + verified download | `verify handoff` + residual download | Split is appropriate only after correcting the false on-disk transport-digest claim (finding 5). |
| **OB-08–11** workspace, Go artifact selection, static ELF/common name, commit date, copied config, canonical hashes | `goprof` and image engine, §6 lines 181–185 | Under-specified at the profile/image boundary. `CanonicalBinary` must carry the common program identity, platform, confined source, and digest needed by OB-10/11; filesystem staging needs a narrow port. |
| **OB-12–17** melange/apko build | APKBuilder/Composer ports, §4 lines 114–115 | Covered structurally. Real argument/output contracts remain CI/tool integration work, not proved by engine mocks. |
| **OB-18–21** deep layout/runtime/SBOM/no-rebuild verification | `rel.Layout` + `LayoutSource`, §9 lines 251–252 | Strong mapping. Ensure exact index bytes are retained, not JSON re-marshaled, because OP-06 hashes the original bytes. Enforce all invariant-6 details, not only platform count. |
| **OB-22** upload OCI artifact | Residual YAML | Correct. |
| **GR-01–03** tag/OCI gate, checkout, tools | Residual/engine per §§6, 8 | The job gate “registry before GitHub release” is absent from the §9 table but remains correctly expressible in caller `needs`; actual `.github/workflows/release.yml:32-73` must stay. GR-01’s digest-pinned caller-repo GHCR reference must remain until the CLI explicitly owns it. |
| **GR-04** App token | `ghapp` | Capability exists but expiry/exposure is unresolved (finding 8). |
| **GR-05/08** metadata and download | `verify handoff` + YAML | Correct after finding 5’s ownership fix. |
| **GR-06/07** draft poll and tag→SHA | `ReleaseStore` + `RefResolver` | Covered. Preserve pagination, 24×5 poll, unique tag match, draft-only start, and exact `github.sha`. |
| **GR-09–12** local closed/signed bundle | `ParseChecksums`, assets engine, BlobVerifier | Intended coverage is good. Split checksum parsing from filesystem closure and stream hashes. Exact certificate identity and issuer are load-bearing. |
| **GR-13** release attestation | Residual YAML | Subject derivation is covered, but migration must explicitly sequence `verify bundle` → successful `actions/attest subject-checksums` → remote upload. Draft slice 4 incorrectly says GR-04–18 as a single replaced range even though GR-13 remains YAML. |
| **GR-14–18** revalidate, clobber expected assets, convergence poll, digest closure, optional undraft/final fetch | `ReleaseStore`/pubgh | Mostly covered, but replacement semantics and post-undraft rerun behavior are undefined (findings 6 and 10). |
| **OP-01/03–08** stable tag, handoff, exact index bytes/digest triple, descriptors, names | rel/verify/layout | Covered. Preserve caller/recorded/computed digest equality and the two platform SBOM paths. |
| **OP-02** tool installation | Residual mise | ORAS disappears if native client wins; Cosign remains pinned. State that native credentials stay in memory so OP-20 logout is intentionally obsolete rather than silently dropped. |
| **OP-09–15** auth, exact precheck, monotonic planning, blob/manifest push, recursive signing | puboci/reg/cosign | Decision logic is well targeted. Missing: actor/credential construction, fresh-state checks at commit, all-three-subject signing proof, and classified partial-write recovery. |
| **OP-16–18** index provenance + two platform SBOM attestations | Residual YAML | Subject shape is represented, but the command topology breaks their position before OP-19 (finding 1). |
| **OP-19** serial tags + per-tag/exact verification | Registry adapter | Covered only after an explicit finalize step. Verify exact and channels from fresh remote state; do not replay a stale plan. |
| **OP-20** logout | No binary login in native design | Reasonable deletion if credentials are passed directly to the oras-go client and never persisted. Document that property. |

### Invariant list

- **Handled:** 1 (upstream draft), 2 (tag/run binding), 5 (byte preservation), 6 (canonical shape), 11 (exact tag plan), 12 (monotonic channels), 15 (verification-only), and 16 (no-cancel residual concurrency). Draft §9 lines 247–260 maps these credibly to inventory lines 310–325.
- **Handled by residual YAML but omitted/blurred in the table:** 3 (registry-before-release caller gate), 13 (serialization is only guaranteed for reusable-workflow users), and 17 (GoReleaser publication disabled twice).
- **Under-specified:** 4 (metadata, download transport digest, and post-download content digest are conflated), 7 (closed bundle lacks a file-source boundary), 8 (subjects are computed but attestation success is external), 9 (GitHub attestation barrier before upload is not explicit), and 10 (expected-name replacement semantics are absent).
- **Broken:** 14, trust metadata before public tags, because a single `publish oci` invocation cannot cross a residual YAML attestation step.

## Failure-mode and rerun analysis

| Scenario | Status | Analysis and required action |
|---|---|---|
| **Tag application fails halfway** | **Undefined; conditionally recoverable** | Draft §9 lines 256–258 promises serial application but not read-after-error or fresh replanning. OP-19 can leave a prefix of tags updated. A rerun is safe only if it discards the old plan, re-resolves every tag, treats same version/digest as complete, preserves newer channels, and resolves ambiguous 5xx writes before proceeding. Never label the initial exit generically safe (inventory lines 320–323). |
| **Rerun of a failed job** | **Code pin handled; remote state not handled** | `actions-constraints.md` lines 166–169 says rerunning only failed jobs reuses the original callee SHA. That prevents code drift. It does not undo partial uploads/tags or an undraft. Artifact same-run checks remain valid within the run. Add command-level reconciliation and structured phase status (draft §2 line 61). |
| **Rerun after a channel already advanced** | **Handled if state is re-read** | `PlanTags` can correctly hold a newer channel and accept the same digest; inventory invariant 12 requires that behavior. It becomes dangerous if a persisted/stale `TagPlan` is applied without re-resolution after attestations or a retry. |
| **Concurrent releases** | **Handled only for one reusable-workflow consumer repository** | Residual repository-wide concurrency with no cancellation preserves the current case (draft §9 lines 257, 260; inventory lines 322, 325). A direct CLI/composite user has no lock, and two consumer repositories overriding the same registry target have different Actions concurrency domains. Declare a single-writer requirement for shared targets or add a real cross-caller lock/CAS strategy later; at minimum recheck state immediately before tag writes. |
| **Registry 5xx/429** | **Undefined** | Content-addressed blob/manifest pushes are naturally replayable, but tag PUT results can be ambiguous. Draft supplies no classified errors, `Retry-After`, backoff limit, or read-after-write check. Implement this with the remote slice, not a generic framework (AGENTS.md E3; inventory OP-13/14/19). |
| **App token expires mid-run** | **Undefined** | `TokenMinter` returns no expiry or refresh capability despite the documented one-hour lifetime (`actions-constraints.md` line 113). Keep the official token action or use a refreshing transport; partial expected-asset uploads must remain reconcilable. |
| **Clock skew** | **Mostly external/live-only, not proved by fake clock** | Core version/tag decisions and build dates use semantic versions and tagged-commit time, so they are not wall-clock dependent. Artifact expiry is server-reported. App JWT creation, OIDC/Cosign, TUF metadata validity, and attestation verification can fail on runner skew. The fake clock at draft §4 line 123 only proves poll schedules; retain live rehearsal coverage and avoid adding local-clock expiry decisions where server state is available. |
| **Consumer pins a stale CLI** | **Actively dangerous when explicitly overridden** | `$/` keeps the default action source aligned, but draft §8 permits `cli-version` override and has no protocol handshake. A stale binary may parse an older staging schema or omit a new invariant while still reporting the requested semantic version. Stamp/verify the required protocol and reject incompatibility before side effects (`actions-constraints.md` lines 181–192, 406). |
| **GitHub upload fails midway** | **Recoverable while still draft, under-specified** | Inventory invariant 10 permits replacing expected names and blocks unexpected ones. Recovery needs exact clobber/delete semantics and remote digest convergence. If undraft already succeeded but the final fetch failed, the current “safe rerun” claim is false (findings 6, 10). |

## Testability claim audit

The proposed split makes **decision logic** laptop-testable, not the whole pipeline. That is still a major improvement, but draft §10 line 274 overstates `go test ./e2e` as “full pipeline flows.” Against in-process fakes it proves CLI wiring, plans, ordering, local verification, and adapter request construction—not GitHub Actions, Sigstore, GHCR, QEMU, or the pinned tool contracts. AGENTS.md T1 explicitly reserves the third layer for live-service end-to-end coverage; the existing rehearsal can satisfy it, but the fake-based `e2e/` directory is more accurately a system/integration layer.

### Are the seams sufficient?

- **Pure planners:** Yes for stable version parsing, exact/channel planning, checksum grammar, subject derivation, and layout predicates—provided exact index bytes are passed through rather than re-marshaled.
- **Engine mocks:** Sufficient for call order, failure short-circuiting, dry-run, and bounded retry decisions after a classified error model exists. They are not evidence that an adapter implements the real external contract.
- **Temp-dir filesystem:** Sufficient for path confinement, regular-file/symlink rejection, closed-set validation, streaming hashes, staging JSON round trips, layout traversal, and byte-preservation. It requires the missing narrow file boundaries from finding 2.
- **In-memory OCI registry:** Sufficient for distribution-protocol blob/manifest/tag behavior and planner integration. It does not prove GHCR authentication, referrers, Cosign recursive signatures, GitHub attestation pushes, GHCR consistency, or registry-specific errors. Draft §12 line 303 correctly retains a scratch-GHCR experiment.
- **Fake GitHub API:** A static sequence of recorded fixtures is insufficient for GR-06 and GR-14–18. The fake must be stateful: paginate, delay draft visibility, delay asset digest/state convergence, allow expected-name replacement, expose duplicate/unexpected assets, return rate limits/5xx/401, and model ambiguous mutations. Recorded sanitized responses can seed schemas, but they cannot be the behavior model.
- **Fake clock:** Sufficient for deterministic poll budgets and cancellation. Insufficient for server `Retry-After`, token refresh, JWT clock skew, or real eventual consistency without classified adapter responses.

### Where a mock can pass while the real tool fails

1. **Cosign:** Draft §4 lines 112–113 reduces verification/signing to one mock call. A mock cannot prove exact `verify-blob` identity/issuer flags, TUF/offline behavior, `--recursive` covering index plus both platform manifests, OIDC acquisition, or referrer storage. Add an adapter contract test using a real pinned Cosign and a fixed verification bundle where feasible; keep recursive signing live against scratch GHCR.
2. **Melange:** `BuildSpec` can look correct while argument order, relative source/repository paths, Docker/QEMU behavior, ephemeral key permissions, exactly-one-APK output, or `--generate-provenance` differs. Test pure argument construction and run a tiny pinned-tool smoke in the rehearsal.
3. **apko:** A mock cannot prove lock/build path context, repeated `--arch`/repository/key flags, annotation encoding, exact output layout, or SBOM filenames. Validate a real tiny build in the pinned environment.
4. **GoReleaser:** A hand-built `artifacts.json` can drift from GoReleaser 2.17.1’s emitted schema and path conventions. Keep a fixture captured from the exact pin and a rehearsal that runs the real command with `--skip=publish`.
5. **go-github:** A permissive fake can accept duplicate upload names even though production requires clobber/delete behavior; it can also omit pagination and eventual asset digests. Use the stateful behavior above.
6. **oras-go vs GHCR:** An in-memory distribution registry will not prove GHCR referrer/signature/attestation interactions. Do not promote the scratch-GHCR spike to optional polish; it is the acceptance proof for slice 3.

### What remains CI/live-only

Actual Actions artifact ZIP digest verification and run ownership, reusable-workflow `$/` behavior and runner minimum, OIDC identity claims, `actions/attest` subject/referrer creation, GitHub App installation selection/token lifetime, GHCR auth/referrers/eventual consistency, Cosign Fulcio/Rekor behavior, QEMU/binfmt, Docker-backed melange/apko, and the caller/callee permission ceiling. The draft already recognizes many of these; its test naming and claims should reflect the boundary honestly.

## Over- and under-engineering

### Delete or defer from slice 1

1. **Do not scaffold the full §5 package tree.** Draft lines 141–162 list roughly twenty production/test packages, but slice 1 needs only CLI wiring, the minimal release/checksum/staging model, the Go artifact selector, narrow file adapters, Actions outputs, and their required generated mocks. Create registry/GitHub/image/clock packages only in the slice that uses them. This follows AGENTS.md R1 and the owner’s prototype-first mandate.
2. **Defer `release.toml`, `config show`, and most config schema until a real non-flag setting exists.** Draft §1 lines 24–25 and §7 lines 193–213 add config-file discovery the template does not have (`template-conventions.md` §2). Slice 1 can inject the Go profile/config path directly. If `config show` is retained later, design secret exclusion first.
3. **Use an explicit `cli-path` for dogfood before inventing `version: local`.** `actions-constraints.md` lines 245–255 already gives the simpler supported shape: the workflow builds the branch binary, and the composite accepts its path. It removes checkout-source ambiguity and lets the action focus on install/verify/invoke. A convenience `local` mode can be added only if it proves useful.
4. **Do not create future-profile generality beyond the current contract.** Remove zero/N-binary and container-only promises at draft lines 54 and 189. Keep the profile seam, but make v1’s staging projection exactly describe what current Go packaging requires.
5. **Do not make every result field stable before it exists.** Keep the top-level JSON envelope/version in slice 1, but stabilize command-specific payloads only as each command ships. Versioning already gives room to learn.
6. **Drop `Asset.Size` unless a current invariant uses it.** The current closed-set contract relies on names, regular-file status, and digests, not local size. Adding a field that `ParseChecksums` cannot populate creates invalid intermediate states.

### Dangerously thin areas

1. Residual-YAML phase barriers and receipts (critical finding 1).
2. Slice-1 filesystem ports and staging artifact placement (findings 2–3).
3. Remote error classification, bounded retries, and ambiguous-write reconciliation (findings 6 and 9).
4. Workflow/CLI protocol compatibility and installer provenance binding (finding 7).
5. Expected-asset replacement semantics under go-github (finding 10).
6. Secret redaction and App-token expiry/refresh (finding 8).
7. Direct-CLI/shared-target concurrency: reusable-workflow serialization is not a property of the binary.

## Missed or too-quickly-dismissed alternatives

1. **OCI prepare/finalize around YAML attestations.** This is not merely an alternative; it is required unless attestations move into the CLI. Cost: one versioned plan artifact and another command invocation. Benefit: preserves invariant 14 without reimplementing GitHub attestation APIs.
2. **Keep `actions/create-github-app-token` as the default.** Draft §12 line 299 treats this as a later fallback. It should be the baseline until native minting proves better. Cost: one residual YAML step and token plumbing. Benefit: the CLI never handles the App private key and need not invent token lifecycle behavior.
3. **Use `cli-path` instead of magic `version: local`.** Constraints already recommend it. Cost: one explicit build step in dogfood jobs. Benefit: exact source is visible, no action-side checkout assumptions, and the same composite works for any caller-built binary.
4. **Keep the ORAS binary behind narrow ports while moving all decisions into pure code.** Draft §12 line 310 dismisses shelling out as making logic untestable, but planner logic does not need to live in the adapter. Cost: harder remote fakes and subprocess parsing. Benefit: exact continuity with current ORAS 1.3.3 behavior and fewer GHCR semantic surprises. Native oras-go remains a reasonable choice, but only after the slice-3 parity spike.
5. **Retain `gh release upload --clobber` only for replacement.** The rest of GitHub release reads/mutations can use go-github. Cost: one subprocess adapter. Benefit: preserves the current convergent replacement contract without reimplementing delete/upload edge cases immediately.
6. **One combined producer artifact instead of two manifests/projections.** Cost: consumers download more bytes and existing workflow outputs change internally. Benefit: one complete staging contract and simpler provenance. Compare it with explicit projected manifests during slice 1; do not let the current two-artifact shape and the claimed one-manifest shape coexist accidentally.
7. **Direct branch-built CLI step for the first slice, composite after the core works.** Cost: slice 1 does not simultaneously prove public installation. Benefit: isolates the architecture experiment from `$`, release stamping, and bootstrap verification. If the composite remains in slice 1, keep its first version to `cli-path` dogfood and test released installation separately.

## Ordered punch list

### Must fix before coding the affected slice

1. **Before slice 1:** choose the staging transport contract across `oci-build-inputs` and `release-assets`; define which projection each downstream command validates and where the manifest is uploaded.
2. **Before slice 1:** add narrow filesystem/use-case boundaries for checksum parsing, bundle closure, canonical Go artifact selection/path confinement, executable metadata, and manifest persistence. Do not add a generic FS wrapper.
3. **Before slice 1:** split `ChecksumSet` parsing from validated `Bundle` construction; remove fields that cannot be validly produced, and constrain the first profile to the actual two-static-ELF contract.
4. **Before slice 1’s composite:** use `cli-path` for dogfood or fully specify/test blank-version fallback, owner guard, runner minimum, source repository derivation, and CLI protocol compatibility. Do not claim same-commit binary provenance without verifying it.
5. **Before slice 2:** correct the handoff ownership model: CLI metadata validation, Action download transport verification, then CLI content verification.
6. **Before slice 3:** split OCI prepare from post-attestation finalize (or move attestations into CLI); finalize must re-read remote state, apply tags serially, and verify all postconditions.
7. **Before slice 3:** replace broad `Registry` with narrow consumer ports and define classified registry errors plus ambiguous-write reconciliation.
8. **Before slice 4:** replace broad `ReleaseStore` with narrow ports and decide official App-token action versus refresh-capable native auth.
9. **Before slice 4:** define expected-asset replacement semantics and post-undraft failure behavior; remove the blanket “exit 3 is safe to rerun” promise.

### Fix during slice 1

1. Preserve PP-01 and PP-03–05 in YAML; replace only PP-06–08 as stated.
2. Verify strict checksum grammar, every payload digest, nonempty control bundle, exactly one Linux amd64 and one Linux arm64 binary, confined regular paths, and executable status.
3. Stream hashes; reject symlinks, directories, duplicate names, controls listed as payloads, and unlisted bundle entries where closure is in scope.
4. Emit the chosen versioned manifest/projections and include them explicitly in the residual upload steps.
5. Keep the result envelope small and versioned; write machine data to stdout only under `--json`, diagnostics to stderr, and Actions outputs through the adapter.
6. Build only slice-1 packages/adapters/mocks; do not scaffold future registry, release, image, token, or clock packages.
7. Run the already-planned rehearsal specifically to prove the real pinned GoReleaser `artifacts.json` and uploaded artifact shapes, not only the txtar fixture.

### Revisit later

1. Native App token minting after a secret-exposure/refresh comparison; otherwise retain the pinned action.
2. oras-go versus the ORAS binary after GHCR parity for digest push, tag resolution, referrers, and partial failure.
3. Native attestation only if it materially simplifies the prepare/finalize barrier and keeps GitHub permission/identity semantics intact.
4. Offline `sigstore-go` bundle verification after measuring TUF/network pain; keep exact identity/issuer behavior as the acceptance contract.
5. Non-Go and container-only profiles only after the staging/image seam has evidence from a second real ecosystem.
6. A config file only when durable settings justify it; keep secrets outside printable effective config.
7. Cross-repository/shared-registry locking only when a real consumer needs a shared target; until then document the single-writer limitation.

## Final verdict

**Approve with changes:** begin slice 1 only after fixing its filesystem and staging-handoff contracts, and reject the current OCI publication command shape because it cannot preserve attestation-before-tag ordering.
