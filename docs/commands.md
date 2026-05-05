# Commands Reference

## drive-agent version

```bash
drive-agent version                                 # Show version
drive-agent --version                               # Show version
drive-agent self version                            # Show version, install path, and drive root
```

Most commands that operate on an initialized drive accept the global `--path <drive-root>` flag. Without it, Drive Agent searches upward from the current directory for `.drive-agent/DRIVE_AGENT_ROOT`.

## drive-agent init

Initialize a drive for drive-agent management.

```bash
drive-agent init                                    # Initialize current directory
drive-agent init --path /Volumes/DevDrive           # Specify path
drive-agent init --path /Volumes/DevDrive --name DevDrive
drive-agent init --repair                           # Repair existing init
drive-agent init --allow-non-volume-path            # Allow non-/Volumes paths
drive-agent init --non-interactive                  # Skip prompts
```

## drive-agent status

Show drive summary including org/project counts, host info, git dirty repos, cleanup estimates.

```bash
drive-agent status
```

## drive-agent doctor

Run comprehensive health checks on drive structure, database integrity, and host tool availability.

```bash
drive-agent doctor
```

## drive-agent org

```bash
drive-agent org add <name>                          # Add organization
drive-agent org add "My Company" --slug my-co       # Custom slug
drive-agent org list                                # List all orgs
```

## drive-agent project

```bash
drive-agent project add                             # Interactive add
drive-agent project add --org personal --name my-app --type nextjs --tags web
drive-agent project add --org company --name api --git git@github.com:co/api.git --clone
drive-agent project list                            # List all projects
drive-agent project list --org personal             # Filter by org
drive-agent project list --tag nextjs               # Filter by tag
drive-agent project path my-app                     # Print path if project slug is unique
drive-agent project path personal/my-app            # Print path
drive-agent project open personal/my-app            # Open in editor
drive-agent project open personal/my-app --editor cursor
drive-agent project reindex                         # Rebuild DB from manifests
drive-agent project reindex --dry-run               # Preview reindex
```

## drive-agent host

```bash
drive-agent host setup                              # Interactive setup
drive-agent host setup --profile developer          # Use profile
drive-agent host setup --profile ai-developer --yes # Non-interactive
drive-agent host setup --dry-run                    # Preview only
drive-agent host doctor                             # Check host tools
drive-agent host packages list                      # List categories
drive-agent host packages list --category core      # List packages in category
drive-agent host packages install git gh jq         # Install specific packages
drive-agent host packages install --category core --yes
drive-agent host packages install --dry-run         # Preview installs
```

## drive-agent git

```bash
drive-agent git status-all                          # Status of all repos
drive-agent git status-all --org personal           # Filter by org
drive-agent git fetch-all                           # Fetch all repos
drive-agent git fetch-all --dry-run                 # Preview
drive-agent git pull-all                            # Pull clean repos
drive-agent git pull-all --org personal             # Filter
drive-agent git pull-all --dry-run                  # Preview
```

## drive-agent cleanup

```bash
drive-agent cleanup                                 # Scan for targets
drive-agent cleanup --dry-run                       # Scan for targets
drive-agent cleanup --apply                         # Delete targets
drive-agent cleanup --apply --yes                   # Skip confirmation
drive-agent cleanup scan                            # Scan for targets
drive-agent cleanup dry-run                         # Same as scan
drive-agent cleanup apply                           # Delete targets
drive-agent cleanup apply --yes                     # Skip confirmation
drive-agent cleanup apply --org personal            # Filter by org
```

## drive-agent backup

```bash
drive-agent backup init --provider restic --repo /Volumes/BackupDrive/restic/devdrive
drive-agent backup status                           # Show configured repo, state, and warnings
drive-agent backup doctor                           # Diagnose backup readiness
drive-agent backup run                              # Run backup
drive-agent backup run --dry-run                    # Show planned backup command
drive-agent backup run --tag manual                 # Add tag
drive-agent backup snapshots                        # List snapshots
drive-agent backup snapshots --json                 # List snapshots as JSON
drive-agent backup check                            # Run restic check
drive-agent backup restore --snapshot latest --target /Volumes/RestoreTest
drive-agent backup restore --snapshot latest --target /Volumes/RestoreTest --dry-run
drive-agent backup excludes list
drive-agent backup excludes add node_modules
drive-agent backup excludes add --project personal/my-app 'apps/*/node_modules'
drive-agent backup excludes list --project personal/my-app
drive-agent backup excludes remove .next
```

## drive-agent self

```bash
drive-agent self version                            # Show version
drive-agent self update                             # Update from GitHub releases
drive-agent self rollback                           # Rollback to a previous backup
```
