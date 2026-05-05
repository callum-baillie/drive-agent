package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
)

type State struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Repos         map[string]RepoState `json:"repos"`
	UpdatedAt     string               `json:"updatedAt"`
}

type RepoState struct {
	LastSuccessfulBackupAt string `json:"lastSuccessfulBackupAt,omitempty"`
	LastSnapshotID         string `json:"lastSnapshotId,omitempty"`
	LastCheckAt            string `json:"lastCheckAt,omitempty"`
	SnapshotCount          int    `json:"snapshotCount,omitempty"`
	LatestSnapshotID       string `json:"latestSnapshotId,omitempty"`
	LastError              string `json:"lastError,omitempty"`
	UpdatedAt              string `json:"updatedAt,omitempty"`
}

func StatePath(driveRoot string) string {
	return filepath.Join(filesystem.AgentPath(driveRoot), "state", StateFileName)
}

func LoadState(driveRoot string) (*State, error) {
	path := StatePath(driveRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("read backup state: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse backup state: %w", err)
	}
	normalizeState(&state)
	return &state, nil
}

func SaveState(driveRoot string, state *State) error {
	normalizeState(state)
	path := StatePath(driveRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create backup state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode backup state: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write backup state: %w", err)
	}
	return nil
}

func NewState() *State {
	return &State{SchemaVersion: 1, Repos: make(map[string]RepoState)}
}

func (s *State) Repo(name string) RepoState {
	normalizeState(s)
	return s.Repos[name]
}

func (s *State) UpdateRepo(name string, repoState RepoState) {
	normalizeState(s)
	now := config.NowISO()
	repoState.UpdatedAt = now
	s.Repos[name] = repoState
	s.UpdatedAt = now
}

func normalizeState(state *State) {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	if state.Repos == nil {
		state.Repos = make(map[string]RepoState)
	}
}
