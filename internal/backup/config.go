package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
)

const (
	ConfigFileName = "backup.json"
	StateFileName  = "backup.json"
	DefaultRepo    = "default"
)

type Config struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Provider      string                `json:"provider"`
	SelectedRepo  string                `json:"selectedRepo"`
	Repos         map[string]Repository `json:"repos"`
	Excludes      []string              `json:"excludes"`
	CreatedAt     string                `json:"createdAt"`
	UpdatedAt     string                `json:"updatedAt"`
}

type Repository struct {
	Name               string `json:"name"`
	Provider           string `json:"provider"`
	Repository         string `json:"repository"`
	AllowSameDriveRepo bool   `json:"allowSameDriveRepo,omitempty"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

func ConfigPath(driveRoot string) string {
	return filepath.Join(filesystem.AgentPath(driveRoot), "config", ConfigFileName)
}

func LoadConfig(driveRoot string) (*Config, error) {
	path := ConfigPath(driveRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read backup config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse backup config: %w", err)
	}
	normalizeConfig(&cfg)
	return &cfg, nil
}

func SaveConfig(driveRoot string, cfg *Config) error {
	normalizeConfig(cfg)
	path := ConfigPath(driveRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create backup config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode backup config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write backup config: %w", err)
	}
	return nil
}

func NewConfig(provider, repoName, repo string, allowSameDrive bool) *Config {
	now := config.NowISO()
	if repoName == "" {
		repoName = DefaultRepo
	}
	cfg := &Config{
		SchemaVersion: 1,
		Provider:      provider,
		SelectedRepo:  repoName,
		Repos:         make(map[string]Repository),
		Excludes:      DefaultExcludes(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	cfg.Repos[repoName] = Repository{
		Name:               repoName,
		Provider:           provider,
		Repository:         repo,
		AllowSameDriveRepo: allowSameDrive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return cfg
}

func (c *Config) SelectedRepository(repoName string) (Repository, error) {
	normalizeConfig(c)
	if repoName == "" {
		repoName = c.SelectedRepo
	}
	if repoName == "" {
		repoName = DefaultRepo
	}
	repo, ok := c.Repos[repoName]
	if !ok {
		return Repository{}, fmt.Errorf("backup repo %q is not configured", repoName)
	}
	return repo, nil
}

func (c *Config) UpsertRepository(repo Repository) {
	normalizeConfig(c)
	now := config.NowISO()
	if repo.Name == "" {
		repo.Name = DefaultRepo
	}
	if repo.Provider == "" {
		repo.Provider = c.Provider
	}
	if repo.CreatedAt == "" {
		repo.CreatedAt = now
	}
	repo.UpdatedAt = now
	c.Repos[repo.Name] = repo
	c.Provider = repo.Provider
	c.SelectedRepo = repo.Name
	c.UpdatedAt = now
}

func normalizeConfig(cfg *Config) {
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = 1
	}
	if cfg.Provider == "" {
		cfg.Provider = "restic"
	}
	if cfg.SelectedRepo == "" {
		cfg.SelectedRepo = DefaultRepo
	}
	if cfg.Repos == nil {
		cfg.Repos = make(map[string]Repository)
	}
	if cfg.Excludes == nil {
		cfg.Excludes = DefaultExcludes()
	} else {
		cfg.Excludes = sortedUnique(append(cfg.Excludes, DefaultExcludes()...))
	}
}
