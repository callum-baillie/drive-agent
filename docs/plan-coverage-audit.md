# Drive Agent Plan Coverage Audit

Audit date: 2026-05-04  
Plan source: `drive-agent-full-plan.md`  
Implementation audited: current `main` checkout plus installed drive layout at `/Volumes/ExternalSSD/.drive-agent`

## Summary

Drive Agent is still aligned with the original plan, but it is an MVP/alpha implementation rather than the full product described in `drive-agent-full-plan.md`.

Estimated coverage:

- Full plan coverage: about 60%.
- MVP roadmap coverage: about 75%.
- Safety-critical MVP coverage: about 80%, with remaining gaps concentrated around broader destructive-action tests, package-manager execution safety, release signing, and restore-test automation.

The implemented core is coherent: drive initialization, external-drive layout, SQLite indexing, org/project management, host profiles, package catalog planning, Git utilities, cleanup scanning, Restic backup support, and self-update/rollback are present. The larger missing areas are advanced host management, workspace/service/port/env systems, scheduled backup and additional backup providers, dashboard, richer cleanup and Git workflows, and full package-manager lifecycle abstraction.

## Coverage Table

| Area | Planned | Current Status | Evidence | Gap / Notes | Priority |
|---|---|---|---|---|---|
| Drive layout | External drive with `Orgs`, `DevData`, `Caches`, `BuildArtifacts`, `Tooling`, `Downloads`, `Inbox`, `Scratch`, `Trash` and `.drive-agent` internals | Implemented | `internal/config/config.go`, `/Volumes/ExternalSSD/.drive-agent` | `.drive-agent/lib`, `.drive-agent/scripts`, deeper template folders, and split catalog files are not created | Medium |
| `.drive-agent` metadata | marker, version, install metadata, bin, config, db, logs, state, backups, locks, templates, catalog, releases | Partially implemented | `/Volumes/ExternalSSD/.drive-agent`, `install.sh` | Uses `config/backup.json` rather than planned `backup.toml`; no `lib`/`scripts`; template subfolders are sparse | Medium |
| Command model | Single Go CLI with grouped commands | Implemented | `internal/cli/root.go`, help audit in `/tmp/drive-agent-help-audit.txt` | Missing planned top-level `package`, `workspace`, `service`, `ports`, `env`, `scan`, `dashboard` groups | High |
| Drive initialization | Non-destructive init, repair, macOS `/Volumes` safety | Implemented | `internal/commands/init/init.go`, `internal/utils/safety_test.go`, `tests/smoke_test.sh` | No migration framework beyond schema v1 | Medium |
| Organization management | add/list/archive/restore/delete/rename | Partially implemented | `internal/commands/org/org.go`, `internal/db/organizations.go` | Only add/list are exposed; DB has `archived` but no command surface | Medium |
| Project management | add/list/path/open/reindex/archive/restore/move/clone/templates | Partially implemented | `internal/commands/project/project.go`, `internal/db/projects.go` | Add/list/path/open/reindex exist; archive/restore/move/template workflows missing | High |
| SQLite schema | drive, hosts, organizations, projects, project_tags, settings, package install records, command_runs | Implemented | `internal/db/db.go`, external DB `.schema` | Schema exists; package records and command_runs have little write coverage today | Medium |
| Project manifests | `.drive-project.toml` as source of truth for rebuild | Partially implemented | `internal/config/config.go`, `project reindex` | Supports id/name/slug/org/type/package manager/tags/git remote/backup excludes; no explicit future metadata map; reindex does not prune stale rows | Medium |
| Host setup | interactive/non-interactive setup, profiles, package installs, shell setup, host state | Partially implemented | `internal/commands/host/host.go`, `internal/commands/host/setup_plan.go` | Profile dry-run, `--file`, cache modes, Docker guidance present; no rich TUI, profile generator, host list/export/current, or detailed package install recording | High |
| Package abstraction | provider registry across OS/package managers | Partially implemented | `internal/packages/providers/*.go` | Homebrew/cask and language-global providers exist; apt/dnf/pacman/winget/chocolatey/scoop are stubs; no search/upgrade/uninstall | High |
| Package catalog | categorized catalog and install preferences | Implemented | `catalog/packages.catalog.json` | Single JSON catalog, not split into packages/categories/managers as planned | Low |
| Host profiles | built-in defaults plus drive-local profiles | Implemented | `profiles/*.json`, `profiles/templates/macos-portable-dev.json`, `/Volumes/ExternalSSD/.drive-agent/config/host-profiles/mac-mini.json` | Host profile generation is manual; exact `mac-mini` profile is drive-local and intentionally not committed | Medium |
| Cache options | host-local, external-drive, disabled/no-change | Implemented | `internal/commands/host/setup_plan.go`, `setup_plan_test.go`, host dry-run | Applies only through host setup plan; no separate host cache command | Low |
| Docker/container storage | bind-mount roots preferred; daemon relocation only with caution | Implemented differently | `internal/commands/host/setup_plan.go`, `docs/host-setup.md` | Implements bind-mount/env guidance and folders; daemon data-root edit is guidance-only, not automated | Low |
| Git tools | status/fetch/pull/push with safety | Partially implemented | `internal/commands/git/git.go` | `status-all`, `fetch-all`, `pull-all` exist; `push-all`, branch/remotes/unpushed/dirty-specific reports missing | Medium |
| Cleanup tools | dry-run first cleanup, apply with confirmation, boundary checks | Partially implemented | `internal/commands/cleanup/cleanup.go`, `internal/utils/safety_test.go` | Core scan/apply exists; no include/older-than/all flags, no soft-delete/trash move, limited tests around cleanup command itself | Medium |
| Backup provider abstraction | Restic/Kopia/rclone/Time Machine abstraction | Partially implemented | `internal/backup`, `internal/backup/restic` | Provider interface and Restic exist; Kopia/rclone/Time Machine are future work | High |
| Restic backup | init/status/run/snapshots/check/restore/excludes/doctor | Implemented | `internal/commands/backup/backup.go`, `internal/backup/restic/provider.go` | Retention/prune, scheduled backup, and automated restore tests missing | High |
| Self-update/rollback | GitHub release update, checksum verification, rollback | Partially implemented | `internal/commands/self`, `install.sh`, `.github/workflows/release.yml`, `.goreleaser.yml` | Checksums and backups exist; no release signing, channels, or post-update migrations | Medium |
| GitHub release flow | CI, GoReleaser, release artifacts and checksums | Implemented | `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.goreleaser.yml` | Signing and richer prerelease channel policy are TODOs | Medium |
| Safety rules | no destructive defaults, path safety, no silent sudo/curl, backup repo safety, shell backups | Partially implemented | `internal/utils/safety.go`, `internal/backup/safety.go`, `install.sh`, tests | Good coverage for core dangerous paths and backup repo/restore targets; package installs can still execute package-manager commands when confirmed; release signing missing | High |
| Docs | README and feature docs | Partially implemented | `README.md`, `docs/*.md` | Current docs cover implemented MVP; full-plan audit gaps were not previously documented in one place | Medium |
| Tests | Unit, smoke, safety, package, backup, self-update | Partially implemented | `go test ./... -list .`, `tests/smoke_test.sh` | Missing command-level tests for org/project/git/cleanup/help, profile generator tests, scheduled backup tests, more install script safety tests | High |
| Future features | workspace, service, ports, env templates, audit scanners, dashboard, language-specific helpers | Deferred / documented | `drive-agent-full-plan.md`, `docs/todos.md` | Not implemented in current alpha | Future |

