package service

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tstech/backend/internal/config"
)

func createMockFileHeader(filename string, content []byte, contentType string) (*multipart.FileHeader, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, err
	}
	_, _ = part.Write(content)
	_ = writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(1024 * 1024)
	if err != nil {
		return nil, err
	}
	return form.File["file"][0], nil
}

func TestSanitizeFolder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "general"},
		{"/", "general"},
		{"..", "general"},
		{"payments", "payments"},
		{"/payments/", "payments"},
		{"projects/deliverables", "projects/deliverables"},
		{"../payments", "payments"},
		{"projects\\assets", "projects/assets"},
		{"  cms/banners  ", "cms/banners"},
	}

	for _, tt := range tests {
		res := sanitizeFolder(tt.input)
		if res != tt.expected {
			t.Errorf("sanitizeFolder(%q) = %q, expected %q", tt.input, res, tt.expected)
		}
	}
}

func TestStorageService_SaveAndGetFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{}
	svc := NewStorageService(cfg)
	svc.uploadDir = tempDir

	// Test upload to payments folder
	fileHeader, err := createMockFileHeader("receipt.png", []byte("fake-image-content"), "image/png")
	if err != nil {
		t.Fatalf("Failed to create mock file header: %v", err)
	}

	result, err := svc.SaveFile(fileHeader, "payments")
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	if result.Folder != "payments" {
		t.Errorf("Expected folder 'payments', got %q", result.Folder)
	}
	if !strings.HasPrefix(result.FileURL, "/uploads/payments/") {
		t.Errorf("Expected URL to start with '/uploads/payments/', got %q", result.FileURL)
	}

	// Verify local file exists in payments subdirectory
	expectedSubpath := strings.TrimPrefix(result.FileURL, "/uploads/")
	localPath := filepath.Join(tempDir, filepath.FromSlash(expectedSubpath))
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Errorf("File was not saved to expected path: %s", localPath)
	}

	// Test GetFile with full relative path
	rc, _, err := svc.GetFile(expectedSubpath)
	if err != nil {
		t.Fatalf("GetFile failed for subpath %q: %v", expectedSubpath, err)
	}
	rc.Close()

	// Test GetFile with legacy filename fallback
	filenameOnly := filepath.Base(expectedSubpath)
	rcLegacy, _, err := svc.GetFile(filenameOnly)
	if err != nil {
		t.Fatalf("GetFile failed for legacy filename %q: %v", filenameOnly, err)
	}
	rcLegacy.Close()
}
