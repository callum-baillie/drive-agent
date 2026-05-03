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
git tag v0.1.0-alpha.1
git push origin v0.1.0-alpha.1
```

## What Happens Next

- Pushing the tag triggers the GoReleaser GitHub Action.
- GitHub Release artifacts (`.tar.gz` and `.zip`) are generated for supported OS/architectures.
- A `checksums.txt` file is generated. This file is critical for the `self update` command to perform SHA256 integrity verification.

*(Note: `checksums.txt` provides integrity, but not signing. Future TODO: Add release signing with cosign/minisign/GPG).*

## After the First Release

Once the first release is published, you can test the `self update` command locally against the real release.