## Implemented

- Portable drive initialization and install workflow with `.drive-agent` marker, version, binary, config, db, logs, state, backups, catalog, and host-profile directories.
- Core command groups: `version`, `init`, `status`, `doctor`, `org`, `project`, `host`, `git`, `cleanup`, `backup`, and `self`.
- SQLite schema for drive, hosts, organizations, projects, project tags, settings, package install records, command runs, and schema version.
- Organization add/list and project add/list/path/open/reindex.
- Project manifests with ID, name, slug, org, type, package manager, tags, Git remote, created timestamp, and project-local backup excludes.
- Built-in host profiles: `minimal`, `developer`, `ai-developer`, `full-stack-saas`, and `mobile`.
- Drive-local host profiles under `.drive-agent/config/host-profiles/`, including the generated `mac-mini` profile on `/Volumes/ExternalSSD`.
- Package catalog with 132 package entries and categories including core, shell, JavaScript, Python, containers, cloud, AI dev, editors, databases, mobile, security, backup, API testing, documentation, productivity, Go/Rust, PHP, compilers, and package managers.
- Host setup dry-run planning for package managers, packages, cache modes, Docker bind-mount guidance, and shell block generation.
- Package provider execution for Homebrew, Homebrew casks, npm, pnpm, bun, pipx, uv, cargo, and `go install`.
- Git status/fetch/pull across projects with org/tag filters and dry-run for mutating fetch/pull commands.
- Cleanup scan/dry-run/apply for generated artifacts inside managed projects.
- Restic backup provider with init/status/run/snapshots/check/restore/excludes/doctor.
- Backup global and per-project excludes, including wildcard preservation and generated Restic exclude file.
- Same-drive backup repository rejection by default, safe restore target validation, and password-source detection without storing plaintext backup passwords.
- Self-update and rollback with GitHub release assets, checksum parsing, archive extraction, binary backup, and rollback listing.
- GitHub Actions CI and GoReleaser release workflow.

