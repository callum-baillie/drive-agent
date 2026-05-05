# Safety Model

Drive Agent is designed with safety as the top priority. No destructive operation should ever run silently.

## Init Safety

- **Never erases, formats, or repartitions a drive**
- Refuses to initialize at dangerous paths:
  - `/`, `/Users`, `/System`, `/Library`, `/usr`, `/etc`, `/var`, `/tmp`, `/opt`
  - User's home directory (`$HOME`)
  - Any system directory
- On macOS, refuses paths outside `/Volumes/` unless `--allow-non-volume-path` is passed
- Refuses if `.drive-agent` already exists unless `--repair` is passed
- Shows confirmation if target has existing files

## Cleanup Safety

- **Defaults to dry-run** — `cleanup`, `cleanup --dry-run`, `cleanup scan`, and `cleanup dry-run` show what would be deleted
- **Requires `cleanup --apply` or `cleanup apply`** for actual deletion
- **Never deletes outside the drive root** — every path is validated with `IsPathInsideDrive()`
- **Never follows symlinks** — symlinked targets are skipped
- **Rejects suspicious paths** — paths containing `..` are refused
- Requires confirmation before deletion unless `--yes`

## Git Safety

- **`git push-all` is not implemented** — too dangerous for automatic use
- `git pull-all` skips dirty repos and detached HEAD by default
- `git fetch-all` is safe (read-only operation)
- Mutating git commands support `--dry-run`

## Host Setup Safety

- **Never installs without explicit consent**
- **Never runs `sudo` silently** — requires user awareness
- **Never runs `curl | bash` silently** — native installers require `requiresExplicitApproval`
- Profile setup supports cache modes:
  - `host-local` leaves package-manager caches unchanged
  - `external-drive` shows planned npm, pnpm, Bun, and Homebrew cache paths
  - `disabled` makes no cache config changes and deletes nothing
- Docker storage setup prefers external bind-mount roots over daemon relocation
- Drive Agent does not edit Docker Desktop or OrbStack daemon storage automatically
- `--yes`/`-y` reduces prompts only for normal package installs and explicit cache/Docker modes; it does not bypass safety checks
- Packages marked `requiresExplicitApproval` remain skipped with `--yes` unless `--include-explicit` is supplied
- `--force` may attempt installs even when a package appears installed, and should only be used after manual review
- Homebrew cask packages can declare `check.appBundles` so manually installed `.app` bundles are treated as installed and skipped by default
- Shell config edits:
  - Create a backup before modification (`.zshrc.drive-agent-backup` or installer-created dated backup)
  - Use marked blocks (`# >>> drive-agent >>>` / `# <<< drive-agent <<<`)
  - Persist portable cache/container exports in a separate block (`# >>> drive-agent storage >>>` / `# <<< drive-agent storage <<<`)
  - Can be identified and removed later
- Supports `--dry-run` to preview all changes
- Package installation always shows exact commands before execution

## Database Safety

- SQLite is an **index, not the source of truth**
- Every project has a `.drive-project.toml` manifest on disk
- Database can be rebuilt from manifests with `project reindex`
- Drive config is stored as `drive.toml` on disk
- Host state is stored as JSON files alongside database records

## Package Safety

- Packages with `requiresExplicitApproval: true` are never auto-installed
- Install plan is always shown before execution
- Category and individual package selection are both supported
- `--dry-run` shows exact commands without running them

## Backup Safety

- Restic passwords are never stored in Drive Agent config, state, logs, or DB records
- S3 access keys must be provided through the current shell environment or a local secret manager, never through docs, config, manifests, logs, or SQLite
- Supported password sources are `RESTIC_PASSWORD` and `RESTIC_PASSWORD_FILE`
- Project-level excludes are stored in `.drive-project.toml` and scoped to that project path in the generated Restic exclude file
- Local repositories inside the source drive are rejected unless `--allow-same-drive-repo` is passed
- Same-drive repositories print a warning that they are not real backups
- Restore refuses active-drive targets and protected system paths
- Restore never deletes target contents automatically
- `backup check` uses `restic check`; expensive full data checks are not the default
