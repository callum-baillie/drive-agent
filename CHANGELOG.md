# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]
### Fixed
- Added global `--path` support for drive-scoped commands.
- Added cleanup flag aliases: `cleanup --dry-run` and `cleanup --apply`.
- Hardened self-update checksum lookup, HTTP timeouts, metadata writes, and rollback backup listing.

## [0.1.0-alpha.3] - 2026-05-04
### Added
- Published GoReleaser artifacts for macOS, Linux, Windows, and `checksums.txt`.
### Fixed
- Release workflow can publish artifacts.

## [0.1.0-alpha.2] - 2026-05-04
### Fixed
- Alpha release publishing pipeline iteration.

## [0.1.0-alpha.1] - 2026-05-03
### Added
- Initial Drive Agent MVP prerelease attempt.