## Partially Implemented

- Host setup: profile use is solid, but richer interactive selection, first-class profile generation, host export/list/current commands, and full host state tracking are missing.
- Package manager abstraction: the registry exists, but non-Homebrew OS package providers are stubs and there is no search/upgrade/uninstall lifecycle.
- Project management: basic indexing and path workflows exist, but archival, restore, move, stale detection, and template scaffolds are missing.
- Cleanup: path-bounded cleanup exists, but planned soft-delete, older-than filtering, include overrides, large-file scans, duplicate scans, and richer logging are missing.
- Backup: Restic is implemented, but retention/prune, schedules, extra providers, secret-manager integrations, and automated restore verification are missing.
- Self-update: checksum verification and rollback exist, but release signing and post-update migrations are not implemented.
- Docs: current docs describe the implemented alpha well, but several future plan areas only existed in the plan until this coverage report.

## Not Implemented

- `drive-agent package ...` as a top-level package command group. Current equivalent is `drive-agent host packages ...`.
- `drive-agent host export`, `host list`, `host current`, or `host profile generate`.
- `drive-agent org rename`, `org archive`, `org restore`, and `org delete`.
- `drive-agent project archive`, `project restore`, `project move`, `project stale`, and template-based project creation.
- Workspace/service/port/env-template systems.
- Local service orchestration and service health registry.
- Port registry and conflict checks.
- Audit scanners such as large-file scan, duplicate scan, secrets scan command, and stale dependency scan.
- Dashboard or TUI.
- Kopia, rclone, rsync, and Time Machine backup providers.
- Backup schedules, retention policies, prune policies, and automated restore-test command.
- macOS Keychain, 1Password, Doppler, or other secret-manager integration for backup passwords.
- Release signing.

## Implemented Differently Than Planned

- Backup config is JSON at `.drive-agent/config/backup.json`; the plan described TOML-style config names.
- Host profiles are installed under `.drive-agent/config/host-profiles/`; the plan also described template-oriented locations under `.drive-agent/templates/host-profiles/`.
- Package catalog is a single `catalog/packages.catalog.json`; the plan described separate package, category, and package-manager catalogs.
- Package commands live under `host packages`, not a standalone `package` namespace.
- Docker support intentionally favors bind-mount root and environment guidance rather than editing Docker Desktop daemon storage automatically.
- SQLite uses the pure-Go `modernc.org/sqlite` driver, which supports CGO-free release builds.
- `git push-all` is intentionally omitted for alpha safety.

## Newly Added Beyond Original Plan

