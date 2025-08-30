package audio

import (
	"math"
)

type OscillatorType int

const (
	OscSine OscillatorType = iota
	OscSaw
	OscSquare
	OscTriangle
	OscNoise
)

func (o OscillatorType) String() string {
	switch o {
	case OscSine:
		return "Sine"
	case OscSaw:
		return "Saw"
	case OscSquare:
		return "Square"
	case OscTriangle:
		return "Triangle"
	case OscNoise:
		return "Noise"
	default:
		return "Unknown"
	}
}

type Oscillator struct {
	Type      OscillatorType
	Frequency float64
	Amplitude float64
	Phase     float64
	phaseInc  float64
}

func NewOscillator(oscType OscillatorType, freq, amp float64) *Oscillator {
	return &Oscillator{
		Type:      oscType,
		Frequency: freq,
		Amplitude: amp,
		Phase:     0,
		phaseInc:  2.0 * math.Pi * freq / float64(SampleRate),
	}
}

func (o *Oscillator) Generate(buffer []float32, params *ParameterManager) {
	// Check for frequency parameter updates
	if newFreq, exists := params.Get("osc1_frequency"); exists {
		o.Frequency = newFreq
		o.phaseInc = 2.0 * math.Pi * newFreq / float64(SampleRate)
	}

	for i := range buffer {
		var sample float64

		switch o.Type {
		case OscSine:
			sample = math.Sin(o.Phase)
		case OscSaw:
			sample = 2.0*(o.Phase/(2.0*math.Pi)) - 1.0
		case OscSquare:
			if o.Phase < math.Pi {
				sample = 1.0
			} else {
				sample = -1.0
			}
		case OscTriangle:
			if o.Phase < math.Pi {
				sample = -1.0 + (2.0 * o.Phase / math.Pi)
			} else {
				sample = 3.0 - (2.0 * o.Phase / math.Pi)
			}
		case OscNoise:
			sample = randFloat64()*2.0 - 1.0
		}

		buffer[i] = float32(sample * o.Amplitude)
		
		o.Phase += o.phaseInc
		if o.Phase >= 2.0*math.Pi {
			o.Phase -= 2.0 * math.Pi
		}
	}
}

// Simple PRNG for noise generation
var randState uint64 = 12345

func randFloat64() float64 {
	randState = randState*6364136223846793005 + 1442695040888963407
	return float64(randState>>32) / float64(1<<32)
}