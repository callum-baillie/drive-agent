package self

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/callumbaillie/drive-agent/internal/config"
)

func TestDetermineAssetName(t *testing.T) {
	config.RepoName = "drive-agent"
	tests := []struct {
		os   string
		arch string
		want string
	}{
		{"darwin", "amd64", "drive-agent_Darwin_x86_64.tar.gz"},
		{"darwin", "arm64", "drive-agent_Darwin_arm64.tar.gz"},
		{"windows", "amd64", "drive-agent_Windows_x86_64.zip"},
		{"linux", "amd64", "drive-agent_Linux_x86_64.tar.gz"},
	}

	for _, tt := range tests {
		got := determineAssetName(tt.os, tt.arch)
		if got != tt.want {
			t.Errorf("determineAssetName(%q, %q) = %q; want %q", tt.os, tt.arch, got, tt.want)
		}
	}
}

func TestParseChecksums(t *testing.T) {
	data := `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  drive-agent_Darwin_arm64.tar.gz
1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  drive-agent_Windows_x86_64.zip`

	tests := []struct {
		asset string
		want  string
		err   bool
	}{
		{"drive-agent_Darwin_arm64.tar.gz", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", false},
		{"drive-agent_Windows_x86_64.zip", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"not-found.zip", "", true},
	}

	for _, tt := range tests {
		got, err := parseChecksums(data, tt.asset)
		if (err != nil) != tt.err {
			t.Errorf("parseChecksums(%q) error = %v, wantErr %v", tt.asset, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseChecksums(%q) = %q, want %q", tt.asset, got, tt.want)
		}
	}
}

func TestExtractZip(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	destPath := filepath.Join(tmpDir, "extracted")

	// Create a dummy zip file
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("drive-agent.exe")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("hello windows"))
	zw.Close()
	f.Close()

	if err := extractZip(zipPath, destPath); err != nil {
		t.Errorf("extractZip failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello windows" {
		t.Errorf("extractZip content = %q; want 'hello windows'", data)
	}
}

func TestExtractTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar.gz")
	destPath := filepath.Join(tmpDir, "extracted")

	// Create a dummy tar.gz file
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := []byte("hello unix")
	hdr := &tar.Header{
		Name: "drive-agent",
		Mode: 0755,
		Size: int64(len(content)),
	}
	tw.WriteHeader(hdr)
	tw.Write(content)
	tw.Close()
	gzw.Close()

	os.WriteFile(tarPath, buf.Bytes(), 0644)

	if err := extractTarGz(tarPath, destPath); err != nil {
		t.Errorf("extractTarGz failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello unix" {
		t.Errorf("extractTarGz content = %q; want 'hello unix'", data)
	}
}
