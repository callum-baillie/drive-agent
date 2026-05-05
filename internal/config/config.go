package config

import "time"

const (
	// AgentDir is the hidden directory name for drive-agent metadata.
	AgentDir = ".drive-agent"

	// MarkerFile is the file that marks a directory as a drive-agent root.
	MarkerFile = "DRIVE_AGENT_ROOT"

	// VersionFile stores the current version.
	VersionFile = "VERSION"

	// DatabaseFile is the SQLite database filename.
	DatabaseFile = "drive-agent.sqlite"

	// ProjectManifest is the filename for project manifests.
	ProjectManifest = ".drive-project.toml"

	// DriveConfigFile is the drive config filename.
	DriveConfigFile = "drive.toml"
)

var (
	// Version is the current drive-agent version. GoReleaser overrides this with ldflags.
	Version = "0.1.0"

	// RepoOwner is the GitHub organization or user hosting the releases
	RepoOwner = "callum-baillie"

	// RepoName is the GitHub repository name hosting the releases
	RepoName = "drive-agent"
)

// DriveLayout defines the top-level directories created during init.
var DriveLayout = []string{
	"Orgs",
	"DevData",
	"Caches",
	"BuildArtifacts",
	"Tooling",
	"Downloads",
	"Inbox",
	"Scratch",
	"Trash",
}

// AgentLayout defines subdirectories under .drive-agent.
var AgentLayout = []string{
	"bin",
	"config",
	"config/host-profiles",
	"db",
	"logs",
	"logs/host-setup",
	"logs/cleanup",
	"logs/backup",
	"logs/git",
	"state",
	"state/hosts",
	"backups",
	"locks",
	"templates",
	"catalog",
	"releases",
}

// OrgLayout defines subdirectories created for each organization.
var OrgLayout = []string{
	"projects",
	"archives",
	"notes",
}

// CleanupTargets are default directories/files to scan for cleanup.
var CleanupTargets = []string{
	"node_modules",
	".next",
	".nuxt",
	".output",
	"dist",
	"build",
	".turbo",
	".vercel",
	".cache",
	"coverage",
	"playwright-report",
	"test-results",
	".expo",
	".expo-shared",
	"android/.gradle",
	"ios/Pods",
	"vendor",
	"target",
	".DS_Store",
}

// DriveConfig represents the drive.toml configuration file.
type DriveConfig struct {
	DriveID       string `toml:"drive_id"`
	Name          string `toml:"name"`
	SchemaVersion int    `toml:"schema_version"`
	DefaultOrg    string `toml:"default_org"`
	CreatedAt     string `toml:"created_at"`
	UpdatedAt     string `toml:"updated_at"`
}

// ProjectManifestData represents the .drive-project.toml file.
type ProjectManifestData struct {
	ID             string              `toml:"id"`
	Name           string              `toml:"name"`
	Slug           string              `toml:"slug"`
	Org            string              `toml:"org"`
	Type           string              `toml:"type"`
	PackageManager string              `toml:"package_manager"`
	Tags           []string            `toml:"tags"`
	GitRemote      string              `toml:"git_remote"`
	CreatedAt      string              `toml:"created_at"`
	Backup         ProjectBackupConfig `toml:"backup,omitempty"`
}

// ProjectBackupConfig contains project-local backup behavior.
type ProjectBackupConfig struct {
	Excludes []string `toml:"excludes,omitempty"`
}

// HostInfo represents detected host information.
type HostInfo struct {
	HostID   string `json:"hostId"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Shell    string `json:"shell"`
}

// HostProfile represents a host setup profile JSON file.
type HostProfile struct {
	SchemaVersion   int                    `json:"schemaVersion"`
	ProfileName     string                 `json:"profileName"`
	Description     string                 `json:"description"`
	Target          ProfileTarget          `json:"target,omitempty"`
	PackageManagers ProfilePackageManagers `json:"packageManagers"`
	Categories      []string               `json:"categories"`
	Packages        ProfilePackages        `json:"packages"`
	Shell           ProfileShell           `json:"shell"`
	Caches          ProfileCaches          `json:"caches"`
	Docker          ProfileDocker          `json:"docker"`
	Safety          ProfileSafety          `json:"safety"`
}

// ProfileTarget specifies OS/arch constraints for a profile.
type ProfileTarget struct {
	OS   string `json:"os,omitempty"`
	Arch string `json:"arch,omitempty"`
}

// ProfilePackageManagers defines package manager preferences.
type ProfilePackageManagers struct {
	InstallMissing bool     `json:"installMissing"`
	Preferred      []string `json:"preferred"`
}

// ProfilePackages defines explicit package includes/excludes.
type ProfilePackages struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// ProfileShell defines shell configuration preferences.
type ProfileShell struct {
	InstallAliases           bool `json:"installAliases"`
	AddLocalBinToPath        bool `json:"addLocalBinToPath"`
	ConfigureDriveAgentAlias bool `json:"configureDriveAgentAlias"`
}

// ProfileCaches defines cache configuration preferences.
type ProfileCaches struct {
	Mode                     string `json:"mode,omitempty"`
	AllowExternalDriveCaches bool   `json:"allowExternalDriveCaches,omitempty"`
	AllowDisableCaches       bool   `json:"allowDisableCaches,omitempty"`
	ExternalDriveRoot        string `json:"externalDriveRoot,omitempty"`
	NpmCachePath             string `json:"npmCachePath,omitempty"`
	PnpmStorePath            string `json:"pnpmStorePath,omitempty"`
	BunCachePath             string `json:"bunCachePath,omitempty"`
	HomebrewCachePath        string `json:"homebrewCachePath,omitempty"`
	ConfigurePnpmStore       bool   `json:"configurePnpmStore"`
	ConfigureNpmCache        bool   `json:"configureNpmCache"`
	ConfigureBunCache        bool   `json:"configureBunCache"`
}

// ProfileDocker defines container storage preferences.
type ProfileDocker struct {
	Mode                           string `json:"mode,omitempty"`
	AllowExternalDriveStorage      bool   `json:"allowExternalDriveStorage,omitempty"`
	ExternalDataRoot               string `json:"externalDataRoot,omitempty"`
	ExternalBuildCacheRoot         string `json:"externalBuildCacheRoot,omitempty"`
	DoNotModifyWithoutConfirmation bool   `json:"doNotModifyWithoutConfirmation,omitempty"`
}

// ProfileSafety defines safety preferences.
type ProfileSafety struct {
	DryRunFirst           bool `json:"dryRunFirst"`
	DryRun                bool `json:"dryRun"`
	RequireConfirmation   bool `json:"requireConfirmation"`
	AllowSudo             bool `json:"allowSudo"`
	AllowNativeInstallers bool `json:"allowNativeInstallers"`
	AllowCurlPipeShell    bool `json:"allowCurlPipeShell"`
}

// NowISO returns the current time in ISO 8601 format.
func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
