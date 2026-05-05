# Restic Backup Provider

Restic is the first implemented Drive Agent backup provider. Drive Agent shells out to `restic`; it does not reimplement backup storage, encryption, snapshots, or restore logic.

## Install Restic

```bash
drive-agent host packages install restic
```

If Restic is missing, backup commands tell you to run that command.

## Initialize

Local repository on a separate backup drive:

```bash
drive-agent backup init --provider restic --repo /Volumes/BackupDrive/restic/devdrive
```

Remote examples:

```bash
drive-agent backup init --provider restic --repo sftp:user@example.com:/backups/devdrive
drive-agent backup init --provider restic --repo s3:s3.amazonaws.com/bucket/devdrive
drive-agent backup init --provider restic --repo s3:http://storage.example.test:9000/bucket/devdrive
```

For S3-compatible providers, keep credentials in the current shell environment only:

```bash
export AWS_ACCESS_KEY_ID='<access-key-id>'
export AWS_SECRET_ACCESS_KEY='<secret-access-key>'
export AWS_DEFAULT_REGION='<region>'

read -s RESTIC_PASSWORD
export RESTIC_PASSWORD
```

Do not put S3 credentials or the Restic password in `backup.json`, `.drive-project.toml`, docs, logs, shell history, or SQLite. The Restic repository URL may include the endpoint, bucket, and path, but not secrets.

Use a named repository if you plan to track multiple destinations:

```bash
drive-agent backup init --provider restic --name local-backup --repo /Volumes/BackupDrive/restic/devdrive
drive-agent backup run --repo local-backup
```

Drive Agent rejects repositories inside the source drive:

```bash
drive-agent backup init --provider restic --repo /Volumes/DevDrive/Backups
```

For disposable tests only:

```bash
drive-agent backup init --provider restic --repo /Volumes/DevDrive/Backups --allow-same-drive-repo
```

## Run

```bash
read -s RESTIC_PASSWORD
export RESTIC_PASSWORD

drive-agent backup run
drive-agent backup run --dry-run
drive-agent backup run --tag manual
drive-agent backup run --exclude node_modules --exclude .next
```

The generated Restic backup command includes:

```bash
restic backup <DriveRoot> \
  --repo <repo> \
  --exclude-file <generated-exclude-file> \
  --tag drive-agent \
  --tag drive:<drive-name-or-id> \
  --tag host:<hostname>
```

`backup run --dry-run` writes the generated exclude file and prints the planned command without creating a snapshot.

## Snapshots And Checks

```bash
drive-agent backup snapshots
drive-agent backup snapshots --json
drive-agent backup check
```

`backup snapshots` uses `restic snapshots --json` internally and parses the result into typed snapshot data.

`backup check` runs `restic check`. It does not run expensive full data checks by default.

## Default Excludes

Drive Agent excludes generated dependency/build artifacts such as:

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
.Spotlight-V100
.TemporaryItems
.Trashes
.fseventsd
.drive-agent/releases/tmp
```

Important source/config paths are not excluded by default, including `.git`, `docs`, `prisma`, `migrations`, `src`, `package.json`, lockfiles, `README.md`, and `.drive-project.toml`.

## Project-Level Excludes

Project-level excludes live in the project manifest and are scoped to that project when the Restic exclude file is generated:

```toml
[backup]
excludes = [
  "node_modules",
  ".next",
  ".turbo",
  "apps/*/node_modules",
  "apps/*/.next",
  "packages/*/dist",
]
```

Use the CLI to manage them:

```bash
drive-agent backup excludes add --project personal/my-app node_modules
drive-agent backup excludes add --project personal/my-app 'apps/*/node_modules'
drive-agent backup excludes list --project personal/my-app
```

Wildcards are preserved for Restic. A project-level `apps/*/node_modules` rule becomes a path-scoped Restic exclude for that project instead of a global drive-wide project rule.
