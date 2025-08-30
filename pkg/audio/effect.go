package audio

import (
	"math"
)

type EffectType int

const (
	EffectDelay EffectType = iota
	EffectReverb
	EffectDistortion
	EffectChorus
	EffectBitCrusher // Industrial effect
)

func (e EffectType) String() string {
	switch e {
	case EffectDelay:
		return "Delay"
	case EffectReverb:
		return "Reverb"
	case EffectDistortion:
		return "Distortion"
	case EffectChorus:
		return "Chorus"
	case EffectBitCrusher:
		return "BitCrusher"
	default:
		return "Unknown"
	}
}

type Effect struct {
	Type       EffectType
	Mix        float64 // Dry/wet mix (0.0 = dry, 1.0 = wet)
	Parameters map[string]float64
	
	// Delay line for time-based effects
	delayLine  []float32
	delayIndex int
	
	// For chorus/flanger
	lfoPhase   float64
	
	// For bit crusher
	sampleHold float32
	holdCount  int
}

func NewEffect(effectType EffectType) *Effect {
	e := &Effect{
		Type:       effectType,
		Mix:        0.5,
		Parameters: make(map[string]float64),
	}
	
	// Initialize based on type
	switch effectType {
	case EffectDelay:
		e.delayLine = make([]float32, SampleRate) // 1 second max delay
		e.Parameters["time"] = 0.25               // 250ms default
		e.Parameters["feedback"] = 0.3
		
	case EffectDistortion:
		e.Parameters["drive"] = 5.0
		e.Parameters["level"] = 0.7
		
	case EffectChorus:
		e.delayLine = make([]float32, SampleRate/10) // 100ms max delay
		e.Parameters["rate"] = 0.5                    // LFO rate in Hz
		e.Parameters["depth"] = 0.3                   // Modulation depth
		e.Parameters["delay"] = 0.02                  // Base delay time
		
	case EffectBitCrusher:
		e.Parameters["bits"] = 8.0      // Bit depth
		e.Parameters["sampleRate"] = 0.5 // Sample rate reduction factor
	}
	
	return e
}

func (e *Effect) Process(buffer []float32, params *ParameterManager) {
	switch e.Type {
	case EffectDelay:
		e.processDelay(buffer)
	case EffectDistortion:
		e.processDistortion(buffer)
	case EffectChorus:
		e.processChorus(buffer)
	case EffectBitCrusher:
		e.processBitCrusher(buffer)
	}
}

func (e *Effect) processDelay(buffer []float32) {
	delayTime := e.Parameters["time"]
	feedback := e.Parameters["feedback"]
	delaySamples := int(delayTime * float64(SampleRate))
	
	// Ensure delay samples doesn't exceed buffer size
	if delaySamples >= len(e.delayLine) {
		delaySamples = len(e.delayLine) - 1
	}
	
	for i := range buffer {
		// Read from delay line
		readIndex := (e.delayIndex - delaySamples + len(e.delayLine)) % len(e.delayLine)
		delayed := e.delayLine[readIndex]
		
		// Mix with input and apply feedback
		wet := delayed
		dry := buffer[i]
		
		// Write to delay line with feedback
		e.delayLine[e.delayIndex] = buffer[i] + delayed*float32(feedback)
		
		// Mix dry and wet signals
		buffer[i] = dry*(1-float32(e.Mix)) + wet*float32(e.Mix)
		
		e.delayIndex = (e.delayIndex + 1) % len(e.delayLine)
	}
}

func (e *Effect) processDistortion(buffer []float32) {
	drive := e.Parameters["drive"]
	level := e.Parameters["level"]
	
	for i := range buffer {
		sample := float64(buffer[i])
		
		// Apply drive
		sample *= drive
		
		// Soft clipping using tanh (smoother than hard clipping)
		sample = math.Tanh(sample)
		
		// Apply output level and mix
		wet := float32(sample * level)
		dry := buffer[i]
		buffer[i] = dry*(1-float32(e.Mix)) + wet*float32(e.Mix)
	}
}

func (e *Effect) processChorus(buffer []float32) {
	rate := e.Parameters["rate"]
	depth := e.Parameters["depth"]
	baseDelay := e.Parameters["delay"]
	
	lfoIncrement := 2.0 * math.Pi * rate / float64(SampleRate)
	
	for i := range buffer {
		// Calculate LFO value for modulation
		lfo := math.Sin(e.lfoPhase) * depth
		e.lfoPhase += lfoIncrement
		if e.lfoPhase >= 2.0*math.Pi {
			e.lfoPhase -= 2.0 * math.Pi
		}
		
		// Calculate modulated delay time
		delayTime := baseDelay + baseDelay*lfo
		delaySamples := delayTime * float64(SampleRate)
		
		// Linear interpolation for fractional delay
		delaySamplesInt := int(delaySamples)
		fraction := delaySamples - float64(delaySamplesInt)
		
		if delaySamplesInt < len(e.delayLine)-1 {
			readIndex1 := (e.delayIndex - delaySamplesInt + len(e.delayLine)) % len(e.delayLine)
			readIndex2 := (readIndex1 - 1 + len(e.delayLine)) % len(e.delayLine)
			
			// Linear interpolation between two samples
			sample1 := e.delayLine[readIndex1]
			sample2 := e.delayLine[readIndex2]
			delayed := sample1*(1-float32(fraction)) + sample2*float32(fraction)
			
			// Write current sample to delay line
			e.delayLine[e.delayIndex] = buffer[i]
			
			// Mix dry and wet
			buffer[i] = buffer[i]*(1-float32(e.Mix)) + delayed*float32(e.Mix)
			
			e.delayIndex = (e.delayIndex + 1) % len(e.delayLine)
		}
	}
}

func (e *Effect) processBitCrusher(buffer []float32) {
	bits := e.Parameters["bits"]
	sampleRateReduction := e.Parameters["sampleRate"]
	
	// Calculate step size for bit reduction
	levels := math.Pow(2, bits)
	stepSize := 2.0 / levels
	
	// Calculate sample hold period
	holdPeriod := int(1.0 / sampleRateReduction)
	
	for i := range buffer {
		// Sample rate reduction
		if e.holdCount == 0 {
			e.sampleHold = buffer[i]
			e.holdCount = holdPeriod
		}
		e.holdCount--
		
		// Bit depth reduction
		sample := float64(e.sampleHold)
		
		// Quantize the signal
		if sample > 0 {
			sample = math.Floor(sample/stepSize) * stepSize
		} else {
			sample = math.Ceil(sample/stepSize) * stepSize
		}
		
		// Mix dry and wet
		wet := float32(sample)
		dry := buffer[i]
		buffer[i] = dry*(1-float32(e.Mix)) + wet*float32(e.Mix)
	}
}

func (e *Effect) Reset() {
	// Clear delay line
	for i := range e.delayLine {
		e.delayLine[i] = 0
	}
	e.delayIndex = 0
	e.lfoPhase = 0
	e.sampleHold = 0
	e.holdCount = 0
}