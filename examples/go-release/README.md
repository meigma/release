# Go release example

This directory models a repository named `example` with module `example.com/meigma/release-consumer` and command `./cmd/example`. It contains the minimum source needed to build GitHub Release assets and a multi-architecture OCI image. It is not a complete CI policy: add the consumer repository's own build, test, review, and branch-protection controls.

See [Configure GitHub Releases](../../docs/how-to/configure-github-releases.md) for credential setup, adoption, and verification. See [Rehearse and recover GitHub Releases](../../docs/how-to/rehearse-and-recover-github-releases.md) before the first publication. Use [Upgrade GitHub Release workflows](../../docs/how-to/upgrade-github-release-workflows.md) to change the pinned revision. The reusable workflow interface is defined in the [GitHub Release contract](../../docs/reference/github-release-contract.md).

## Files to copy

Copy these release files into an existing Go repository, preserving their paths:

- `.github/workflows/release-please.yml`
- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `apko.yaml`
- `melange.yaml`
- `.release-please-manifest.json`
- `release-please-config.json`
- `mise.toml`
- `mise.lock`

To reproduce the complete minimal consumer in a new empty repository, also copy:

- `go.mod`
- `cmd/example/main.go`

Do not copy this README into the consumer repository.

## Values to replace

Replace these project-specific example values:

- `example.com/meigma/release-consumer` in `go.mod` with the consumer's module path.
- `./cmd/example` in `.goreleaser.yaml` with the consumer command package.
- Project name, build ID, archive ID, binary name, and Release Please package name `example` with the consumer's project and binary names.
- Package name, description, license, and installed command path in `melange.yaml`.
- Package name, entrypoint, image annotations, and source URL in `apko.yaml`.
- The literal command name and default output in `cmd/example/main.go` if you copy the sample command.
- Branch name `main` in `.github/workflows/release-please.yml` if the consumer uses another default branch.
- `initial-version` value `0.1.0` in `release-please-config.json` if the first intended release differs.
- Manifest value `0.0.0` in `.release-please-manifest.json` if the consumer already has a release. Use its latest released version without the `v` prefix.
- Linker variables `main.version` and `main.commit` in `.goreleaser.yaml` if the consumer command exposes version data through different variables. The copied sample defines both variables and prints `example <version> (<commit>)` for `--version`.

Keep these contract values unchanged:

- all three reusable workflow references at `f880a4cefdad74cf65d22688cd64e7238390f52d`;
- `checksum-signing-workflow-ref` value `meigma/release/.github/workflows/go-pre-publish.yml@f880a4cefdad74cf65d22688cd64e7238390f52d`;
- organization variable `MEIGMA_RELEASE_APP_CLIENT_ID`;
- organization secret `MEIGMA_RELEASE_APP_PRIVATE_KEY`; and
- the locked Go 1.26.6, GoReleaser 2.17.1, Syft 1.51.0, Cosign 3.1.3, GitHub CLI 2.97.0, Melange 0.59.1, and apko 1.2.37 versions unless the shared workflow contract is deliberately updated.

The caller sets `publish-release: false` so the first run leaves a populated draft. After inspecting that draft, change the input to `true` and follow the recovery guide to publish through the same tag and release.

The OCI job retains the signed APK repository, apko lock, SPDX files, and OCI layout in the `oci-image` workflow artifact. It does not publish the image to a registry.
