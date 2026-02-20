package cmd

import (
	"testing"

	"github.com/rnwolfe/mine/internal/config"
)

func TestResolveSystemInstructions(t *testing.T) {
	tests := []struct {
		name           string
		cfg            config.AIConfig
		subcommand     string
		flagValue      string
		flagChanged    bool
		builtinDefault string
		want           string
	}{
		{
			name:           "no custom config, flag not set — returns builtin default",
			cfg:            config.AIConfig{},
			subcommand:     "review",
			flagValue:      "",
			flagChanged:    false,
			builtinDefault: "built-in reviewer prompt",
			want:           "built-in reviewer prompt",
		},
		{
			name:           "flag set overrides all config",
			cfg:            config.AIConfig{SystemInstructions: "global", ReviewSystemInstructions: "review-config"},
			subcommand:     "review",
			flagValue:      "flag-override",
			flagChanged:    true,
			builtinDefault: "built-in",
			want:           "flag-override",
		},
		{
			name:           "flag set to empty string disables system instructions",
			cfg:            config.AIConfig{SystemInstructions: "global", ReviewSystemInstructions: "review-config"},
			subcommand:     "review",
			flagValue:      "",
			flagChanged:    true,
			builtinDefault: "built-in",
			want:           "",
		},
		{
			name:           "subcommand config takes precedence over global",
			cfg:            config.AIConfig{SystemInstructions: "global", AskSystemInstructions: "ask-config"},
			subcommand:     "ask",
			flagValue:      "",
			flagChanged:    false,
			builtinDefault: "",
			want:           "ask-config",
		},
		{
			name:           "global config used when no subcommand config",
			cfg:            config.AIConfig{SystemInstructions: "global"},
			subcommand:     "ask",
			flagValue:      "",
			flagChanged:    false,
			builtinDefault: "",
			want:           "global",
		},
		{
			name:           "global config used when subcommand config is empty",
			cfg:            config.AIConfig{SystemInstructions: "global", ReviewSystemInstructions: ""},
			subcommand:     "review",
			flagValue:      "",
			flagChanged:    false,
			builtinDefault: "built-in",
			want:           "global",
		},
		{
			name:           "review subcommand config overrides global and builtin",
			cfg:            config.AIConfig{SystemInstructions: "global", ReviewSystemInstructions: "review-config"},
			subcommand:     "review",
			flagValue:      "",
			flagChanged:    false,
			builtinDefault: "built-in",
			want:           "review-config",
		},
		{
			name:           "commit subcommand config overrides global and builtin",
			cfg:            config.AIConfig{SystemInstructions: "global", CommitSystemInstructions: "commit-config"},
			subcommand:     "commit",
			flagValue:      "",
			flagChanged:    false,
			builtinDefault: "built-in",
			want:           "commit-config",
		},
		{
			name:           "no config at all, no flag — returns empty for ask",
			cfg:            config.AIConfig{},
			subcommand:     "ask",
			flagValue:      "",
			flagChanged:    false,
			builtinDefault: "",
			want:           "",
		},
		{
			name:           "flag overrides subcommand config",
			cfg:            config.AIConfig{CommitSystemInstructions: "commit-config"},
			subcommand:     "commit",
			flagValue:      "flag-override",
			flagChanged:    true,
			builtinDefault: "built-in",
			want:           "flag-override",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSystemInstructions(&tt.cfg, tt.subcommand, tt.flagValue, tt.flagChanged, tt.builtinDefault)
			if got != tt.want {
				t.Errorf("resolveSystemInstructions() = %q, want %q", got, tt.want)
			}
		})
	}
}
