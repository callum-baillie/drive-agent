package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectOS returns the current operating system.
func DetectOS() string {
	return runtime.GOOS
}

// DetectArch returns the current CPU architecture.
func DetectArch() string {
	return runtime.GOARCH
}

// DetectShell returns the current user's default shell.
func DetectShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return filepath.Base(shell)
	}
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "unknown"
}

// DetectHostname returns the system hostname.
func DetectHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// HostID returns a slugified hostname for use as an identifier.
func HostID() string {
	hostname := DetectHostname()
	hostname = strings.ToLower(hostname)
	hostname = strings.ReplaceAll(hostname, " ", "-")
	hostname = strings.ReplaceAll(hostname, ".", "-")
	hostname = strings.TrimSuffix(hostname, "-local")
	return hostname
}

// IsCommandAvailable checks if a command exists in PATH.
func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// CommandVersion runs a command with --version and returns stdout.
func CommandVersion(name string, args ...string) string {
	if len(args) == 0 {
		args = []string{"--version"}
	}
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CommandWhich returns the path of a command.
func CommandWhich(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// RunCommand runs a command and returns its combined output.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// RunCommandInDir runs a command in a specific directory.
func RunCommandInDir(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// IsOnPath checks if a specific path is already in the user's PATH.
func IsOnPath(targetPath string) bool {
	pathEnv := os.Getenv("PATH")
	paths := filepath.SplitList(pathEnv)
	for _, p := range paths {
		if p == targetPath {
			return true
		}
	}
	return false
}

// ShellConfigFile returns the config file path for the detected shell.
func ShellConfigFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	shell := DetectShell()
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		// Check for .bash_profile first (macOS preference), then .bashrc
		if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
			return filepath.Join(home, ".bash_profile")
		}
		return filepath.Join(home, ".bashrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return ""
	}
}

const shellBlockStart = "# >>> drive-agent >>>"
const shellBlockEnd = "# <<< drive-agent <<<"

// ShellBlock generates a marked block for shell config.
func ShellBlock(content string) string {
	return fmt.Sprintf("%s\n%s\n%s", shellBlockStart, content, shellBlockEnd)
}

// ShellBlockOptions controls optional drive-agent shell exports.
type ShellBlockOptions struct {
	NpmCachePath      string
	BunCachePath      string
	HomebrewCachePath string
	ContainerDataPath string
	DockerCachePath   string
}

// ShellQuote returns a POSIX-shell-safe single-quoted value.
func ShellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// ShellBlockContent returns the standard drive-agent shell configuration.
func ShellBlockContent(driveRoot string) string {
	return ShellBlockContentWithOptions(driveRoot, ShellBlockOptions{})
}

// ShellBlockContentWithOptions returns the standard drive-agent shell
// configuration plus optional cache/storage exports.
func ShellBlockContentWithOptions(driveRoot string, options ShellBlockOptions) string {
	binPath := filepath.Join(driveRoot, ".drive-agent", "bin")
	lines := []string{
		fmt.Sprintf(`export PATH=%s:"$PATH"`, ShellQuote(binPath)),
		`alias da="drive-agent"`,
		`alias drive="drive-agent"`,
		``,
		`# drive-agent shell helpers`,
		`da-cd() { cd "$(drive-agent project path "$1")" ; }`,
		`da-open() { drive-agent project open "$1" ; }`,
	}
	if options.NpmCachePath != "" || options.BunCachePath != "" || options.HomebrewCachePath != "" || options.ContainerDataPath != "" || options.DockerCachePath != "" {
		lines = append(lines, "", "# drive-agent portable cache and container roots")
	}
	if options.NpmCachePath != "" {
		lines = append(lines, fmt.Sprintf("export npm_config_cache=%s", ShellQuote(options.NpmCachePath)))
	}
	if options.BunCachePath != "" {
		lines = append(lines, fmt.Sprintf("export BUN_INSTALL_CACHE_DIR=%s", ShellQuote(options.BunCachePath)))
	}
	if options.HomebrewCachePath != "" {
		lines = append(lines, fmt.Sprintf("export HOMEBREW_CACHE=%s", ShellQuote(options.HomebrewCachePath)))
	}
	if options.ContainerDataPath != "" {
		lines = append(lines, fmt.Sprintf("export DRIVE_AGENT_CONTAINER_DATA=%s", ShellQuote(options.ContainerDataPath)))
	}
	if options.DockerCachePath != "" {
		lines = append(lines, fmt.Sprintf("export DRIVE_AGENT_DOCKER_BUILD_CACHE=%s", ShellQuote(options.DockerCachePath)))
	}
	return strings.Join(lines, "\n")
}

// ShellBlockAlreadyInstalled checks whether the marked block is already present
// in the given shell config file. Used to make host setup idempotent.
func ShellBlockAlreadyInstalled(configPath string) (bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(data), shellBlockStart), nil
}

// AppendShellBlock appends the drive-agent marked block to the shell config.
// It creates a timestamped backup first and refuses if the block already exists.
func AppendShellBlock(configPath, driveRoot string) error {
	// Guard: refuse if block already installed
	installed, err := ShellBlockAlreadyInstalled(configPath)
	if err != nil {
		return fmt.Errorf("read shell config: %w", err)
	}
	if installed {
		return ErrShellBlockAlreadyPresent
	}

	// Create timestamped backup
	backupPath, err := backupFile(configPath)
	if err != nil {
		return fmt.Errorf("backup shell config: %w", err)
	}
	_ = backupPath // caller can log this

	// Append block
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open shell config: %w", err)
	}
	defer f.Close()

	content := ShellBlockContent(driveRoot)
	block := ShellBlock(content)
	_, err = fmt.Fprintln(f, "\n"+block)
	return err
}

// BackupPathFor returns what the backup path will be for a given file.
func BackupPathFor(configPath string) string {
	return configPath + ".drive-agent-backup"
}

// backupFile creates a timestamped backup of a file.
// Returns the path of the backup file created.
func backupFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Nothing to back up yet
			return "", nil
		}
		return "", err
	}
	// Use a date-based name, not timestamp, so repeated runs on the same day
	// overwrite the same backup rather than creating many.
	backupPath := BackupPathFor(path)
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", err
	}
	return backupPath, nil
}

// ErrShellBlockAlreadyPresent is returned when the block is already in the config.
var ErrShellBlockAlreadyPresent = fmt.Errorf("drive-agent shell block is already present in config")
