package ai

import (
	"fmt"
	"sync"
)

var (
	mu        sync.RWMutex
	providers = make(map[string]ProviderFactory)
)

// ProviderFactory creates a new provider instance with the given API key.
type ProviderFactory func(apiKey string) (Provider, error)

// Register adds a provider factory to the registry.
func Register(name string, factory ProviderFactory) {
	mu.Lock()
	defer mu.Unlock()
	providers[name] = factory
}

// GetProvider returns a configured provider by name.
func GetProvider(name, apiKey string) (Provider, error) {
	mu.RLock()
	factory, ok := providers[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	return factory(apiKey)
}

// ListProviders returns all registered provider names.
func ListProviders() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}
