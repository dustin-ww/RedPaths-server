package adapter

import (
	"RedPaths-server/pkg/adapter/scan"
	"RedPaths-server/pkg/interfaces"
	"fmt"
	"sync"
)

type AdapterRegistry struct {
	adapters map[string]interfaces.ToolAdapter
	mutex    sync.RWMutex
}

var (
	factory     *AdapterRegistry
	factoryOnce sync.Once
)

func GetAdapterFactory() *AdapterRegistry {
	factoryOnce.Do(func() {
		factory = &AdapterRegistry{
			adapters: make(map[string]interfaces.ToolAdapter),
		}

		factory.RegisterAdapter(scan.NewNmapAdapter())
	})

	return factory
}

func (f *AdapterRegistry) RegisterAdapter(adapter interfaces.ToolAdapter) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.adapters[adapter.GetName()] = adapter
}

func (f *AdapterRegistry) GetAdapter(name string) (interfaces.ToolAdapter, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	adapter, exists := f.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter with name '%s' not found", name)
	}

	return adapter, nil
}

func (f *AdapterRegistry) GetScanAdapter(name string) (interfaces.ScanAdapter, error) {
	adapter, err := f.GetAdapter(name)
	if err != nil {
		return nil, err
	}

	scanAdapter, ok := adapter.(interfaces.ScanAdapter)
	if !ok {
		return nil, fmt.Errorf("adapter '%s' is not a ScanAdapter", name)
	}

	return scanAdapter, nil
}

func (f *AdapterRegistry) ListAvailableAdapters() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var names []string
	for name := range f.adapters {
		names = append(names, name)
	}

	return names
}
