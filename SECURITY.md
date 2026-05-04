# Security Policy

## Reporting a Vulnerability

Please do **NOT** report security vulnerabilities via public GitHub Issues or Discussions.

Security contact is still being finalized for the alpha. Until then, use GitHub's private vulnerability reporting flow if it is enabled for the repository, or contact the maintainer out of band before publishing details.

Include a description of the vulnerability, the steps to reproduce it, and any potential impact. We will try to address it as soon as possible.

## DO NOT POST SECRETS PUBLICLY

Ensure you do not accidentally include any `GITHUB_TOKEN`, passwords, SSH keys, or API tokens in bug reports or issues.

## Safety-Sensitive Areas

When contributing or auditing, please pay special attention to the following areas:
- **Path Validation:** (e.g., `IsDangerousPath` preventing `rm -rf` on system files).
- **Cleanup:** Dry-run boundaries and symlink evaluation.
- **Shell Modification:** Idempotent updates to `.bashrc` or `.zshrc`.
- **Install / Update:** Checksum integrity and atomic updates.
- **Backup / Restore:** Ensuring data integrity.
- **Package Installation:** Validating input passed to shell execution (`brew`, `npm`).
