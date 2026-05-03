# Drive Agent — Portable Development Drive Manager Plan

## 1. Overview

`drive-agent` is a portable development-drive management system that lives on an external drive and helps configure, organize, maintain, and back up development work across multiple host machines.

The drive should contain the agent, project registry, configuration, package catalogs, setup profiles, logs, and drive-local metadata. The host machine should contain installed applications, package managers, runtimes, editors, Docker/OrbStack, shell configuration, and other normal host-level development tools.

The goal is to make the external drive a self-describing, portable development workspace while keeping every host computer clean, predictable, and usable without permanently depending on the external drive for core system functionality.

---

## 2. Core Principles

1. **The drive should be self-describing.**  
   A new host should be able to inspect the drive and understand its structure, projects, organizations, profiles, and setup history.

2. **The host should remain a normal host.**  
   Apps, package managers, Docker/OrbStack, system tools, editors, and runtimes install on the host, not on the external drive.

3. **The drive should remain usable without the agent.**  
   Projects are normal folders. Git repos are normal Git repos. The database improves the experience but should not be the only source of truth.

4. **SQLite is an index, not the source of truth.**  
   The agent should also write project manifests to disk so the registry can be rebuilt if the database is lost or corrupted.

5. **Safety first.**  
   No destructive operation should run silently. Init should never erase the drive. Cleanup should dry-run first. Git push-all should require confirmation.

6. **Interactive and automated flows should both be supported.**  
   Every major workflow should have an interactive CLI UI and non-interactive flags/profile support.

7. **Cross-platform by design, macOS first.**  
   Initial implementation can prioritize macOS, but the architecture should support Windows and Linux later.

---

## 3. Target Command

Primary command:

```bash
drive-agent
```

Recommended aliases after host setup:

```bash
alias da="drive-agent"
alias drive="drive-agent"
```

Example usage:

```bash
drive-agent init
drive-agent host setup
drive-agent org add roamar
drive-agent project add
drive-agent git status-all
drive-agent cleanup scan
drive-agent backup run
```

---

## 4. Recommended Drive Layout

Assume the external drive is mounted at:

```text
/Volumes/DevDrive
```

Recommended structure:

```text
/Volumes/DevDrive
├── .drive-agent
│   ├── bin
│   │   ├── drive-agent
│   │   └── drive-agent-core
│   ├── lib
│   ├── scripts
│   ├── templates
│   │   ├── projects
│   │   ├── host-profiles
│   │   └── shell
│   ├── catalog
│   │   ├── packages.catalog.json
│   │   ├── categories.catalog.json
│   │   └── package-managers.catalog.json
│   ├── config
│   │   ├── drive.toml
│   │   ├── cleanup-rules.toml
│   │   ├── backup.toml
│   │   └── defaults.toml
│   ├── db
│   │   └── drive-agent.sqlite
│   ├── logs
│   │   ├── host-setup
│   │   ├── cleanup
│   │   ├── backup
│   │   └── git
│   ├── state
│   │   └── hosts
│   ├── backups
│   ├── locks
│   ├── releases
│   ├── VERSION
│   └── DRIVE_AGENT_ROOT
│
├── Orgs
│   ├── personal
│   │   ├── projects
│   │   ├── archives
│   │   └── notes
│   ├── roamar
│   │   ├── projects
│   │   ├── archives
│   │   └── notes
│   └── jaspersclassroom
│       ├── projects
│       ├── archives
│       └── notes
│
├── DevData
│   ├── docker
│   ├── postgres
│   ├── redis
│   ├── minio
│   ├── mailpit
│   └── local-services
│
├── Caches
│   ├── pnpm
│   ├── npm
│   ├── yarn
│   ├── bun
│   ├── composer
│   ├── pip
│   ├── uv
│   ├── cargo
│   ├── turbo
│   └── next
│
├── BuildArtifacts
│   ├── expo
│   ├── eas
│   ├── android
│   ├── xcode
│   ├── next
│   └── releases
│
├── Tooling
│   ├── scripts
│   ├── templates
│   ├── snippets
│   └── local-bin
│
├── Downloads
│   ├── sdks
│   ├── installers
│   └── docs
│
├── Inbox
├── Scratch
└── Trash
```

---

## 5. Folder Responsibilities

### `.drive-agent`

The hidden control system for the drive.

Stores:

- agent executable/bootstrap script
- package catalog
- host profiles
- SQLite database
- logs
- lock files
- templates
- self-update state
- backup configuration
- host setup records

### `Orgs`

Stores organizations and their projects.

Example:

```text
/Volumes/DevDrive/Orgs/roamar/projects/user-web
/Volumes/DevDrive/Orgs/roamar/projects/mobile
/Volumes/DevDrive/Orgs/jaspersclassroom/projects/app
/Volumes/DevDrive/Orgs/personal/projects/live-chat-saas
```

### `DevData`

Stores local service state that does not belong inside an individual repository.

Examples:

- Docker/OrbStack bind mounts
- local Postgres data
- Redis data
- MinIO data
- Mailpit/maildev data
- local service volumes

### `Caches`

Stores optional package/build caches that can be redirected from the host.

Examples:

- pnpm store
- npm cache
- Bun cache
- Composer cache
- uv cache
- Cargo cache
- Turborepo cache

These should be optional because some users may prefer host-local caches.

### `BuildArtifacts`

