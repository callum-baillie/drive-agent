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

For less-interactive setup, pass explicit cache and Docker choices with `--yes`:

```bash
drive-agent host setup \
  --path /Volumes/ExternalSSD \
  --profile mac-mini \
  --yes \
  --cache-mode external-drive \
  --docker-mode bind-mounts
```

`--yes` also has short form `-y`. It accepts normal package installs and selected cache/storage changes when the mode is explicit. It does not bypass safety checks and does not install packages marked `requiresExplicitApproval`.

To include explicit-approval packages such as `playwright-cli`, opt in separately:

```bash
drive-agent host setup \
  --path /Volumes/ExternalSSD \
  --profile mac-mini \
  --yes \
  --include-explicit \
  --cache-mode external-drive \
  --docker-mode bind-mounts
```

Use `--force` only when you intentionally want Drive Agent to attempt an install even though a catalog check says the package or app is already present. For Homebrew casks, manually installed app bundles may still require manual cleanup or review before Homebrew can adopt or reinstall the cask.

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

These exports are written to a separate idempotent storage block so they can be
added or updated even when the main Drive Agent PATH/alias block already exists:

```bash
# >>> drive-agent storage >>>
export HOMEBREW_CACHE='/Volumes/ExternalSSD/Caches/homebrew'
export BUN_INSTALL_CACHE_DIR='/Volumes/ExternalSSD/Caches/bun'
# <<< drive-agent storage <<<
```

Homebrew itself should stay installed on the host. Only the Homebrew cache is portable.

## Docker And Container Storage

Profile Docker mode can be:

- `prompt`: ask during interactive setup.
- `default`: keep Docker's default storage.
- `bind-mounts`: create external project/container roots and export environment variables.
- `daemon-guidance`: documentation-only guidance for daemon relocation. `daemon` and `daemon-data-root` are accepted aliases.

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

When `--docker-mode bind-mounts` is applied, these exports are written into the
same `drive-agent storage` shell block as the cache exports. Dry-run prints the
planned block without editing shell files.

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

Host setup uses each catalog entry's `check.command` and `check.appBundles` before planning an install. This lets Drive Agent skip apps that were installed manually or by another source. For example, `vscode` checks both `code --version` and `/Applications/Visual Studio Code.app`, so a manually installed VS Code app is treated as already installed instead of triggering `brew install --cask visual-studio-code`.

If a dry-run shows an unexpected manager, fix the catalog entry before running
real setup. For example, Turborepo is intentionally mapped to npm/pnpm as the
global package `turbo`, not a Homebrew formula.

## Recommended Node/React/Next.js And Coding-Agent Tools

Use host profiles for small tools that are useful across many repositories: `ripgrep`, `fd`, `ast-grep`, `git-delta`, `lazygit`, `biome`, `knip`, `depcheck`, `svgo`, and `imagemagick`.

Keep project frameworks and runtime dependencies in each repository's `package.json`. `next`, `react`, `react-dom`, `tailwindcss`, `vite`, `vitest`, `jest`, `eslint-config-next`, `@playwright/test`, and `sharp` should normally be project dependencies so versions stay tied to the app that uses them.

`playwright-cli` is optional host tooling only. Drive Agent marks it as explicit-approval because real Playwright test suites should manage `@playwright/test` and browser downloads per project.

## Safety

- Dry-run first for profile setup.
- Package install plans show exact commands.
- Shell config edits use a marked block and backup.
- Packages with `requiresExplicitApproval` are not auto-installed.
- Cache and Docker storage changes are explicit; disabled/no-change mode does not delete or rewrite existing config.
- No silent `sudo` or `curl | bash`.
