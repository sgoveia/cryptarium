package detector

import "sync"

var (
	registryMu sync.RWMutex
	registry   []Detector
)

// Register adds d to the process-wide detector registry.
// Intended for init() in detector subpackages.
func Register(d Detector) {
	if d == nil {
		panic("detector: Register nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, existing := range registry {
		if existing.Name() == d.Name() {
			return // idempotent across packages that blank-import detectors
		}
	}
	registry = append(registry, d)
}

// All returns a snapshot of registered detectors in registration order.
func All() []Detector {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Detector, len(registry))
	copy(out, registry)
	return out
}
