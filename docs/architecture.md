# Architecture

## Overview

Drive Agent is a Go CLI application using the Cobra command framework. It manages an external development drive's structure, project registry, and host machine setup.

## Directory Structure

```
drive-agent/
├── cmd/drive-agent/main.go      # Entrypoint
├── internal/
│   ├── cli/                      # Root command and top-level commands
│   │   ├── root.go              # Command registration
│   │   └── status.go            # status + doctor commands
│   ├── commands/                 # Command implementations
│   │   ├── init/                # Drive initialization
│   │   ├── org/                 # Organization management
│   │   ├── project/             # Project management
│   │   ├── host/                # Host setup and packages
│   │   ├── git/                 # Git bulk operations
│   │   ├── cleanup/             # Build artifact cleanup
│   │   ├── backup/              # Backup command wiring
│   │   └── self/                # Self-update and rollback
│   ├── backup/                   # Backup provider/config/safety/state
│   │   └── restic/              # Restic CLI provider
│   ├── config/                   # Types, constants, layout definitions
│   ├── db/                       # SQLite database layer
│   ├── filesystem/               # Path resolution, directory ops
│   ├── packages/
│   │   ├── catalog/             # Package catalog parser
│   │   └── providers/           # Package manager abstractions
│   ├── shell/                    # OS/shell detection, command execution
│   ├── ui/                       # Terminal output, colors, prompts
│   └── utils/                    # Slugs, safety, IDs, formatting
├── catalog/                      # Package catalog JSON
├── profiles/                     # Host setup profiles
├── tests/                        # Smoke tests
└── docs/                         # Documentation
```

## Key Design Decisions

### SQLite as Index
The database is an index that can be rebuilt from `.drive-project.toml` manifests. This means projects remain usable without drive-agent and the database can be regenerated if lost.

### Provider Pattern for Package Managers
Package managers are abstracted behind a `Provider` interface. Each provider implements detection, availability checks, and install commands. New managers can be added by implementing the interface.

### Safety by Default
Destructive operations (cleanup, host installs) default to dry-run or require explicit confirmation. Path safety validation prevents operations outside the drive root.

### Drive Root Detection
`FindDriveRoot()` walks upward from the current directory or from the global `--path` override looking for `.drive-agent/DRIVE_AGENT_ROOT`. This allows commands to work from any subdirectory or from scripts that pass the drive root explicitly.

## Database Schema

Tables: `drive`, `hosts`, `organizations`, `projects`, `project_tags`, `settings`, `package_install_records`, `command_runs`, `schema_version`

Schema version is tracked for future migrations.

## Data Flow

```
User Command → Cobra CLI → Command Handler → DB/Filesystem/Shell → Output (UI)
```

Package installs:
```
Catalog → Provider Registry → Best Available Provider → Shell Exec → Log
```

Backups:
```
Backup command → Backup config/state → Provider abstraction → Restic CLI → Logs/state
```
