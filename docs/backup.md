# Backup

## Current Status

The backup system is **stubbed** in the MVP. Commands exist but print guided setup instructions rather than executing backups.

## Available Commands

```bash
drive-agent backup status    # Shows which backup tools are installed
drive-agent backup init      # Planned: initialize backup config
drive-agent backup run       # Planned: run a backup
drive-agent backup check     # Planned: verify backup integrity
```

## Manual Setup Guide

### Using Restic (Recommended)

```bash
# Install
brew install restic

# Initialize repository
restic init -r /Volumes/BackupDrive/restic-repo

# Run backup
restic backup /Volumes/DevDrive \
  --exclude node_modules \
  --exclude .next \
  --exclude .turbo \
  --exclude .cache \
  --exclude dist \
  --exclude build \
  --exclude coverage \
  --exclude .DS_Store \
  --exclude .expo \
  --exclude "android/.gradle" \
  --exclude "ios/Pods" \
  --exclude vendor \
  --exclude target

# Check integrity
restic check -r /Volumes/BackupDrive/restic-repo

# List snapshots
restic snapshots -r /Volumes/BackupDrive/restic-repo
```

### Using Kopia

```bash
brew install kopia
kopia repository create filesystem --path /Volumes/BackupDrive/kopia-repo
kopia snapshot create /Volumes/DevDrive
```

## Future Plan

The backup system will support provider adapters for:
- **restic** — Encrypted, deduplicated backups
- **kopia** — Fast, encrypted backup with GUI
- **rclone** — Cloud storage sync
- **rsync** — Local file sync
- **Time Machine** — macOS native

The internal structure (`internal/commands/backup/`) is designed to accommodate these providers.
