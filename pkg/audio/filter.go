package audio

import (
	"math"
)

type FilterType int

const (
	FilterLowPass FilterType = iota
	FilterHighPass
	FilterBandPass
	FilterNotch
)

func (f FilterType) String() string {
	switch f {
	case FilterLowPass:
		return "LowPass"
	case FilterHighPass:
		return "HighPass"
	case FilterBandPass:
		return "BandPass"
	case FilterNotch:
		return "Notch"
	default:
		return "Unknown"
	}
}

type Filter struct {
	Type      FilterType
	Frequency float64
	Resonance float64
	
	// State variables for biquad filter
	a0, a1, a2 float64
	b0, b1, b2 float64
	x1, x2     float64
	y1, y2     float64
}

func NewLowPassFilter(freq, resonance float64) *Filter {
	f := &Filter{
		Type:      FilterLowPass,
		Frequency: freq,
		Resonance: resonance,
	}
	f.updateCoefficients()
	return f
}

func NewHighPassFilter(freq, resonance float64) *Filter {
	f := &Filter{
		Type:      FilterHighPass,
		Frequency: freq,
		Resonance: resonance,
	}
	f.updateCoefficients()
	return f
}

func (f *Filter) updateCoefficients() {
	// Butterworth filter coefficients
	omega := 2.0 * math.Pi * f.Frequency / float64(SampleRate)
	sin := math.Sin(omega)
	cos := math.Cos(omega)
	
	// Ensure resonance is in valid range (0.1 to 10)
	q := f.Resonance
	if q < 0.1 {
		q = 0.1
	}
	if q > 10.0 {
		q = 10.0
	}
	
	alpha := sin / (2.0 * q)
	
	switch f.Type {
	case FilterLowPass:
		f.b0 = (1.0 - cos) / 2.0
		f.b1 = 1.0 - cos
		f.b2 = (1.0 - cos) / 2.0
		f.a0 = 1.0 + alpha
		f.a1 = -2.0 * cos
		f.a2 = 1.0 - alpha
		
	case FilterHighPass:
		f.b0 = (1.0 + cos) / 2.0
		f.b1 = -(1.0 + cos)
		f.b2 = (1.0 + cos) / 2.0
		f.a0 = 1.0 + alpha
		f.a1 = -2.0 * cos
		f.a2 = 1.0 - alpha
		
	case FilterBandPass:
		f.b0 = alpha
		f.b1 = 0.0
		f.b2 = -alpha
		f.a0 = 1.0 + alpha
		f.a1 = -2.0 * cos
		f.a2 = 1.0 - alpha
		
	case FilterNotch:
		f.b0 = 1.0
		f.b1 = -2.0 * cos
		f.b2 = 1.0
		f.a0 = 1.0 + alpha
		f.a1 = -2.0 * cos
		f.a2 = 1.0 - alpha
	}
	
	// Normalize coefficients
	f.b0 /= f.a0
	f.b1 /= f.a0
	f.b2 /= f.a0
	f.a1 /= f.a0
	f.a2 /= f.a0
}

func (f *Filter) Process(buffer []float32, params *ParameterManager) {
	// Check for filter parameter updates
	if newFreq, exists := params.Get("filter_frequency"); exists {
		f.Frequency = newFreq
		f.updateCoefficients()
	}
	if newRes, exists := params.Get("filter_resonance"); exists {
		f.Resonance = newRes
		f.updateCoefficients()
	}
	
	for i := range buffer {
		x0 := float64(buffer[i])
		
		// Biquad filter difference equation
		y0 := f.b0*x0 + f.b1*f.x1 + f.b2*f.x2 - f.a1*f.y1 - f.a2*f.y2
		
		// Update state variables
		f.x2 = f.x1
		f.x1 = x0
		f.y2 = f.y1
		f.y1 = y0
		
		// Apply soft clipping to prevent filter instability
		if y0 > 1.0 {
			y0 = 1.0
		} else if y0 < -1.0 {
			y0 = -1.0
		}
		
		buffer[i] = float32(y0)
	}
}

func (f *Filter) Reset() {
	f.x1 = 0
	f.x2 = 0
	f.y1 = 0
	f.y2 = 0
}