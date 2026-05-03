#!/usr/bin/env bash
set -euo pipefail

# Drive Agent Installer
# Builds and installs drive-agent onto an external drive.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DRIVE_ROOT=""
BINARY_PATH=""
SKIP_SHELL="false"
DRY_RUN="false"
YES="false"

function usage() {
    echo "Usage: ./install.sh --drive <path> [options]"
    echo ""
    echo "Options:"
    echo "  --drive <path>      Target drive root (e.g. /Volumes/DevDrive) (required)"
    echo "  --binary <path>     Use an existing binary instead of building from source"
    echo "  --skip-shell        Skip configuring the host shell"
    echo "  --yes               Skip confirmation prompts"
    echo "  --dry-run           Show what would be done without making changes"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --drive) DRIVE_ROOT="$2"; shift 2 ;;
        --binary) BINARY_PATH="$2"; shift 2 ;;
        --skip-shell) SKIP_SHELL="true"; shift ;;
        --yes) YES="true"; shift ;;
        --dry-run) DRY_RUN="true"; shift ;;
        -h|--help) usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

if [[ -z "$DRIVE_ROOT" ]]; then
    echo "Error: --drive is required."
    usage
fi

echo "=== Drive Agent Installer ==="
echo ""

# 1. Safety Checks
if [[ "$OSTYPE" == "darwin"* ]]; then
    # Strict path check for macOS: must be in /Volumes
    if [[ "$DRIVE_ROOT" != /Volumes/* ]]; then
        # Explicit test-only escape hatch
        if [[ "${ALLOW_TEST_DRIVE:-0}" != "1" ]]; then
            echo "Error: On macOS, target drive must be under /Volumes/."
            echo "Refusing to install to: $DRIVE_ROOT"
            echo "For local fake-drive tests, use an APFS disk image mounted to /Volumes."
            exit 1
        fi
    fi
fi

# Basic path validation (block root, home, system paths)
# This also handles descendants by checking string prefixes.
# Ensure trailing slash for prefix matching to avoid blocking /usr-local when blocking /usr.
DANGEROUS=("/Users/" "/System/" "/Library/" "/private/" "/etc/" "/bin/" "/usr/" "/opt/")
CHECK_PATH="${DRIVE_ROOT%/}/" # add trailing slash

if [[ "$CHECK_PATH" == "/" ]]; then
    if [[ "${ALLOW_TEST_DRIVE:-0}" != "1" ]]; then
        echo "Error: Dangerous install target inside protected path: /"
        exit 1
    fi
fi

for dp in "${DANGEROUS[@]}"; do
    if [[ "$CHECK_PATH" == "$dp"* ]]; then
        if [[ "${ALLOW_TEST_DRIVE:-0}" != "1" ]]; then
            echo "Error: Dangerous install target inside protected path: $dp"
            exit 1
        fi
    fi
done

# 2. Build or Find Binary
if [[ -z "$BINARY_PATH" ]]; then
    if ! command -v go &> /dev/null; then
        echo "Error: Go is required to build drive-agent. Install Go or use --binary."
        exit 1
    fi
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[Dry Run] Would build drive-agent from source..."
        BINARY_PATH="$SCRIPT_DIR/drive-agent"
    else
        echo "Building drive-agent..."
        cd "$SCRIPT_DIR"
        CGO_ENABLED=1 go build -o drive-agent ./cmd/drive-agent
        BINARY_PATH="$SCRIPT_DIR/drive-agent"
        echo "Built successfully."
    fi
fi

if [[ "$DRY_RUN" == "false" && ! -f "$BINARY_PATH" ]]; then
    echo "Error: Binary not found at $BINARY_PATH"
    exit 1
fi

VERSION="unknown"
if [[ "$DRY_RUN" == "false" ]]; then
    VERSION=$("$BINARY_PATH" --version | awk '{print $3}') || VERSION="unknown"
fi

# 3. Directories Setup
AGENT_DIR="$DRIVE_ROOT/.drive-agent"
echo "Target drive: $DRIVE_ROOT"

if [[ "$DRY_RUN" == "true" ]]; then
    echo "[Dry Run] Would create directories in $AGENT_DIR: bin, releases, backups, logs, state, config"
else
    mkdir -p "$AGENT_DIR/"{bin,releases,backups,logs,state,config}
fi

# 4. Install Binary & Backups
TARGET_BIN="$AGENT_DIR/bin/drive-agent"

if [[ -f "$TARGET_BIN" ]]; then
    TIMESTAMP=$(date +%Y%m%d%H%M%S)
    BACKUP_BIN="$AGENT_DIR/backups/drive-agent-$TIMESTAMP"
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[Dry Run] Would back up existing binary to $BACKUP_BIN"
    else
        mv "$TARGET_BIN" "$BACKUP_BIN"
        echo "Backed up existing binary to .drive-agent/backups/drive-agent-$TIMESTAMP"
    fi
fi

if [[ "$DRY_RUN" == "true" ]]; then
    echo "[Dry Run] Would copy $BINARY_PATH to $TARGET_BIN"
    echo "[Dry Run] Would write $AGENT_DIR/VERSION ($VERSION)"
    echo "[Dry Run] Would write $AGENT_DIR/install.json"
else
    cp "$BINARY_PATH" "$TARGET_BIN"
    chmod +x "$TARGET_BIN"
    echo "$VERSION" > "$AGENT_DIR/VERSION"
    
    BACKUP_VAL="null"
    if [[ -n "${BACKUP_BIN:-}" ]]; then
        BACKUP_VAL="\"$BACKUP_BIN\""
    fi

    cat > "$AGENT_DIR/install.json" <<EOF
{
  "installed_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "version": "$VERSION",
  "method": "install.sh",
  "install_path": "$TARGET_BIN",
  "drive_root": "$DRIVE_ROOT",
  "source_binary": "$BINARY_PATH",
  "os": "$(uname -s)",
  "arch": "$(uname -m)",
  "repo_owner": "callumbaillie",
  "repo_name": "drive-agent",
  "previous_backup": $BACKUP_VAL
}
EOF
    echo "Installed drive-agent to $TARGET_BIN"
fi

# 5. Shell Setup
if [[ "$SKIP_SHELL" == "false" ]]; then
    SHELL_RC=""
    if [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        if [[ -f "$HOME/.bash_profile" ]]; then
            SHELL_RC="$HOME/.bash_profile"
        else
            SHELL_RC="$HOME/.bashrc"
        fi
    fi
    
    if [[ -n "$SHELL_RC" ]]; then
        if grep -q ">>> drive-agent >>>" "$SHELL_RC" 2>/dev/null; then
            echo "Shell block already present in $SHELL_RC (skipping)"
        else
            DO_SHELL="$YES"
            if [[ "$YES" == "false" && "$DRY_RUN" == "false" ]]; then
                read -p "Install shell aliases and PATH to $SHELL_RC? (Y/n) " resp
                if [[ "$resp" == "" || "$resp" == "Y" || "$resp" == "y" ]]; then
                    DO_SHELL="true"
                fi
            fi
            
            if [[ "$DRY_RUN" == "true" ]]; then
                echo "[Dry Run] Would append shell block to $SHELL_RC"
            elif [[ "$DO_SHELL" == "true" ]]; then
                BACKUP_RC="${SHELL_RC}.drive-agent-backup-$(date +%Y-%m-%d)"
                cp "$SHELL_RC" "$BACKUP_RC"
                
                # Append block (quoting DRIVE_ROOT properly)
                cat >> "$SHELL_RC" <<EOF

# >>> drive-agent >>>
export PATH="${DRIVE_ROOT}/.drive-agent/bin:\$PATH"
alias da="drive-agent"
alias drive="drive-agent"

# drive-agent shell helpers
da-cd() { cd "\$(drive-agent project path "\$1")" ; }
da-open() { drive-agent project open "\$1" ; }
# <<< drive-agent <<<
EOF
                echo "Backed up $SHELL_RC to $BACKUP_RC"
                echo "Shell config updated. Restart your shell or run: source $SHELL_RC"
            fi
        fi
    fi
fi

echo ""
echo "Installation complete!"
echo "If this is a new drive, run: drive-agent init --path \"$DRIVE_ROOT\""
