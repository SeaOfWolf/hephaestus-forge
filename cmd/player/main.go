package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall" 
	"time"

	"github.com/SeaOfWolf/hephaestus-forge/pkg/audio"
)

func main() {
	fmt.Println("🔥 Hephaestus Forge - Starting...")
	
	// Create audio engine
	engine := audio.NewAudioEngine()
	
	// Start audio processing
	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start audio engine: %v", err)
	}
	defer engine.Stop()
	
	// Get parameter manager for real-time control
	params := engine.GetParameterManager()
	
	// Demonstrate real-time parameter changes
	go func() {
		frequencies := []float64{440.0, 660.0, 880.0, 330.0, 440.0}
		
		for i, freq := range frequencies {
			time.Sleep(3 * time.Second)
			params.Set("osc1_frequency", freq)
			fmt.Printf("🔨 Forged frequency: %.1f Hz (%d/5)\n", freq, i+1)
			
			// Also demonstrate oscillator direct control
			// (In a real implementation, this would go through the parameter system)
		}
		
		fmt.Println("✨ Forging sequence complete - press Ctrl+C to exit")
	}()
	
	// Set up graceful shutdown
	fmt.Println("🎵 Hephaestus Forge running - press Ctrl+C to stop")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n👋 Shutting down gracefully...")
	}