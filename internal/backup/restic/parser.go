package restic

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/callum-baillie/drive-agent/internal/backup"
)

type snapshotJSON struct {
	ID       string   `json:"id"`
	ShortID  string   `json:"short_id"`
	Time     string   `json:"time"`
	Hostname string   `json:"hostname"`
	Paths    []string `json:"paths"`
	Tags     []string `json:"tags"`
}

func ParseSnapshotsJSON(data []byte) ([]backup.Snapshot, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}

	var raw []snapshotJSON
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, fmt.Errorf("parse restic snapshots JSON: %w", err)
	}

	snapshots := make([]backup.Snapshot, 0, len(raw))
	for _, item := range raw {
		id := item.ID
		if id == "" {
			id = item.ShortID
		}
		snapshots = append(snapshots, backup.Snapshot{
			ID:       id,
			ShortID:  item.ShortID,
			Time:     item.Time,
			Hostname: item.Hostname,
			Paths:    item.Paths,
			Tags:     item.Tags,
			RawID:    item.ID,
		})
	}
	return snapshots, nil
}
