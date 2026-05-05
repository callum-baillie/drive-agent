package providers

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/callum-baillie/drive-agent/internal/shell"
)

// PnpmGlobal implements Provider for global pnpm packages.
type PnpmGlobal struct{}

func (p *PnpmGlobal) ID() string            { return "pnpm" }
func (p *PnpmGlobal) Name() string          { return "pnpm (global)" }
func (p *PnpmGlobal) SupportedOS() []string { return []string{"darwin", "linux", "windows"} }
func (p *PnpmGlobal) IsAvailable() bool     { return shell.IsCommandAvailable("pnpm") }
func (p *PnpmGlobal) ManagerPath() string   { return shell.CommandWhich("pnpm") }
func (p *PnpmGlobal) ManagerVersion() string {
	return shell.CommandVersion("pnpm", "--version")
}
func (p *PnpmGlobal) InstallManager(bool) (string, error) {
	return "pnpm is usually installed via Homebrew, Corepack, or npm", fmt.Errorf("install pnpm first")
}
func (p *PnpmGlobal) IsPackageInstalled(packageName string) bool {
	out, err := shell.RunCommand("pnpm", "list", "-g", "--depth=0", packageName)
	return err == nil && strings.Contains(out, packageName)
}
func (p *PnpmGlobal) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("pnpm add -g %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("pnpm", "add", "-g", packageName)
	if err != nil {
		return cmd, fmt.Errorf("pnpm add -g %s: %s", packageName, out)
	}
	return cmd, nil
}
func (p *PnpmGlobal) InstallPackages(packages []string, dryRun bool) (string, error) {
	args := append([]string{"add", "-g"}, packages...)
	cmd := "pnpm " + strings.Join(args, " ")
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("pnpm", args...)
	if err != nil {
		return cmd, fmt.Errorf("pnpm add -g: %s", out)
	}
	return cmd, nil
}

// BunGlobal implements Provider for global Bun packages.
type BunGlobal struct{}

func (b *BunGlobal) ID() string            { return "bun" }
func (b *BunGlobal) Name() string          { return "Bun (global)" }
func (b *BunGlobal) SupportedOS() []string { return []string{"darwin", "linux", "windows"} }
func (b *BunGlobal) IsAvailable() bool     { return shell.IsCommandAvailable("bun") }
func (b *BunGlobal) ManagerPath() string   { return shell.CommandWhich("bun") }
func (b *BunGlobal) ManagerVersion() string {
	return shell.CommandVersion("bun", "--version")
}
func (b *BunGlobal) InstallManager(bool) (string, error) {
	return "bun is usually installed via Homebrew or the official installer", fmt.Errorf("install bun first")
}
func (b *BunGlobal) IsPackageInstalled(packageName string) bool {
	out, err := shell.RunCommand("bun", "pm", "ls", "-g")
	return err == nil && strings.Contains(out, packageName)
}
func (b *BunGlobal) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("bun add --global %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("bun", "add", "--global", packageName)
	if err != nil {
		return cmd, fmt.Errorf("bun add --global %s: %s", packageName, out)
	}
	return cmd, nil
}
func (b *BunGlobal) InstallPackages(packages []string, dryRun bool) (string, error) {
	args := append([]string{"add", "--global"}, packages...)
	cmd := "bun " + strings.Join(args, " ")
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("bun", args...)
	if err != nil {
		return cmd, fmt.Errorf("bun add --global: %s", out)
	}
	return cmd, nil
}

// Pipx implements Provider for Python applications installed with pipx.
type Pipx struct{}

func (p *Pipx) ID() string            { return "pipx" }
func (p *Pipx) Name() string          { return "pipx" }
func (p *Pipx) SupportedOS() []string { return []string{"darwin", "linux", "windows"} }
func (p *Pipx) IsAvailable() bool     { return shell.IsCommandAvailable("pipx") }
func (p *Pipx) ManagerPath() string   { return shell.CommandWhich("pipx") }
func (p *Pipx) ManagerVersion() string {
	return shell.CommandVersion("pipx", "--version")
}
func (p *Pipx) InstallManager(bool) (string, error) {
	return "pipx is usually installed via Homebrew or Python packaging", fmt.Errorf("install pipx first")
}
func (p *Pipx) IsPackageInstalled(packageName string) bool {
	out, err := shell.RunCommand("pipx", "list")
	return err == nil && strings.Contains(out, packageName)
}
func (p *Pipx) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("pipx install %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("pipx", "install", packageName)
	if err != nil {
		return cmd, fmt.Errorf("pipx install %s: %s", packageName, out)
	}
	return cmd, nil
}
func (p *Pipx) InstallPackages(packages []string, dryRun bool) (string, error) {
	for _, pkg := range packages {
		if _, err := p.InstallPackage(pkg, dryRun); err != nil {
			return "", err
		}
	}
	return "pipx install " + strings.Join(packages, " && pipx install "), nil
}

