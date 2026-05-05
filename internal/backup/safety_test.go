package backup

import "testing"

func TestValidateRepositoryRejectsSameDrive(t *testing.T) {
	safety, err := ValidateRepository("/Volumes/DevDrive", "/Volumes/DevDrive/Backups/restic", false)
	if err == nil {
		t.Fatal("expected same-drive repo rejection")
	}
	if !safety.SameDrive {
		t.Fatal("expected SameDrive=true")
	}
}

func TestValidateRepositoryAllowsSameDriveWithWarning(t *testing.T) {
	safety, err := ValidateRepository("/Volumes/DevDrive", "/Volumes/DevDrive/Backups/restic", true)
	if err != nil {
		t.Fatalf("ValidateRepository: %v", err)
	}
	if !safety.SameDrive || len(safety.Warnings) == 0 {
		t.Fatalf("safety = %+v, want same-drive warning", safety)
	}
}

func TestValidateRepositoryRemote(t *testing.T) {
	for _, repo := range []string{"sftp:user@example.com:/backups/devdrive", "s3:s3.amazonaws.com/bucket/devdrive"} {
		safety, err := ValidateRepository("/Volumes/DevDrive", repo, false)
		if err != nil {
			t.Fatalf("ValidateRepository(%q): %v", repo, err)
		}
		if safety.IsLocal {
			t.Fatalf("repo %q detected as local", repo)
		}
	}
}

func TestValidateRepositoryRejectsDangerousLocalPath(t *testing.T) {
	if _, err := ValidateRepository("/Volumes/DevDrive", "/tmp/restic-repo", false); err == nil {
		t.Fatal("expected dangerous repo path rejection")
	}
}

func TestValidateRestoreTargetSafety(t *testing.T) {
	if err := ValidateRestoreTarget("/Volumes/DevDrive", "/tmp/RestoreTest"); err == nil {
		t.Fatal("expected /tmp restore target rejection")
	}
	if err := ValidateRestoreTarget("/Volumes/DevDrive", "/Volumes/DevDrive/restore"); err == nil {
		t.Fatal("expected active-drive restore target rejection")
	}
	if err := ValidateRestoreTarget("/Volumes/DevDrive", "/Volumes/RestoreTest"); err != nil {
		t.Fatalf("safe restore target rejected: %v", err)
	}
}
