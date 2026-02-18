package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- parseKeyValue ---

func TestParseKeyValue_SpaceSeparated(t *testing.T) {
	key, val := parseKeyValue("HostName 192.168.1.1")
	if key != "HostName" {
		t.Fatalf("expected key 'HostName', got %q", key)
	}
	if val != "192.168.1.1" {
		t.Fatalf("expected val '192.168.1.1', got %q", val)
	}
}

func TestParseKeyValue_EqualsSeparated(t *testing.T) {
	key, val := parseKeyValue("User=ryan")
	if key != "User" {
		t.Fatalf("expected key 'User', got %q", key)
	}
	if val != "ryan" {
		t.Fatalf("expected val 'ryan', got %q", val)
	}
}

func TestParseKeyValue_Comment(t *testing.T) {
	key, val := parseKeyValue("# this is a comment")
	if key != "" || val != "" {
		t.Fatalf("expected empty key/val for comment, got %q %q", key, val)
	}
}

func TestParseKeyValue_Empty(t *testing.T) {
	key, val := parseKeyValue("")
	if key != "" || val != "" {
		t.Fatalf("expected empty key/val for empty line, got %q %q", key, val)
	}
}

func TestParseKeyValue_InlineComment(t *testing.T) {
	_, val := parseKeyValue("Port 2222 # non-standard")
	if val != "2222" {
		t.Fatalf("expected '2222', got %q", val)
	}
}

// --- parseConfig ---

func TestParseConfig_Basic(t *testing.T) {
	config := `
Host myserver
    HostName 10.0.0.1
    User admin
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
`
	hosts, err := parseConfig(strings.NewReader(config))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	h := hosts[0]
	if h.Alias != "myserver" {
		t.Fatalf("expected alias 'myserver', got %q", h.Alias)
	}
	if h.Hostname != "10.0.0.1" {
		t.Fatalf("expected hostname '10.0.0.1', got %q", h.Hostname)
	}
	if h.User != "admin" {
		t.Fatalf("expected user 'admin', got %q", h.User)
	}
	if h.Port != "2222" {
		t.Fatalf("expected port '2222', got %q", h.Port)
	}
	if h.KeyFile != "~/.ssh/id_ed25519" {
		t.Fatalf("expected keyfile '~/.ssh/id_ed25519', got %q", h.KeyFile)
	}
}

func TestParseConfig_MultipleHosts(t *testing.T) {
	config := `
Host web
    HostName web.example.com
    User deploy

Host db
    HostName 10.0.0.5
    Port 2222
`
	hosts, err := parseConfig(strings.NewReader(config))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Alias != "web" {
		t.Fatalf("expected 'web', got %q", hosts[0].Alias)
	}
	if hosts[1].Alias != "db" {
		t.Fatalf("expected 'db', got %q", hosts[1].Alias)
	}
}

func TestParseConfig_SkipsWildcard(t *testing.T) {
	config := `
Host *
    ServerAliveInterval 60

Host myserver
    HostName 10.0.0.1
`
	hosts, err := parseConfig(strings.NewReader(config))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host (wildcard skipped), got %d", len(hosts))
	}
	if hosts[0].Alias != "myserver" {
		t.Fatalf("expected 'myserver', got %q", hosts[0].Alias)
	}
}

func TestParseConfig_HandlesComments(t *testing.T) {
	config := `
# global comment

Host webserver
    # inline comment
    HostName web.example.com
    User ubuntu
`
	hosts, err := parseConfig(strings.NewReader(config))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].Hostname != "web.example.com" {
		t.Fatalf("expected hostname 'web.example.com', got %q", hosts[0].Hostname)
	}
}

