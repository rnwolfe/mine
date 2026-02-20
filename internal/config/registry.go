package config

import (
	"fmt"
	"sort"
	"strings"
)

// KeyType represents the data type of a config key.
type KeyType string

const (
	KeyTypeString KeyType = "string"
	KeyTypeInt    KeyType = "int"
	KeyTypeBool   KeyType = "bool"
)

// KeyEntry describes a known, settable config key.
type KeyEntry struct {
	// Type is the value's data type (string, int, bool).
	Type KeyType
	// Desc is a human-readable description shown in `mine config list`.
	Desc string
	// DefaultStr is the string representation of the default/zero value.
	DefaultStr string

	// get returns the current value as a string.
	get func(*Config) string
	// set validates and applies the value to cfg, returning an error on type mismatch.
	set func(cfg *Config, value string) error
	// unset resets the key to its schema default.
	unset func(cfg *Config)
}

// Get returns the current value of the key as a string.
func (e *KeyEntry) Get(cfg *Config) string { return e.get(cfg) }

// Set validates and sets the value, returning a descriptive error on type mismatch.
func (e *KeyEntry) Set(cfg *Config, value string) error { return e.set(cfg, value) }

// Unset resets the key to its schema default.
func (e *KeyEntry) Unset(cfg *Config) { e.unset(cfg) }

// SchemaKeys is the authoritative registry of all settable config keys.
// Keys use dot-notation matching the TOML section structure.
var SchemaKeys = map[string]*KeyEntry{
	"user.name": {
		Type:       KeyTypeString,
		Desc:       "Display name",
		DefaultStr: "",
		get:        func(cfg *Config) string { return cfg.User.Name },
		set:        func(cfg *Config, v string) error { cfg.User.Name = v; return nil },
		unset:      func(cfg *Config) { cfg.User.Name = "" },
	},
	"user.email": {
		Type:       KeyTypeString,
		Desc:       "Email address",
		DefaultStr: "",
		get:        func(cfg *Config) string { return cfg.User.Email },
		set:        func(cfg *Config, v string) error { cfg.User.Email = v; return nil },
		unset:      func(cfg *Config) { cfg.User.Email = "" },
	},
	"shell.default_shell": {
		Type:       KeyTypeString,
		Desc:       "Default shell path (e.g. /bin/bash)",
		DefaultStr: envOr("SHELL", "/bin/bash"),
		get:        func(cfg *Config) string { return cfg.Shell.DefaultShell },
		set:        func(cfg *Config, v string) error { cfg.Shell.DefaultShell = v; return nil },
		unset:      func(cfg *Config) { cfg.Shell.DefaultShell = envOr("SHELL", "/bin/bash") },
	},
	"ai.provider": {
		Type:       KeyTypeString,
		Desc:       "AI provider (claude, openai, gemini, openrouter)",
		DefaultStr: "claude",
		get:        func(cfg *Config) string { return cfg.AI.Provider },
		set:        func(cfg *Config, v string) error { cfg.AI.Provider = v; return nil },
		unset:      func(cfg *Config) { cfg.AI.Provider = "claude" },
	},
	"ai.model": {
		Type:       KeyTypeString,
		Desc:       "AI model name",
		DefaultStr: DefaultModel,
		get:        func(cfg *Config) string { return cfg.AI.Model },
		set:        func(cfg *Config, v string) error { cfg.AI.Model = v; return nil },
		unset:      func(cfg *Config) { cfg.AI.Model = DefaultModel },
	},
	"ai.system_instructions": {
		Type:       KeyTypeString,
		Desc:       "Default system instructions for all AI commands",
		DefaultStr: "",
		get:        func(cfg *Config) string { return cfg.AI.SystemInstructions },
		set:        func(cfg *Config, v string) error { cfg.AI.SystemInstructions = v; return nil },
		unset:      func(cfg *Config) { cfg.AI.SystemInstructions = "" },
	},
	"ai.ask_system_instructions": {
		Type:       KeyTypeString,
		Desc:       "System instructions for `mine ai ask`",
		DefaultStr: "",
		get:        func(cfg *Config) string { return cfg.AI.AskSystemInstructions },
		set:        func(cfg *Config, v string) error { cfg.AI.AskSystemInstructions = v; return nil },
		unset:      func(cfg *Config) { cfg.AI.AskSystemInstructions = "" },
	},
	"ai.review_system_instructions": {
		Type:       KeyTypeString,
		Desc:       "System instructions for `mine ai review`",
		DefaultStr: "",
		get:        func(cfg *Config) string { return cfg.AI.ReviewSystemInstructions },
		set:        func(cfg *Config, v string) error { cfg.AI.ReviewSystemInstructions = v; return nil },
		unset:      func(cfg *Config) { cfg.AI.ReviewSystemInstructions = "" },
	},
	"ai.commit_system_instructions": {
		Type:       KeyTypeString,
		Desc:       "System instructions for `mine ai commit`",
		DefaultStr: "",
		get:        func(cfg *Config) string { return cfg.AI.CommitSystemInstructions },
		set:        func(cfg *Config, v string) error { cfg.AI.CommitSystemInstructions = v; return nil },
		unset:      func(cfg *Config) { cfg.AI.CommitSystemInstructions = "" },
	},
	"analytics": {
		Type:       KeyTypeBool,
		Desc:       "Enable anonymous usage analytics",
		DefaultStr: "true",
		get:        func(cfg *Config) string { return fmt.Sprintf("%t", cfg.Analytics.IsEnabled()) },
		set: func(cfg *Config, v string) error {
			b, err := ParseBoolValue(v)
			if err != nil {
				return fmt.Errorf("invalid value %q for analytics: %w", v, err)
			}
			cfg.Analytics.Enabled = BoolPtr(b)
			return nil
		},
		unset: func(cfg *Config) { cfg.Analytics.Enabled = BoolPtr(true) },
	},
}

// ValidKeyNames returns the sorted list of all known config key names.
func ValidKeyNames() []string {
	names := make([]string, 0, len(SchemaKeys))
	for k := range SchemaKeys {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// LookupKey returns the KeyEntry for a known config key.
func LookupKey(key string) (*KeyEntry, bool) {
	entry, ok := SchemaKeys[key]
	return entry, ok
}

// ParseBoolValue accepts common boolean string representations.
// Valid truthy values: true, 1, yes, on.
// Valid falsy values: false, 0, no, off.
func ParseBoolValue(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("not a boolean: %q (use one of: true/false, 1/0, yes/no, on/off)", s)
	}
}