// UvTool implements Provider for uv-managed Python tools.
type UvTool struct{}

func (u *UvTool) ID() string            { return "uv" }
func (u *UvTool) Name() string          { return "uv tool" }
func (u *UvTool) SupportedOS() []string { return []string{"darwin", "linux", "windows"} }
func (u *UvTool) IsAvailable() bool     { return shell.IsCommandAvailable("uv") }
func (u *UvTool) ManagerPath() string   { return shell.CommandWhich("uv") }
func (u *UvTool) ManagerVersion() string {
	return shell.CommandVersion("uv", "--version")
}
func (u *UvTool) InstallManager(bool) (string, error) {
	return "uv is usually installed via Homebrew or Python packaging", fmt.Errorf("install uv first")
}
func (u *UvTool) IsPackageInstalled(packageName string) bool {
	out, err := shell.RunCommand("uv", "tool", "list")
	return err == nil && strings.Contains(out, packageName)
}
func (u *UvTool) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("uv tool install %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("uv", "tool", "install", packageName)
	if err != nil {
		return cmd, fmt.Errorf("uv tool install %s: %s", packageName, out)
	}
	return cmd, nil
}
func (u *UvTool) InstallPackages(packages []string, dryRun bool) (string, error) {
	for _, pkg := range packages {
		if _, err := u.InstallPackage(pkg, dryRun); err != nil {
			return "", err
		}
	}
	return "uv tool install " + strings.Join(packages, " && uv tool install "), nil
}

// CargoInstall implements Provider for cargo-installed Rust binaries.
type CargoInstall struct{}

func (c *CargoInstall) ID() string            { return "cargo" }
func (c *CargoInstall) Name() string          { return "cargo install" }
func (c *CargoInstall) SupportedOS() []string { return []string{"darwin", "linux", "windows"} }
func (c *CargoInstall) IsAvailable() bool     { return shell.IsCommandAvailable("cargo") }
func (c *CargoInstall) ManagerPath() string   { return shell.CommandWhich("cargo") }
func (c *CargoInstall) ManagerVersion() string {
	return shell.CommandVersion("cargo", "--version")
}
func (c *CargoInstall) InstallManager(bool) (string, error) {
	return "cargo is installed with Rust/rustup", fmt.Errorf("install Rust first")
}
func (c *CargoInstall) IsPackageInstalled(packageName string) bool {
	out, err := shell.RunCommand("cargo", "install", "--list")
	return err == nil && strings.Contains(out, packageName)
}
func (c *CargoInstall) InstallPackage(packageName string, dryRun bool) (string, error) {
	cmd := fmt.Sprintf("cargo install %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("cargo", "install", packageName)
	if err != nil {
		return cmd, fmt.Errorf("cargo install %s: %s", packageName, out)
	}
	return cmd, nil
}
func (c *CargoInstall) InstallPackages(packages []string, dryRun bool) (string, error) {
	args := append([]string{"install"}, packages...)
	cmd := "cargo " + strings.Join(args, " ")
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("cargo", args...)
	if err != nil {
		return cmd, fmt.Errorf("cargo install: %s", out)
	}
	return cmd, nil
}

// GoInstall implements Provider for Go binaries installed with go install.
type GoInstall struct{}

func (g *GoInstall) ID() string            { return "go-install" }
func (g *GoInstall) Name() string          { return "go install" }
func (g *GoInstall) SupportedOS() []string { return []string{"darwin", "linux", "windows"} }
func (g *GoInstall) IsAvailable() bool     { return shell.IsCommandAvailable("go") }
func (g *GoInstall) ManagerPath() string   { return shell.CommandWhich("go") }
func (g *GoInstall) ManagerVersion() string {
	return shell.CommandVersion("go", "version")
}
func (g *GoInstall) InstallManager(bool) (string, error) {
	return "go install requires Go to be installed first", fmt.Errorf("install Go first")
}
func (g *GoInstall) IsPackageInstalled(packageName string) bool {
	parts := strings.Split(packageName, "/")
	name := strings.TrimSuffix(parts[len(parts)-1], "@latest")
	_, err := exec.LookPath(name)
	return err == nil
}
func (g *GoInstall) InstallPackage(packageName string, dryRun bool) (string, error) {
	if !strings.Contains(packageName, "@") {
		packageName += "@latest"
	}
	cmd := fmt.Sprintf("go install %s", packageName)
	if dryRun {
		return cmd, nil
	}
	out, err := shell.RunCommand("go", "install", packageName)
	if err != nil {
		return cmd, fmt.Errorf("go install %s: %s", packageName, out)
	}
	return cmd, nil
}
func (g *GoInstall) InstallPackages(packages []string, dryRun bool) (string, error) {
	for _, pkg := range packages {
		if _, err := g.InstallPackage(pkg, dryRun); err != nil {
			return "", err
		}
	}
	return "go install " + strings.Join(packages, " && go install "), nil
}
