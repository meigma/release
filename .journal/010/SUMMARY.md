---
id: "010"
title: Cross-organization release adoption
date: 2026-08-22
status: complete
repos_touched: [meigma/release]
related_sessions: ["008", "009"]
---

## Goal
Inventory every supported release destination, assess whether an outside GitHub organization can use `meigma/release` for one or more Go applications, remove the blockers found by that review, and replace the Meigma-specific documentation with a reusable adoption path.

## Outcome
The goal was met. PR 63 made package-repository signer policy explicit, removed the producer-repository coupling that blocked shared-workflow consumers, dual-licensed the repository under Apache-2.0 OR MIT, and replaced the existing documentation with a focused cross-organization Diátaxis set. An adopting organization can use all five publishers with its own GitHub App, destination repositories, Cloudflare R2 origin, and signing material, while the supported topology remains one application per repository. GitHub CI, Nix, and Kusari checks passed before squash merge as commit `2d524ae`.

## Key Decisions
- Store the full checksum certificate identity and attestation signer in each producer policy instead of deriving them from the producer repository -> reusable workflows may sign for repositories in other organizations without weakening exact workflow and revision checks.
- Make the policy schema a clean cutover -> strict YAML decoding rejects the ambiguous `checksum_workflow` and `attestation_workflow` fields rather than preserving two trust models.
- Keep one immutable release-unit revision across workflows, the setup action, CLI, and signer identities -> adopters cannot accidentally compose incompatible release components.
- Keep adopter credentials and destination ownership external to `meigma/release` -> no Meigma-owned App, private key, tap, bucket, receiver, R2 bucket, or signing key is required.
- Consolidate the README plus 17 documents into one tutorial, six how-to guides, two references, and one explanation -> each document serves one Diátaxis mode and duplicated setup, workflow, and recovery material has one maintained home.
- Use Apache-2.0 OR MIT throughout source and distribution metadata -> outside organizations have explicit reusable licensing terms.

## Changes
- `internal/stage/pkgrepo` and `internal/adapter/ghattest` - added strict `checksum_identity` and `attestation_signer` parsing, carried explicit trust values through publication, and allowed a verified signer repository to differ from the producer.
- `LICENSE-APACHE`, `LICENSE-MIT`, and release/package metadata - added full dual-license texts and consistent `Apache-2.0 OR MIT` declarations.
- `README.md` and `docs/` - replaced Meigma-specific setup and four overlapping contracts with the approved cross-organization tutorial, operator guides, references, and architecture explanation.
- `examples/go-release/` - added all supported publishers, one release-unit SHA placeholder, neutral adopter-owned destinations, and disabled-by-default publication controls.
- `examples/nix-release-cli/` - advanced the consumer example from `v0.1.3` to `v0.1.16`.

## Open Threads
- Release PR 64 must publish `v0.1.17` before consumers can select a released workflow unit containing the new package-policy schema.
- `meigma/pkgs` must migrate its producer policy to `checksum_identity` and `attestation_signer` before pinning a release that contains the clean cutover.
- Local `stash@{0}` in the main checkout preserves eight unexpected pre-merge worktree edits under `session-close: preserve pre-merge main worktree changes`; it was intentionally not reapplied over the merged implementation.

## Lessons
- A workflow interface can be repository-agnostic while its derived trust identity still prevents cross-organization use; external-adopter review must trace the final signer values, not only workflow inputs.
- Copyable examples are part of the supported product surface. Mixed immutable revisions and omitted optional publishers can make implemented capabilities effectively unavailable.

## References
- External-adoption review: `agent://CrossOrgReleaseReview`
- Implementation and documentation: https://github.com/meigma/release/pull/63
- Squash merge: `2d524ae89785f3de26c240d32dda2b45b45be85b`
- Follow-up release: https://github.com/meigma/release/pull/64
- Native package repository architecture and production context: `.journal/008/SUMMARY.md` and `.journal/009/SUMMARY.md`
