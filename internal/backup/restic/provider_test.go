package restic

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/callum-baillie/drive-agent/internal/backup"
)

type fakeRunner struct {
	installed bool
	calls     int
	name      string
	args      []string
	output    string
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	if f.installed {
		return "/usr/bin/" + file, nil
	}
	return "", errors.New("missing")
}

func (f *fakeRunner) Run(ctx context.Context, name string, args []string, logPath string) (CommandResult, error) {
	f.calls++
	f.name = name
	f.args = append([]string(nil), args...)
	return CommandResult{Output: f.output}, nil
}

func TestResticCommandConstruction(t *testing.T) {
	args := BackupArgs("/Volumes/Dev Drive", "/Volumes/Backup/restic", "/tmp/excludes.txt", []string{"drive-agent", "host:test"}, true)
	cmd := FormatCommand("restic", args)
	for _, want := range []string{"backup", "'/Volumes/Dev Drive'", "--exclude-file", "--tag", "--dry-run"} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("command %q missing %q", cmd, want)
		}
	}
}

func TestProviderDryRunDoesNotCallRunner(t *testing.T) {
	runner := &fakeRunner{installed: false}
	provider := &Provider{Runner: runner, LogRoot: t.TempDir()}
	plan, err := provider.Backup(context.Background(), backup.BackupRequest{
		DriveRoot:   "/Volumes/DevDrive",
		DriveLabel:  "DevDrive",
		HostLabel:   "host",
		Repo:        backup.Repository{Repository: "/Volumes/Backup/restic"},
		ExcludeFile: "/Volumes/DevDrive/.drive-agent/state/backup/restic-excludes.txt",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("Backup dry-run: %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("runner called %d times", runner.calls)
	}
	if !strings.Contains(plan.Command, "--dry-run") {
		t.Fatalf("plan command missing dry-run: %s", plan.Command)
	}
}

func TestProviderMissingRestic(t *testing.T) {
	runner := &fakeRunner{installed: false}
	provider := &Provider{Runner: runner, LogRoot: t.TempDir()}
	_, err := provider.Init(context.Background(), backup.Repository{Repository: "/Volumes/Backup/restic"}, false)
	if err == nil || !strings.Contains(err.Error(), "restic is not installed") {
		t.Fatalf("err = %v, want missing restic", err)
	}
}

func TestProviderSnapshotsParsesJSON(t *testing.T) {
	runner := &fakeRunner{installed: true, output: `[{"time":"2026-05-04T12:00:00Z","id":"abcdef123456","short_id":"abcdef12","hostname":"host","paths":["/Volumes/DevDrive"],"tags":["drive-agent"]}]`}
	provider := &Provider{Runner: runner, LogRoot: t.TempDir()}
	snapshots, err := provider.Snapshots(context.Background(), backup.Repository{Repository: "/Volumes/Backup/restic"})
	if err != nil {
		t.Fatalf("Snapshots: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].ID != "abcdef123456" {
		t.Fatalf("snapshots = %+v", snapshots)
	}
}

func TestParseSnapshotsJSONEmpty(t *testing.T) {
	snapshots, err := ParseSnapshotsJSON([]byte(" \n"))
	if err != nil {
		t.Fatalf("ParseSnapshotsJSON: %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("snapshots = %+v", snapshots)
	}
}

func TestWriteLogRedactsPasswordValues(t *testing.T) {
	t.Setenv("RESTIC_PASSWORD", "super-secret")
	logPath := filepath.Join(t.TempDir(), "backup.log")
	if err := writeLog(logPath, "restic", []string{"backup", "/Volumes/DevDrive"}, "output contains super-secret"); err != nil {
		t.Fatalf("writeLog: %v", err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "super-secret") {
		t.Fatalf("log was not redacted: %s", data)
	}
}
