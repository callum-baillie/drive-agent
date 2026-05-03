# TODOs and Known Issues

## P0 — Deferred / Tracked

### [TODO-001] Evaluate `modernc.org/sqlite` for pure-Go releases

**Current state:** Drive Agent uses `github.com/mattn/go-sqlite3` which requires CGO.

**Tradeoff:**
- `go-sqlite3` is the most battle-tested SQLite binding for Go. It wraps the official SQLite C library exactly, giving 100% SQLite feature parity and the best performance.
- However, it **requires CGO** (`CGO_ENABLED=1`), which means:
  - Cross-compilation requires a C cross-compiler toolchain for the target OS/arch
  - The resulting binary is dynamically linked to system C libraries (normally fine on macOS/Linux)
  - CI pipelines need to install `gcc` or equivalent
  - Pure-Go `go install` (without CGO) will fail

**Alternative: `modernc.org/sqlite`**
- A pure-Go port of SQLite (auto-translated from C)
- Enables `CGO_ENABLED=0` builds, simplifying cross-compilation
- Slightly larger binary, slightly lower performance (~10-30% slower on write-heavy workloads)
- API is nearly identical (drop-in replacement for the `database/sql` driver)
- Actively maintained; used in production by projects like CockroachDB tooling

**Recommendation:** Evaluate `modernc.org/sqlite` when adding a CI release pipeline (GitHub Actions). The pure-Go binary is easier to distribute cross-platform. For single-user CLI tools on macOS, the performance difference is negligible.

**Migration path:**
```go
// Current (go-sqlite3):
import _ "github.com/mattn/go-sqlite3"
db, _ := sql.Open("sqlite3", path)

// Future (modernc):
import _ "modernc.org/sqlite"
db, _ := sql.Open("sqlite", path)
```

The only change needed is the import path and driver name string.

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

### [TODO-004] Self-update from GitHub releases

Currently stubbed. Implementation plan:
1. Query `https://api.github.com/repos/callum-baillie/drive-agent/releases/latest`
2. Download the appropriate binary for `GOOS/GOARCH`
3. Verify SHA256 checksum against the release's `checksums.txt`
4. Backup current binary to `.drive-agent/releases/drive-agent-v<version>`
5. Atomic swap: write to temp file, then `os.Rename` to final location
6. Run any pending schema migrations

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
