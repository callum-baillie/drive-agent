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
| core | Essential CLI tools | ~10 |
| backup | Backup tools | ~1 |
| shell | Search, navigation, productivity | ~10 |
| javascript | Node.js, package managers, tools | ~10 |
| python | Python runtime and tools | ~6 |
| php | PHP runtime and tools | ~2 |
| go-rust | Go and Rust toolchains | ~3 |
| compilers | Build tools and compilers | ~2 |
| containers | Docker, Kubernetes, etc. | ~7 |
| databases | Database servers and clients | ~8 |
| cloud | Cloud provider CLIs | ~9 |
| ai-dev | AI developer tools | ~14 |
| editors | Code editors and IDEs | ~7 |
| mobile | Mobile development tools | ~6 |
| api-testing | API clients and testing | ~5 |
| security | Security scanning tools and security apps | ~7 |
| documentation | Writing and documentation | ~2 |
| productivity | Apps, browsers, terminals, and media tools | ~20 |

## Source Normalization

Host profiles should describe the desired install source, not just where a tool happened to come from on one Mac.

- Prefer Homebrew formulae for stable developer CLIs such as `gh`, `restic`, `terraform`, `trivy`, `checkov`, `stripe-cli`, and database clients.
- Prefer Homebrew casks for GUI apps such as VS Code, ChatGPT, Postman, OrbStack, Docker Desktop, browsers, terminals, and productivity apps.
- Keep npm/pnpm/bun for JavaScript-specific global CLIs where that is the normal source.
- Keep pipx/uv for isolated Python tools when Homebrew is not the better source.
- Keep cargo and `go install` for language-specific binaries such as Rust or Go tools.
- Avoid listing the same tool in multiple managers unless there is a clear reason.

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
