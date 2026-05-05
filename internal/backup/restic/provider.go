package restic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/callum-baillie/drive-agent/internal/backup"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
)

type CommandResult struct {
	Output string
}

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args []string, logPath string) (CommandResult, error)
}

type ExecRunner struct{}

func (ExecRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (ExecRunner) Run(ctx context.Context, name string, args []string, logPath string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	output := backup.RedactSensitive(string(out))
	if logPath != "" {
		if writeErr := writeLog(logPath, name, args, output); writeErr != nil && err == nil {
			err = writeErr
		}
	}
	if err != nil {
		return CommandResult{Output: output}, fmt.Errorf("%s failed: %w", backup.RedactSensitive(FormatCommand(name, args)), err)
	}
	return CommandResult{Output: output}, nil
}

type Provider struct {
	Runner  Runner
	LogRoot string
}

func NewProvider(driveRoot string) *Provider {
	return &Provider{
		Runner:  ExecRunner{},
		LogRoot: filepath.Join(filesystem.AgentPath(driveRoot), "logs", "backup"),
	}
}

func (p *Provider) Name() string {
	return "restic"
}

func (p *Provider) IsInstalled() bool {
	runner := p.runner()
	_, err := runner.LookPath("restic")
	return err == nil
}

func (p *Provider) Init(ctx context.Context, repo backup.Repository, dryRun bool) (backup.Plan, error) {
	args := InitArgs(repo.Repository)
	return p.runOrPlan(ctx, "init", args, dryRun)
}

func (p *Provider) Backup(ctx context.Context, req backup.BackupRequest) (backup.Plan, error) {
	tags := []string{"drive-agent", "drive:" + req.DriveLabel, "host:" + req.HostLabel}
	tags = append(tags, req.ExtraTags...)
	args := BackupArgs(req.DriveRoot, req.Repo.Repository, req.ExcludeFile, tags, req.DryRun)
	return p.runOrPlan(ctx, "backup", args, req.DryRun)
}

func (p *Provider) Snapshots(ctx context.Context, repo backup.Repository) ([]backup.Snapshot, error) {
	args := SnapshotsArgs(repo.Repository, true)
	plan, err := p.runOrPlan(ctx, "snapshots", args, false)
	if err != nil {
		return nil, err
	}
	return ParseSnapshotsJSON([]byte(plan.Output))
}

func (p *Provider) Check(ctx context.Context, repo backup.Repository, dryRun bool) (backup.Plan, error) {
	args := CheckArgs(repo.Repository)
	return p.runOrPlan(ctx, "check", args, dryRun)
}

func (p *Provider) Restore(ctx context.Context, req backup.RestoreRequest) (backup.Plan, error) {
	if err := ValidateSnapshot(req.Snapshot); err != nil {
		return backup.Plan{}, err
	}
	args := RestoreArgs(req.Repo.Repository, req.Snapshot, req.Target, req.DryRun)
	return p.runOrPlan(ctx, "restore", args, req.DryRun)
}

func (p *Provider) runOrPlan(ctx context.Context, operation string, args []string, dryRun bool) (backup.Plan, error) {
	cmd := FormatCommand("restic", args)
	plan := backup.Plan{Command: backup.RedactSensitive(cmd)}
	if dryRun {
		return plan, nil
	}

	runner := p.runner()
	if _, err := runner.LookPath("restic"); err != nil {
		return plan, fmt.Errorf("restic is not installed; run: drive-agent host packages install restic")
	}

	logPath, err := p.logPath(operation)
	if err != nil {
		return plan, err
	}
	result, err := runner.Run(ctx, "restic", args, logPath)
	plan.LogPath = logPath
	plan.Output = result.Output
	return plan, err
}

func (p *Provider) runner() Runner {
	if p.Runner == nil {
		return ExecRunner{}
	}
	return p.Runner
}

func (p *Provider) logPath(operation string) (string, error) {
	if p.LogRoot == "" {
		p.LogRoot = filepath.Join(".", ".drive-agent", "logs", "backup")
	}
	if err := os.MkdirAll(p.LogRoot, 0755); err != nil {
		return "", fmt.Errorf("create backup log dir: %w", err)
	}
	return filepath.Join(p.LogRoot, fmt.Sprintf("%s-%s.log", time.Now().UTC().Format("20060102T150405Z"), operation)), nil
}

func writeLog(path, name string, args []string, output string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	content := fmt.Sprintf("command: %s\n\n%s", backup.RedactSensitive(FormatCommand(name, args)), backup.RedactSensitive(output))
	return os.WriteFile(path, []byte(content), 0644)
}