Stores intentional generated outputs.

Examples:

- Expo/EAS builds
- Android artifacts
- Xcode archives
- Next.js exported builds
- release bundles

### `Tooling`

Stores reusable scripts, snippets, templates, and local helper binaries.

### `Inbox`

Temporary landing area for files not yet organized.

### `Scratch`

Temporary working area.

### `Trash`

Soft-delete area for agent-managed operations when possible.

---

## 6. First-Class Concepts

### Drive

Represents the external drive itself.

Example metadata:

```toml
drive_id = "devdrive-2026-05"
name = "DevDrive"
schema_version = 1
created_at = "2026-05-02T00:00:00Z"
default_org = "personal"
```

### Host

Represents a computer that has used this drive.

Example:

```json
{
    "hostId": "callums-mac-mini",
    "hostname": "Callums-Mac-mini",
    "os": "macos",
    "arch": "arm64",
    "shell": "zsh",
    "lastSeenAt": "2026-05-02T00:00:00Z",
    "setupCompleted": true
}
```

### Organization

Represents a project owner or grouping.

Examples:

```text
personal
roamar
jaspersclassroom
client-acme
```

### Project

Represents a Git repo or local folder managed by the agent.

Example metadata:

```toml
id = "proj_roamar_mobile"
name = "Mobile App"
slug = "mobile"
org = "roamar"
type = "expo"
package_manager = "pnpm"
tags = ["mobile", "expo", "react-native"]
git_remote = "git@github.com:roamar/mobile.git"
created_at = "2026-05-02T00:00:00Z"
```

### Package Catalog

A drive-stored catalog that maps friendly package IDs to platform/package-manager-specific install methods.

### Host Profile

A JSON file that defines a desired setup for a specific host or type of host.

---

## 7. Command Structure

Top-level command groups:

```bash
drive-agent init
drive-agent status
drive-agent doctor

drive-agent host
drive-agent org
drive-agent project
drive-agent git
drive-agent cleanup
drive-agent backup
drive-agent package
drive-agent service
drive-agent workspace
drive-agent scan
drive-agent self
```

---

## 8. Init Command

### Purpose

Initialize a drive non-destructively.

### Command

```bash
drive-agent init
drive-agent init --path /Volumes/DevDrive
drive-agent init --path /Volumes/DevDrive --name DevDrive
drive-agent init --path /Volumes/DevDrive --repair
drive-agent init --path /Volumes/DevDrive --non-interactive
```

### What It Creates

```text
.drive-agent/
Orgs/
DevData/
Caches/
BuildArtifacts/
Tooling/
Downloads/
Inbox/
Scratch/
Trash/
```

### Safety Rules

The init command must refuse to run if:

- target path is `/`
- target path is `/Users`
- target path is the current user's home directory
- target path is not inside `/Volumes` on macOS unless `--allow-non-volume-path` is passed
- `.drive-agent` already exists and neither `--repair` nor `--reinit` was passed
- target path contains existing files and the user does not confirm non-destructive setup

### Important Rule

`drive-agent init` must never erase, format, or repartition a drive.

---

## 9. Host Setup System

### Purpose

Configure the current host machine to work well with the drive.

The agent lives on the drive. Packages install on the host.

### Commands

```bash
drive-agent host setup
drive-agent host setup --profile developer
drive-agent host setup --profile ai-developer
drive-agent host setup --profile minimal
drive-agent host setup --file ./host-profile.json
drive-agent host setup --yes
drive-agent host setup --dry-run

drive-agent host doctor
drive-agent host export
drive-agent host list
drive-agent host current
```

---

## 10. Host Setup Responsibilities

Host setup should be able to:

1. Detect the current operating system.
2. Detect CPU architecture.
3. Detect shell.
4. Detect installed package managers.
5. Install missing package managers with consent.
6. Install selected host packages.
7. Configure shell PATH and aliases.
8. Configure optional package caches.
9. Configure optional development service locations.
10. Record host setup state on the drive.
11. Verify setup with health checks.

---

## 11. Interactive Host Setup UI

Interactive setup should guide the user through:

```text
Welcome to drive-agent host setup

Detected:
- OS: macOS
- Arch: arm64
- Shell: zsh
- Drive: /Volumes/DevDrive
- Homebrew: not installed
- Git: installed
- Docker: not installed
- Node: installed
- pnpm: not installed

What would you like to set up?

[ ] Package managers
[ ] Core developer tools
[ ] JavaScript/TypeScript
[ ] Python
[ ] PHP/Laravel
[ ] Go/Rust/Java/.NET
[ ] Docker/containers
[ ] Databases
[ ] Cloud CLIs
[ ] AI developer tools
[ ] Editors/IDEs
[ ] Mobile development
[ ] Security/testing tools
[ ] Productivity apps
[ ] Media/tools
[ ] Configure package caches
[ ] Configure shell aliases
```

The final confirmation screen should show exact planned commands before running anything.

Example:

```text
Planned actions:

Package managers:
- Install Homebrew

Packages:
- brew install git gh jq ripgrep fd fzf
- brew install --cask cursor visual-studio-code orbstack chatgpt
- npm install -g @openai/codex

Shell:
- Add ~/.local/bin to PATH
- Add drive-agent aliases
- Configure pnpm store at /Volumes/DevDrive/Caches/pnpm

Continue? [y/N]
```

---

