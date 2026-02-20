package config

import (
	"sort"
	"testing"
)

func TestValidKeyNames_NonEmpty(t *testing.T) {
	names := ValidKeyNames()
	if len(names) == 0 {
		t.Fatal("expected non-empty key list")
	}
}

func TestValidKeyNames_Sorted(t *testing.T) {
	names := ValidKeyNames()
	if !sort.StringsAreSorted(names) {
		t.Fatalf("expected sorted key names, got %v", names)
	}
}

func TestValidKeyNames_ContainsKnownKeys(t *testing.T) {
	expected := []string{"analytics", "ai.model", "ai.provider", "user.name", "user.email"}
	names := ValidKeyNames()
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	for _, want := range expected {
		if !nameSet[want] {
			t.Errorf("ValidKeyNames missing expected key %q", want)
		}
	}
}

func TestLookupKey_Known(t *testing.T) {
	entry, ok := LookupKey("user.name")
	if !ok {
		t.Fatal("expected user.name to be found")
	}
	if entry.Type != KeyTypeString {
		t.Fatalf("expected string type for user.name, got %q", entry.Type)
	}
}

func TestLookupKey_Unknown(t *testing.T) {
	_, ok := LookupKey("not.a.real.key")
	if ok {
		t.Fatal("expected unknown key to return false")
	}
}

func TestParseBoolValue_TrueVariants(t *testing.T) {
	for _, v := range []string{"true", "1", "yes", "on", "TRUE", "YES", "On"} {
		b, err := ParseBoolValue(v)
		if err != nil {
			t.Errorf("ParseBoolValue(%q): unexpected error: %v", v, err)
		}
		if !b {
			t.Errorf("ParseBoolValue(%q): expected true", v)
		}
	}
}

func TestParseBoolValue_FalseVariants(t *testing.T) {
	for _, v := range []string{"false", "0", "no", "off", "FALSE", "NO", "Off"} {
		b, err := ParseBoolValue(v)
		if err != nil {
			t.Errorf("ParseBoolValue(%q): unexpected error: %v", v, err)
		}
		if b {
			t.Errorf("ParseBoolValue(%q): expected false", v)
		}
	}
}

func TestParseBoolValue_Invalid(t *testing.T) {
	for _, v := range []string{"maybe", "yep", "nope", "", "2", "tru"} {
		_, err := ParseBoolValue(v)
		if err == nil {
			t.Errorf("ParseBoolValue(%q): expected error for invalid bool", v)
		}
	}
}

func TestSetGetUnset_StringKey(t *testing.T) {
	cfg := &Config{}
	entry, ok := LookupKey("user.name")
	if !ok {
		t.Fatal("user.name not found in registry")
	}

	if err := entry.Set(cfg, "Alice"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := entry.Get(cfg); got != "Alice" {
		t.Fatalf("Get: expected 'Alice', got %q", got)
	}

	entry.Unset(cfg)
	if got := entry.Get(cfg); got != "" {
		t.Fatalf("Unset: expected '', got %q", got)
	}
}

func TestSetGetUnset_BoolKey(t *testing.T) {
	cfg := &Config{Analytics: AnalyticsConfig{Enabled: BoolPtr(true)}}
	entry, ok := LookupKey("analytics")
	if !ok {
		t.Fatal("analytics not found in registry")
	}

	if err := entry.Set(cfg, "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := entry.Get(cfg); got != "false" {
		t.Fatalf("Get: expected 'false', got %q", got)
	}

	entry.Unset(cfg)
	if got := entry.Get(cfg); got != "true" {
		t.Fatalf("Unset: expected 'true', got %q", got)
	}
}

func TestSet_BoolInvalidType(t *testing.T) {
	cfg := &Config{}
	entry, ok := LookupKey("analytics")
	if !ok {
		t.Fatal("analytics not found in registry")
	}

	err := entry.Set(cfg, "notabool")
	if err == nil {
		t.Fatal("expected error for invalid bool value")
	}
}

func TestSetGetUnset_AIProvider(t *testing.T) {
	cfg := &Config{AI: AIConfig{Provider: "claude"}}
	entry, ok := LookupKey("ai.provider")
	if !ok {
		t.Fatal("ai.provider not found in registry")
	}

	if err := entry.Set(cfg, "openai"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := entry.Get(cfg); got != "openai" {
		t.Fatalf("Get: expected 'openai', got %q", got)
	}

	entry.Unset(cfg)
	if got := entry.Get(cfg); got != "claude" {
		t.Fatalf("Unset: expected 'claude', got %q", got)
	}
}

func TestSetGetUnset_AISystemInstructions(t *testing.T) {
	cfg := &Config{}
	entry, ok := LookupKey("ai.system_instructions")
	if !ok {
		t.Fatal("ai.system_instructions not found in registry")
	}

	if err := entry.Set(cfg, "Always respond concisely."); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := entry.Get(cfg); got != "Always respond concisely." {
		t.Fatalf("Get: expected instructions, got %q", got)
	}

	entry.Unset(cfg)
	if got := entry.Get(cfg); got != "" {
		t.Fatalf("Unset: expected empty string, got %q", got)
	}
}

func TestAllSchemaKeys_GetSetUnsetDoNotPanic(t *testing.T) {
	cfg := defaultConfig()
	for key, entry := range SchemaKeys {
		// Verify Get doesn't panic.
		_ = entry.Get(cfg)

		// Verify Unset doesn't panic.
		entry.Unset(cfg)

		// Verify Get after Unset doesn't panic.
		_ = entry.Get(cfg)

		// Verify Set with the default doesn't fail for string keys.
		if entry.Type == KeyTypeString {
			if err := entry.Set(cfg, entry.DefaultStr); err != nil {
				t.Errorf("key %q: Set with default value %q failed: %v", key, entry.DefaultStr, err)
			}
		}
	}
}

func TestAllSchemaKeys_HaveDesc(t *testing.T) {
	for key, entry := range SchemaKeys {
		if entry.Desc == "" {
			t.Errorf("key %q has empty Desc", key)
		}
	}
}

func TestAllSchemaKeys_HaveValidType(t *testing.T) {
	for key, entry := range SchemaKeys {
		switch entry.Type {
		case KeyTypeString, KeyTypeInt, KeyTypeBool:
			// valid
		default:
			t.Errorf("key %q has invalid Type %q", key, entry.Type)
		}
	}
}

func TestKeyEntry_DotNotation(t *testing.T) {
	// Verify all known keys use dot-notation or are at the top level.
	for key := range SchemaKeys {
		if key == "" {
			t.Error("found empty key in SchemaKeys")
		}
	}
}

func TestRoundTrip_UserEmail(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")

	entry, ok := LookupKey("user.email")
	if !ok {
		t.Fatal("user.email not found")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := entry.Set(cfg, "alice@example.com"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}

	if got := entry.Get(loaded); got != "alice@example.com" {
		t.Fatalf("round-trip failed: expected 'alice@example.com', got %q", got)
	}
}