- Generated Mac mini host profile on the external drive: `/Volumes/ExternalSSD/.drive-agent/config/host-profiles/mac-mini.json`.
- Sanitized portable macOS profile template in `profiles/templates/macos-portable-dev.json`.
- External cache planning for npm, pnpm, bun, and Homebrew as profile-driven dry-run output.
- External Docker/container bind-mount root planning through `DRIVE_AGENT_CONTAINER_DATA` and `DRIVE_AGENT_DOCKER_BUILD_CACHE`.
- Restic S3-compatible repository support and sensitive URL user-info redaction.
- Required backup safety excludes for `.env`, Terraform state, `.terraform`, temp folders, and macOS metadata.
- Per-project backup excludes stored in `.drive-project.toml` and scoped into the generated Restic exclude file.
- Fake-drive smoke testing with APFS disk image guidance.

## Safety Review Against Plan

| Safety Rule | Current Coverage | Evidence | Gap / Notes | Priority |
|---|---|---|---|---|
| Refuse dangerous drive roots | Implemented | `internal/utils/safety.go`, `install.sh`, `utils` tests | Shell installer has separate path logic from Go safety helpers | Medium |
| Prefer `/Volumes` on macOS | Implemented | `init --allow-non-volume-path`, `install.sh` checks | Test escape hatch exists for fake drives | Low |
| No destructive init | Implemented | `init` creates/repairs layout without formatting | No formatter/erase code exists | Low |
| Cleanup defaults dry-run | Implemented | `cleanup` help and implementation | Add command-level cleanup tests | Medium |
| Cleanup symlink/boundary safety | Partially implemented | `utils` symlink/path tests, cleanup code | Need more cleanup-specific apply tests | High |
| Same-drive backup repo rejection | Implemented | `internal/backup/safety_test.go` | `--allow-same-drive-repo` is explicit | Low |
| Restore target safety | Implemented | `internal/backup/safety_test.go` | Automated restore test command missing | High |
| Shell config backups and idempotent blocks | Implemented | `internal/shell/detect_test.go`, `install.sh` | Fish shell support pending | Medium |
| No plaintext backup secrets | Implemented | password source detection, docs, redaction tests | Secret-manager integrations missing | Medium |
| No silent sudo | Partially implemented | Profile safety fields, package confirmation flow | Package providers do not centrally enforce sudo-free behavior beyond configured commands and confirmation | High |
| No silent `curl | bash` | Partially implemented | Homebrew package entry requires explicit approval and profile disallows curl-pipe-shell by default | No generic command policy engine | High |
| No package installs without confirmation | Implemented | `host setup`, `host packages install --dry-run/--yes` | Needs more tests with fake runners | Medium |
| Self-update checksum verification | Implemented | `internal/commands/self/self_test.go` | Release signing missing | Medium |
| Rollback backup behavior | Implemented | `install.sh`, `self rollback`, tests for backup listing | No migration rollback handling yet | Medium |

## CLI Command Coverage

Actual top-level commands from help:

- `version`
- `init`
- `status`
- `doctor`
- `org`
- `project`
- `host`
- `git`
- `cleanup`
- `backup`
- `self`

Implemented nested commands audited:

- `org add`, `org list`
- `project add`, `project list`, `project path`, `project open`, `project reindex`
- `git status-all`, `git fetch-all`, `git pull-all`
- `cleanup scan`, `cleanup dry-run`, `cleanup apply`, plus root `cleanup --dry-run` and `cleanup --apply`
- `host setup`, `host doctor`, `host packages list`, `host packages install`
- `backup init`, `backup status`, `backup run`, `backup snapshots`, `backup check`, `backup restore`, `backup excludes list/add/remove`, `backup doctor`
- `self version`, `self update`, `self rollback`

Planned commands missing or renamed:

