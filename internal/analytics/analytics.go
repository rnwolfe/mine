// Package analytics provides lightweight, anonymous usage tracking for mine.
//
// Analytics are enabled by default and can be opted out via config:
//
//	mine config set analytics false
//
// Data collected: installation ID (random UUID), mine version, OS/arch,
// command name (not arguments), and date (day granularity). No PII is ever sent.
// Pings are fire-and-forget: non-blocking, fail silently, and deduplicated daily.
package analytics

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/rnwolfe/mine/internal/version"
)

// DefaultEndpoint is the analytics ingest URL.
// Override with MINE_ANALYTICS_ENDPOINT env var for testing.
const DefaultEndpoint = "https://analytics.mine.rwolfe.io/v1/events"

// Payload is the data sent to the analytics endpoint.
type Payload struct {
	InstallID string `json:"install_id"`
	Version   string `json:"version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Command   string `json:"command"`
	Date      string `json:"date"`
}

// client is a shared HTTP client with a short timeout to avoid
// blocking command execution.
var client = &http.Client{
	Timeout: 2 * time.Second,
}

// Ping sends an analytics event for the given command.
// It is designed to be called in a goroutine, for example:
//
//	go analytics.Ping(db, cmd, enabled, analytics.DefaultEndpoint)
//
// Ping is a no-op when:
//   - analytics is disabled in config
//   - the same command was already pinged today (daily dedup)
//   - any error occurs (fails silently)
func Ping(conn *sql.DB, command string, enabled bool, endpoint string) {
	if !enabled {
		return
	}

	today := time.Now().Format("2006-01-02")
	dedupKey := fmt.Sprintf("analytics:last_ping:%s", command)

	// Check daily dedup
	var lastPing string
	err := conn.QueryRow("SELECT value FROM kv WHERE key = ?", dedupKey).Scan(&lastPing)
	if err == nil && lastPing == today {
		return // Already pinged today
	}

	// Get or create installation ID
	installID, err := GetOrCreateID()
	if err != nil {
		return // Fail silently
	}

	payload := Payload{
		InstallID: installID,
		Version:   version.Short(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Command:   command,
		Date:      today,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return // Network error — fail silently
	}
	resp.Body.Close()

	// Only record dedup on success — transient server errors should allow retry
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return
	}

	// Record successful ping for dedup (upsert)
	_, _ = conn.Exec(
		`INSERT INTO kv (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		dedupKey, today,
	)
}

// ShouldShowNotice checks whether the one-time privacy notice needs to be displayed.
// Returns true if the notice has not been shown yet, false if it was already shown.
// Call MarkNoticeShown after actually displaying the notice.
func ShouldShowNotice(conn *sql.DB) bool {
	const noticeKey = "analytics:notice_shown"

	var shown string
	err := conn.QueryRow("SELECT value FROM kv WHERE key = ?", noticeKey).Scan(&shown)
	if err == nil && shown == "true" {
		return false
	}

	return true
}

// MarkNoticeShown records that the privacy notice has been displayed.
// Call this after actually printing the notice to avoid marking it shown
// before the user sees it.
func MarkNoticeShown(conn *sql.DB) {
	const noticeKey = "analytics:notice_shown"
	_, _ = conn.Exec(
		`INSERT INTO kv (key, value, updated_at) VALUES (?, 'true', CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = 'true', updated_at = CURRENT_TIMESTAMP`,
		noticeKey,
	)
}

// BuildPayload constructs an analytics payload without sending it.
// Exported for testing.
func BuildPayload(installID, command string) Payload {
	return Payload{
		InstallID: installID,
		Version:   version.Short(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Command:   command,
		Date:      time.Now().Format("2006-01-02"),
	}
}
