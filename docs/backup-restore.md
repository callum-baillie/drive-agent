# Backup Restore Guide

Restores are intentionally cautious. Drive Agent asks Restic to restore a snapshot, but validates the target first and never deletes existing files.

## Dry Run First

```bash
drive-agent backup restore --snapshot latest --target /Volumes/RestoreTest --dry-run
```

Then run the restore:

```bash
drive-agent backup restore --snapshot latest --target /Volumes/RestoreTest
```

You can restore a specific snapshot:

```bash
drive-agent backup restore --snapshot <snapshot-id> --target /Volumes/RestoreTest
```

## Target Rules

Allowed:

- A separate mounted drive or disk image under `/Volumes/...` on macOS
- An empty restore target is preferred
- A non-empty target is allowed only with a warning; Drive Agent does not delete existing files

Rejected:

- The active Drive Agent source drive
- `/`, `$HOME`, `/Users`, `/home`, `/System`, `/Library`, `/private`, `/usr`, `/opt`, `/tmp`, `/var`, or descendants
- macOS targets outside `/Volumes`

## Restore Test

Perform a restore test before trusting the backup. A safe pattern on macOS is:

```bash
hdiutil create -size 20g -fs APFS -volname DriveAgentRestoreTest /tmp/DriveAgentRestoreTest.dmg
hdiutil attach /tmp/DriveAgentRestoreTest.dmg

drive-agent backup restore --snapshot latest --target /Volumes/DriveAgentRestoreTest --dry-run
drive-agent backup restore --snapshot latest --target /Volumes/DriveAgentRestoreTest

hdiutil detach /Volumes/DriveAgentRestoreTest
rm /tmp/DriveAgentRestoreTest.dmg
```