- Planned `package` command is implemented as `host packages`.
- Planned `backup exclude` command is implemented as plural `backup excludes`.
- Missing host commands: `host export`, `host list`, `host current`, `host profile generate`.
- Missing org commands: `rename`, `archive`, `restore`, `delete`.
- Missing project commands: `archive`, `restore`, `move`, `stale`, template commands.
- Missing Git commands: `push-all`, `branch-all`, `remotes`, `dirty`, `unpushed`.
- Missing cleanup/scanner commands: large files, duplicate files, dependency caches, include/older-than/all filters.
- Missing future groups: workspace, service, ports, env, dashboard, audit/scan.

Help text review:

- `host setup --help` includes `--profile`, `--file`, `--cache-mode`, `--docker-mode`, and `--dry-run`.
- `init --help` includes `--non-interactive`.
- Backup help explains Restic as first provider, no plaintext backup passwords, dry-run behavior, per-project excludes, and restore-test recommendations.
- A small dry-run wording mismatch was fixed during this audit: dry-run host setup now ends with `Host setup plan complete` instead of `Host setup complete`.

## Architecture Coverage

External installed layout exists for:

- `.drive-agent/bin`
- `.drive-agent/config`
- `.drive-agent/config/host-profiles`
- `.drive-agent/db`
- `.drive-agent/logs`
- `.drive-agent/logs/backup`
- `.drive-agent/logs/cleanup`
- `.drive-agent/logs/git`
- `.drive-agent/logs/host-setup`
- `.drive-agent/state`
- `.drive-agent/state/backup`
- `.drive-agent/state/hosts`
- `.drive-agent/backups`
- `.drive-agent/locks`
- `.drive-agent/templates`
- `.drive-agent/catalog`
- `.drive-agent/releases`
- `.drive-agent/VERSION`
- `.drive-agent/install.json`
- `.drive-agent/DRIVE_AGENT_ROOT`

Planned architecture folders not currently present or not populated:

- `.drive-agent/lib`
- `.drive-agent/scripts`
- `.drive-agent/templates/projects`
- `.drive-agent/templates/host-profiles`
- `.drive-agent/templates/shell`
- split `categories.catalog.json`
- split `package-managers.catalog.json`
- `backup.toml`, `cleanup-rules.toml`, and `defaults.toml`

Code structure coverage:

- Present: `internal/cli`, `internal/commands`, `internal/config`, `internal/db`, `internal/filesystem`, `internal/packages`, `internal/shell`, `internal/ui`, `internal/utils`, `internal/backup`.
- The implementation is flatter than the plan in a few places, but the current package boundaries are understandable and match the alpha scope.

## Database and Manifest Audit

External SQLite schema includes all planned core tables:

- `drive`
- `hosts`
- `organizations`
- `projects`
- `project_tags`
- `settings`
- `package_install_records`
- `command_runs`
- `schema_version`

Project manifest support:

- `id`: supported
- `name`: supported
- `slug`: supported
- `org`: supported
- `project type`: supported as `type`
- `package manager`: supported
- `tags`: supported
- `Git remote`: supported
- `backup excludes`: supported under `[backup] excludes`
- `future metadata`: not generalized yet

The database can be partially rebuilt from manifests through `project reindex`. It can discover project manifests and add missing rows, but it does not yet repair/prune stale DB rows or rebuild every future metadata field.

## Host Setup Coverage

| Feature | Status | Notes |
|---|---|---|
| Interactive setup | Partially implemented | Simple prompts exist; no rich checkbox TUI |
| Non-interactive setup | Partially implemented | `--yes`, `--profile`, `--file`, and `--dry-run`; no broad `--non-interactive` flag for host setup |
| `--profile` | Implemented | Finds bundled and drive-local profiles |
| `--file` | Implemented | Loads explicit JSON file |
| `--dry-run` | Implemented | Shows package/cache/Docker/shell plan without mutating host config |
| Package manager installation | Partially implemented | Package installs are supported; package-manager bootstrap is limited and approval-driven |
| Package categories | Implemented | Catalog categories and profile categories work |
| Package catalog | Implemented | 132 packages |
| Host profiles | Implemented | Built-ins plus drive-local `mac-mini` |
| Generated profile support | Implemented | `mac-mini` profile validated from external drive |
| Homebrew/cask/npm/pnpm/bun/pipx/uv/cargo/go providers | Implemented | Provider registry has install planning/execution for these managers |
| Cache modes | Implemented | `prompt`, `host-local`, `external-drive`, `disabled` |
| Docker bind-mount/storage options | Implemented differently | Bind-mount/env plan exists; daemon relocation is not automated |
| Shell configuration | Implemented | Idempotent block with path quoting and backups |
| Host state tracking | Partially implemented | Host DB upsert and JSON state exist; no package/action history UI |
| Sudo/curl/shell edit safety | Partially implemented | Prompts and profile safety fields exist; command policy needs more centralized enforcement |

