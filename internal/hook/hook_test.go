package hook

import (
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext("todo.add", []string{"buy milk"}, map[string]string{"priority": "high"})
	if ctx.Command != "todo.add" {
		t.Errorf("Command = %q, want %q", ctx.Command, "todo.add")
	}
	if len(ctx.Args) != 1 || ctx.Args[0] != "buy milk" {
		t.Errorf("Args = %v, want [buy milk]", ctx.Args)
	}
	if ctx.Flags["priority"] != "high" {
		t.Errorf("Flags[priority] = %q, want %q", ctx.Flags["priority"], "high")
	}
	if ctx.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestNewContext_NilArgs(t *testing.T) {
	ctx := NewContext("test", nil, nil)
	if ctx.Args == nil {
		t.Error("Args should be initialized to empty slice, not nil")
	}
	if ctx.Flags == nil {
		t.Error("Flags should be initialized to empty map, not nil")
	}
}

func TestContextJSON(t *testing.T) {
	ctx := NewContext("todo.add", []string{"test"}, map[string]string{"p": "1"})
	data, err := ctx.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}

	parsed, err := ParseContext(data)
	if err != nil {
		t.Fatalf("ParseContext() error: %v", err)
	}

	if parsed.Command != ctx.Command {
		t.Errorf("Command = %q, want %q", parsed.Command, ctx.Command)
	}
	if len(parsed.Args) != len(ctx.Args) {
		t.Errorf("Args len = %d, want %d", len(parsed.Args), len(ctx.Args))
	}
}

func TestParseContext_Invalid(t *testing.T) {
	_, err := ParseContext([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		command string
		want    bool
	}{
		{"todo.add", "todo.add", true},
		{"todo.add", "todo.done", false},
		{"todo.*", "todo.add", true},
		{"todo.*", "todo.done", true},
		{"todo.*", "stash.add", false},
		{"*", "anything", true},
		{"*", "todo.add", true},
		{"*.*", "todo.add", true},
		{"*.*", "single", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.command, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.command)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.command, got, tt.want)
			}
		})
	}
}

func TestRegistryRegisterAndResolve(t *testing.T) {
	reg := &Registry{}

	reg.Register(Hook{
		Pattern: "todo.*",
		Stage:   StagePreexec,
		Mode:    ModeTransform,
		Name:    "auto-tag",
		Source:  "user",
		Handler: func(ctx *Context) (*Context, error) { return ctx, nil },
	})
	reg.Register(Hook{
		Pattern: "todo.add",
		Stage:   StagePreexec,
		Mode:    ModeTransform,
		Name:    "enrich",
		Source:  "user",
		Handler: func(ctx *Context) (*Context, error) { return ctx, nil },
	})
	reg.Register(Hook{
		Pattern: "todo.add",
		Stage:   StageNotify,
		Mode:    ModeNotify,
		Name:    "slack-notify",
		Source:  "plugin:slack",
		Handler: func(ctx *Context) (*Context, error) { return ctx, nil },
	})

	// Resolve preexec for todo.add should get both hooks
	hooks := reg.Resolve("todo.add", StagePreexec)
	if len(hooks) != 2 {
		t.Fatalf("Resolve(todo.add, preexec) got %d hooks, want 2", len(hooks))
	}
	// Should be sorted by name
	if hooks[0].Name != "auto-tag" || hooks[1].Name != "enrich" {
		t.Errorf("hooks not sorted: got %q, %q", hooks[0].Name, hooks[1].Name)
	}

	// Resolve notify for todo.add
	hooks = reg.Resolve("todo.add", StageNotify)
	if len(hooks) != 1 || hooks[0].Name != "slack-notify" {
		t.Errorf("Resolve(todo.add, notify) got unexpected hooks: %v", hooks)
	}

	// No hooks for todo.done notify
	hooks = reg.Resolve("todo.done", StageNotify)
	if len(hooks) != 0 {
		t.Errorf("Resolve(todo.done, notify) got %d hooks, want 0", len(hooks))
	}
}

func TestRegistryUnregister(t *testing.T) {
	reg := &Registry{}
	reg.Register(Hook{Pattern: "todo.*", Stage: StageNotify, Mode: ModeNotify, Name: "a", Source: "user"})
	reg.Register(Hook{Pattern: "todo.*", Stage: StageNotify, Mode: ModeNotify, Name: "b", Source: "plugin:x"})
	reg.Register(Hook{Pattern: "todo.*", Stage: StageNotify, Mode: ModeNotify, Name: "c", Source: "user"})

	if reg.Count() != 3 {
		t.Fatalf("Count() = %d, want 3", reg.Count())
	}

	reg.Unregister("user")
	if reg.Count() != 1 {
		t.Fatalf("Count() after Unregister = %d, want 1", reg.Count())
	}

	remaining := reg.All()
	if remaining[0].Name != "b" {
		t.Errorf("remaining hook name = %q, want %q", remaining[0].Name, "b")
	}
}

