package cmd

import (
	"sync/atomic"
	"testing"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/spf13/cobra"
)

// hookWrapTestEnv sets up an isolated XDG environment for the test.
func hookWrapTestEnv(t *testing.T) {
	t.Helper()
	configTestEnv(t)
}

// makeCounter returns a hook.Handler that increments an atomic counter on each call.
func makeCounter(n *atomic.Int32) hook.Handler {
	return func(ctx *hook.Context) (*hook.Context, error) {
		n.Add(1)
		return ctx, nil
	}
}

// stubCmd returns a minimal cobra.Command suitable for passing to wrapped handlers in tests.
func stubCmd() *cobra.Command {
	return &cobra.Command{Use: "test"}
}

func TestHookWrap_VersionFiresHook(t *testing.T) {
	hookWrapTestEnv(t)

	reg := &hook.Registry{}
	var called atomic.Int32
	if err := reg.Register(hook.Hook{
		Pattern: "version",
		Stage:   hook.StagePreexec,
		Mode:    hook.ModeTransform,
		Name:    "test-version-hook",
		Source:  "test",
		Handler: makeCounter(&called),
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	wrapped := hook.WrapWith(reg, "version", runVersion)
	if err := wrapped(stubCmd(), nil); err != nil {
		t.Fatalf("wrapped runVersion: %v", err)
	}

	if called.Load() != 1 {
		t.Errorf("hook called %d times, want 1", called.Load())
	}
}

func TestHookWrap_AboutFiresHook(t *testing.T) {
	hookWrapTestEnv(t)

	reg := &hook.Registry{}
	var called atomic.Int32
	if err := reg.Register(hook.Hook{
		Pattern: "about",
		Stage:   hook.StagePreexec,
		Mode:    hook.ModeTransform,
		Name:    "test-about-hook",
		Source:  "test",
		Handler: makeCounter(&called),
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	wrapped := hook.WrapWith(reg, "about", runAbout)
	if err := wrapped(stubCmd(), nil); err != nil {
		t.Fatalf("wrapped runAbout: %v", err)
	}

	if called.Load() != 1 {
		t.Errorf("hook called %d times, want 1", called.Load())
	}
}

func TestHookWrap_PluginListFiresHook(t *testing.T) {
	hookWrapTestEnv(t)

	reg := &hook.Registry{}
	var called atomic.Int32
	if err := reg.Register(hook.Hook{
		Pattern: "plugin.list",
		Stage:   hook.StagePreexec,
		Mode:    hook.ModeTransform,
		Name:    "test-plugin-list-hook",
		Source:  "test",
		Handler: makeCounter(&called),
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	wrapped := hook.WrapWith(reg, "plugin.list", runPluginList)
	if err := wrapped(stubCmd(), nil); err != nil {
		t.Fatalf("wrapped runPluginList: %v", err)
	}

	if called.Load() != 1 {
		t.Errorf("hook called %d times, want 1", called.Load())
	}
}

func TestHookWrap_HookListFiresHook(t *testing.T) {
	hookWrapTestEnv(t)

	reg := &hook.Registry{}
	var called atomic.Int32
	if err := reg.Register(hook.Hook{
		Pattern: "hook.list",
		Stage:   hook.StagePreexec,
		Mode:    hook.ModeTransform,
		Name:    "test-hook-list-hook",
		Source:  "test",
		Handler: makeCounter(&called),
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	wrapped := hook.WrapWith(reg, "hook.list", runHookList)
	if err := wrapped(stubCmd(), nil); err != nil {
		t.Fatalf("wrapped runHookList: %v", err)
	}

	if called.Load() != 1 {
		t.Errorf("hook called %d times, want 1", called.Load())
	}
}

func TestHookWrap_NoHooksRegistered_StillRuns(t *testing.T) {
	hookWrapTestEnv(t)

	// Empty registry — fast path must still execute the command without hooks.
	reg := &hook.Registry{}

	wrapped := hook.WrapWith(reg, "version", runVersion)
	if err := wrapped(stubCmd(), nil); err != nil {
		t.Fatalf("wrapped runVersion (no hooks): %v", err)
	}
}
