package registry

import (
	"errors"
	"sync"
	"testing"
)

var testRegistries = map[testing.TB]*Registry{}

var registryLock sync.Mutex

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

func (r *Registry) ForceRegisterCapability(c Capability) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.capabilities[c.ID()] = c
}

func (r *Registry) GetCapability(id string) (Capability, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	c, ok := r.capabilities[id]
	if !ok {
		return nil, errors.New("capability not found: " + id)
	}
	return c, nil
}
