package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rnwolfe/mine/internal/hook"
)

// ProtocolVersion is the current plugin protocol version.
const ProtocolVersion = "1.0.0"

// InvocationType identifies what kind of invocation is being made.
type InvocationType string

const (
	InvocationHook      InvocationType = "hook"
	InvocationCommand   InvocationType = "command"
	InvocationLifecycle InvocationType = "lifecycle"
)

// Invocation is the JSON envelope sent to plugin binaries on stdin.
type Invocation struct {
	ProtocolVersion string         `json:"protocol_version"`
	Type            InvocationType `json:"type"`
	Stage           string         `json:"stage,omitempty"`
	Mode            string         `json:"mode,omitempty"`
	Event           string         `json:"event,omitempty"`
	Command         string         `json:"command,omitempty"`
	Context         *hook.Context  `json:"context,omitempty"`
	Args            []string       `json:"args,omitempty"`
	Flags           map[string]string `json:"flags,omitempty"`
}

// Response is the JSON response from a plugin for transform hooks.
type Response struct {
	Status  string        `json:"status"`
	Context *hook.Context `json:"context,omitempty"`
	Error   string        `json:"error,omitempty"`
	Code    string        `json:"code,omitempty"`
}

// RegisterPluginHooks registers all hooks from installed plugins into the hook registry.
func RegisterPluginHooks() error {
	plugins, err := List()
	if err != nil {
		return err
	}

	for _, p := range plugins {
		if !p.Enabled {
			continue
		}

		binPath := filepath.Join(p.Dir, p.Manifest.Entrypoint())
		source := "plugin:" + p.Manifest.Plugin.Name

		for _, hd := range p.Manifest.Hooks {
			stage := hook.Stage(hd.Stage)
			mode := hook.Mode(hd.Mode)

			timeout := hook.DefaultTransformTimeout
			if mode == hook.ModeNotify {
				timeout = hook.DefaultNotifyTimeout
			}
			if hd.Timeout != "" {
				if d, err := time.ParseDuration(hd.Timeout); err == nil {
					timeout = d
				}
			}

			handler := pluginHookHandler(binPath, stage, mode, timeout, p.Manifest.Permissions)

			if err := hook.Register(hook.Hook{
				Pattern: hd.Command,
				Stage:   stage,
				Mode:    mode,
				Name:    fmt.Sprintf("%s:%s:%s", p.Manifest.Plugin.Name, hd.Command, hd.Stage),
				Source:  source,
				Handler: handler,
				Timeout: timeout,
			}); err != nil {
				return fmt.Errorf("registering hook for plugin %s: %w", p.Manifest.Plugin.Name, err)
			}
		}
	}

	return nil
}

// pluginHookHandler creates a hook.Handler that invokes a plugin binary.
func pluginHookHandler(binPath string, stage hook.Stage, mode hook.Mode, timeout time.Duration, perms Permissions) hook.Handler {
	return func(ctx *hook.Context) (*hook.Context, error) {
		inv := Invocation{
			ProtocolVersion: ProtocolVersion,
			Type:            InvocationHook,
			Stage:           string(stage),
			Mode:            string(mode),
			Context:         ctx,
		}

		invJSON, err := json.Marshal(inv)
		if err != nil {
			return nil, fmt.Errorf("serializing invocation: %w", err)
		}

		execCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cmd := exec.CommandContext(execCtx, binPath)
		cmd.Stdin = bytes.NewReader(invJSON)
		cmd.Env = buildPluginEnv(perms)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if execCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("plugin timed out after %s", timeout)
			}
			errMsg := stderr.String()
			if errMsg != "" {
				return nil, fmt.Errorf("plugin error: %s", errMsg)
			}
			return nil, fmt.Errorf("plugin failed: %w", err)
		}

		if mode == hook.ModeNotify {
			return ctx, nil
		}

		// Parse transform response
		output := stdout.Bytes()
		if len(output) == 0 {
			return ctx, nil
		}

		var resp Response
		if err := json.Unmarshal(output, &resp); err != nil {
			return nil, fmt.Errorf("parsing plugin response: %w", err)
		}

		if resp.Status == "error" {
			return nil, fmt.Errorf("plugin error: %s", resp.Error)
		}

		if resp.Context != nil {
			return resp.Context, nil
		}

		return ctx, nil
	}
}

// RunCommand executes a plugin's custom command.
func RunCommand(p *InstalledPlugin, cmdName string, args []string) error {
	binPath := filepath.Join(p.Dir, p.Manifest.Entrypoint())

	inv := Invocation{
		ProtocolVersion: ProtocolVersion,
		Type:            InvocationCommand,
		Command:         cmdName,
		Args:            args,
	}

	invJSON, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("serializing command invocation: %w", err)
	}

	cmd := exec.Command(binPath)
	cmd.Stdin = bytes.NewReader(invJSON)
	cmd.Env = buildPluginEnv(p.Manifest.Permissions)

	// Commands get raw terminal access
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// SendLifecycleEvent sends a lifecycle event to a plugin.
func SendLifecycleEvent(p *InstalledPlugin, event string) error {
	binPath := filepath.Join(p.Dir, p.Manifest.Entrypoint())

	inv := Invocation{
		ProtocolVersion: ProtocolVersion,
		Type:            InvocationLifecycle,
		Event:           event,
	}

	invJSON, err := json.Marshal(inv)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath)
	cmd.Stdin = bytes.NewReader(invJSON)
	cmd.Env = buildPluginEnv(p.Manifest.Permissions)

	return cmd.Run()
}
