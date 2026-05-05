# Host Setup

## Overview

Host setup configures the current machine to work with a Drive Agent external drive. The agent and profiles live on the drive; host tools and GUI apps still install on each Mac.

Host setup is designed for broad reproducibility, not perfect cloning. It does not copy secrets, logins, license state, app settings, SSH keys, browser profiles, private keychains, certificates, or credentials.

## Interactive Setup

```bash
drive-agent host setup --path /Volumes/ExternalSSD
```

This detects OS, architecture, shell, and installed tools, then guides you through shell configuration and any profile-selected package/cache/storage plans.

## Profile-Based Setup

Bundled profiles and drive-local profiles use the same schema:

```bash
drive-agent host setup --path /Volumes/ExternalSSD --profile developer --dry-run
drive-agent host setup --path /Volumes/ExternalSSD --profile mac-mini --dry-run
drive-agent host setup --path /Volumes/ExternalSSD --profile mac-mini
```

Drive-local profiles live here:

```text
/Volumes/ExternalSSD/.drive-agent/config/host-profiles/
```

The repo includes sanitized examples under `profiles/templates/`. Do not commit exact host-specific profiles unless they have been reviewed and scrubbed.

You can also pass a file directly:

```bash
drive-agent host setup \
  --path /Volumes/ExternalSSD \
  --file /Volumes/ExternalSSD/.drive-agent/config/host-profiles/mac-mini.json \
  --dry-run
```

Dry-run output shows package managers, packages/apps to install, cache choices, Docker/container storage choices, shell changes, and commands that would run.

## Generating A Host Profile

To turn an existing Mac into a reusable profile:

1. Audit `/Applications`, `~/Applications`, Homebrew formulae/casks, global npm/pnpm/bun packages, pipx/uv tools, cargo installs, Go-installed binaries, and Mac App Store apps if `mas` is present.
2. Normalize the result instead of mirroring messy state. Prefer Homebrew formulae for stable CLIs, Homebrew casks for GUI apps, npm/pnpm/bun for JavaScript-specific global CLIs, pipx/uv for Python tools, cargo for Rust binaries, and `go install` for Go binaries.
3. Write the profile to `.drive-agent/config/host-profiles/<name>.json`.
4. Run `host setup --profile <name> --dry-run` before applying it on any Mac.

## Cache Options

Profile cache mode can be:

- `prompt`: ask during interactive setup.
- `host-local`: keep each host's existing package-manager cache behavior.
- `external-drive`: plan external cache paths on the drive.
- `disabled`: do not configure caches and do not delete existing caches.

External-drive defaults:

```text
/Volumes/ExternalSSD/Caches/npm
/Volumes/ExternalSSD/Caches/pnpm
/Volumes/ExternalSSD/Caches/bun
/Volumes/ExternalSSD/Caches/homebrew
```

Drive Agent plans:

```bash
npm config set cache /Volumes/ExternalSSD/Caches/npm
pnpm config set store-dir /Volumes/ExternalSSD/Caches/pnpm
```

For Bun and Homebrew, Drive Agent exposes shell exports:

```bash
export BUN_INSTALL_CACHE_DIR=/Volumes/ExternalSSD/Caches/bun
export HOMEBREW_CACHE=/Volumes/ExternalSSD/Caches/homebrew
```

Homebrew itself should stay installed on the host. Only the Homebrew cache is portable.

## Docker And Container Storage

Profile Docker mode can be:

- `prompt`: ask during interactive setup.
- `default`: keep Docker's default storage.
- `bind-mounts`: create external project/container roots and export environment variables.
- `daemon`: documentation-only guidance for daemon relocation.

The preferred default for portability is bind mounts:

```text
/Volumes/ExternalSSD/DevData/containers
/Volumes/ExternalSSD/DevData/docker-build-cache
```

Shell exports:

```bash
export DRIVE_AGENT_CONTAINER_DATA=/Volumes/ExternalSSD/DevData/containers
export DRIVE_AGENT_DOCKER_BUILD_CACHE=/Volumes/ExternalSSD/DevData/docker-build-cache
```

Relocating Docker Desktop's whole disk image or daemon `data-root` is not the default. Docker Desktop on macOS and OrbStack do not behave exactly like a Linux Docker daemon, and UI-managed storage settings are usually safer than editing `~/.docker/daemon.json` directly. If daemon editing is ever implemented, it must back up existing JSON, merge carefully, require explicit confirmation, and avoid automatic restarts.

## Moving Between Two Macs

On each Mac:

```bash
/Volumes/ExternalSSD/.drive-agent/bin/drive-agent host setup \
  --path /Volumes/ExternalSSD \
  --profile mac-mini \
  --dry-run
```

Review the plan, then run without `--dry-run` when ready. Each Mac keeps its own secrets, logins, app settings, and service state, while the external drive keeps projects, Drive Agent config, host profiles, optional caches, and optional bind-mount data roots.

## Package Installation

```bash
drive-agent host packages install git gh jq pnpm cursor --yes
drive-agent host packages install --category core,shell,javascript --dry-run
drive-agent host packages list
drive-agent host packages list --category ai-dev
```

## Safety

- Dry-run first for profile setup.
- Package install plans show exact commands.
- Shell config edits use a marked block and backup.
- Packages with `requiresExplicitApproval` are not auto-installed.
- Cache and Docker storage changes are explicit; disabled/no-change mode does not delete or rewrite existing config.
- No silent `sudo` or `curl | bash`.
