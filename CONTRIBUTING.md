# Contributing to Drive Agent

Thank you for your interest in contributing to Drive Agent!

## Local Setup

To build and test Drive Agent locally, you will need Go 1.21+ installed and CGO enabled (since we use go-sqlite3).

```bash
# Clone the repo
git clone https://github.com/callum-baillie/drive-agent.git
cd drive-agent

# Build the binary
go build -o drive-agent ./cmd/drive-agent
```

## Running Tests

Before submitting any Pull Requests, please ensure all tests pass:

```bash
go test ./...
go vet ./...
go build ./cmd/drive-agent
bash tests/smoke_test.sh
```

## Coding Style

- Use `gofmt` to format your code.
- Write clear and concise commit messages.
- Prefer simplicity and readability over cleverness.

## Safety Expectations

Drive Agent manages user files on an external drive. Safety is paramount:
1. **Never perform destructive actions without confirmation.**
2. **Never escape the drive boundary.** Rely on `IsDangerousPath` to validate directory traversal.
3. **Dry-runs first.** Add `--dry-run` to all state-mutating features where possible.

## Opening Pull Requests

- Keep PRs small and focused on a single issue or feature.
- Follow the Pull Request template provided.
- Ensure the CI pipeline passes before requesting a review.
