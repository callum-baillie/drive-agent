package cli

import (
	"bytes"
	"testing"

	"github.com/callum-baillie/drive-agent/internal/config"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}

	want := "drive-agent v" + config.Version + "\n"
	if out.String() != want {
		t.Fatalf("version output = %q, want %q", out.String(), want)
	}
}