## 12. Non-Interactive Host Setup

Examples:

```bash
drive-agent host setup --profile developer --yes

drive-agent host setup \
  --package-managers homebrew,npm,uv \
  --categories core,shell,javascript,ai-dev \
  --yes

drive-agent host packages install git gh jq pnpm cursor codex-cli claude-code --yes

drive-agent host setup --file /Volumes/DevDrive/.drive-agent/config/host-profiles/callum-mac.json --yes

drive-agent host setup --file ./host-profile.json --dry-run
```

---

## 13. Host Profile JSON

Example:

```json
{
    "schemaVersion": 1,
    "profileName": "callum-mac-dev",
    "target": {
        "os": "macos",
        "arch": "arm64"
    },
    "packageManagers": {
        "installMissing": true,
        "preferred": [
            "homebrew",
            "homebrew-cask",
            "npm",
            "uv",
            "cargo"
        ]
    },
    "categories": [
        "core",
        "shell",
        "javascript",
        "python",
        "containers",
        "cloud",
        "ai-dev",
        "editors",
        "databases"
    ],
    "packages": {
        "include": [
            "git",
            "gh",
            "jq",
            "ripgrep",
            "fd",
            "fzf",
            "pnpm",
            "node",
            "uv",
            "orbstack",
            "cursor",
            "vscode",
            "chatgpt",
            "codex-cli",
            "claude-code"
        ],
        "exclude": [
            "xcode",
            "android-studio"
        ]
    },
    "shell": {
        "installAliases": true,
        "addLocalBinToPath": true,
        "configureDriveAgentAlias": true
    },
    "caches": {
        "configurePnpmStore": true,
        "configureNpmCache": false,
        "configureBunCache": false
    },
    "safety": {
        "dryRun": false,
        "requireConfirmation": true,
        "allowSudo": true,
        "allowNativeInstallers": false,
        "allowCurlPipeShell": false
    }
}
```

---

## 14. Package Manager Abstraction

The agent should use package manager providers.

Provider interface:

```text
PackageManagerProvider
├── id
├── supported_os
├── detect()
├── install_manager()
├── is_available()
├── search_package()
├── install_package()
├── upgrade_package()
├── uninstall_package()
└── is_package_installed()
```

Supported package managers:

### macOS

```text
homebrew
homebrew-cask
mas
npm
pnpm
bun
uv
pipx
cargo
go
mise
asdf
nix
```

### Windows

```text
winget
chocolatey
scoop
npm
pnpm
bun
uv
pipx
cargo
go
mise
nix-via-wsl
```

### Linux

```text
apt
dnf
pacman
zypper
apk
homebrew-linux
npm
pnpm
bun
uv
pipx
cargo
go
mise
asdf
nix
```

---

## 15. Package Manager Priority

### macOS

Recommended priority:

```text
1. Homebrew
2. Homebrew Cask
3. Mac App Store via mas
4. npm/pnpm/bun for JavaScript CLIs
5. uv/pipx for Python CLIs
6. mise/asdf for runtimes
7. cargo for Rust CLIs
8. go install for Go CLIs
9. Nix as optional advanced mode
```

### Windows

Recommended priority:

```text
1. Winget
2. Chocolatey
3. Scoop
4. npm/pnpm/bun
5. uv/pipx
6. mise/asdf where available
7. cargo
8. go install
```

### Linux

Recommended priority:

```text
1. Native package manager: apt/dnf/pacman/zypper/apk
2. Homebrew on Linux as optional
3. mise/asdf for runtimes
4. npm/pnpm/bun
5. uv/pipx
6. cargo
7. go install
8. Nix as optional advanced mode
```

---

## 16. Package Catalog Design

Global package catalog:

```text
.drive-agent/catalog/packages.catalog.json
```

Example package entry:

```json
{
    "id": "git",
    "name": "Git",
    "category": "core",
    "description": "Version control CLI",
    "kind": "cli",
    "default": true,
    "installPreference": [
        "homebrew",
        "winget",
        "chocolatey",
        "scoop",
        "apt",
        "dnf",
        "pacman"
    ],
    "install": {
        "homebrew": {
            "type": "formula",
            "name": "git"
        },
        "winget": {
            "id": "Git.Git"
        },
        "chocolatey": {
            "id": "git"
        },
        "scoop": {
            "id": "git"
        },
        "apt": {
            "id": "git"
        },
        "dnf": {
            "id": "git"
        },
        "pacman": {
            "id": "git"
        }
    },
    "check": {
        "command": "git --version"
    }
}
```

Example AI package entry:

```json
{
    "id": "codex-cli",
    "name": "OpenAI Codex CLI",
    "category": "ai-dev",
    "description": "OpenAI terminal coding agent",
    "kind": "cli",
    "installPreference": [
        "homebrew",
        "npm"
    ],
    "install": {
        "homebrew": {
            "type": "cask",
            "name": "codex"
        },
        "npm": {
            "global": true,
            "name": "@openai/codex"
        }
    },
    "check": {
        "command": "codex --version"
    }
}
```

Native installer example:

