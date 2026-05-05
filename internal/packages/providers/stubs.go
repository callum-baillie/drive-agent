package providers

import "fmt"

// StubProvider represents a package manager that is scaffolded but not yet implemented.
type StubProvider struct {
	id          string
	name        string
	supportedOS []string
}

func (s *StubProvider) ID() string                     { return s.id }
func (s *StubProvider) Name() string                   { return s.name }
func (s *StubProvider) SupportedOS() []string          { return s.supportedOS }
func (s *StubProvider) IsAvailable() bool              { return false }
func (s *StubProvider) ManagerPath() string            { return "" }
func (s *StubProvider) ManagerVersion() string         { return "" }
func (s *StubProvider) IsPackageInstalled(string) bool { return false }

func (s *StubProvider) InstallManager(dryRun bool) (string, error) {
	return "", fmt.Errorf("provider %q is not yet implemented", s.id)
}

func (s *StubProvider) InstallPackage(packageName string, dryRun bool) (string, error) {
	return "", fmt.Errorf("provider %q is not yet implemented", s.id)
}

func (s *StubProvider) InstallPackages(packages []string, dryRun bool) (string, error) {
	return "", fmt.Errorf("provider %q is not yet implemented", s.id)
}

// stubProviders returns all stub providers for future implementation.
func stubProviders() []*StubProvider {
	return []*StubProvider{
		{id: "winget", name: "Winget", supportedOS: []string{"windows"}},
		{id: "chocolatey", name: "Chocolatey", supportedOS: []string{"windows"}},
		{id: "scoop", name: "Scoop", supportedOS: []string{"windows"}},
		{id: "apt", name: "apt", supportedOS: []string{"linux"}},
		{id: "dnf", name: "dnf", supportedOS: []string{"linux"}},
		{id: "pacman", name: "pacman", supportedOS: []string{"linux"}},
		{id: "zypper", name: "zypper", supportedOS: []string{"linux"}},
		{id: "apk", name: "apk", supportedOS: []string{"linux"}},
		{id: "mise", name: "mise", supportedOS: []string{"darwin", "linux"}},
		{id: "asdf", name: "asdf", supportedOS: []string{"darwin", "linux"}},
		{id: "nix", name: "Nix", supportedOS: []string{"darwin", "linux"}},
	}
}
