package audio

import (
	"sync"
	"sync/atomic"
)

// ParameterManager handles thread-safe parameter updates
// between the UI/control thread and the audio processing thread
type ParameterManager struct {
	mu     sync.RWMutex
	params map[string]float64
	
	// Atomic parameters for lock-free access in hot paths
	atomicParams map[string]*atomic.Value
	
	// Parameter smoothing for avoiding clicks
	smoothing    map[string]*ParameterSmoother
	smoothingMu  sync.RWMutex
}

// ParameterSmoother provides smooth parameter transitions
type ParameterSmoother struct {
	current  float64
	target   float64
	rate     float64 // Smoothing rate (0.0 to 1.0)
	active   bool
}

func NewParameterManager() *ParameterManager {
	return &ParameterManager{
		params:       make(map[string]float64),
		atomicParams: make(map[string]*atomic.Value),
		smoothing:    make(map[string]*ParameterSmoother),
	}
}

// Set updates a parameter value
func (pm *ParameterManager) Set(key string, value float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.params[key] = value
	
	// Update atomic parameter if it exists
	if atomicParam, exists := pm.atomicParams[key]; exists {
		atomicParam.Store(value)
	}
	
	// Update smoother target if smoothing is enabled
	pm.smoothingMu.RLock()
	if smoother, exists := pm.smoothing[key]; exists {
		smoother.target = value
		smoother.active = true
	}
	pm.smoothingMu.RUnlock()
}

// Get retrieves a parameter value
func (pm *ParameterManager) Get(key string) (float64, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	val, exists := pm.params[key]
	return val, exists
}

// GetWithDefault retrieves a parameter with a default value
func (pm *ParameterManager) GetWithDefault(key string, defaultValue float64) float64 {
	if val, exists := pm.Get(key); exists {
		return val
	}
	return defaultValue
}

// GetSmoothed retrieves a smoothed parameter value
func (pm *ParameterManager) GetSmoothed(key string, defaultValue float64) float64 {
	pm.smoothingMu.RLock()
	defer pm.smoothingMu.RUnlock()
	
	if smoother, exists := pm.smoothing[key]; exists && smoother.active {
		// Apply exponential smoothing
		smoother.current += (smoother.target - smoother.current) * smoother.rate
		
		// Check if we've reached the target
		if absFloat64(smoother.target-smoother.current) < 0.0001 {
			smoother.current = smoother.target
			smoother.active = false
		}
		
		return smoother.current
	}
	
	return pm.GetWithDefault(key, defaultValue)
}

// RegisterAtomic creates an atomic parameter for lock-free access
func (pm *ParameterManager) RegisterAtomic(key string, initialValue float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	atomicVal := &atomic.Value{}
	atomicVal.Store(initialValue)
	pm.atomicParams[key] = atomicVal
	pm.params[key] = initialValue
}

// GetAtomic retrieves an atomic parameter value (lock-free)
func (pm *ParameterManager) GetAtomic(key string) (float64, bool) {
	if atomicParam, exists := pm.atomicParams[key]; exists {
		return atomicParam.Load().(float64), true
	}
	return 0, false
}

// EnableSmoothing enables parameter smoothing for a given key
func (pm *ParameterManager) EnableSmoothing(key string, rate float64) {
	pm.smoothingMu.Lock()
	defer pm.smoothingMu.Unlock()
	
	currentValue := pm.GetWithDefault(key, 0.0)
	pm.smoothing[key] = &ParameterSmoother{
		current: currentValue,
		target:  currentValue,
		rate:    rate,
		active:  false,
	}
}

// BatchSet updates multiple parameters atomically
func (pm *ParameterManager) BatchSet(updates map[string]float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for key, value := range updates {
		pm.params[key] = value
		
		// Update atomic parameter if it exists
		if atomicParam, exists := pm.atomicParams[key]; exists {
			atomicParam.Store(value)
		}
	}
}

// GetAll returns a copy of all parameters
func (pm *ParameterManager) GetAll() map[string]float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	result := make(map[string]float64)
	for k, v := range pm.params {
		result[k] = v
	}
	return result
}

// Helper function for absolute value
func absFloat64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}