```json
{
    "id": "claude-code",
    "name": "Claude Code",
    "category": "ai-dev",
    "description": "Anthropic Claude coding agent",
    "kind": "cli",
    "installPreference": [
        "native",
        "homebrew",
        "winget"
    ],
    "install": {
        "native": {
            "macos": "curl -fsSL https://claude.ai/install.sh | bash",
            "windows": "irm https://claude.ai/install.ps1 | iex"
        },
        "homebrew": {
            "name": "claude-code"
        },
        "winget": {
            "id": "Anthropic.ClaudeCode"
        }
    },
    "check": {
        "command": "claude --version"
    },
    "requiresExplicitApproval": true
}
```

---

## 17. Package Categories

### Package Managers

```text
homebrew
macports
mas
winget
chocolatey
scoop
nix
mise
asdf
npm
pnpm
yarn
bun
uv
pipx
cargo
go
```

### Core CLI Tools

```text
git
gh
jq
yq
curl
wget
rsync
openssl
gnupg
age
sops
direnv
tree
watch
coreutils
findutils
gnu-sed
grep
htop
btop
fastfetch
shellcheck
shfmt
```

### Search, Navigation, Shell Productivity

```text
ripgrep
fd
fzf
bat
eza
zoxide
starship
tmux
lazygit
delta
tig
glow
mdcat
hyperfine
tokei
cloc
dust
duf
ncdu
```

### JavaScript / TypeScript

```text
node
nvm
fnm
volta
pnpm
yarn
bun
deno
typescript
tsx
turbo
vercel
netlify-cli
wrangler
eslint
prettier
npm-check-updates
```

### Python

```text
python
uv
pipx
poetry
ruff
black
mypy
pyright
ipython
jupyterlab
pytest
```

### PHP / Laravel

```text
php
composer
laravel-installer
symfony-cli
wp-cli
php-cs-fixer
```

### Go / Rust / Java / .NET

```text
go
goreleaser
rust
rustup
cargo-binstall
java
openjdk
maven
gradle
dotnet
```

### Compilers and Build Tools

```text
xcode-command-line-tools
cmake
ninja
make
gcc
llvm
pkg-config
autoconf
automake
libtool
```

### Containers and Local Infrastructure

```text
docker
orbstack
podman
colima
lima
docker-compose
lazydocker
dive
hadolint
kubectl
k9s
helm
kind
minikube
tilt
skaffold
```

### Databases and Local Services

```text
postgresql
mysql
mariadb
sqlite
redis
valkey
mongodb
clickhouse
duckdb
supabase-cli
neonctl
psql
pgcli
mycli
dbmate
atlas
prisma
```

### Cloud and Infrastructure CLIs

```text
awscli
azure-cli
google-cloud-sdk
doctl
vercel
netlify-cli
wrangler
flyctl
render-cli
heroku
stripe-cli
terraform
tofu
terragrunt
pulumi
ansible
```

### AI Developer Tools

```text
codex-cli
codex-app
claude-code
chatgpt
cursor
vscode
windsurf
aider
continue
opencode
ollama
lm-studio
jan
llama.cpp
huggingface-cli
repomix
ast-grep
```

### Editors and IDEs

```text
cursor
vscode
vscode-insiders
windsurf
zed
jetbrains-toolbox
intellij-idea
phpstorm
webstorm
pycharm
android-studio
xcode
sublime-text
vim
neovim
emacs
```

### Mobile Development

```text
xcode
xcode-command-line-tools
android-studio
android-platform-tools
watchman
cocoapods
fastlane
expo-cli
eas-cli
maestro
```

### API, Testing, and Security

```text
postman
insomnia
bruno
httpie
xh
curlie
grpcurl
k6
playwright
vitest
snyk
trivy
grype
syft
semgrep
gitleaks
detect-secrets
zap
nmap
mkcert
```

### Documentation and Writing

```text
obsidian
notion
logseq
typora
mark-text
pandoc
mermaid-cli
drawio
figma
imageoptim
```

### Browsers

```text
google-chrome
firefox
arc
brave-browser
microsoft-edge
opera
```

### Generic Productivity and Media Apps

```text
vlc
iina
mpv
handbrake
keka
the-unarchiver
raycast
alfred
rectangle
alt-tab
hiddenbar
stats
iterm2
warp
ghostty
wezterm
alacritty
1password
bitwarden
```

---

## 18. Default Host Profiles

### `minimal.json`

Installs:

```text
homebrew
git
gh
jq
ripgrep
fd
fzf
bat
eza
zoxide
starship
vscode or cursor
```

### `developer.json`

Installs:

```text
minimal profile
node
pnpm
uv
python
docker or orbstack
postgresql client
redis client
stripe-cli
wrangler
vercel
cursor
vscode
```

### `ai-developer.json`

Installs:

```text
developer profile
codex-cli
claude-code
chatgpt
cursor
windsurf optional
aider
ollama
lm-studio optional
repomix
ast-grep
```

### `full-stack-saas.json`

Installs:

```text
developer profile
awscli
google-cloud-sdk
azure-cli optional
terraform or tofu
supabase-cli
neonctl
docker or orbstack
postman or bruno
snyk
trivy
gitleaks
semgrep
```

### `mobile.json`

Installs:

```text
developer profile
xcode
android-studio
android-platform-tools
watchman
cocoapods
fastlane
expo-cli
eas-cli
maestro
```

---

## 19. Host State Tracking

Per-host state should be stored on the drive:

```text
/Volumes/DevDrive/.drive-agent/state/hosts/callums-mac-mini.json
```

Example:

