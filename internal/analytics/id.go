package analytics

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/rnwolfe/mine/internal/config"
)

// idFileName is the name of the file storing the installation ID.
const idFileName = "analytics_id"

// GetOrCreateID returns the installation ID, generating one if it doesn't exist.
// The ID is a UUIDv4 stored as a plain text file in the XDG data directory.
func GetOrCreateID() (string, error) {
	idPath := idFilePath()

	data, err := os.ReadFile(idPath)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if isValidUUID(id) {
			return id, nil
		}
		// Invalid/corrupt — fall through to regenerate
	}

	return generateAndSaveID(idPath)
}

// generateAndSaveID creates a new UUIDv4 and persists it.
func generateAndSaveID(path string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	id := uuid.New().String()
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", err
	}
	return id, nil
}

// idFilePath returns the full path to the analytics ID file.
func idFilePath() string {
	return filepath.Join(config.GetPaths().DataDir, idFileName)
}

// isValidUUID checks if a string is a valid UUID.
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
