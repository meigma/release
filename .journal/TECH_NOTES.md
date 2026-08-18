# Technical Notes

- `main` contains the GitHub Release and OCI delivery MVP at squash commit `5566640c061c5e36f3715e0a1b57eaf69646a0ba` from PR #2.
- The release boundary is `GoReleaser canonical assets -> digest-checked Actions artifacts -> independent GitHub Release and OCI publishers`; privileged publishers do not rebuild consumer source.
- OCI packaging is `canonical Linux binary -> Melange-signed APK repository -> locked apko multi-platform layout`. The publisher signs the index and platform manifests, creates index provenance and per-platform SPDX attestations, then assigns immutable and monotonic tags.
- GitHub Release publication remains draft until its assets and any required digest-pinned OCI image are verified. Release mutation uses `MEIGMA_RELEASE_APP_CLIENT_ID` and `MEIGMA_RELEASE_APP_PRIVATE_KEY`.
- Current documentation and example callers pin pre-squash workflow revision `fb8c8098ff27968fb3070e928c00e925f38c698e`. Move consumers to a reviewed revision reachable from `main` before external adoption.
- Release Please opened PR #3 for `v0.1.0`; it was intentionally left unmerged at session close.
- Homebrew, MacPorts, Nix, Scoop, generalized installer, and mise delivery remain unimplemented.