```json
{
    "hostId": "callums-mac-mini",
    "hostname": "Callums-Mac-mini",
    "os": "macos",
    "arch": "arm64",
    "shell": "zsh",
    "lastSetupAt": "2026-05-02T21:00:00Z",
    "packageManagers": {
        "homebrew": {
            "installed": true,
            "path": "/opt/homebrew/bin/brew",
            "version": "..."
        }
    },
    "installedPackages": {
        "git": {
            "manager": "homebrew",
            "installedAt": "2026-05-02T21:00:00Z"
        },
        "codex-cli": {
            "manager": "npm",
            "installedAt": "2026-05-02T21:00:00Z"
        }
    }
}
```

This state file should be treated as a cache. The real host should always be verified by `drive-agent host doctor`.

---

## 20. Host Setup Safety Rules

1. Never install host packages during `drive-agent init`.
2. Never install package managers without explicit consent.
3. Always support `--dry-run`.
4. Never run `sudo` silently.
5. Never run `curl | bash` or equivalent silently.
6. Never install both Docker Desktop and OrbStack by default.
7. Never overwrite shell config without a backup.
8. Always log every command run.
9. Always show package source and package manager before install.
10. Always allow package exclusions.
11. Always support non-interactive JSON-driven setup.
12. Always make native installers opt-in with explicit approval.

Shell config modifications should be wrapped in a marked block:

```bash
# >>> drive-agent >>>
export PATH="$HOME/.local/bin:$PATH"
alias da="drive-agent"
# <<< drive-agent <<<
```

Before modifying shell files, create backups:

```text
~/.zshrc.drive-agent-backup-2026-05-02
~/.bashrc.drive-agent-backup-2026-05-02
```

---

## 21. Organization Commands

### Commands

```bash
drive-agent org add roamar
drive-agent org add "Jasper's Classroom" --slug jaspersclassroom
drive-agent org list
drive-agent org rename roamar roamar-old
drive-agent org archive roamar
drive-agent org restore roamar
drive-agent org delete roamar
```

### Created Structure

```text
/Volumes/DevDrive/Orgs/roamar
├── projects
├── archives
└── notes
```

### Safety

- `org delete` should be soft-delete by default.
- Deleting an org containing projects should require explicit confirmation.
- Archive should move the org or mark it archived without deleting data.

---

## 22. Project Commands

### Interactive Add

```bash
drive-agent project add
```

Interactive flow:

```text
Project name?
Organization?
Project type?
Package manager?
Git remote?
Clone now or create empty folder?
Tags?
Open in editor?
```

### Flag-Based Add

```bash
drive-agent project add \
  --org roamar \
  --name user-web \
  --type nextjs \
  --package-manager pnpm \
  --git git@github.com:roamar/user-web.git \
  --tags web,nextjs,typescript \
  --clone
```

### Additional Commands

```bash
drive-agent project list
drive-agent project list --org roamar
drive-agent project list --tag nextjs
drive-agent project open user-web
drive-agent project path user-web
drive-agent project archive user-web
drive-agent project restore user-web
drive-agent project move user-web --org personal
drive-agent project reindex
drive-agent project stale --older-than 90d
```

### Project Manifest

Each project should have a manifest:

```text
/Volumes/DevDrive/Orgs/roamar/projects/user-web/.drive-project.toml
```

Example:

```toml
id = "proj_roamar_user_web"
name = "User Web"
slug = "user-web"
org = "roamar"
type = "nextjs"
package_manager = "pnpm"
tags = ["web", "nextjs", "typescript"]
git_remote = "git@github.com:roamar/user-web.git"
created_at = "2026-05-02T00:00:00Z"
```

---

## 23. Project Templates

Commands:

```bash
drive-agent project add --template nextjs
drive-agent project add --template laravel
drive-agent project add --template expo
drive-agent project add --template hono-api
drive-agent template list
drive-agent template add
```

Template path:

```text
.drive-agent/templates/projects
├── nextjs
├── expo
├── laravel
├── hono-api
├── worker
└── cli
```

Templates can support variables:

```text
{{project_name}}
{{project_slug}}
{{org_slug}}
{{package_manager}}
{{created_at}}
```

---

## 24. Workspace Commands

Purpose: generate editor workspaces for related projects.

Commands:

```bash
drive-agent workspace create roamar \
  --projects api,user-web,mobile,worker

drive-agent workspace open roamar
drive-agent workspace list
drive-agent workspace delete roamar
```

Output:

```text
/Volumes/DevDrive/Workspaces/roamar.code-workspace
```

---

## 25. Shell CD Helpers

A normal CLI process cannot change the parent shell's current directory, so host setup should install shell functions.

Example:

```bash
da-cd() {
  cd "$(drive-agent project path "$1")"
}
```

Usage:

```bash
da-cd user-web
da-cd roamar/user-web
```

Additional helper:

```bash
da-open() {
  cursor "$(drive-agent project path "$1")"
}
```

---

## 26. Git Tools

### Commands

```bash
drive-agent git status-all
drive-agent git fetch-all
drive-agent git pull-all
drive-agent git push-all
drive-agent git branch-all
drive-agent git remotes
drive-agent git dirty
drive-agent git unpushed
```

### Filtering

```bash
drive-agent git status-all --org roamar
drive-agent git pull-all --org roamar
drive-agent git push-all --tag nextjs
drive-agent git fetch-all --project user-web
```

### Pull Safety

