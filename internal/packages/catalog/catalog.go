package catalog

import (
	"encoding/json"
	"fmt"
	"os"
)

// Catalog represents the full package catalog.
type Catalog struct {
	SchemaVersion int       `json:"schemaVersion"`
	Packages      []Package `json:"packages"`
}

// Package represents a single package entry in the catalog.
type Package struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	Category          string                   `json:"category"`
	Description       string                   `json:"description"`
	Kind              string                   `json:"kind"` // "cli", "gui", "runtime", "service"
	Default           bool                     `json:"default,omitempty"`
	InstallPreference []string                 `json:"installPreference"`
	Install           map[string]InstallConfig `json:"install"`
	Check             *CheckConfig             `json:"check,omitempty"`
	RequiresApproval  bool                     `json:"requiresExplicitApproval,omitempty"`
}

// InstallConfig holds installation details for a specific package manager.
type InstallConfig struct {
	Type    string `json:"type,omitempty"`    // "formula", "cask"
	Name    string `json:"name,omitempty"`    // Package name for this manager
	ID      string `json:"id,omitempty"`      // Alternative ID field (winget, choco, etc.)
	Global  bool   `json:"global,omitempty"`  // For npm/pnpm global installs
	MacOS   string `json:"macos,omitempty"`   // Native install command for macOS
	Windows string `json:"windows,omitempty"` // Native install command for Windows
	Linux   string `json:"linux,omitempty"`   // Native install command for Linux
}

// CheckConfig defines how to verify a package is installed.
type CheckConfig struct {
	Command    string   `json:"command"`
	AppBundles []string `json:"appBundles,omitempty"`
}

// LoadCatalog reads and parses a package catalog JSON file.
func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}

	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return &cat, nil
}

// GetPackage returns a package by ID.
func (c *Catalog) GetPackage(id string) *Package {
	for i := range c.Packages {
		if c.Packages[i].ID == id {
			return &c.Packages[i]
		}
	}
	return nil
}

// GetByCategory returns packages in a specific category.
func (c *Catalog) GetByCategory(category string) []Package {
	var result []Package
	for _, p := range c.Packages {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}

// Categories returns all unique categories in the catalog.
func (c *Catalog) Categories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, p := range c.Packages {
		if !seen[p.Category] {
			seen[p.Category] = true
			cats = append(cats, p.Category)
		}
	}
	return cats
}

// GetInstallName returns the package name for a specific manager, or empty string if not available.
func (p *Package) GetInstallName(managerID string) string {
	cfg, ok := p.Install[managerID]
	if !ok {
		return ""
	}
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.ID != "" {
		return cfg.ID
	}
	return p.ID
}

// AvailableOn returns the list of package managers that can install this package.
func (p *Package) AvailableOn() []string {
	var managers []string
	for mgr := range p.Install {
		managers = append(managers, mgr)
	}
	return managers
}
