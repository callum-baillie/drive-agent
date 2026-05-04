# Release Process

Drive Agent uses GitHub Actions and GoReleaser to automate cross-platform builds and releases.

## Basic Release Flow

```bash
git checkout main
git pull
go test ./...
go vet ./...
go build ./cmd/drive-agent
bash tests/smoke_test.sh

# Tag the release
git tag v0.1.0-alpha.3
git push origin v0.1.0-alpha.3
```

## What Happens Next

- Pushing the tag triggers the GoReleaser GitHub Action.
- GitHub Release artifacts (`.tar.gz` and `.zip`) are generated for supported OS/architectures.
- A `checksums.txt` file is generated. This file is critical for the `self update` command to perform SHA256 integrity verification.
- The alpha artifact matrix is Darwin arm64, Darwin amd64, Linux arm64, Linux amd64, and Windows amd64. Windows arm64 is not published.

*(Note: `checksums.txt` provides integrity, but not signing. Future TODO: Add release signing with cosign/minisign/GPG).*

## After the First Release

Once the first release is published, you can test the `self update` command locally against the real release.

## Recovering from a Failed Alpha Release

If a tag exists but the GitHub Release was not created, leave the failed tag in place and create the next alpha tag. The early alpha tags are retained for this reason; `v0.1.0-alpha.3` is the current release verified by the audit flow.