Default behavior:

- skip dirty repos
- skip detached HEAD
- show branch and remote
- show skipped repos
- support dry-run

Example:

```bash
drive-agent git pull-all --org roamar --dry-run
drive-agent git pull-all --org roamar
```

### Push Safety

Default behavior:

- require confirmation
- skip repos without upstream
- skip detached HEAD
- show commits to push
- show remote target
- support dry-run

Example:

```bash
drive-agent git push-all --org roamar --dry-run
drive-agent git push-all --org roamar
```

---

## 27. Cleanup System

### Commands

```bash
drive-agent cleanup scan
drive-agent cleanup --dry-run
drive-agent cleanup --apply
drive-agent cleanup --project user-web
drive-agent cleanup --org roamar
drive-agent cleanup --all
drive-agent cleanup --include node_modules,.next
drive-agent cleanup --older-than 30d
```

### Default Cleanup Targets

```text
node_modules
.next
.nuxt
.output
dist
build
.turbo
.vercel
.cache
coverage
playwright-report
test-results
.expo
.expo-shared
android/.gradle
ios/Pods
vendor
target
.DS_Store
```

### Default Behavior

`cleanup` should default to dry-run.

Example scan output:

```text
Found removable cache/build directories:

1. /Orgs/roamar/projects/user-web/node_modules       1.8 GB
2. /Orgs/roamar/projects/user-web/.next              650 MB
3. /Orgs/roamar/projects/mobile/node_modules         2.1 GB
4. /Orgs/roamar/projects/api/coverage                120 MB

Total reclaimable: 4.67 GB
```

Apply:

```bash
drive-agent cleanup --apply
```

### Cleanup Safety Rules

1. Never clean outside the drive root.
2. Never follow symlinks by default.
3. Never delete unknown folders without a matching rule.
4. Always support dry-run.
5. Require confirmation for large deletions.
6. Require confirmation for `node_modules` unless `--yes`.
7. Log all deleted paths.
8. Support a soft-delete mode into `/Trash` where feasible.

---

## 28. Health Checks

### Commands

```bash
drive-agent doctor
drive-agent health
drive-agent host doctor
drive-agent project doctor
drive-agent backup doctor
```

### Drive Checks

```text
Drive is mounted
Drive marker exists
Database exists
Required folders exist
Drive is writable
Available space is above threshold
Drive path is expected
APFS/exFAT/NTFS filesystem detection
No unexpected broken symlinks
```

### Database Checks

```text
SQLite opens successfully
Schema version is current
Migrations are applied
Projects in DB exist on disk
Project manifests match DB rows
Duplicate slugs
Orphan project folders
Missing organizations
```

### Host Checks

```text
drive-agent is on PATH
Shell aliases installed
Homebrew/winget/chocolatey/scoop status
Git available
GitHub CLI available
Docker/OrbStack available
Node/pnpm/npm/bun available
Python/uv available
Editor available
Package caches configured
```

### Git Checks

```text
Dirty repos
Unpushed commits
Missing remotes
Missing upstream branches
Detached HEAD repos
Stale branches
Large files
```

### Backup Checks

```text
Backup tool installed
Backup destination reachable
Last successful backup
Backup excludes configured
Restore test status
```

---

## 29. Backup System

The external drive is active storage, not a backup. It needs to be backed up.

Recommended strategy:

```text
Active data:
- External NVMe DevDrive

Local backup:
- Separate physical backup disk
- Time Machine or clone/snapshot utility

Offsite/cloud backup:
- Restic, Kopia, Backblaze B2, S3-compatible storage, Google Drive, Dropbox, etc.
```

### Backup Commands

```bash
drive-agent backup init
drive-agent backup run
drive-agent backup status
drive-agent backup check
drive-agent backup restore
drive-agent backup test-restore
drive-agent backup exclude add node_modules
drive-agent backup exclude list
```

### Recommended Backup Tools

Support adapters for:

```text
restic
kopia
rclone
rsync
time-machine-helper
```

### Default Backup Excludes

```text
node_modules
.next
.nuxt
.turbo
.cache
dist
build
coverage
.DS_Store
.expo
android/.gradle
ios/Pods
vendor
target
```

### Important Rule

A backup should not be considered valid until a restore test has succeeded.

---

## 30. Database Design

SQLite database path:

```text
.drive-agent/db/drive-agent.sqlite
```

### Schema Draft

```sql
CREATE TABLE drive (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE hosts (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    os TEXT NOT NULL,
    arch TEXT,
    shell TEXT,
    last_seen_at TEXT,
    setup_completed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    path TEXT NOT NULL,
    notes TEXT,
    archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    git_remote TEXT,
    default_branch TEXT,
    project_type TEXT,
    package_manager TEXT,
    framework TEXT,
    archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_opened_at TEXT,
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE project_tags (
    project_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (project_id, tag),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE package_install_records (
    id TEXT PRIMARY KEY,
    host_id TEXT NOT NULL,
    package_id TEXT NOT NULL,
    manager TEXT NOT NULL,
    installed_at TEXT NOT NULL,
    status TEXT NOT NULL,
    version TEXT,
    FOREIGN KEY (host_id) REFERENCES hosts(id)
);

CREATE TABLE command_runs (
    id TEXT PRIMARY KEY,
    command TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    log_path TEXT
);
```

---

## 31. Reindexing

Command:

