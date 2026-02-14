package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/config"
)

// PermissionSummary formats permissions for display during install.
func PermissionSummary(perms Permissions) []string {
	var lines []string

	if perms.Network {
		lines = append(lines, "Network: outbound access")
	}
	if len(perms.Filesystem) > 0 {
		lines = append(lines, fmt.Sprintf("Filesystem: %s", strings.Join(perms.Filesystem, ", ")))
	}
	if perms.Store {
		lines = append(lines, "Store: read/write mine database")
	}
	if perms.ConfigRead {
		lines = append(lines, "Config: read mine configuration")
	}
	if perms.ConfigWrite {
		lines = append(lines, "Config: write mine configuration")
	}
	if len(perms.EnvVars) > 0 {
		lines = append(lines, fmt.Sprintf("Environment: %s", strings.Join(perms.EnvVars, ", ")))
	}

	if len(lines) == 0 {
		lines = append(lines, "No special permissions required")
	}

	return lines
}

// HasEscalation checks if new permissions exceed current permissions.
func HasEscalation(current, proposed Permissions) []string {
	var escalations []string

	if !current.Network && proposed.Network {
		escalations = append(escalations, "NEW: network access")
	}
	if !current.Store && proposed.Store {
		escalations = append(escalations, "NEW: database access")
	}
	if !current.ConfigWrite && proposed.ConfigWrite {
		escalations = append(escalations, "NEW: config write access")
	}

	// Check new filesystem paths
	currentPaths := make(map[string]bool)
	for _, p := range current.Filesystem {
		currentPaths[p] = true
	}
	for _, p := range proposed.Filesystem {
		if !currentPaths[p] {
			escalations = append(escalations, fmt.Sprintf("NEW: filesystem access to %s", p))
		}
	}

	// Check new env vars
	currentVars := make(map[string]bool)
	for _, v := range current.EnvVars {
		currentVars[v] = true
	}
	for _, v := range proposed.EnvVars {
		if !currentVars[v] {
			escalations = append(escalations, fmt.Sprintf("NEW: environment variable %s", v))
		}
	}

	return escalations
}

// buildPluginEnv constructs the environment variables for a plugin subprocess.
// Only declared env vars are passed through. Always includes PATH and HOME.
func buildPluginEnv(perms Permissions) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}

	// Pass through declared env vars
	for _, v := range perms.EnvVars {
		if val := os.Getenv(v); val != "" {
			env = append(env, v+"="+val)
		}
	}

	// Pass mine paths if config_read is allowed
	if perms.ConfigRead {
		paths := config.GetPaths()
		env = append(env, "MINE_CONFIG_DIR="+paths.ConfigDir)
		env = append(env, "MINE_DATA_DIR="+paths.DataDir)
	}

	return env
}

// AuditLogPath returns the path to the plugin audit log.
func AuditLogPath() string {
	return filepath.Join(config.GetPaths().DataDir, "plugin-audit.log")
}

// AuditLog appends an entry to the plugin audit log.
func AuditLog(pluginName, action, detail string) error {
	path := AuditLogPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := fmt.Sprintf("%s plugin=%s action=%s %s\n",
		time.Now().UTC().Format(time.RFC3339),
		pluginName,
		action,
		detail,
	)

	_, err = f.WriteString(entry)
	return err
}
