# Drive Agent

**Portable development drive manager** — a CLI tool that lives on an external drive and helps configure, organize, maintain, and back up development work across multiple host machines.

> **Status:** MVP / Alpha
> **Safety Warning:** This project is under active development. While it enforces strict path safety, it is not yet production-stable. Use at your own risk.

## Core Features
- 🚀 **Portable Dev Environment:** Move your code between machines instantly.
- 📦 **Host Setup:** Automatically install tools (Homebrew, npm, Node, Docker) on a new host based on profiles.
- 🧹 **Idempotent Cleanup:** Safely reclaim space by scanning for `node_modules`, `dist`, `.next`, etc.
- 🔄 **Self-Update:** Securely update the agent in-place via GitHub releases.
- 🛡️ **Safety-First:** Strict validation prevents destructive operations outside the drive root.

## Quick Start (Real External Drive)

```bash
# 1. Initialize your external drive
./drive-agent init --path /Volumes/DevDrive

# 2. Install Drive Agent to the drive
./install.sh --drive /Volumes/DevDrive

# 3. Open a new terminal or source your shell config, then add an organization
drive-agent org add personal

# 4. Add a project
drive-agent project add --org personal --name my-app --type nextjs

# 5. Check everything
drive-agent status
drive-agent doctor
```

*(Placeholder: A single-line curl script for GitHub release installs will be provided after the first public release).*

## Updates and Rollback

Drive Agent can safely update itself in-place via GitHub releases:
```bash
# Fetch the latest release and update securely
drive-agent self update

# Check available backups and revert if needed
drive-agent self rollback --list
drive-agent self rollback
```

## First Safe Test Run (Fake Drive Testing)

If you want to test Drive Agent safely on your local macOS machine without using a physical external drive, you can use an APFS disk image to simulate a real drive mounted at `/Volumes`:

```bash
# 1. Create and mount a 2GB APFS disk image
hdiutil create -size 2g -fs APFS -volname DriveAgentTest /tmp/DriveAgentTest.dmg
hdiutil attach /tmp/DriveAgentTest.dmg

# 2. Build from source
go build -o drive-agent ./cmd/drive-agent

# 3. Initialize the drive
./drive-agent init --path /Volumes/DriveAgentTest --name DriveAgentTest

# 4. Install the binary to the drive
ALLOW_TEST_DRIVE=1 ./install.sh --drive /Volumes/DriveAgentTest --binary ./drive-agent --skip-shell --yes

# 5. Verify the installation
/Volumes/DriveAgentTest/.drive-agent/bin/drive-agent self version
/Volumes/DriveAgentTest/.drive-agent/bin/drive-agent doctor --path /Volumes/DriveAgentTest

# 6. Cleanup
hdiutil detach /Volumes/DriveAgentTest
rm /tmp/DriveAgentTest.dmg
```

## Known Limitations

- **Package managers**: Only Homebrew, Homebrew Cask, and npm global are fully implemented. Other managers are scaffolded as stub providers.
- **Backup**: Commands exist but print guided setup instructions rather than executing backups automatically.
- **Self-update**: Requires the `callumbaillie/drive-agent` repository to have a tagged GitHub Release with GoReleaser artifacts.
- **Interactive host setup**: Uses simple terminal prompts, not a rich TUI selector with checkboxes.
- **git push-all**: Intentionally not implemented (safety requirement).
- **Disk space reporting**: `status` doesn't show free/used disk space natively.
- **Shell config**: `install.sh` handles bash/zsh; fish support is pending.
- **Cross-platform**: macOS-first. Windows/Linux paths and managers are scaffolded but untested.

## Documentation

- [Installation Guide](docs/install.md)
- [Self Update & Rollback](docs/self-update.md)
- [Fake Drive Testing](docs/fake-drive-testing.md)
- [Release Process](docs/release-process.md)
- [Safety Model](docs/safety.md)

## Philosophy

- **The drive is self-describing.** A new host can inspect the drive and understand its structure.
- **The host stays normal.** Apps and tools install on the host, not the drive.
- **Projects are just folders.** Git repos work without drive-agent. The database is an index, not the source of truth.
- **Safety first.** No destructive operation runs silently. Cleanup defaults to dry-run.

