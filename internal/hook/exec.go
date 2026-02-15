package hook

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ExecHandler creates a Handler that runs an external executable.
// For transform hooks, it passes Context JSON on stdin and reads modified Context from stdout.
// For notify hooks, it passes Context JSON on stdin and discards output.
func ExecHandler(path string, mode Mode, timeout time.Duration) Handler {
	if timeout == 0 {
		if mode == ModeTransform {
			timeout = DefaultTransformTimeout
		} else {
			timeout = DefaultNotifyTimeout
		}
	}

	return func(ctx *Context) (*Context, error) {
		input, err := ctx.JSON()
		if err != nil {
			return nil, fmt.Errorf("serializing context: %w", err)
		}

		execCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cmd := exec.CommandContext(execCtx, path)
		cmd.Stdin = bytes.NewReader(input)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if execCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("hook timed out after %s", timeout)
			}
			errMsg := stderr.String()
			if errMsg != "" {
				return nil, fmt.Errorf("hook failed: %s", errMsg)
			}
			return nil, fmt.Errorf("hook failed: %w", err)
		}

		if mode == ModeNotify {
			return ctx, nil
		}

		// Parse transform response
		output := stdout.Bytes()
		if len(output) == 0 {
			return ctx, nil
		}

		result, err := ParseContext(output)
		if err != nil {
			return nil, fmt.Errorf("parsing hook output: %w", err)
		}
		return result, nil
	}
}