Host setup dry-run result:

- Found `/Volumes/ExternalSSD/.drive-agent/config/host-profiles/mac-mini.json`.
- Printed detected host, installed tools, package manager plan, package plan, cache plan, Docker/container storage plan, and shell block.
- Showed current and planned npm/pnpm cache/store paths.
- Quoted external drive paths safely.
- Ended in dry-run mode without mutating host config.

## Package Catalog/Profile Audit

Catalog summary:

- 132 package entries.
- Categories: `ai-dev`, `api-testing`, `backup`, `cloud`, `compilers`, `containers`, `core`, `databases`, `documentation`, `editors`, `go-rust`, `javascript`, `mobile`, `package-managers`, `php`, `productivity`, `python`, `security`, `shell`.

Built-in profiles:

- `minimal`: 11 packages; core and shell.
- `developer`: 27 packages; general dev with JavaScript, Python, containers, databases.
- `ai-developer`: 28 packages; AI coding tools plus dev baseline.
- `full-stack-saas`: 40 packages; cloud, security, infra, API testing, database, AI.
- `mobile`: 22 packages; mobile and Expo/React Native tooling.
- Template `macos-portable-dev`: 20 packages; sanitized portable example.

Gaps:

- No dedicated `categories.catalog.json` or `package-managers.catalog.json`.
- No catalog command to validate or explain normalization choices from a live host audit.
- Package IDs are generally stable, but some categories differ from the plan names, such as `go-rust` and `api-testing`.

## Backup Coverage Audit

Implemented:

- Backup provider abstraction.
- Restic provider.
- Backup config file at `.drive-agent/config/backup.json`.
- Backup state file at `.drive-agent/state/backup.json`.
- Backup logs under `.drive-agent/logs/backup`.
- Global excludes and generated Restic exclude file.
- Per-project excludes in `.drive-project.toml`.
- S3-compatible Restic repository support.
- Password source detection through environment or password file without storing plaintext passwords.
- Dry-run backup.
- Snapshots, check, restore, doctor.
- Same-drive repo rejection by default.
- Project-level wildcard excludes.
- Docs and tests for core backup safety.

Verified backup dry-run hygiene:

- `backup status` did not print credentials.
- `backup run --dry-run --tag plan-audit` did not create a snapshot.
- Generated exclude file includes required sensitive/default excludes from the recent backup safety work.
- Per-project excludes are scoped into the generated exclude file.

Planned but not implemented:

- Kopia provider.
- rclone provider.
- Time Machine integration.
- Scheduled backups.
- Retention/prune policies.
- macOS Keychain, 1Password, Doppler, or similar password integrations.
- Automated restore-test command and reporting.

## Self-Update/Release Coverage Audit

Implemented:

- `install.sh` installs/updates the external-drive binary and writes `VERSION` plus `install.json`.
- Existing installed binary is backed up before replacement.
- GoReleaser builds macOS/Linux/Windows artifacts and checksums.
- GitHub Actions release workflow runs GoReleaser on version tags.
- `self version`, `self update`, and `self rollback` exist.
- Update flow verifies checksums.
- Archive extraction is implemented in Go for zip and tar.gz.
- Unsupported OS/arch behavior is handled through asset selection.

Gaps:

- No release signing verification.
- No explicit release channel model beyond latest/specific version behavior.
- No post-update migration framework.
- No rollback of schema migrations.