## Commands

### Core
| Command | Description |
|---------|-------------|
| `drive-agent --version` | Show version |
| `drive-agent init` | Initialize a drive (non-destructive) |
| `drive-agent status` | Show drive status summary |
| `drive-agent doctor` | Run health checks |

### Organizations
| Command | Description |
|---------|-------------|
| `drive-agent org add <name>` | Add an organization |
| `drive-agent org list` | List organizations |

### Projects
| Command | Description |
|---------|-------------|
| `drive-agent project add` | Add a project (interactive or flags) |
| `drive-agent project list` | List projects |
| `drive-agent project path <org/project>` | Print project path |
| `drive-agent project open <org/project>` | Open in editor |
| `drive-agent project reindex` | Rebuild database from manifests |

### Host Setup
| Command | Description |
|---------|-------------|
| `drive-agent host setup` | Interactive host setup |
| `drive-agent host setup --profile developer` | Setup from profile |
| `drive-agent host doctor` | Check host tools |
| `drive-agent host packages list` | List available packages |
| `drive-agent host packages install <pkg...>` | Install packages |

### Git Utilities
| Command | Description |
|---------|-------------|
| `drive-agent git status-all` | Git status across all projects |
| `drive-agent git fetch-all` | Fetch all projects |
| `drive-agent git pull-all` | Pull all clean projects |

### Cleanup
| Command | Description |
|---------|-------------|
| `drive-agent cleanup scan` | Scan for removable build artifacts |
| `drive-agent cleanup dry-run` | Show cleanup plan |
| `drive-agent cleanup apply` | Delete targets (with confirmation) |

### Backup (Stubs)
| Command | Description |
|---------|-------------|
| `drive-agent backup status` | Show backup tool availability |
| `drive-agent backup init` | Initialize backup (planned) |
| `drive-agent backup run` | Run backup (planned) |
| `drive-agent backup check` | Verify backup (planned) |

### Self-Management
| Command | Description |
|---------|-------------|
| `drive-agent self version` | Show version |
| `drive-agent self update` | Update binary |
| `drive-agent self rollback` | Rollback to backup |

## Drive Layout

```
/Volumes/YourDrive
├── .drive-agent/          # Agent metadata, database, config
│   ├── bin/               # Agent binary
│   ├── config/            # drive.toml and settings
│   ├── db/                # SQLite database
│   ├── logs/              # Operation logs
│   ├── state/hosts/       # Per-host state files
│   ├── catalog/           # Package catalog
│   ├── DRIVE_AGENT_ROOT   # Marker file
│   └── VERSION            # Current version
├── Orgs/                  # Organizations and projects
├── DevData/               # Local service data
├── Caches/                # Package caches
├── BuildArtifacts/        # Build outputs
├── Tooling/               # Scripts and templates
├── Downloads/             # SDKs and installers
├── Inbox/                 # Unsorted files
├── Scratch/               # Temporary work
└── Trash/                 # Soft-delete area
```

## Profiles

Pre-built setup profiles in `profiles/`:
- **minimal.json** — Essential CLI tools only
- **developer.json** — General dev with Node, Python, Docker
- **ai-developer.json** — AI coding tools (Codex, Claude, Cursor, etc.)
- **full-stack-saas.json** — Cloud, security, infrastructure
- **mobile.json** — Xcode, Android Studio, React Native

## Building

```bash
# Requires Go 1.21+ and CGO
CGO_ENABLED=1 go build -o drive-agent ./cmd/drive-agent

# Run tests
go test ./...

# Run vet
go vet ./...

# Run smoke test
bash tests/smoke_test.sh
```

## Safety

See [docs/safety.md](docs/safety.md) for the full safety model.

**Key guarantees:**
- `init` never erases or formats the drive
- `init` refuses dangerous paths (/, /Users, $HOME, system dirs)
- Cleanup defaults to dry-run and never deletes outside the drive root
- Cleanup never follows symlinks
- Git push-all is not implemented (requires explicit confirmation in future)
- Host setup never installs without consent
- No silent `sudo` or `curl | bash`
- Shell config edits create timestamped backups with marked blocks

## License

MIT
