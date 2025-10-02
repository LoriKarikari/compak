package pkg

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const testComposeFilename = "docker-compose.yaml"

func TestDownloadComposeFile(t *testing.T) {
	composeContent := `services:
  app:
    image: nginx:alpine
    ports:
      - "8080:80"
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(composeContent)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, testComposeFilename)

	err := downloadComposeFile(server.URL, destPath)
	if err != nil {
		t.Fatalf("downloadComposeFile failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(data) != composeContent {
		t.Errorf("Downloaded content mismatch.\nGot:\n%s\nWant:\n%s", string(data), composeContent)
	}
}

func TestDownloadComposeFileInvalidYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("this is not valid yaml: [[[")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, testComposeFilename)

	err := downloadComposeFile(server.URL, destPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestDownloadComposeFileHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, testComposeFilename)

	err := downloadComposeFile(server.URL, destPath)
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

func TestDownloadComposeFileTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, testComposeFilename)

	err := downloadComposeFile(server.URL, destPath)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}
