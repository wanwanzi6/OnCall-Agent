package upload

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	ErrEmptyFileName       = errors.New("file name is required")
	ErrInvalidFileName     = errors.New("file name is invalid")
	ErrFileTooLarge        = errors.New("file size exceeds limit")
	ErrUnsupportedFileType = errors.New("unsupported file type")
	ErrUnsafePath          = errors.New("unsafe upload path")
)

type Policy struct {
	UploadDir        string
	MaxFileSizeBytes int64
	AllowedExts      []string
}

func SanitizeFileName(name string) (string, error) {
	raw := strings.TrimSpace(name)
	if raw == "" {
		return "", ErrEmptyFileName
	}
	if strings.Contains(raw, "..") || strings.ContainsAny(raw, `/\`) {
		return "", ErrInvalidFileName
	}
	name = filepath.Base(raw)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "", ErrEmptyFileName
	}

	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune('_')
		default:
			b.WriteRune('_')
		}
	}

	sanitized := strings.Trim(b.String(), "._-")
	if sanitized == "" {
		return "", ErrInvalidFileName
	}
	return sanitized, nil
}

func ValidateSize(size, max int64) error {
	if max > 0 && size > max {
		return fmt.Errorf("%w: max=%d actual=%d", ErrFileTooLarge, max, size)
	}
	return nil
}

func ValidateExtension(fileName string, allowed []string) error {
	ext := strings.ToLower(filepath.Ext(fileName))
	for _, item := range allowed {
		allowedExt := strings.ToLower(strings.TrimSpace(item))
		if allowedExt == "" {
			continue
		}
		if !strings.HasPrefix(allowedExt, ".") {
			allowedExt = "." + allowedExt
		}
		if ext == allowedExt {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedFileType, ext)
}

func SafeDestination(uploadDir, fileName string) (string, error) {
	if uploadDir == "" {
		return "", ErrUnsafePath
	}
	cleanDir, err := filepath.Abs(filepath.Clean(uploadDir))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnsafePath, err)
	}
	target := filepath.Join(cleanDir, fileName)
	cleanTarget, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnsafePath, err)
	}

	rel, err := filepath.Rel(cleanDir, cleanTarget)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", ErrUnsafePath
	}
	return cleanTarget, nil
}

func Validate(policy Policy, fileName string, size int64) (string, string, error) {
	sanitized, err := SanitizeFileName(fileName)
	if err != nil {
		return "", "", err
	}
	if err := ValidateExtension(sanitized, policy.AllowedExts); err != nil {
		return "", "", err
	}
	if err := ValidateSize(size, policy.MaxFileSizeBytes); err != nil {
		return "", "", err
	}
	target, err := SafeDestination(policy.UploadDir, sanitized)
	if err != nil {
		return "", "", err
	}
	return sanitized, target, nil
}
