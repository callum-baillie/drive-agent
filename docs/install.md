# Installation Guide

Drive Agent is designed to be installed directly onto your external development drive. This allows you to plug the drive into any machine and immediately have access to your environment.

## 1. Initializing the Drive

Before installing the agent, initialize the root structure on your drive:

```bash
# Example for a drive named "DevDrive"
./drive-agent init --path /Volumes/DevDrive
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

## 3. Host Setup

If you skipped the shell configuration during installation, or if you plug the drive into a new computer, you can run the setup command directly:

```bash
/Volumes/DevDrive/.drive-agent/bin/drive-agent host setup
```

This will safely add the agent to your PATH and install any required host packages based on your profile.
