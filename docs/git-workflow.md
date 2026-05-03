# Git Workflow

Keep it simple:

- `main` is protected and stable.
- No direct commits to `main` after initial setup.
- No force pushes to `main`.
- Feature branches use `feature/<short-name>`.
- Fixes use `fix/<short-name>`.
- Docs use `docs/<short-name>`.
- PRs must pass CI before merge.
- Squash merge is preferred to keep the history clean.
