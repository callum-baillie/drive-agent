package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/packages/catalog"
	"github.com/callum-baillie/drive-agent/internal/packages/providers"
	"github.com/callum-baillie/drive-agent/internal/shell"
)

const (
	cacheModePrompt      = "prompt"
	cacheModeHostLocal   = "host-local"
	cacheModeExternal    = "external-drive"
	cacheModeDisabled    = "disabled"
	dockerModePrompt     = "prompt"
	dockerModeDefault    = "default"
	dockerModeBindMounts = "bind-mounts"
	dockerModeDaemon     = "daemon"
)

type packageAction struct {
	ID          string
	Name        string
	ManagerID   string
	PackageName string
	Command     string
	Installed   bool
	SkipReason  string
}

type packagePlan struct {
	Managers []managerPlan
	Actions  []packageAction
}

type managerPlan struct {
	ID        string
	Name      string
	Available bool
	Path      string
	Version   string
}

type setupAction struct {
	Title       string
	Command     string
	Run         []string
	Path        string
	Current     string
	Planned     string
	RequiresRun bool
}

func normalizeCacheMode(mode string) (string, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "", cacheModePrompt:
		return cacheModePrompt, nil
	case cacheModeHostLocal, "local":
		return cacheModeHostLocal, nil
	case cacheModeExternal, "external":
		return cacheModeExternal, nil
	case cacheModeDisabled, "disabled/no-change", "no-change", "none":
		return cacheModeDisabled, nil
	default:
		return "", fmt.Errorf("unknown cache mode %q", mode)
	}
}

func normalizeDockerMode(mode string) (string, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "", dockerModePrompt:
		return dockerModePrompt, nil
	case dockerModeDefault, "host-local":
		return dockerModeDefault, nil
	case dockerModeBindMounts, "bind-mount", "external-bind-mounts":
		return dockerModeBindMounts, nil
	case dockerModeDaemon, "daemon-data-root":
		return dockerModeDaemon, nil
	default:
		return "", fmt.Errorf("unknown Docker mode %q", mode)
	}
}

func defaultProfileCachePaths(profile *config.HostProfile, driveRoot string) config.ProfileCaches {
	caches := profile.Caches
	if caches.ExternalDriveRoot == "" {
		caches.ExternalDriveRoot = driveRoot
	}
	if caches.NpmCachePath == "" {
		caches.NpmCachePath = filepath.Join(caches.ExternalDriveRoot, "Caches", "npm")
	}
	if caches.PnpmStorePath == "" {
		caches.PnpmStorePath = filepath.Join(caches.ExternalDriveRoot, "Caches", "pnpm")
	}
	if caches.BunCachePath == "" {
		caches.BunCachePath = filepath.Join(caches.ExternalDriveRoot, "Caches", "bun")
	}
	if caches.HomebrewCachePath == "" {
		caches.HomebrewCachePath = filepath.Join(caches.ExternalDriveRoot, "Caches", "homebrew")
	}
	return caches
}

func defaultProfileDockerPaths(profile *config.HostProfile, driveRoot string) config.ProfileDocker {
	docker := profile.Docker
	if docker.ExternalDataRoot == "" {
		docker.ExternalDataRoot = filepath.Join(driveRoot, "DevData", "containers")
	}
	if docker.ExternalBuildCacheRoot == "" {
		docker.ExternalBuildCacheRoot = filepath.Join(driveRoot, "DevData", "docker-build-cache")
	}
	return docker
}

