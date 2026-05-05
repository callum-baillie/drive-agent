package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
)

var defaultExcludes = []string{
	".env",
	".env.*",
	"**/.env",
	"**/.env.*",
	"*.tfstate",
	"*.tfstate.backup",
	"**/*.tfstate",
	"**/*.tfstate.backup",
	".terraform",
	"**/.terraform",
	"tmp",
	"tmp/**",
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
	".Spotlight-V100",
	".TemporaryItems",
	".Trashes",
	".fseventsd",
	".drive-agent/releases/tmp",
}

func DefaultExcludes() []string {
	out := append([]string(nil), defaultExcludes...)
	sort.Strings(out)
	return out
}

func AddExclude(excludes []string, pattern string) ([]string, bool, error) {
	pattern, err := normalizeExclude(pattern)
	if err != nil {
		return excludes, false, err
	}
	for _, existing := range excludes {
		if existing == pattern {
			return sortedUnique(excludes), false, nil
		}
	}
	excludes = append(excludes, pattern)
	return sortedUnique(excludes), true, nil
}

func RemoveExclude(excludes []string, pattern string) ([]string, bool, error) {
	pattern, err := normalizeExclude(pattern)
	if err != nil {
		return excludes, false, err
	}
	var out []string
	removed := false
	for _, existing := range excludes {
		if existing == pattern {
			removed = true
			continue
		}
		out = append(out, existing)
	}
	return sortedUnique(out), removed, nil
}

func WriteExcludeFile(driveRoot string, excludes []string) (string, error) {
	dir := filepath.Join(filesystem.AgentPath(driveRoot), "state", "backup")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create backup state dir: %w", err)
	}
	path := filepath.Join(dir, "restic-excludes.txt")
	data := strings.Join(sortedUnique(excludes), "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return "", fmt.Errorf("write exclude file: %w", err)
	}
	return path, nil
}

type ProjectExcludeSet struct {
	OrgSlug     string
	ProjectSlug string
	ProjectPath string
	Patterns    []string
}

func LoadProjectManifestExcludes(projectPath string) ([]string, error) {
	manifestPath := filepath.Join(projectPath, config.ProjectManifest)
	var manifest config.ProjectManifestData
	if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
		return nil, fmt.Errorf("read project manifest excludes: %w", err)
	}
	return sortedUnique(manifest.Backup.Excludes), nil
}

func SaveProjectManifestExcludes(projectPath string, excludes []string) error {
	manifestPath := filepath.Join(projectPath, config.ProjectManifest)
	var manifest config.ProjectManifestData
	if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
		return fmt.Errorf("read project manifest: %w", err)
	}

	var normalized []string
	for _, pattern := range excludes {
		clean, err := normalizeProjectExclude(pattern)
		if err != nil {
			return err
		}
		normalized = append(normalized, clean)
	}
	manifest.Backup.Excludes = sortedUnique(normalized)

	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("open project manifest for write: %w", err)
	}
	defer file.Close()
	if err := toml.NewEncoder(file).Encode(manifest); err != nil {
		return fmt.Errorf("write project manifest: %w", err)
	}
	return nil
}

func AddProjectExclude(projectPath string, pattern string) (bool, error) {
	excludes, err := LoadProjectManifestExcludes(projectPath)
	if err != nil {
		return false, err
	}
	next, changed, err := addProjectExclude(excludes, pattern)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	return true, SaveProjectManifestExcludes(projectPath, next)
}

func RemoveProjectExclude(projectPath string, pattern string) (bool, error) {
	excludes, err := LoadProjectManifestExcludes(projectPath)
	if err != nil {
		return false, err
	}
	pattern, err = normalizeProjectExclude(pattern)
	if err != nil {
		return false, err
	}
	var next []string
	removed := false
	for _, existing := range excludes {
		if existing == pattern {
			removed = true
			continue
		}
		next = append(next, existing)
	}
	if !removed {
		return false, nil
	}
	return true, SaveProjectManifestExcludes(projectPath, next)
}

func MergeProjectExcludes(base []string, projects []ProjectExcludeSet) ([]string, error) {
	out := append([]string(nil), base...)
	for _, project := range projects {
		for _, pattern := range project.Patterns {
			scoped, err := ScopeProjectExclude(project.ProjectPath, pattern)
			if err != nil {
				return nil, err
			}
			out = append(out, scoped)
		}
	}
	return sortedUnique(out), nil
}

func ScopeProjectExclude(projectPath, pattern string) (string, error) {
	pattern, err := normalizeProjectExclude(pattern)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(projectPath) == "" {
		return "", fmt.Errorf("project path is required")
	}
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("resolve project path: %w", err)
	}
	return filepath.ToSlash(filepath.Join(absProjectPath, filepath.FromSlash(pattern))), nil
}

func addProjectExclude(excludes []string, pattern string) ([]string, bool, error) {
	pattern, err := normalizeProjectExclude(pattern)
	if err != nil {
		return excludes, false, err
	}
	for _, existing := range excludes {
		if existing == pattern {
			return sortedUnique(excludes), false, nil
		}
	}
	excludes = append(excludes, pattern)
	return sortedUnique(excludes), true, nil
}

func MergeExcludes(base, extra []string) ([]string, error) {
	out := append([]string(nil), base...)
	for _, pattern := range extra {
		var err error
		out, _, err = AddExclude(out, pattern)
		if err != nil {
			return nil, err
		}
	}
	return sortedUnique(out), nil
}

func normalizeExclude(pattern string) (string, error) {
	pattern = strings.TrimSpace(pattern)
	pattern = strings.TrimPrefix(pattern, "./")
	if pattern == "" {
		return "", fmt.Errorf("exclude pattern is required")
	}
	clean := filepath.Clean(pattern)
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("exclude pattern must be relative to the drive root")
	}
	clean = filepath.ToSlash(clean)
	if clean == config.AgentDir || strings.HasPrefix(clean, config.AgentDir+"/") && clean != ".drive-agent/releases/tmp" {
		return "", fmt.Errorf("refusing to exclude drive-agent metadata path %q", clean)
	}
	return clean, nil
}

func normalizeProjectExclude(pattern string) (string, error) {
	pattern = strings.TrimSpace(pattern)
	pattern = strings.TrimPrefix(pattern, "./")
	if pattern == "" {
		return "", fmt.Errorf("exclude pattern is required")
	}
	if strings.Contains(pattern, "\x00") {
		return "", fmt.Errorf("exclude pattern contains invalid characters")
	}
	clean := filepath.Clean(pattern)
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("project exclude pattern must be relative to the project root")
	}
	return filepath.ToSlash(clean), nil
}

func sortedUnique(values []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
