package upload

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSanitizeFileName(t *testing.T) {
	got, err := SanitizeFileName("告警 处理手册.md")
	if err != nil {
		t.Fatalf("SanitizeFileName returned error: %v", err)
	}
	if got != "告警_处理手册.md" {
		t.Fatalf("sanitized = %q", got)
	}
}

func TestSanitizeFileNameRejectsTraversal(t *testing.T) {
	if _, err := SanitizeFileName("../evil.md"); !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("error = %v, want ErrInvalidFileName", err)
	}
}

func TestValidateExtension(t *testing.T) {
	if err := ValidateExtension("runbook.md", []string{".md", ".txt"}); err != nil {
		t.Fatalf("ValidateExtension returned error: %v", err)
	}
	if err := ValidateExtension("runbook.pdf", []string{".md", ".txt"}); !errors.Is(err, ErrUnsupportedFileType) {
		t.Fatalf("error = %v, want ErrUnsupportedFileType", err)
	}
}

func TestValidateSize(t *testing.T) {
	if err := ValidateSize(11, 10); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("error = %v, want ErrFileTooLarge", err)
	}
}

func TestSafeDestinationStaysUnderUploadDir(t *testing.T) {
	dir := t.TempDir()
	target, err := SafeDestination(dir, "runbook.txt")
	if err != nil {
		t.Fatalf("SafeDestination returned error: %v", err)
	}
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		t.Fatalf("Rel returned error: %v", err)
	}
	if rel != "runbook.txt" {
		t.Fatalf("rel = %q, want runbook.txt", rel)
	}
}
