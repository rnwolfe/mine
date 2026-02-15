// Package hook provides the command execution pipeline for mine's plugin system.
//
// Commands traverse four stages: prevalidate → preexec → postexec → notify.
// Hooks are either transform (modify data) or notify (fire-and-forget side effects).
// The pipeline is a no-op when no hooks are registered, ensuring zero overhead.
package hook

import (
	"encoding/json"
	"fmt"
	"time"
)

// Stage identifies when a hook runs in the pipeline.
type Stage string

const (
	StagePrevalidate Stage = "prevalidate"
	StagePreexec     Stage = "preexec"
	StagePostexec    Stage = "postexec"
	StageNotify      Stage = "notify"
)

// AllStages is the execution order for the pipeline.
var AllStages = []Stage{StagePrevalidate, StagePreexec, StagePostexec, StageNotify}

// Mode determines how a hook interacts with the pipeline.
type Mode string

const (
	ModeTransform Mode = "transform" // receives and returns modified Context
	ModeNotify    Mode = "notify"    // receives Context, no response expected
)

// Context carries data through the hook pipeline.
type Context struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Flags     map[string]string `json:"flags"`
	Result    any               `json:"result,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// NewContext creates a Context for the given command invocation.
func NewContext(command string, args []string, flags map[string]string) *Context {
	if args == nil {
		args = []string{}
	}
	if flags == nil {
		flags = map[string]string{}
	}
	return &Context{
		Command:   command,
		Args:      args,
		Flags:     flags,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// JSON serializes the context for passing to hook executables.
func (c *Context) JSON() ([]byte, error) {
	return json.Marshal(c)
}

// ParseContext deserializes a Context from JSON.
func ParseContext(data []byte) (*Context, error) {
	var ctx Context
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("parsing hook context: %w", err)
	}
	return &ctx, nil
}

// Hook defines a single hook registration.
type Hook struct {
	// Pattern is the command pattern this hook matches (e.g. "todo.add", "todo.*", "*").
	Pattern string
	// Stage is when this hook runs.
	Stage Stage
	// Mode is how this hook interacts with the pipeline.
	Mode Mode
	// Name is a human-readable identifier for this hook.
	Name string
	// Source identifies where this hook came from (e.g. "user", "plugin:obsidian").
	Source string
	// Handler executes the hook. For transform hooks, it may modify the context.
	// For notify hooks, the returned context is ignored.
	Handler Handler
	// Timeout is the maximum duration for this hook to execute.
	// Zero means use the default (5s for transform, 30s for notify).
	Timeout time.Duration
}

// Handler is the function signature for hook execution.
// It receives the current context and returns a potentially modified context.
type Handler func(ctx *Context) (*Context, error)

// DefaultTransformTimeout is the default timeout for transform hooks.
const DefaultTransformTimeout = 5 * time.Second

// DefaultNotifyTimeout is the default timeout for notify hooks.
const DefaultNotifyTimeout = 30 * time.Second
