# Host Setup

## Overview

Host setup configures the current machine to work with the development drive. The agent lives on the drive; packages install on the host.

## Interactive Setup

```bash
drive-agent host setup
```

This detects your OS, architecture, shell, and installed tools, then guides you through:
1. Shell configuration (PATH, aliases)
2. Package manager detection
3. Tool installation (future: full interactive selector)

## Profile-Based Setup

```bash
drive-agent host setup --profile minimal
drive-agent host setup --profile developer
drive-agent host setup --profile ai-developer
drive-agent host setup --profile full-stack-saas
drive-agent host setup --profile mobile
```

Profiles are JSON files in `profiles/` that define:
- Package managers to use
- Categories to install
- Specific packages to include/exclude
- Shell configuration preferences
- Safety preferences

## Package Installation

```bash
# Install specific packages
drive-agent host packages install git gh jq pnpm cursor --yes

# Install by category
drive-agent host packages install --category core,shell,javascript --dry-run

# List available packages
drive-agent host packages list
drive-agent host packages list --category ai-dev
```

## Safety

- Never installs without showing what will happen
- `--dry-run` shows exact commands
- Shell config creates backups before editing
- Uses marked blocks for easy removal
- Packages with `requiresExplicitApproval` are never auto-installed
- No silent `sudo` or `curl | bash`

## Supported Package Managers

### Fully Implemented (MVP)
- **Homebrew** (formulae)
- **Homebrew Cask** (GUI apps)
- **npm** (global installs)

### Scaffolded (Future)
- winget, chocolatey, scoop (Windows)
- apt, dnf, pacman, zypper, apk (Linux)
- pnpm, bun, uv, pipx, cargo, go install, mise, asdf, nix
