package restic

import (
	"fmt"
	"strings"
)

func InitArgs(repo string) []string {
	return []string{"-r", repo, "init"}
}

func BackupArgs(source, repo, excludeFile string, tags []string, dryRun bool) []string {
	args := []string{"-r", repo, "backup", source}
	if excludeFile != "" {
		args = append(args, "--exclude-file", excludeFile)
	}
	for _, tag := range tags {
		if tag != "" {
			args = append(args, "--tag", tag)
		}
	}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return args
}

func SnapshotsArgs(repo string, jsonOutput bool) []string {
	args := []string{"-r", repo, "snapshots"}
	if jsonOutput {
		args = append(args, "--json")
	}
	return args
}

func CheckArgs(repo string) []string {
	return []string{"-r", repo, "check"}
}

func RestoreArgs(repo, snapshot, target string, dryRun bool) []string {
	args := []string{"-r", repo, "restore", snapshot, "--target", target}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return args
}

func FormatCommand(name string, args []string) string {
	parts := append([]string{name}, args...)
	for i, part := range parts {
		parts[i] = shellQuote(part)
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.ContainsAny(s, " \t\n'\"$`\\") {
		return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
	}
	return s
}

func ValidateSnapshot(snapshot string) error {
	if strings.TrimSpace(snapshot) == "" {
		return fmt.Errorf("snapshot is required")
	}
	return nil
}
