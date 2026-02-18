package ssh

import (
	"testing"
)

// --- ParsePortSpec ---

func TestParsePortSpec_Simple(t *testing.T) {
	local, remoteAddr, err := ParsePortSpec("8080:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if local != "8080" {
		t.Fatalf("expected local '8080', got %q", local)
	}
	// For "local:port" form, remoteAddr should be "localhost:port".
	if remoteAddr != "localhost:80" {
		t.Fatalf("expected remoteAddr 'localhost:80', got %q", remoteAddr)
	}
}

func TestParsePortSpec_WithHost(t *testing.T) {
	local, remoteAddr, err := ParsePortSpec("5433:db.internal:5432")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if local != "5433" {
		t.Fatalf("expected local '5433', got %q", local)
	}
	// For "local:host:port" form, remoteAddr should be "host:port".
	if remoteAddr != "db.internal:5432" {
		t.Fatalf("expected remoteAddr 'db.internal:5432', got %q", remoteAddr)
	}
	// Verify that the full PortSpec is constructed correctly via internal parser.
	ps, err := parsePortSpec("5433:db.internal:5432")
	if err != nil {
		t.Fatalf("unexpected error from parsePortSpec: %v", err)
	}
	if ps.RemoteHost != "db.internal" {
		t.Fatalf("expected remote host 'db.internal', got %q", ps.RemoteHost)
	}
	if ps.RemotePort != "5432" {
		t.Fatalf("expected remote port '5432', got %q", ps.RemotePort)
	}
}

func TestParsePortSpec_Empty(t *testing.T) {
	_, _, err := ParsePortSpec("")
	if err == nil {
		t.Fatal("expected error for empty spec")
	}
}

func TestParsePortSpec_NoColon(t *testing.T) {
	_, _, err := ParsePortSpec("8080")
	if err == nil {
		t.Fatal("expected error for missing colon")
	}
}

func TestParsePortSpec_MissingLocal(t *testing.T) {
	_, _, err := ParsePortSpec(":80")
	if err == nil {
		t.Fatal("expected error for missing local port")
	}
}

func TestParsePortSpec_MissingRemote(t *testing.T) {
	_, _, err := ParsePortSpec("8080:")
	if err == nil {
		t.Fatal("expected error for missing remote port")
	}
}

// --- StartTunnel (stubbed) ---

func TestStartTunnel_Stubbed(t *testing.T) {
	original := StartTunnelFunc
	defer func() { StartTunnelFunc = original }()

	var gotAlias, gotSpec string
	StartTunnelFunc = func(alias, portSpec string) error {
		gotAlias = alias
		gotSpec = portSpec
		return nil
	}

	if err := StartTunnel("myserver", "8080:80"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAlias != "myserver" {
		t.Fatalf("expected alias 'myserver', got %q", gotAlias)
	}
	if gotSpec != "8080:80" {
		t.Fatalf("expected spec '8080:80', got %q", gotSpec)
	}
}
