package hook

import (
	"path/filepath"
	"sort"
	"sync"
)

// Registry holds registered hooks and resolves which hooks apply to a command.
type Registry struct {
	mu    sync.RWMutex
	hooks []Hook
}

// DefaultRegistry is the global hook registry.
var DefaultRegistry = &Registry{}

// Register adds a hook to the registry.
func (r *Registry) Register(h Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, h)
}

// Unregister removes all hooks from the given source.
func (r *Registry) Unregister(source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	filtered := r.hooks[:0]
	for _, h := range r.hooks {
		if h.Source != source {
			filtered = append(filtered, h)
		}
	}
	r.hooks = filtered
}

// Resolve returns all hooks matching the command and stage, sorted by name.
func (r *Registry) Resolve(command string, stage Stage) []Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []Hook
	for _, h := range r.hooks {
		if h.Stage != stage {
			continue
		}
		if matchPattern(h.Pattern, command) {
			matched = append(matched, h)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Name < matched[j].Name
	})
	return matched
}

// HasHooks returns true if any hooks are registered for the given command.
func (r *Registry) HasHooks(command string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range r.hooks {
		if matchPattern(h.Pattern, command) {
			return true
		}
	}
	return false
}

// All returns all registered hooks.
func (r *Registry) All() []Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Hook, len(r.hooks))
	copy(out, r.hooks)
	return out
}

// Count returns the number of registered hooks.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.hooks)
}

// Register adds a hook to the default registry.
func Register(h Hook) {
	DefaultRegistry.Register(h)
}

// matchPattern checks if a command matches a hook pattern.
// Patterns support dotted notation and wildcards:
//   - "todo.add" matches only "todo.add"
//   - "todo.*"   matches "todo.add", "todo.done", etc.
//   - "*"        matches everything
func matchPattern(pattern, command string) bool {
	matched, _ := filepath.Match(pattern, command)
	return matched
}
