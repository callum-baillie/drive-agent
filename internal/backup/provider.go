package backup

import "context"

type PasswordSource struct {
	Configured bool
	Source     string
	Warning    string
}

type Provider interface {
	Name() string
	IsInstalled() bool
	Init(ctx context.Context, repo Repository, dryRun bool) (Plan, error)
	Backup(ctx context.Context, req BackupRequest) (Plan, error)
	Snapshots(ctx context.Context, repo Repository) ([]Snapshot, error)
	Check(ctx context.Context, repo Repository, dryRun bool) (Plan, error)
	Restore(ctx context.Context, req RestoreRequest) (Plan, error)
}

type Plan struct {
	Command string
	LogPath string
	Output  string
}

type BackupRequest struct {
	DriveRoot   string
	DriveLabel  string
	HostLabel   string
	Repo        Repository
	ExcludeFile string
	ExtraTags   []string
	DryRun      bool
}

type RestoreRequest struct {
	Repo     Repository
	Snapshot string
	Target   string
	DryRun   bool
}

type Snapshot struct {
	ID       string   `json:"id"`
	ShortID  string   `json:"shortId,omitempty"`
	Time     string   `json:"time"`
	Hostname string   `json:"hostname"`
	Paths    []string `json:"paths"`
	Tags     []string `json:"tags,omitempty"`
	Summary  string   `json:"summary,omitempty"`
	RawID    string   `json:"-"`
}
