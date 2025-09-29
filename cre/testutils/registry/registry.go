package registry

import (
	"errors"
	"sync"
	"testing"
)

var testRegistries = map[testing.TB]*Registry{}

var registryLock sync.Mutex

// GetRegistry returns a Registry instance scoped to the provided testing.TB, creating one if it doesn't exist.
// The registry is used internally by the test utilities to manage mock capabilities.
func GetRegistry(tb testing.TB) *Registry {
	registryLock.Lock()
	defer registryLock.Unlock()
	if r, ok := testRegistries[tb]; ok {
		return r
	}

	r := &Registry{tb: tb, capabilities: map[string]Capability{}}
	testRegistries[tb] = r
	tb.Cleanup(func() {
		delete(testRegistries, tb)
	})
	return r
}

// Registry is meant to be used with GetRegistry, do not use it directly.
type Registry struct {
	capabilities map[string]Capability
	tb           testing.TB
	lock         sync.Mutex
}

// RegisterCapability is meant to be called by generated mock code to register the mock with the test.
// It returns an error if a capability with the same ID is already registered.
func (r *Registry) RegisterCapability(c Capability) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	_, ok := r.capabilities[c.ID()]
	if ok {
		return errors.New("capability already exists: " + c.ID())
	}
	r.capabilities[c.ID()] = c
	return nil
}

// ForceRegisterCapability is like RegisterCapability but it overwrites any existing capability with the same ID.
func (r *Registry) ForceRegisterCapability(c Capability) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.capabilities[c.ID()] = c
}

// GetCapability is meant to be used by generated code to retrieve a registered capability.
// It retrieves a registered capability by its ID.
// It returns an error if no capability with the given ID is found.
func (r *Registry) GetCapability(id string) (Capability, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	c, ok := r.capabilities[id]
	if !ok {
		return nil, errors.New("capability not found: " + id)
	}
	return c, nil
}
