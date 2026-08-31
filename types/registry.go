package types

import (
	"fmt"
	"sort"
	"sync"
)

// ProviderFactory constructs a Provider from a config entry.
type ProviderFactory func(cfg ProviderConfig) (Provider, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]ProviderFactory{}
)

// Register adds a provider factory under name. Provider packages call
// this from their own init(). Returns an error -- never panics -- on a
// nil factory or a name that's already registered, so callers can assert
// on the return value without recover().
func Register(name string, factory ProviderFactory) error {
	if factory == nil {
		return fmt.Errorf("saka: Register(%q): nil factory", name)
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		return fmt.Errorf("saka: Register(%q): already registered", name)
	}
	registry[name] = factory
	return nil
}

// Lookup returns the factory registered under name, and whether one
// exists.
func Lookup(name string) (ProviderFactory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	return f, ok
}

// Registered returns the sorted names of every currently registered
// provider.
func Registered() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
