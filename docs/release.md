# Release Process

Drive Agent uses GitHub Actions and GoReleaser to automate cross-platform builds and releases.

## Automated Releases

Releases are triggered automatically when a new Git tag starting with `v` is pushed to the repository.

```bash
git tag v0.1.0
git push origin v0.1.0
```

## GitHub Actions CI

The `.github/workflows/ci.yml` workflow runs on all pull requests and pushes to `main`. It executes:
- `go vet ./...`
- `go test -v ./...`
- `go build ./cmd/drive-agent`
- `tests/smoke_test.sh` (end-to-end integration test)

## Release Artifacts

The `.github/workflows/release.yml` workflow uses GoReleaser to build the current alpha artifact matrix:

- `drive-agent_Darwin_arm64.tar.gz`
- `drive-agent_Darwin_x86_64.tar.gz`
- `drive-agent_Linux_arm64.tar.gz`
- `drive-agent_Linux_x86_64.tar.gz`
- `drive-agent_Windows_x86_64.zip`
- `checksums.txt`

Windows arm64 is intentionally not published for the alpha release.

### Checksums

GoReleaser automatically generates a `checksums.txt` file containing SHA256 hashes of all artifacts. This file is critical for the `self update` command to verify the integrity of downloaded binaries before applying updates.

**Note:** Currently, this provides *integrity* checking (SHA256 checksum verification) but not publisher authenticity.
**TODO:** Add cryptographic release signing (using `cosign`, `minisign`, or GPG) to verify authenticity before applying self-updates.