func buildCachePlan(profile *config.HostProfile, driveRoot, mode string) ([]setupAction, shell.ShellBlockOptions, error) {
	mode, err := normalizeCacheMode(mode)
	if err != nil {
		return nil, shell.ShellBlockOptions{}, err
	}
	caches := defaultProfileCachePaths(profile, driveRoot)
	if mode == cacheModePrompt {
		return []setupAction{
			{Title: "Cache mode", Planned: "prompt for host-local, external-drive, or disabled/no-change"},
			{Title: "External npm cache option", Command: "npm config set cache " + shell.ShellQuote(caches.NpmCachePath), Current: commandOutput("npm", "config", "get", "cache"), Planned: caches.NpmCachePath},
			{Title: "External pnpm store option", Command: "pnpm config set store-dir " + shell.ShellQuote(caches.PnpmStorePath), Current: commandOutput("pnpm", "config", "get", "store-dir"), Planned: caches.PnpmStorePath},
			{Title: "External Bun cache option", Planned: "BUN_INSTALL_CACHE_DIR=" + caches.BunCachePath},
			{Title: "External Homebrew cache option", Current: os.Getenv("HOMEBREW_CACHE"), Planned: "HOMEBREW_CACHE=" + caches.HomebrewCachePath},
		}, shell.ShellBlockOptions{}, nil
	}
	if mode == cacheModeHostLocal || mode == cacheModeDisabled {
		return []setupAction{{
			Title:   "Cache configuration",
			Planned: "leave package-manager cache configuration unchanged",
		}}, shell.ShellBlockOptions{}, nil
	}

	actions := []setupAction{
		{Title: "Create npm cache directory", Path: caches.NpmCachePath, Command: "mkdir -p " + shell.ShellQuote(caches.NpmCachePath), RequiresRun: true},
		{Title: "Create pnpm store directory", Path: caches.PnpmStorePath, Command: "mkdir -p " + shell.ShellQuote(caches.PnpmStorePath), RequiresRun: true},
		{Title: "Create Bun cache directory", Path: caches.BunCachePath, Command: "mkdir -p " + shell.ShellQuote(caches.BunCachePath), RequiresRun: true},
		{Title: "Create Homebrew cache directory", Path: caches.HomebrewCachePath, Command: "mkdir -p " + shell.ShellQuote(caches.HomebrewCachePath), RequiresRun: true},
		{Title: "Configure npm cache", Command: "npm config set cache " + shell.ShellQuote(caches.NpmCachePath), Run: []string{"npm", "config", "set", "cache", caches.NpmCachePath}, Current: commandOutput("npm", "config", "get", "cache"), Planned: caches.NpmCachePath, RequiresRun: shell.IsCommandAvailable("npm")},
		{Title: "Configure pnpm store", Command: "pnpm config set store-dir " + shell.ShellQuote(caches.PnpmStorePath), Run: []string{"pnpm", "config", "set", "store-dir", caches.PnpmStorePath}, Current: commandOutput("pnpm", "config", "get", "store-dir"), Planned: caches.PnpmStorePath, RequiresRun: shell.IsCommandAvailable("pnpm")},
		{Title: "Export Bun cache", Planned: "BUN_INSTALL_CACHE_DIR=" + caches.BunCachePath},
		{Title: "Export Homebrew cache", Current: os.Getenv("HOMEBREW_CACHE"), Planned: "HOMEBREW_CACHE=" + caches.HomebrewCachePath},
	}
	return actions, shell.ShellBlockOptions{
		NpmCachePath:      caches.NpmCachePath,
		BunCachePath:      caches.BunCachePath,
		HomebrewCachePath: caches.HomebrewCachePath,
	}, nil
}

func buildDockerPlan(profile *config.HostProfile, driveRoot, mode string) ([]setupAction, shell.ShellBlockOptions, error) {
	mode, err := normalizeDockerMode(mode)
	if err != nil {
		return nil, shell.ShellBlockOptions{}, err
	}
	docker := defaultProfileDockerPaths(profile, driveRoot)
	if mode == dockerModePrompt {
		return []setupAction{
			{Title: "Docker storage mode", Planned: "prompt for default storage, external bind-mount roots, or daemon data-root guidance"},
			{Title: "External container root option", Command: "mkdir -p " + shell.ShellQuote(docker.ExternalDataRoot), Planned: docker.ExternalDataRoot},
			{Title: "External build cache option", Command: "mkdir -p " + shell.ShellQuote(docker.ExternalBuildCacheRoot), Planned: docker.ExternalBuildCacheRoot},
			{Title: "Container env option", Planned: "DRIVE_AGENT_CONTAINER_DATA=" + docker.ExternalDataRoot},
			{Title: "Build cache env option", Planned: "DRIVE_AGENT_DOCKER_BUILD_CACHE=" + docker.ExternalBuildCacheRoot},
		}, shell.ShellBlockOptions{}, nil
	}
	if mode == dockerModeDefault {
		return []setupAction{{
			Title:   "Docker storage",
			Planned: "leave Docker storage unchanged",
		}}, shell.ShellBlockOptions{}, nil
	}
	if mode == dockerModeDaemon {
		return []setupAction{
			{Title: "Docker daemon relocation", Planned: "manual-only; Drive Agent will not edit Docker Desktop/OrbStack daemon storage automatically"},
			{Title: "Recommended fallback", Planned: "use external bind mounts at " + docker.ExternalDataRoot},
		}, shell.ShellBlockOptions{}, nil
	}

	actions := []setupAction{
		{Title: "Create container data root", Path: docker.ExternalDataRoot, Command: "mkdir -p " + shell.ShellQuote(docker.ExternalDataRoot), RequiresRun: true},
		{Title: "Create Docker build cache root", Path: docker.ExternalBuildCacheRoot, Command: "mkdir -p " + shell.ShellQuote(docker.ExternalBuildCacheRoot), RequiresRun: true},
		{Title: "Export container data root", Planned: "DRIVE_AGENT_CONTAINER_DATA=" + docker.ExternalDataRoot},
		{Title: "Export Docker build cache root", Planned: "DRIVE_AGENT_DOCKER_BUILD_CACHE=" + docker.ExternalBuildCacheRoot},
	}
	return actions, shell.ShellBlockOptions{
		ContainerDataPath: docker.ExternalDataRoot,
		DockerCachePath:   docker.ExternalBuildCacheRoot,
	}, nil
}

