# Fake Drive Testing (macOS APFS)

Testing Drive Agent on a real external drive is best, but when you want an isolated environment to verify behavior without a physical drive, you can mount an APFS disk image. This behaves identically to a real external drive (mounted under `/Volumes`), meaning all safety constraints natively apply.

## 1. Create and mount an APFS disk image

```bash
# Create a 2GB APFS image
hdiutil create -size 2g -fs APFS -volname DriveAgentTest /tmp/DriveAgentTest.dmg

# Attach it (mounts to /Volumes/DriveAgentTest)
hdiutil attach /tmp/DriveAgentTest.dmg
```

## 2. Initialize the Drive

Build the agent locally and initialize the structure on your new disk image:

```bash
go build -o drive-agent ./cmd/drive-agent
./drive-agent init --path /Volumes/DriveAgentTest --name DriveAgentTest
```

## 3. Install the Agent

Use the install script to install the agent into the `.drive-agent/bin` directory on the disk image:

```bash
# Preview what the installer will do
./install.sh --drive /Volumes/DriveAgentTest --binary ./drive-agent --dry-run

# Install it and skip updating your real ~/.zshrc
./install.sh --drive /Volumes/DriveAgentTest --binary ./drive-agent --skip-shell --yes
```

## 4. Test Commands

Verify the agent works correctly from its installation path:

```bash
/Volumes/DriveAgentTest/.drive-agent/bin/drive-agent self version
/Volumes/DriveAgentTest/.drive-agent/bin/drive-agent doctor --path /Volumes/DriveAgentTest
```

## 5. Cleanup

Once testing is complete, detach the volume and remove the image:

```bash
hdiutil detach /Volumes/DriveAgentTest
rm /tmp/DriveAgentTest.dmg
```