```bash
drive-agent project reindex
```

Purpose:

- scan for `.drive-project.toml`
- repair missing database rows
- detect project folders without manifests
- detect database rows whose folders are missing
- optionally regenerate manifests from Git metadata

Options:

```bash
drive-agent project reindex --dry-run
drive-agent project reindex --repair
drive-agent project reindex --org roamar
```

---

## 32. Self-Update System

The agent will be hosted on GitHub.

Commands:

```bash
drive-agent self version
drive-agent self update
drive-agent self update --channel stable
drive-agent self update --channel beta
drive-agent self rollback
```

### Update Flow

1. Check current version.
2. Fetch latest release metadata.
3. Download update to temp folder.
4. Verify checksum.
5. Back up current binary/scripts.
6. Swap binary atomically.
7. Run migrations if needed.
8. Confirm new version.
9. Record update in logs.

### Rollback

Keep previous releases in:

```text
.drive-agent/releases
```

Rollback command:

```bash
drive-agent self rollback
```

---

## 33. Local Service Management

Optional but useful.

Commands:

```bash
drive-agent service list
drive-agent service add postgres --org roamar
drive-agent service start postgres
drive-agent service stop postgres
drive-agent service status
drive-agent service logs postgres
```

Useful for:

- Postgres
- Redis
- MinIO
- Mailpit
- LocalStack
- Meilisearch
- Typesense
- Elasticsearch/OpenSearch

Service state should live under:

```text
/Volumes/DevDrive/DevData/local-services
```

---

## 34. Port Registry

Useful when many apps run locally.

Commands:

```bash
drive-agent ports list
drive-agent ports reserve roamar-api 4000
drive-agent ports reserve roamar-user-web 3000
drive-agent ports release roamar-api
```

Database table idea:

```sql
CREATE TABLE ports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    port INTEGER NOT NULL UNIQUE,
    project_id TEXT,
    notes TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

---

## 35. Environment Template System

Purpose: create safe `.env.example` or `.env.local` scaffolds without storing production secrets.

Commands:

```bash
drive-agent env template list
drive-agent env template add nextjs
drive-agent env init roamar/user-web
drive-agent env validate roamar/user-web
```

Safety:

- Do not store real production secrets in SQLite.
- Prefer `.env.example` and `.env.local.template`.
- Encourage 1Password, Doppler, Infisical, Apple Keychain, cloud secret managers, or provider secret stores.

---

## 36. Audit and Scanner Tools

### Repo Audit

```bash
drive-agent audit repos
```

Checks:

```text
missing README
missing license
missing .gitignore
dirty worktree
unpushed commits
large files
secrets-looking files
old branches
missing package manager lockfile
```

### Large File Scanner

```bash
drive-agent scan large-files --min 100MB
```

### Duplicate Scanner

```bash
drive-agent scan duplicates
```

### Stale Project Scanner

```bash
drive-agent project stale --older-than 90d
```

---

## 37. Dashboard

Command:

```bash
drive-agent dashboard
```

Example output:

```text
DevDrive

Drive:
- Path: /Volumes/DevDrive
- Free space: 712 GB
- Used space: 288 GB

Projects:
- Total: 48
- Active: 39
- Archived: 9

Git:
- Dirty repos: 6
- Unpushed repos: 3
- Missing upstream: 2

Cleanup:
- Reclaimable cache/build data: 38 GB

Backup:
- Last backup: 2026-05-01 22:10
- Last restore test: 2026-04-28 19:00

