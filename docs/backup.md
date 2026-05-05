# Backup

Drive Agent supports real backups through Restic as the first provider. Restic remains the backup engine; Drive Agent handles safe configuration, excludes, status, and restore guardrails.

## Commands

```bash
drive-agent backup init --provider restic --repo /Volumes/BackupDrive/restic/devdrive
drive-agent backup status
drive-agent backup doctor
drive-agent backup run
drive-agent backup snapshots
drive-agent backup snapshots --json
drive-agent backup check
drive-agent backup restore --snapshot latest --target /Volumes/RestoreTest
drive-agent backup excludes list
drive-agent backup excludes add node_modules
drive-agent backup excludes add --project personal/my-app 'apps/*/node_modules'
drive-agent backup excludes list --project personal/my-app
drive-agent backup excludes remove .next
```

## Passwords

Drive Agent never stores Restic passwords in config, state, logs, or the database.

Supported password sources:

- `RESTIC_PASSWORD`
- `RESTIC_PASSWORD_FILE`

Do not put passwords directly in shell history. Use a prompt:

```bash
read -s RESTIC_PASSWORD
export RESTIC_PASSWORD
```

Future options to consider: macOS Keychain, 1Password CLI, Doppler, and other secret managers.

## Files

- Config: `.drive-agent/config/backup.json`
- State: `.drive-agent/state/backup.json`
- Logs: `.drive-agent/logs/backup/`
- Generated Restic exclude file: `.drive-agent/state/backup/restic-excludes.txt`
- Project excludes: `Orgs/<org>/projects/<project>/.drive-project.toml`

## Excludes

Backup excludes come from three places:

1. Global defaults in `.drive-agent/config/backup.json`
2. Project-level excludes in each `.drive-project.toml`
3. One-off CLI values from `backup run --exclude`

Global excludes are relative to the drive root. Project-level excludes are relative to the project root and are scoped to that project path when Drive Agent writes the Restic exclude file, so `node_modules` on one project does not become a project-specific rule for every other project.

Project manifest example:

```toml
[backup]
excludes = [
  "node_modules",
  ".next",
  ".turbo",
  "apps/*/node_modules",
  "apps/*/.next",
  "packages/*/dist",
]
```

Restic wildcard/glob patterns are preserved. Use source-safe excludes for generated dependencies and build output; do not exclude `.git`, lockfiles, source directories, migrations, docs, or project manifests unless you intentionally want them absent from backups.

Project-level CLI:

```bash
drive-agent backup excludes add --project personal/my-app node_modules
drive-agent backup excludes add --project personal/my-app 'apps/*/node_modules'
drive-agent backup excludes list --project personal/my-app
```

## Safety

- Local backup repositories inside the source drive are rejected by default.
- Same-drive repositories require `--allow-same-drive-repo` and print a warning that this is not a real backup.
- Backup dry-run does not require a password and prints the planned Restic command.
- Backup config stores repository locations and excludes only; do not store credentials in config, docs, logs, manifests, or SQLite.
- Restore refuses active-drive targets and protected system paths.
- Restore never deletes target contents automatically.
- `backup check` runs `restic check` only; expensive full data checks are not the default.

Before trusting backups, perform a restore test to a separate drive or disk image.
