package backup

import (
	"strings"
	"testing"
)

func TestDetectPasswordSource(t *testing.T) {
	t.Setenv("RESTIC_PASSWORD", "")
	t.Setenv("RESTIC_PASSWORD_FILE", "")
	if source := DetectPasswordSource(); source.Configured {
		t.Fatalf("source = %+v, want missing", source)
	}

	t.Setenv("RESTIC_PASSWORD", "secret")
	if source := DetectPasswordSource(); !source.Configured || source.Source != "RESTIC_PASSWORD" {
		t.Fatalf("source = %+v", source)
	}
}

func TestRedactSensitive(t *testing.T) {
	t.Setenv("RESTIC_PASSWORD", "super-secret")
	t.Setenv("RESTIC_PASSWORD_FILE", "/secret/file")
	t.Setenv("AWS_ACCESS_KEY_ID", "access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-secret-key")

	got := RedactSensitive("RESTIC_PASSWORD=super-secret file=/secret/file key=access-key secret=aws-secret-key")
	if got == "RESTIC_PASSWORD=super-secret file=/secret/file" {
		t.Fatal("expected redaction")
	}
	if containsSecret(got) {
		t.Fatalf("redacted string still contains secret: %q", got)
	}
}

func TestRedactSensitiveRedactsURLUserInfo(t *testing.T) {
	got := RedactSensitive("restic -r rest:http://user:pass@example.com/repo backup /Volumes/DevDrive")
	if strings.Contains(got, "user:pass") {
		t.Fatalf("redacted string still contains URL credentials: %q", got)
	}
	if !strings.Contains(got, "http://[REDACTED]@example.com/repo") {
		t.Fatalf("redacted string did not preserve safe URL shape: %q", got)
	}
}

func containsSecret(s string) bool {
	return strings.Contains(s, "super-secret") || strings.Contains(s, "/secret/file") || strings.Contains(s, "access-key") || strings.Contains(s, "aws-secret-key")
}