func TestParseConfig_Empty(t *testing.T) {
	hosts, err := parseConfig(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hosts != nil {
		t.Fatalf("expected nil for empty config, got %v", hosts)
	}
}

func TestParseConfig_CaseInsensitiveKeys(t *testing.T) {
	config := `
Host testhost
    hostname myhost.example.com
    user testuser
    port 9922
`
	hosts, err := parseConfig(strings.NewReader(config))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	h := hosts[0]
	if h.Hostname != "myhost.example.com" {
		t.Fatalf("expected hostname, got %q", h.Hostname)
	}
	if h.User != "testuser" {
		t.Fatalf("expected user 'testuser', got %q", h.User)
	}
	if h.Port != "9922" {
		t.Fatalf("expected port '9922', got %q", h.Port)
	}
}

// --- Host.Description ---

func TestHostDescription_Full(t *testing.T) {
	h := Host{
		Alias:    "myserver",
		Hostname: "10.0.0.1",
		User:     "ubuntu",
		Port:     "2222",
	}
	desc := h.Description()
	if !strings.Contains(desc, "10.0.0.1") {
		t.Fatalf("expected hostname in description, got %q", desc)
	}
	if !strings.Contains(desc, "ubuntu@") {
		t.Fatalf("expected user in description, got %q", desc)
	}
	if !strings.Contains(desc, ":2222") {
		t.Fatalf("expected port in description, got %q", desc)
	}
}

func TestHostDescription_DefaultPort(t *testing.T) {
	h := Host{
		Alias:    "myserver",
		Hostname: "10.0.0.1",
		Port:     "22",
	}
	desc := h.Description()
	if strings.Contains(desc, ":22") {
		t.Fatalf("default port 22 should not appear in description, got %q", desc)
	}
}

func TestHostDescription_Empty(t *testing.T) {
	h := Host{Alias: "bare"}
	desc := h.Description()
	if desc != "" {
		t.Fatalf("expected empty description for bare host, got %q", desc)
	}
}

// --- tui.Item interface ---

func TestHostFilterValue(t *testing.T) {
	h := Host{Alias: "myserver"}
	if h.FilterValue() != "myserver" {
		t.Fatalf("FilterValue should return alias, got %q", h.FilterValue())
	}
}

func TestHostTitle(t *testing.T) {
	h := Host{Alias: "myserver"}
	if h.Title() != "myserver" {
		t.Fatalf("Title should return alias, got %q", h.Title())
	}
}

// --- FindHost ---

func TestFindHost_Found(t *testing.T) {
	hosts := []Host{
		{Alias: "web"},
		{Alias: "db"},
	}
	h, err := FindHost("db", hosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Alias != "db" {
		t.Fatalf("expected 'db', got %q", h.Alias)
	}
}

func TestFindHost_CaseInsensitive(t *testing.T) {
	hosts := []Host{{Alias: "MyServer"}}
	h, err := FindHost("myserver", hosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Alias != "MyServer" {
		t.Fatalf("expected 'MyServer', got %q", h.Alias)
	}
}

func TestFindHost_NotFound(t *testing.T) {
	hosts := []Host{{Alias: "web"}}
	_, err := FindHost("unknown", hosts)
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

// --- formatHostBlock ---

func TestFormatHostBlock_Full(t *testing.T) {
	h := Host{
		Alias:    "myserver",
		Hostname: "10.0.0.1",
		User:     "ubuntu",
		Port:     "2222",
		KeyFile:  "~/.ssh/id_ed25519",
	}
	block := formatHostBlock(h)
	if !strings.Contains(block, "Host myserver") {
		t.Fatalf("expected 'Host myserver' in block, got:\n%s", block)
	}
	if !strings.Contains(block, "HostName 10.0.0.1") {
		t.Fatalf("expected HostName in block")
	}
	if !strings.Contains(block, "User ubuntu") {
		t.Fatalf("expected User in block")
	}
	if !strings.Contains(block, "Port 2222") {
		t.Fatalf("expected Port in block")
	}
	if !strings.Contains(block, "IdentityFile ~/.ssh/id_ed25519") {
		t.Fatalf("expected IdentityFile in block")
	}
}

func TestFormatHostBlock_MinimalAlias(t *testing.T) {
	h := Host{Alias: "bare"}
	block := formatHostBlock(h)
	if !strings.Contains(block, "Host bare") {
		t.Fatalf("expected 'Host bare', got:\n%s", block)
	}
	// Should not have empty fields
	if strings.Contains(block, "HostName") {
		t.Fatalf("unexpected HostName in bare block")
	}
}

// --- AppendHostTo / RemoveHostFrom ---

func TestAppendAndRemoveHost(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	h := Host{
		Alias:    "testhost",
		Hostname: "192.168.1.100",
		User:     "ubuntu",
	}

	// Append
	if err := AppendHostTo(configPath, h); err != nil {
		t.Fatalf("AppendHostTo failed: %v", err)
	}

	// Verify it appears
	hosts, err := ReadHostsFrom(configPath)
	if err != nil {
		t.Fatalf("ReadHostsFrom failed: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host after append, got %d", len(hosts))
	}
	if hosts[0].Alias != "testhost" {
		t.Fatalf("expected 'testhost', got %q", hosts[0].Alias)
	}

	// Remove
	if err := RemoveHostFrom(configPath, "testhost"); err != nil {
		t.Fatalf("RemoveHostFrom failed: %v", err)
	}

	// Verify it's gone
	hosts, err = ReadHostsFrom(configPath)
	if err != nil {
		t.Fatalf("ReadHostsFrom failed after remove: %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("expected 0 hosts after remove, got %d", len(hosts))
	}
}

func TestRemoveHostFrom_NotFound(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	// Write a config with a different host
	if err := os.WriteFile(configPath, []byte("Host other\n    HostName 10.0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := RemoveHostFrom(configPath, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent host")
	}
}

func TestRemoveHostFrom_NoConfig(t *testing.T) {
	err := RemoveHostFrom("/nonexistent/path/config", "anyhost")
	if err == nil {
		t.Fatal("expected error when config doesn't exist")
	}
}

// --- removeHostBlock ---

func TestRemoveHostBlock_PreservesOtherHosts(t *testing.T) {
	content := `
Host web
    HostName web.example.com
    User deploy

Host db
    HostName 10.0.0.5
    Port 2222

Host cache
    HostName cache.internal
`
	result, removed := removeHostBlock(content, "db")
	if !removed {
		t.Fatal("expected removed=true")
	}

	// web and cache should remain
	hosts, err := parseConfig(strings.NewReader(result))
	if err != nil {
		t.Fatalf("parsing after remove: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts after removing 'db', got %d", len(hosts))
	}
	names := []string{hosts[0].Alias, hosts[1].Alias}
	for _, n := range names {
		if n == "db" {
			t.Fatal("db should have been removed")
		}
	}
}

func TestRemoveHostBlock_NotFound(t *testing.T) {
	content := "Host web\n    HostName web.example.com\n"
	_, removed := removeHostBlock(content, "nonexistent")
	if removed {
		t.Fatal("expected removed=false for nonexistent host")
	}
}

// --- ReadHostsFrom missing file ---

func TestReadHostsFrom_MissingFile(t *testing.T) {
	hosts, err := ReadHostsFrom("/nonexistent/path/config")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if hosts != nil {
		t.Fatalf("expected nil hosts for missing file")
	}
}
