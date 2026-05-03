#!/usr/bin/env bash
set -euo pipefail

# Drive Agent Smoke Test
# Builds the CLI and runs basic commands against a temporary directory.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TMPDIR_BASE="${PROJECT_ROOT}/tests/tmp-smoke-$$"
BINARY="${PROJECT_ROOT}/drive-agent"

cleanup() {
    echo ""
    echo "=== Cleaning up ==="
    rm -rf "$TMPDIR_BASE"
    rm -f "$BINARY"
    echo "Done."
}
trap cleanup EXIT

echo "=== Drive Agent Smoke Test ==="
echo ""

# Build
echo "--- Building drive-agent ---"
cd "$PROJECT_ROOT"
CGO_ENABLED=1 go build -o "$BINARY" ./cmd/drive-agent
echo "Binary built: $BINARY"
echo ""

# Version
echo "--- Testing: version ---"
$BINARY --version
echo ""

# Self version
echo "--- Testing: self version ---"
$BINARY self version
echo ""

# Create temp drive directory
mkdir -p "$TMPDIR_BASE/TestDrive"
DRIVE_PATH="$TMPDIR_BASE/TestDrive"

# Init (using --allow-non-volume-path since we're in /tmp)
echo "--- Testing: init ---"
$BINARY init --path "$DRIVE_PATH" --name "SmokeTestDrive" --allow-non-volume-path --non-interactive
echo ""

# Status
echo "--- Testing: status ---"
cd "$DRIVE_PATH"
$BINARY status
echo ""

# Doctor
echo "--- Testing: doctor ---"
$BINARY doctor
echo ""

# Org add
echo "--- Testing: org add ---"
$BINARY org add "personal"
$BINARY org add "Test Company" --slug test-company
echo ""

# Org list
echo "--- Testing: org list ---"
$BINARY org list
echo ""

# Project add
echo "--- Testing: project add ---"
$BINARY project add --org personal --name "My Website" --type nextjs --package-manager pnpm --tags web,nextjs
echo ""

# Project list
echo "--- Testing: project list ---"
$BINARY project list
echo ""

# Project path
echo "--- Testing: project path ---"
$BINARY project path personal/my-website
echo ""

# Project reindex (dry-run)
echo "--- Testing: project reindex --dry-run ---"
$BINARY project reindex --dry-run
echo ""

# Git status-all
echo "--- Testing: git status-all ---"
$BINARY git status-all || true
echo ""

# Cleanup scan
echo "--- Testing: cleanup scan ---"
# Create a fake node_modules to scan
mkdir -p "$DRIVE_PATH/Orgs/personal/projects/my-website/node_modules"
echo "fake" > "$DRIVE_PATH/Orgs/personal/projects/my-website/node_modules/fake.txt"
$BINARY cleanup scan
echo ""

# Host doctor
echo "--- Testing: host doctor ---"
$BINARY host doctor
echo ""

# Backup status
echo "--- Testing: backup status ---"
$BINARY backup status
echo ""

# Self update
echo "--- Testing: self update ---"
$BINARY self update --dry-run || true
echo ""

echo "=== All smoke tests passed ==="
