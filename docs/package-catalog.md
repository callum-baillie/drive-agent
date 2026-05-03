# Package Catalog

## Overview

The package catalog (`catalog/packages.catalog.json`) maps friendly package IDs to platform-specific installation methods. It supports multiple package managers per package and tracks availability per platform.

## Structure

Each package entry has:

```json
{
  "id": "git",
  "name": "Git",
  "category": "core",
  "description": "Version control CLI",
  "kind": "cli",
  "default": true,
  "installPreference": ["homebrew", "apt", "winget"],
  "install": {
    "homebrew": { "type": "formula", "name": "git" },
    "apt": { "id": "git" },
    "winget": { "id": "Git.Git" }
  },
  "check": { "command": "git --version" }
}
```

## Categories

| Category | Description | Count |
|----------|-------------|-------|
| package-managers | Package managers themselves | 1 |
| core | Essential CLI tools | ~8 |
| shell | Search, navigation, productivity | ~10 |
| javascript | Node.js, package managers, tools | ~8 |
| python | Python runtime and tools | ~5 |
| php | PHP runtime and tools | ~2 |
| go-rust | Go and Rust toolchains | ~2 |
| compilers | Build tools and compilers | ~2 |
| containers | Docker, Kubernetes, etc. | ~7 |
| databases | Database servers and clients | ~5 |
| cloud | Cloud provider CLIs | ~6 |
| ai-dev | AI developer tools | ~10 |
| editors | Code editors and IDEs | ~3 |
| mobile | Mobile development tools | ~6 |
| api-testing | API clients and testing | ~5 |
| security | Security scanning tools | ~4 |
| documentation | Writing and documentation | ~2 |
| productivity | Apps and media tools | ~8 |

## Package Kinds

- `cli` — Command-line tool
- `gui` — Desktop application
- `runtime` — Language runtime
- `service` — Background service

## Adding Packages

Edit `catalog/packages.catalog.json` and add an entry following the schema. Ensure:
1. `id` is unique and lowercase
2. `installPreference` lists managers in priority order
3. `install` has entries for each manager that supports this package
4. `check.command` verifies installation

## Safety

Packages with `requiresExplicitApproval: true` (like `claude-code`, `xcode`, `android-studio`) are never auto-installed even with `--yes`.
