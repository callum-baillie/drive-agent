# TODOs and Known Issues

## P0 — Deferred / Tracked

### [TODO-001] Track pure-Go SQLite release behavior

**Current state:** Drive Agent uses `modernc.org/sqlite`, a pure-Go SQLite driver, so release builds can run with `CGO_ENABLED=0`.

**Tradeoff:**
- The pure-Go driver simplifies cross-compilation and avoids C toolchain drift in GitHub Actions.
- Slightly larger binary, slightly lower performance (~10-30% slower on write-heavy workloads)
- API usage remains behind `database/sql`, so the application surface is unchanged.

**Current DB open path:**
```go
import _ "modernc.org/sqlite"
db, _ := sql.Open("sqlite", "file:/path/to/drive-agent.sqlite?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)")
```

Keep DB tests covering WAL, foreign keys, and busy timeout behavior before changing drivers again.

---

### [TODO-002] `project reindex --repair` to prune missing DB rows

Currently `reindex` reports DB entries whose folders are missing on disk but
does not delete them. A future `--repair` flag should offer to prune stale entries.

---

### [TODO-003] Full backup provider implementation (restic adapter)

See `docs/backup.md` for the manual setup guide. The restic adapter should:
1. Wrap `restic init`, `restic backup`, `restic check`, `restic snapshots`
2. Store repo config in `.drive-agent/config/backup.toml`
3. Support multiple backup targets (local, S3, Backblaze B2, rclone)
4. Create a LaunchAgent (macOS) or systemd unit (Linux) for scheduled backups

---

### [TODO-004] Self-update release signing

Self-update now downloads GitHub release assets and verifies SHA256 checksums from `checksums.txt`. It does not yet verify publisher authenticity.

Future work:
1. Add release signing with cosign, minisign, or GPG.
2. Verify signatures before applying a downloaded update.
3. Run any pending schema migrations after a successful update.

---

### [TODO-005] Rich interactive TUI for `host setup` and `host packages install`

Use `github.com/charmbracelet/bubbletea` for a checkbox-based package selector.
The current implementation uses simple readline prompts.

---

### [TODO-006] `git push-all` (explicit per-repo confirmation)

Not implemented for safety. If added, must require explicit per-repo confirmation,
show the remote and branch before each push, and default to `--dry-run`.

---

### [TODO-007] Port registry for local development

Track which projects use which local ports (e.g., 3000, 5432, 8080) in the database
to prevent conflicts when running multiple projects simultaneously.