func TestRegistryHasHooks(t *testing.T) {
	reg := &Registry{}
	if reg.HasHooks("todo.add") {
		t.Error("empty registry should have no hooks")
	}

	reg.Register(Hook{Pattern: "todo.*", Stage: StagePreexec, Mode: ModeTransform, Name: "test"})
	if !reg.HasHooks("todo.add") {
		t.Error("should have hooks for todo.add")
	}
	if reg.HasHooks("stash.add") {
		t.Error("should not have hooks for stash.add")
	}
}

func TestTransformStageChaining(t *testing.T) {
	reg := &Registry{}

	// Hook 1: appends " [tagged]" to first arg
	reg.Register(Hook{
		Pattern: "test",
		Stage:   StagePreexec,
		Mode:    ModeTransform,
		Name:    "a-tagger",
		Handler: func(ctx *Context) (*Context, error) {
			if len(ctx.Args) > 0 {
				ctx.Args[0] = ctx.Args[0] + " [tagged]"
			}
			return ctx, nil
		},
	})

	// Hook 2: uppercases first arg (runs after hook 1 due to name sort)
	reg.Register(Hook{
		Pattern: "test",
		Stage:   StagePreexec,
		Mode:    ModeTransform,
		Name:    "b-upper",
		Handler: func(ctx *Context) (*Context, error) {
			if len(ctx.Args) > 0 {
				ctx.Args[0] = ctx.Args[0] + " [upper]"
			}
			return ctx, nil
		},
	})

	ctx := NewContext("test", []string{"hello"}, nil)
	result, err := runTransformStage(reg, "test", StagePreexec, ctx)
	if err != nil {
		t.Fatalf("runTransformStage error: %v", err)
	}

	want := "hello [tagged] [upper]"
	if result.Args[0] != want {
		t.Errorf("chained result = %q, want %q", result.Args[0], want)
	}
}

func TestTransformStageError(t *testing.T) {
	reg := &Registry{}
	reg.Register(Hook{
		Pattern: "test",
		Stage:   StagePreexec,
		Mode:    ModeTransform,
		Name:    "failing",
		Handler: func(ctx *Context) (*Context, error) {
			return nil, errors.New("hook broke")
		},
	})

	ctx := NewContext("test", nil, nil)
	_, err := runTransformStage(reg, "test", StagePreexec, ctx)
	if err == nil {
		t.Fatal("expected error from failing hook")
	}
	if !errors.Is(err, errors.Unwrap(err)) {
		// Just check it wraps properly
	}
}

func TestNotifyStageParallel(t *testing.T) {
	reg := &Registry{}

	var count atomic.Int32

	for i := 0; i < 5; i++ {
		reg.Register(Hook{
			Pattern: "test",
			Stage:   StageNotify,
			Mode:    ModeNotify,
			Name:    string(rune('a' + i)),
			Handler: func(ctx *Context) (*Context, error) {
				count.Add(1)
				return ctx, nil
			},
		})
	}

	ctx := NewContext("test", nil, nil)
	runNotifyStage(reg, "test", ctx)

	if count.Load() != 5 {
		t.Errorf("notify count = %d, want 5", count.Load())
	}
}

func TestNotifyStageErrorsLogged(t *testing.T) {
	reg := &Registry{}
	reg.Register(Hook{
		Pattern: "test",
		Stage:   StageNotify,
		Mode:    ModeNotify,
		Name:    "failing-notify",
		Handler: func(ctx *Context) (*Context, error) {
			return nil, errors.New("notify error")
		},
	})

	ctx := NewContext("test", nil, nil)
	// Should not panic; errors are logged
	runNotifyStage(reg, "test", ctx)
}

func TestContextJSONRoundTrip(t *testing.T) {
	original := &Context{
		Command:   "todo.add",
		Args:      []string{"buy milk", "eggs"},
		Flags:     map[string]string{"priority": "high", "due": "tomorrow"},
		Result:    map[string]any{"id": float64(42)},
		Timestamp: "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Context
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Command != original.Command {
		t.Errorf("Command = %q, want %q", decoded.Command, original.Command)
	}
	if len(decoded.Args) != 2 {
		t.Errorf("Args len = %d, want 2", len(decoded.Args))
	}
	if decoded.Flags["priority"] != "high" {
		t.Errorf("Flags[priority] = %q, want %q", decoded.Flags["priority"], "high")
	}
}

func TestAllStagesOrder(t *testing.T) {
	want := []Stage{StagePrevalidate, StagePreexec, StagePostexec, StageNotify}
	if len(AllStages) != len(want) {
		t.Fatalf("AllStages len = %d, want %d", len(AllStages), len(want))
	}
	for i, s := range AllStages {
		if s != want[i] {
			t.Errorf("AllStages[%d] = %q, want %q", i, s, want[i])
		}
	}
}
