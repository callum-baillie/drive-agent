package backup

import (
	"os"
	"regexp"
	"strings"
)

var urlUserInfoPattern = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)([^/@\s]+@)`)

func DetectPasswordSource() PasswordSource {
	if os.Getenv("RESTIC_PASSWORD") != "" {
		return PasswordSource{Configured: true, Source: "RESTIC_PASSWORD"}
	}
	if os.Getenv("RESTIC_PASSWORD_FILE") != "" {
		return PasswordSource{Configured: true, Source: "RESTIC_PASSWORD_FILE"}
	}
	return PasswordSource{
		Configured: false,
		Warning:    "No Restic password source configured. Set RESTIC_PASSWORD or RESTIC_PASSWORD_FILE.",
	}
}

func RedactSensitive(input string) string {
	redacted := input
	for _, key := range []string{"RESTIC_PASSWORD", "RESTIC_PASSWORD_FILE", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"} {
		if value := os.Getenv(key); value != "" {
			redacted = strings.ReplaceAll(redacted, value, "[REDACTED]")
		}
	}
	redacted = urlUserInfoPattern.ReplaceAllString(redacted, "${1}[REDACTED]@")
	return redacted
}
