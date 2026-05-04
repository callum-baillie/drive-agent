package self

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/callum-baillie/drive-agent/internal/config"
)

func TestDetermineAssetName(t *testing.T) {
	config.RepoName = "drive-agent"
	tests := []struct {
		os   string
		arch string
		want string
		err  bool
	}{
		{"darwin", "amd64", "drive-agent_Darwin_x86_64.tar.gz", false},
		{"darwin", "arm64", "drive-agent_Darwin_arm64.tar.gz", false},
		{"windows", "amd64", "drive-agent_Windows_x86_64.zip", false},
		{"linux", "amd64", "drive-agent_Linux_x86_64.tar.gz", false},
		{"linux", "arm64", "drive-agent_Linux_arm64.tar.gz", false},
		{"windows", "arm64", "", true},
		{"freebsd", "amd64", "", true},
		{"linux", "386", "", true},
	}

	for _, tt := range tests {
		got, err := determineAssetName(tt.os, tt.arch)
		if (err != nil) != tt.err {
			t.Errorf("determineAssetName(%q, %q) error = %v, wantErr %v", tt.os, tt.arch, err, tt.err)
		}
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

func TestFirstUsableReleaseSkipsDrafts(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v0.1.0-alpha.3", Draft: true},
		{TagName: "v0.1.0-alpha.2", Prerelease: true},
	}

	release, ok := firstUsableRelease(releases)
	if !ok {
		t.Fatal("firstUsableRelease returned ok=false")
	}
	if release.TagName != "v0.1.0-alpha.2" {
		t.Fatalf("release = %q, want v0.1.0-alpha.2", release.TagName)
	}
}

func TestFirstUsableReleaseEmpty(t *testing.T) {
	if _, ok := firstUsableRelease([]githubRelease{{TagName: "v0.1.0-alpha.3", Draft: true}}); ok {
		t.Fatal("firstUsableRelease returned ok=true for only draft releases")
	}
}

func TestListBackupsEmpty(t *testing.T) {
	backups, err := listBackups(t.TempDir())
	if err != nil {
		t.Fatalf("listBackups: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("backups = %v, want empty", backups)
	}
}

func TestListBackupsFiltersAndSorts(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"notes.txt", "drive-agent-v0.1.0-b", "drive-agent-v0.1.0-a"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "drive-agent-v0.1.0-dir"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	backups, err := listBackups(tmpDir)
	if err != nil {
		t.Fatalf("listBackups: %v", err)
	}
	want := []string{"drive-agent-v0.1.0-a", "drive-agent-v0.1.0-b"}
	if len(backups) != len(want) {
		t.Fatalf("backups = %v, want %v", backups, want)
	}
	for i := range want {
		if backups[i] != want[i] {
			t.Fatalf("backups = %v, want %v", backups, want)
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
