package workflow

import (
	"fmt"
	"sync"
)

// Factory creates a new Workflow instance with the given workDir
type Factory func(workDir string) Workflow

var (
	registry = make(map[string]Factory)
	mu       sync.RWMutex
)

// Register adds a workflow factory to the registry
// Typically called in init() of each workflow implementation
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("workflow %q already registered", name))
	}
	registry[name] = factory
}

// Get returns a new Workflow instance for the given name and workDir
// Returns nil if the workflow is not registered
func Get(name string, workDir string) Workflow {
	mu.RLock()
	defer mu.RUnlock()
	factory, exists := registry[name]
	if !exists {
		return nil
	}
	return factory(workDir)
}

// List returns all registered workflow names
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Exists checks if a workflow is registered
func Exists(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, exists := registry[name]
	return exists
}
