# Installation Guide

Drive Agent is designed to be installed directly onto your external development drive. This allows you to plug the drive into any machine and immediately have access to your environment.

## 1. Initializing the Drive

Before installing the agent, initialize the root structure on your drive:

```bash
# Example for a drive named "DevDrive"
./drive-agent init --path /Volumes/DevDrive --name DevDrive --non-interactive
```

This creates the `.drive-agent` directory, database, and project folders.

## 2. Running the Installer

Use the provided `install.sh` script to build the agent and install it to the drive:

```bash
./install.sh --drive /Volumes/DevDrive
```

The script will:
1. Build `drive-agent` from source.
2. Verify the target is safe (e.g. not a system directory).
3. Copy the binary to `/Volumes/DevDrive/.drive-agent/bin/drive-agent`.
4. Prompt to configure your host shell (adds a PATH entry and aliases).

### Installer Options

- `--drive <path>`: The target drive root (required).
- `--binary <path>`: Skip building and use an existing binary.
- `--skip-shell`: Do not prompt to modify `.zshrc` or `.bashrc`.
- `--yes`: Accept all prompts automatically.
- `--dry-run`: Show what would be done without modifying the disk.

For a locally built binary:

```bash
go build -o ./drive-agent ./cmd/drive-agent
./install.sh --drive /Volumes/DevDrive --binary ./drive-agent --skip-shell --yes
```

Verify the installed binary from the drive:

```bash
/Volumes/DevDrive/.drive-agent/bin/drive-agent self version
/Volumes/DevDrive/.drive-agent/bin/drive-agent doctor --path /Volumes/DevDrive
```

## 3. Copying An Existing Repository Into A Managed Project

Create the organization and project first so Drive Agent writes the database row and `.drive-project.toml` manifest:

```bash
/Volumes/DevDrive/.drive-agent/bin/drive-agent org add MyOrg --path /Volumes/DevDrive
/Volumes/DevDrive/.drive-agent/bin/drive-agent project add \
  --path /Volumes/DevDrive \
  --org myorg \
  --name my-app \
  --type turborepo \
  --package-manager pnpm
```

Then copy source files with `rsync`. Dry-run first, and do not use `--delete` for the first copy:

```bash
rsync -avhn --progress \
  --exclude node_modules \
  --exclude .next \
  --exclude .turbo \
  --exclude dist \
  --exclude build \
  /path/to/existing-repo/ \
  /Volumes/DevDrive/Orgs/myorg/projects/my-app/

rsync -avh --progress \
  --exclude node_modules \
  --exclude .next \
  --exclude .turbo \
  --exclude dist \
  --exclude build \
  /path/to/existing-repo/ \
  /Volumes/DevDrive/Orgs/myorg/projects/my-app/
```

After copying, verify that `.drive-project.toml` still exists. If a source copy overwrites or removes it, recreate the project manifest from Drive Agent metadata or run `project reindex` after restoring a correct manifest.

## 4. Host Setup

If you skipped the shell configuration during installation, or if you plug the drive into a new computer, you can run the setup command directly:

```bash
/Volumes/DevDrive/.drive-agent/bin/drive-agent host setup
```

This will safely add the agent to your PATH and install any required host packages based on your profile.
