package providers

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/callumbaillie/drive-agent/internal/shell"
)

// Homebrew implements Provider for Homebrew formulae.
type Homebrew struct{}

func (h *Homebrew) ID() string            { return "homebrew" }
func (h *Homebrew) Name() string          { return "Homebrew" }
func (h *Homebrew) SupportedOS() []string { return []string{"darwin", "linux"} }

func (h *Homebrew) IsAvailable() bool {
	return shell.IsCommandAvailable("brew")
}

func (h *Homebrew) ManagerPath() string {
	return shell.CommandWhich("brew")
}

func (h *Homebrew) ManagerVersion() string {
	v := shell.CommandVersion("brew")
	if strings.HasPrefix(v, "Homebrew ") {
		return strings.TrimPrefix(v, "Homebrew ")
	}
	return v
}

func (h *Homebrew) InstallManager(dryRun bool) (string, error) {
	cmd := `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
	if dryRun {
		return cmd, nil
	}
	return cmd, fmt.Errorf("Homebrew installation requires manual execution for safety.\nRun: %s", cmd)
}

func (h *Homebrew) IsPackageInstalled(packageName string) bool {
	err := exec.Command("brew", "list", "--formula", packageName).Run()
	return err == nil
}

func (h *Homebrew) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("brew install %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("brew", "install", packageName)
	if err != nil {
		return cmd, fmt.Errorf("brew install %s: %s", packageName, out)
	}
	return cmd, nil
}

func (h *Homebrew) InstallPackages(packages []string, dryRun bool) (string, error) {
	args := append([]string{"install"}, packages...)
	cmd := "brew " + strings.Join(args, " ")
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("brew", args...)
	if err != nil {
		return cmd, fmt.Errorf("brew install: %s", out)
	}
	return cmd, nil
}

// HomebrewCask implements Provider for Homebrew Cask (GUI apps).
type HomebrewCask struct{}

func (h *HomebrewCask) ID() string            { return "homebrew-cask" }
func (h *HomebrewCask) Name() string          { return "Homebrew Cask" }
func (h *HomebrewCask) SupportedOS() []string { return []string{"darwin"} }

func (h *HomebrewCask) IsAvailable() bool {
	return shell.IsCommandAvailable("brew")
}

func (h *HomebrewCask) ManagerPath() string {
	return shell.CommandWhich("brew")
}

func (h *HomebrewCask) ManagerVersion() string {
	hb := &Homebrew{}
	return hb.ManagerVersion()
}

func (h *HomebrewCask) InstallManager(dryRun bool) (string, error) {
	hb := &Homebrew{}
	return hb.InstallManager(dryRun)
}

func (h *HomebrewCask) IsPackageInstalled(packageName string) bool {
	err := exec.Command("brew", "list", "--cask", packageName).Run()
	return err == nil
}

func (h *HomebrewCask) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("brew install --cask %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("brew", "install", "--cask", packageName)
	if err != nil {
		return cmd, fmt.Errorf("brew install --cask %s: %s", packageName, out)
	}
	return cmd, nil
}

func (h *HomebrewCask) InstallPackages(packages []string, dryRun bool) (string, error) {
	args := []string{"install", "--cask"}
	args = append(args, packages...)
	cmd := "brew " + strings.Join(args, " ")
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("brew", args...)
	if err != nil {
		return cmd, fmt.Errorf("brew install --cask: %s", out)
	}
	return cmd, nil
}

// NpmGlobal implements Provider for global npm packages.
type NpmGlobal struct{}

func (n *NpmGlobal) ID() string            { return "npm" }
func (n *NpmGlobal) Name() string          { return "npm (global)" }
func (n *NpmGlobal) SupportedOS() []string { return []string{"darwin", "linux", "windows"} }

func (n *NpmGlobal) IsAvailable() bool {
	return shell.IsCommandAvailable("npm")
}

func (n *NpmGlobal) ManagerPath() string {
	return shell.CommandWhich("npm")
}

func (n *NpmGlobal) ManagerVersion() string {
	return shell.CommandVersion("npm", "--version")
}

func (n *NpmGlobal) InstallManager(dryRun bool) (string, error) {
	return "npm is usually installed via Node.js", fmt.Errorf("install Node.js first to get npm")
}

func (n *NpmGlobal) IsPackageInstalled(packageName string) bool {
	out, err := shell.RunCommand("npm", "list", "-g", "--depth=0", packageName)
	if err != nil {
		return false
	}
	return strings.Contains(out, packageName)
}

func (n *NpmGlobal) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("npm install -g %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("npm", "install", "-g", packageName)
	if err != nil {
		return cmd, fmt.Errorf("npm install -g %s: %s", packageName, out)
	}
	return cmd, nil
}

func (n *NpmGlobal) InstallPackages(packages []string, dryRun bool) (string, error) {
	args := append([]string{"install", "-g"}, packages...)
	cmd := "npm " + strings.Join(args, " ")
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("npm", args...)
	if err != nil {
		return cmd, fmt.Errorf("npm install -g: %s", out)
	}
	return cmd, nil
}
