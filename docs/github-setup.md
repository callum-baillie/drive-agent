# GitHub Setup Recommendations

When setting up the Drive Agent repository on GitHub, we recommend the following settings to keep the workflow clean, safe, and aligned with our Git workflow.

## General Repo Settings

- **Repository visibility:** private initially
- **Default branch:** main
- **Issues:** enabled
- **Discussions:** optional/off for now
- **Wiki:** off
- **Projects:** off for now
- **Security alerts:** on
- **Dependabot alerts:** on
- **Dependabot security updates:** on

## Merge Button Settings

- **Merge strategy:** squash merge (Disable merge commits and rebase merging)
- **Automatically delete head branches:** enabled

## Branch Protection Policy (`main`)

It's highly recommended to protect the `main` branch. Go to Settings -> Branches -> Add branch protection rule:

- **Require a pull request before merging:** enabled
- **Require status checks to pass before merging:** enabled (select the `test` jobs from `ci.yml`)
- **Require branches to be up to date before merging:** enabled
- **Require conversation resolution before merging:** enabled
- **Do not allow bypassing the above settings:** enabled
- **Restrict who can push to matching branches:** enabled (nobody except through PRs)
- **Allow force pushes:** disabled
- **Allow deletions:** disabled

*(Optional later: Require signed commits, Require code owner review, Require linear history).*

## Description & Topics

**Suggested Description:**
Portable developer-drive manager for external SSD workflows, host setup, project organization, cleanup, and safe self-updates.

**Suggested Topics:**
`go`, `cli`, `developer-tools`, `macos`, `external-drive`, `dev-environment`, `automation`
