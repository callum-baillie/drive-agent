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

### [TODO-003] Backup provider expansion and scheduling

Restic is implemented as the first provider. Future backup work should:
1. Add managed schedules with LaunchAgent (macOS) or systemd timers (Linux)
2. Add additional provider adapters where useful (kopia, rclone, rsync, Time Machine)
3. Add secret-manager integrations for macOS Keychain, 1Password, Doppler, and similar tools
4. Add richer repository health metrics and retention/prune policy management

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

### [TODO-011] First-class host profile generator

Host setup can consume drive-local profiles, but profile generation is still a
manual audit-and-normalize workflow. Add a read-only command such as
`drive-agent host profile generate --name <name> --dry-run` that audits safe
developer package sources, maps them through the package catalog, reports
normalization decisions, and writes to `.drive-agent/config/host-profiles/`
only after confirmation.

---

### [TODO-006] `git push-all` (explicit per-repo confirmation)

Not implemented for safety. If added, must require explicit per-repo confirmation,
show the remote and branch before each push, and default to `--dry-run`.

---

### [TODO-007] Port registry for local development

Track which projects use which local ports (e.g., 3000, 5432, 8080) in the database
to prevent conflicts when running multiple projects simultaneously.

---

### [TODO-008] Centralize canonical path safety for `install.sh`

The Go CLI resolves symlinks and blocks protected paths in `internal/utils/safety.go`.
`install.sh` has its own shell-level checks for early install safety, but it should be
refactored to share or invoke the same canonical validation logic before copying
binaries or editing shell profiles.

---

### [TODO-009] Add contexts/timeouts around external command execution

Git, editor, and package-manager calls currently use `os/exec` without command
contexts. Add bounded timeouts or cancellation for long-running commands while
preserving interactive package install behavior where user input may be required.

---

### [TODO-010] Replace cleanup's broad `..` substring check with component-aware validation

Cleanup already validates targets against the resolved drive root and skips
symlinks. The extra `strings.Contains(cleanPath, "..")` guard is conservative
and may reject legitimate names containing two dots. Replace it with
component-aware traversal detection or remove it after equivalent tests prove
the drive-boundary checks cover the intended risk.