func buildPackagePlan(profile *config.HostProfile, cat *catalog.Catalog, registry *providers.Registry) packagePlan {
	plan := packagePlan{}
	for _, mgrID := range profile.PackageManagers.Preferred {
		if mgr, ok := registry.Get(mgrID); ok {
			plan.Managers = append(plan.Managers, managerPlan{
				ID:        mgr.ID(),
				Name:      mgr.Name(),
				Available: mgr.IsAvailable(),
				Path:      mgr.ManagerPath(),
				Version:   mgr.ManagerVersion(),
			})
		}
	}

	excluded := make(map[string]bool)
	for _, id := range profile.Packages.Exclude {
		excluded[id] = true
	}
	for _, id := range profile.Packages.Include {
		if excluded[id] {
			continue
		}
		pkg := cat.GetPackage(id)
		if pkg == nil {
			plan.Actions = append(plan.Actions, packageAction{ID: id, SkipReason: "unknown package"})
			continue
		}
		action := packageAction{ID: pkg.ID, Name: pkg.Name}
		if isPackageCheckInstalled(pkg) {
			action.Installed = true
			plan.Actions = append(plan.Actions, action)
			continue
		}
		if pkg.RequiresApproval {
			action.SkipReason = "requires explicit approval"
			plan.Actions = append(plan.Actions, action)
			continue
		}
		mgr, managerID := chooseProvider(pkg, profile.PackageManagers.Preferred, registry)
		if mgr == nil {
			action.SkipReason = "no available provider"
			plan.Actions = append(plan.Actions, action)
			continue
		}
		action.ManagerID = managerID
		action.PackageName = pkg.GetInstallName(managerID)
		action.Command, _ = mgr.InstallPackage(action.PackageName, true)
		plan.Actions = append(plan.Actions, action)
	}
	return plan
}

func chooseProvider(pkg *catalog.Package, preferred []string, registry *providers.Registry) (providers.Provider, string) {
	for _, mgrID := range preferred {
		if _, ok := pkg.Install[mgrID]; !ok {
			continue
		}
		mgr, ok := registry.Get(mgrID)
		if ok && mgr.IsAvailable() {
			return mgr, mgrID
		}
	}
	for _, mgrID := range pkg.InstallPreference {
		mgr, ok := registry.Get(mgrID)
		if ok && mgr.IsAvailable() {
			return mgr, mgrID
		}
	}
	return nil, ""
}

func isPackageCheckInstalled(pkg *catalog.Package) bool {
	if pkg.Check == nil || strings.TrimSpace(pkg.Check.Command) == "" {
		return false
	}
	return exec.Command("sh", "-c", pkg.Check.Command).Run() == nil
}

func commandOutput(name string, args ...string) string {
	if !shell.IsCommandAvailable(name) {
		return "not installed"
	}
	out, err := shell.RunCommand(name, args...)
	if err != nil {
		return strings.TrimSpace(out)
	}
	return out
}

func applySetupActions(actions []setupAction) error {
	for _, action := range actions {
		if !action.RequiresRun || action.Command == "" {
			continue
		}
		if strings.HasPrefix(action.Command, "mkdir -p ") && action.Path != "" {
			if err := os.MkdirAll(action.Path, 0755); err != nil {
				return fmt.Errorf("%s: %w", action.Title, err)
			}
			continue
		}
		parts := action.Run
		if len(parts) == 0 {
			parts = strings.Fields(action.Command)
		}
		if len(parts) == 0 {
			continue
		}
		out, err := shell.RunCommand(parts[0], parts[1:]...)
		if err != nil {
			return fmt.Errorf("%s: %s", action.Title, out)
		}
	}
	return nil
}

func mergeShellBlockOptions(a, b shell.ShellBlockOptions) shell.ShellBlockOptions {
	if b.NpmCachePath != "" {
		a.NpmCachePath = b.NpmCachePath
	}
	if b.BunCachePath != "" {
		a.BunCachePath = b.BunCachePath
	}
	if b.HomebrewCachePath != "" {
		a.HomebrewCachePath = b.HomebrewCachePath
	}
	if b.ContainerDataPath != "" {
		a.ContainerDataPath = b.ContainerDataPath
	}
	if b.DockerCachePath != "" {
		a.DockerCachePath = b.DockerCachePath
	}
	return a
}
