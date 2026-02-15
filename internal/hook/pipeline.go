package hook

import (
	"fmt"
	"log"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Wrap wraps a Cobra RunE function with the hook pipeline.
// When no hooks are registered for the command, this is a zero-cost no-op.
//
// Usage:
//
//	var myCmd = &cobra.Command{RunE: hook.Wrap("todo.add", runTodoAdd)}
func Wrap(command string, fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return WrapWith(DefaultRegistry, command, fn)
}

// WrapWith wraps a Cobra RunE function using a specific registry.
func WrapWith(reg *Registry, command string, fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Fast path: no hooks registered at all
		if reg.Count() == 0 {
			return fn(cmd, args)
		}

		// No hooks for this specific command
		if !reg.HasHooks(command) {
			return fn(cmd, args)
		}

		// Build context from command invocation
		flags := extractFlags(cmd)
		ctx := NewContext(command, args, flags)

		// Stage 1: prevalidate (transform)
		var err error
		ctx, err = runTransformStage(reg, command, StagePrevalidate, ctx)
		if err != nil {
			return fmt.Errorf("hook prevalidate failed: %w", err)
		}

		// Stage 2: preexec (transform)
		ctx, err = runTransformStage(reg, command, StagePreexec, ctx)
		if err != nil {
			return fmt.Errorf("hook preexec failed: %w", err)
		}

		// Execute the actual command
		if err := fn(cmd, ctx.Args); err != nil {
			return err
		}

		// Stage 3: postexec (transform)
		ctx, err = runTransformStage(reg, command, StagePostexec, ctx)
		if err != nil {
			return fmt.Errorf("hook postexec failed: %w", err)
		}

		// Stage 4: notify (fire-and-forget)
		runNotifyStage(reg, command, ctx)

		return nil
	}
}

// runTransformStage runs all transform hooks for a stage sequentially.
// Each hook receives the context from the previous hook (chaining).
func runTransformStage(reg *Registry, command string, stage Stage, ctx *Context) (*Context, error) {
	hooks := reg.Resolve(command, stage)
	for _, h := range hooks {
		if h.Mode == ModeNotify {
			continue
		}
		result, err := h.Handler(ctx)
		if err != nil {
			return ctx, fmt.Errorf("hook %q (%s): %w", h.Name, stage, err)
		}
		if result != nil {
			ctx = result
		}
	}
	return ctx, nil
}

// runNotifyStage runs all notify hooks concurrently. Errors are logged via
// log.Printf rather than the ui package because notify hooks run in goroutines
// where concurrent writes to styled terminal output could interleave.
func runNotifyStage(reg *Registry, command string, ctx *Context) {
	hooks := reg.Resolve(command, StageNotify)
	if len(hooks) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, h := range hooks {
		wg.Add(1)
		go func(h Hook) {
			defer wg.Done()
			if _, err := h.Handler(ctx); err != nil {
				log.Printf("notify hook %q error: %v", h.Name, err)
			}
		}(h)
	}
	wg.Wait()
}

// extractFlags extracts changed flag values from a Cobra command.
func extractFlags(cmd *cobra.Command) map[string]string {
	flags := make(map[string]string)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			flags[f.Name] = f.Value.String()
		}
	})
	return flags
}
