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

- **Defaults to dry-run** — `cleanup scan` and `cleanup dry-run` show what would be deleted
- **Requires `cleanup apply`** for actual deletion
- **Never deletes outside the drive root** — every path is validated with `IsPathInsideDrive()`
- **Never follows symlinks** — symlinked targets are skipped
- **Rejects suspicious paths** — paths containing `..` are refused
- Requires confirmation before deletion unless `--yes`

## Git Safety

- **`git push-all` is not implemented** — too dangerous for automatic use
- `git pull-all` skips dirty repos and detached HEAD by default
- `git fetch-all` is safe (read-only operation)
- All commands support `--dry-run`

## Host Setup Safety

- **Never installs without explicit consent**
- **Never runs `sudo` silently** — requires user awareness
- **Never runs `curl | bash` silently** — native installers require `requiresExplicitApproval`
- Shell config edits:
  - Create timestamped backup before modification (`.zshrc.drive-agent-backup-2026-05-03`)
  - Use marked blocks (`# >>> drive-agent >>>` / `# <<< drive-agent <<<`)
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
