package utils

import (
	"fmt"

	"github.com/google/uuid"
)

// NewID generates a new UUID string.
func NewID() string {
	return uuid.New().String()
}

// ProjectID generates a project ID from org slug and project slug.
func ProjectID(orgSlug, projectSlug string) string {
	return fmt.Sprintf("proj_%s_%s", orgSlug, projectSlug)
}

// OrgID generates an organization ID from the slug.
func OrgID(slug string) string {
	return fmt.Sprintf("org_%s", slug)
}
