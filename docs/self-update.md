# Self Update & Rollback

Drive Agent can securely update itself in place, directly on your external drive.

## Checking Version

To see your current version and the installation path on the drive:

```bash
drive-agent self version
```

## Updating

To fetch the latest release from GitHub and apply it:

```bash
drive-agent self update
```

You can also update to a specific version or test the update with a dry-run:

```bash
drive-agent self update --version v0.1.1
drive-agent self update --dry-run
drive-agent self update --yes
```

### Update Security & Safety

1. **Path Validation:** The update command refuses to run if the binary is not located in the `.drive-agent/bin` directory.
2. **Integrity Check:** Downloads `checksums.txt` from the GitHub Release and performs SHA256 checksum verification on the downloaded archive to ensure the file was not corrupted during transit. (Note: Authenticity verification via cryptographic signatures is planned for a future release).
3. **Automatic Backups:** Copies your existing binary to `.drive-agent/backups/drive-agent-<timestamp>` before replacing it.
4. **Atomic Swap:** Attempts to `mv` the new binary into place. If it fails, it automatically restores the backup.

## Rollback

If an update breaks functionality, you can easily revert to a previous version stored on the drive:

```bash
# List available backups
drive-agent self rollback --list

# Revert to the most recent backup
drive-agent self rollback

# Revert to a specific backup
drive-agent self rollback --backup drive-agent-v0.1.0-20231015120000
```