## Docs Coverage

Reviewed docs:

- `README.md`
- `docs/architecture.md`
- `docs/backup.md`
- `docs/backup-restic.md`
- `docs/backup-restore.md`
- `docs/commands.md`
- `docs/host-setup.md`
- `docs/install.md`
- `docs/mvp-summary.md`
- `docs/package-catalog.md`
- `docs/release-process.md`
- `docs/release.md`
- `docs/safety.md`
- `docs/self-update.md`
- `docs/todos.md`

Current docs cover the implemented alpha feature set. This report fills the main docs gap: a plan-vs-implementation coverage matrix. Existing docs appropriately warn that the project is alpha and do not claim perfect environment reproducibility.

Docs gaps remaining:

- Add a public roadmap derived from this audit.
- Add command examples for every future-deferred feature once implemented.
- Add explicit docs for package-provider stubs and what is macOS-first today.
- Add a restore-test runbook with expected output once restore-test automation exists.

## Test Coverage

Current test areas from `go test ./... -list .`:

- Backup config, defaults, excludes, project-scoped excludes, password source detection, sensitive redaction, backup repository safety, restore target safety.
- Restic command construction, dry-run runner behavior, missing Restic behavior, snapshots parsing, log redaction.
- CLI version command.
- Host profile parsing, generated Mac mini profile validity, cache mode parsing, external cache plan, disabled cache plan, Docker bind-mount plan, dry-run non-mutation.
- Self-update asset names, checksum parsing, release selection, backup listing, archive extraction.
- Database schema, SQLite pragmas, org/project CRUD, host upsert, drive record, init directory creation.
- Filesystem root finding and size helpers.
- Package catalog loading, package source normalization, profile parsing.
- Shell block detection, idempotency, markers, and path quoting with spaces.
- General path safety, `/Volumes` checks, drive-boundary checks, symlink traversal, formatting, slug validation.

Important missing tests:

- Command-level tests for `org`, `project`, `git`, and `cleanup`.
- Cleanup apply tests with symlink and path traversal fixtures.
- Host package installation tests with fake command runners for every provider.
- Install script safety tests, especially canonical path edge cases.
- Profile generator tests after a generator exists.
- Scheduled backup/retention/prune tests after those features exist.
- Restore-test automation tests after the command exists.
- Release signing verification tests after signing is added.

## Recommended Next Work

Top 10 recommended next tasks:

1. Add command-level tests for cleanup apply, including symlink, traversal, and out-of-drive fixtures.
2. Add fake-runner tests for host package installs across Homebrew, cask, npm, pnpm, bun, pipx, uv, cargo, and `go install`.
3. Implement `drive-agent host profile generate --name <name> --dry-run` to formalize the manual Mac audit workflow.
4. Add `project reindex --repair` to prune stale DB rows after confirmation.
5. Add org/project archive and restore commands.
6. Add backup retention/prune policy support for Restic.
7. Add an automated `backup restore-test` workflow that restores to a separate safe target and records results.
8. Centralize installer path safety by sharing or invoking the Go safety logic from `install.sh`.
9. Add release signing verification before `self update` applies a downloaded binary.
10. Decide whether to keep `host packages` as the package command namespace or add the planned top-level `package` alias/group.

## Validation

Validation after this audit passed:

- `go mod tidy`
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/drive-agent`
- `bash tests/smoke_test.sh`

Additional spot check:

- `go build -o ./drive-agent ./cmd/drive-agent`
- `./drive-agent host setup --path /Volumes/ExternalSSD --profile mac-mini --dry-run`

The local dry-run found the drive-local profile and ended with `Host setup plan complete` plus `(dry-run - no changes were made)`.

## Release Recommendation

A new alpha release is recommended after review and commit because the current implementation adds meaningful user-facing features: Restic backup provider, per-project backup excludes, portable host profiles, cache/Docker setup planning, and improved safety/docs. Local tags already include `v0.1.0-alpha.5`, so the next alpha tag should be `v0.1.0-alpha.6` unless a remote tag check shows that tag already exists.
