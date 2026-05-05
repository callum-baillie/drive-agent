package providers

// Provider defines the interface for package manager providers.
type Provider interface {
	// ID returns the unique identifier for this provider.
	ID() string

	// Name returns the human-readable name.
	Name() string

	// SupportedOS returns the operating systems this provider supports.
	SupportedOS() []string

	// IsAvailable checks if this package manager is installed on the host.
	IsAvailable() bool

	// ManagerPath returns the path to the package manager binary.
	ManagerPath() string

	// ManagerVersion returns the version of the package manager.
	ManagerVersion() string

	// InstallManager provides instructions or installs the package manager itself.
	// Returns the command that would be run (for dry-run display).
	InstallManager(dryRun bool) (string, error)

	// IsPackageInstalled checks if a specific package is installed.
	IsPackageInstalled(packageName string) bool

	// InstallPackage installs a package.
	// Returns the command that was/would be run.
	InstallPackage(packageName string, dryRun bool) (string, error)

	// InstallPackages installs multiple packages at once.
	// Returns the command that was/would be run.
	InstallPackages(packages []string, dryRun bool) (string, error)
}

// Registry holds all known package manager providers.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry with all known providers.
func NewRegistry() *Registry {
	r := &Registry{
		providers: make(map[string]Provider),
	}

	// Register built-in providers
	r.Register(&Homebrew{})
	r.Register(&HomebrewCask{})
	r.Register(&NpmGlobal{})
	r.Register(&PnpmGlobal{})
	r.Register(&BunGlobal{})
	r.Register(&Pipx{})
	r.Register(&UvTool{})
	r.Register(&CargoInstall{})
	r.Register(&GoInstall{})

	// Register stub providers for future implementation
	for _, stub := range stubProviders() {
		r.Register(stub)
	}

	return r
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.providers[p.ID()] = p
}

// Get returns a provider by ID.
func (r *Registry) Get(id string) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

// Available returns all providers that are available on this system.
func (r *Registry) Available() []Provider {
	var result []Provider
	for _, p := range r.providers {
		if p.IsAvailable() {
			result = append(result, p)
		}
	}
	return result
}

// All returns all registered providers.
func (r *Registry) All() map[string]Provider {
	return r.providers
}
