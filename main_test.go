package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsCriticalPath(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"/", true},
		{"/usr", true},
		{"/bin", true},
		{"/etc", true},
		{"/home", true},
		{"/root", true},
		{"/var", true},
		{"/opt", true},
		{"/usr/local/go", false},
		{"/home/user/.gvm/gos/go1.21", false},
		{"C:\\", true},
		{"C:\\Windows", true},
		{"C:\\Program Files", true},
		{"C:\\Users", true},
		{"C:\\Go", false},
	}

	for _, tc := range testCases {
		result := isCriticalPath(tc.path)
		if result != tc.expected {
			t.Errorf("isCriticalPath(%s) = %v, expected %v", tc.path, result, tc.expected)
		}
	}
}

func TestGenerateSecurityHash(t *testing.T) {
	hash1 := generateSecurityHash()
	hash2 := generateSecurityHash()

	if len(hash1) != 8 {
		t.Errorf("Expected hash length 8, got %d", len(hash1))
	}

	if hash1 == hash2 {
		t.Error("Expected different hashes, got same hash")
	}
}

func TestCheckPermissions(t *testing.T) {
	// Test permission checking (this will vary by system)
	err := checkPermissions()
	// Should not panic or crash
	if err != nil && runtime.GOOS != "windows" {
		t.Logf("Permission check failed (expected on non-root): %v", err)
	}
}

func TestDetectGoInstallations(t *testing.T) {
	installations := detectGoInstallations()

	// Should return a slice (may be empty)
	if installations == nil {
		t.Error("Expected non-nil installations slice")
	}

	// Verify installation structure
	for i, install := range installations {
		if install.Path == "" {
			t.Errorf("Installation %d has empty path", i)
		}
		if install.Source == "" {
			t.Errorf("Installation %d has empty source", i)
		}
		if !install.Verified {
			t.Errorf("Installation %d not verified", i)
		}
	}
}

func TestGetGoVersion(t *testing.T) {
	// Test with non-existent path
	version := getGoVersion("/non/existent/path")
	if version != "unknown version" {
		t.Errorf("Expected 'unknown version', got '%s'", version)
	}
}

func TestGetDirSize(t *testing.T) {
	// Create a temporary directory with known content
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size := getDirSize(tempDir)
	if size < int64(len(testContent)) {
		t.Errorf("Expected size >= %d, got %d", len(testContent), size)
	}
}

func TestGetPermissions(t *testing.T) {
	// Test with current directory
	permissions := getPermissions(".")
	if permissions == "unknown" {
		t.Error("Expected valid permissions, got 'unknown'")
	}
	if permissions == "" {
		t.Error("Expected non-empty permissions string")
	}
}

func TestCreateBackup(t *testing.T) {
	// Test backup creation with temporary directory
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	backupDir := filepath.Join(tempDir, "backup")

	// Create source directory and file
	err := os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Test backup creation
	err = createBackup(sourceDir, backupDir)
	if err != nil {
		t.Logf("Backup creation failed (may be expected if tar not available): %v", err)
	}
}

func TestNewLogger(t *testing.T) {
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Error("Expected non-nil logger")
	}

	// Test logging
	logger.Log("TEST", "This is a test message")

	// Verify log file exists
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".fugo")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}
}

func TestGoInstallationStruct(t *testing.T) {
	installation := GoInstallation{
		Path:        "/usr/local/go",
		Version:     "go version go1.21.0 linux/amd64",
		Source:      "official",
		Size:        1024,
		Permissions: "drwxr-xr-x",
		Verified:    true,
	}

	if installation.Path != "/usr/local/go" {
		t.Error("GoInstallation path not set correctly")
	}
	if installation.Version != "go version go1.21.0 linux/amd64" {
		t.Error("GoInstallation version not set correctly")
	}
	if installation.Source != "official" {
		t.Error("GoInstallation source not set correctly")
	}
	if installation.Size != 1024 {
		t.Error("GoInstallation size not set correctly")
	}
	if installation.Permissions != "drwxr-xr-x" {
		t.Error("GoInstallation permissions not set correctly")
	}
	if !installation.Verified {
		t.Error("GoInstallation should be verified")
	}
}

// Benchmark tests for performance-critical functions
func BenchmarkDetectGoInstallations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		detectGoInstallations()
	}
}

func BenchmarkGenerateSecurityHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateSecurityHash()
	}
}

func BenchmarkIsCriticalPath(b *testing.B) {
	testPaths := []string{"/", "/usr", "/usr/local/go", "/home/user/.gvm"}
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			isCriticalPath(path)
		}
	}
}
