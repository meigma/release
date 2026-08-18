# Meigma release workflows

This repository defines the reusable workflows and repository contract that Meigma uses to build, sign, attest, and publish Go binaries through GitHub Releases.

Current documented automation covers GitHub Releases only. The repository does not yet provide OCI publication, package-manager publication, native package repositories, or installer distribution.

## Documentation

- [Configure GitHub releases](docs/how-to/configure-github-releases.md)
- [Rehearse and recover GitHub releases](docs/how-to/rehearse-and-recover-github-releases.md)
- [Upgrade GitHub release workflows](docs/how-to/upgrade-github-release-workflows.md)
- [GitHub release contract reference](docs/reference/github-release-contract.md)
- [Copyable Go release example](examples/go-release/)

Consumer repositories call the reusable workflows at a full commit SHA. The documented revision is `5be87cc60f2f11ac11fe401d8129c7644edc17ca`.
