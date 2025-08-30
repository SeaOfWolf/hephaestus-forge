package audio

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/gordonklaus/portaudio"
)

const (
	SampleRate = 44100
	FrameSize  = 512
	Channels   = 2 // Stereo output
)

// AudioEngine manages the entire audio processing pipeline
type AudioEngine struct {
	stream    *portaudio.Stream
	isRunning int32 // atomic bool

	// Audio processing chain
	oscillators []*Oscillator
	filters     []*Filter
	effects     []*Effect

	// Real-time parameter control
	params *ParameterManager // Changed from 'param' to 'params' for consistency
}

// NewAudioEngine creates a new audio engine instance
func NewAudioEngine() *AudioEngine {
	engine := &AudioEngine{
		oscillators: make([]*Oscillator, 0, 8), // Support up to 8 oscillators
		filters:     make([]*Filter, 0, 4),      // Support up to 4 filters
		effects:     make([]*Effect, 0, 8),      // Support up to 8 effects
		params:      NewParameterManager(),
	}

	// Add default oscillator
	osc := NewOscillator(OscSine, 440.0, 0.3)
	engine.AddOscillator(osc)

	// Add default low-pass filter
	filter := NewLowPassFilter(1000.0, 0.7) // 1kHz cutoff, 0.7 resonance
	engine.AddFilter(filter)

	return engine
}

// AddOscillator adds an oscillator to the processing chain
func (ae *AudioEngine) AddOscillator(osc *Oscillator) {
	ae.oscillators = append(ae.oscillators, osc)
	log.Printf("Added %s oscillator at %.1f Hz", osc.Type.String(), osc.Frequency)
}

// AddFilter adds a filter to the processing chain
func (ae *AudioEngine) AddFilter(filter *Filter) {
	ae.filters = append(ae.filters, filter)
	log.Printf("Added %s filter", filter.Type.String())
}

// AddEffect adds an effect to the processing chain
func (ae *AudioEngine) AddEffect(effect *Effect) {
	ae.effects = append(ae.effects, effect)
	log.Printf("Added %s effect", effect.Type.String())
}

// processAudio is the main audio processing callback
func (ae *AudioEngine) processAudio(out [][]float32) {
	if !ae.IsRunning() {
		// Output silence when stopped
		for ch := 0; ch < len(out); ch++ {
			for i := range out[ch] {
				out[ch][i] = 0
			}
		}
		return
	}

	frameCount := len(out[0])

	// Generate audio from oscillators
	mixBuffer := make([]float32, frameCount)
	for _, osc := range ae.oscillators {
		tempBuffer := make([]float32, frameCount)
		osc.Generate(tempBuffer, ae.params)
		
		// Mix oscillator output
		for i := range mixBuffer {
			mixBuffer[i] += tempBuffer[i]
		}
	}

	// Apply filters
	for _, filter := range ae.filters {
		filter.Process(mixBuffer, ae.params)
	}

	// Apply effects
	for _, effect := range ae.effects {
		effect.Process(mixBuffer, ae.params)
	}

	// Copy to output channels (stereo)
	for ch := 0; ch < len(out); ch++ {
		copy(out[ch], mixBuffer)
	}
}

// Start begins audio processing
func (ae *AudioEngine) Start() error {
	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize PortAudio: %v", err)
	}

	stream, err := portaudio.OpenDefaultStream(
		0,                   // input channels
		Channels,            // output channels
		SampleRate,          // sample rate as float64 (PortAudio expects float64)
		FrameSize,
		ae.processAudio,
	)
	if err != nil {
		portaudio.Terminate()
		return fmt.Errorf("failed to open audio stream: %v", err)
	}

	ae.stream = stream
	atomic.StoreInt32(&ae.isRunning, 1)

	if err := ae.stream.Start(); err != nil {
		return fmt.Errorf("failed to start audio stream: %v", err)
	}

	log.Printf("🔥 Hephaestus Forge started (SR: %d Hz, Buffer: %d frames)", SampleRate, FrameSize)
	return nil
}

// Stop ends audio processing
func (ae *AudioEngine) Stop() {
	atomic.StoreInt32(&ae.isRunning, 0)

	if ae.stream != nil {
		ae.stream.Stop()
		ae.stream.Close()
	}

	portaudio.Terminate()
	log.Printf("🔇 Hephaestus Forge stopped")
}

// IsRunning returns whether the engine is currently processing audio
func (ae *AudioEngine) IsRunning() bool {
	return atomic.LoadInt32(&ae.isRunning) != 0
}

// GetParameterManager returns the parameter manager for real-time control
func (ae *AudioEngine) GetParameterManager() *ParameterManager {
	return ae.params
}