Host:
- Current host: Callums-Mac-mini
- Setup profile: ai-developer
- Health: warning
```

---

## 38. Implementation Language Recommendation

### MVP

A shell script MVP is acceptable for:

- init
- folder creation
- simple org/project commands
- Git status loops
- cleanup scan
- PATH setup

### Long-Term

Build the core CLI in Go.

Reasons:

- single binary
- cross-platform
- strong filesystem support
- good interactive CLI libraries
- easy JSON/TOML parsing
- easy command execution
- easy GitHub releases
- no dependency on Node/Python being installed first

Recommended structure:

```text
drive-agent
├── README.md
├── install.sh
├── src
│   ├── main.go
│   ├── cli
│   ├── commands
│   │   ├── init
│   │   ├── host
│   │   ├── org
│   │   ├── project
│   │   ├── git
│   │   ├── cleanup
│   │   ├── backup
│   │   ├── package
│   │   ├── service
│   │   └── self
│   ├── db
│   ├── filesystem
│   ├── config
│   ├── shell
│   ├── packages
│   │   ├── catalog
│   │   └── providers
│   │       ├── homebrew
│   │       ├── winget
│   │       ├── chocolatey
│   │       ├── scoop
│   │       ├── npm
│   │       ├── uv
│   │       ├── cargo
│   │       └── mise
│   ├── host
│   │   ├── detect
│   │   └── setup
│   ├── ui
│   ├── logging
│   └── utils
├── catalog
├── templates
├── migrations
├── scripts
└── tests
```

---

## 39. MVP Roadmap

### Phase 1: Drive Foundation

Build:

```bash
drive-agent init
drive-agent status
drive-agent doctor
drive-agent host setup
```

Deliverables:

- non-destructive drive init
- folder layout
- SQLite database
- marker file
- PATH/alias setup
- basic host detection

### Phase 2: Organizations and Projects

Build:

```bash
drive-agent org add
drive-agent org list
drive-agent project add
drive-agent project list
drive-agent project open
drive-agent project path
drive-agent project reindex
```

Deliverables:

- project manifests
- SQLite registry
- project scanning
- editor opening support

### Phase 3: Host Provisioning

Build:

```bash
drive-agent host setup --interactive
drive-agent host setup --file profile.json
drive-agent host packages install
drive-agent host package-managers install
drive-agent host doctor
```

Deliverables:

- package catalog
- package manager providers
- interactive package selector
- non-interactive JSON setup
- host state records

### Phase 4: Git Tools

Build:

```bash
drive-agent git status-all
drive-agent git fetch-all
drive-agent git pull-all
drive-agent git push-all
```

Deliverables:

- filtering by org/project/tag
- dry-run
- safe dirty repo handling
- push confirmation

### Phase 5: Cleanup

Build:

```bash
drive-agent cleanup scan
drive-agent cleanup --dry-run
drive-agent cleanup --apply
```

Deliverables:

- cleanup rules
- size calculation
- dry-run output
- safe delete/apply mode

### Phase 6: Backup

Build:

```bash
drive-agent backup init
drive-agent backup run
drive-agent backup status
drive-agent backup check
drive-agent backup test-restore
```

Deliverables:

- restic/kopia/rclone adapter plan
- backup excludes
- backup logs
- restore test

### Phase 7: Self-Update

Build:

```bash
drive-agent self version
drive-agent self update
drive-agent self rollback
```

Deliverables:

- GitHub release update flow
- checksum verification
- rollback support
- migration runner

### Phase 8: Advanced Utilities

Build:

```bash
drive-agent dashboard
drive-agent audit repos
drive-agent scan large-files
drive-agent scan duplicates
drive-agent workspace create
drive-agent service start
drive-agent ports reserve
```

---

## 40. Critical Safety Requirements

These should be treated as non-negotiable:

```text
Never erase the drive.
Never delete without dry-run first.
Never run destructive cleanup outside the drive root.
Never assume /Volumes/DevDrive exists.
Never trust symlinks during cleanup unless explicitly allowed.
Never push all repos without confirmation.
Never overwrite existing projects.
Never store production secrets in SQLite.
Never install host packages during drive init.
Never run sudo silently.
Never run native installers silently.
Always keep project folders usable without drive-agent.
Always allow database rebuild from project manifests.
Always log host setup, cleanup, Git, backup, and update operations.
Always support --dry-run for destructive or system-changing commands.
```

---

## 41. Example End-to-End Flow

### First-time drive setup

```bash
/Volumes/DevDrive/.drive-agent/bin/drive-agent init --path /Volumes/DevDrive
```

### Host setup

```bash
drive-agent host setup
```

or:

```bash
drive-agent host setup \
  --profile ai-developer \
  --yes
```

### Add organizations

```bash
drive-agent org add roamar
drive-agent org add jaspersclassroom
drive-agent org add personal
```

### Add a project

```bash
drive-agent project add \
  --org roamar \
  --name user-web \
  --type nextjs \
  --package-manager pnpm \
  --git git@github.com:roamar/user-web.git \
  --tags web,nextjs,typescript \
  --clone
```

### Open a project

```bash
drive-agent project open roamar/user-web --editor cursor
```

### Check repos

```bash
drive-agent git status-all --org roamar
```

### Pull clean repos

```bash
drive-agent git pull-all --org roamar
```

### Scan cleanup

```bash
drive-agent cleanup scan
```

### Apply cleanup

```bash
drive-agent cleanup --apply
```

### Run backup

```bash
drive-agent backup run
```

### Check drive health

```bash
drive-agent doctor
```

---

## 42. Codex/Cursor Build Prompt

Use this prompt when ready to generate the first implementation:

```text
Build a portable external-drive development manager called drive-agent.

The tool should live on an external drive under .drive-agent and provide a command called drive-agent. The initial implementation should prioritize macOS but be architected for Windows/Linux later.

Implement Phase 1 and Phase 2 only:

Phase 1:
- drive-agent init
- drive-agent status
- drive-agent doctor
- drive-agent host setup for PATH/alias only

Phase 2:
- drive-agent org add/list
- drive-agent project add/list/path/open/reindex
- SQLite registry
- .drive-project.toml manifests

Critical requirements:
- init must be non-destructive and must never erase/format anything
- refuse dangerous root paths
- create a .drive-agent/DRIVE_AGENT_ROOT marker
- database is an index, not the source of truth
- project manifests must allow rebuilding the DB
- all destructive/system-changing commands need dry-run support where applicable
- no package installs yet
- no backup implementation yet
- no cleanup apply yet

Use a clean command architecture so later phases can add host provisioning, package catalogs, cleanup, backup, Git bulk operations, and self-update.
```

---

## 43. Final Recommended Direction

Start with a small but safe MVP:

```text
init
host setup for PATH/alias
org add/list
project add/list/open/path/reindex
git status-all
cleanup scan
```

Then add host provisioning and package management once the project model feels right.

The most valuable long-term feature is not just “installing tools” or “cleaning folders”; it is the combination of:

```text
portable drive metadata
host profiles
project registry
safe Git operations
safe cleanup
rebuildable database
backup awareness
```

That combination turns the external NVMe into a durable, portable development workspace rather than just a folder full of projects.
