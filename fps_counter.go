package main

import (
	"fmt"
	"os"
	"time"
)

// Helper function
// Package level variables for FPS counter
var (
	fpsCounter     int
	fpsLastPrint   time.Time
	fpsInitialized bool
)

func printFPS() {
	// Initialize on first call
	if !fpsInitialized {
		fpsLastPrint = time.Now()
		fpsInitialized = true
	}

	// Increment the counter
	fpsCounter++

	// Check if a second has passed
	now := time.Now()
	if now.Sub(fpsLastPrint).Seconds() >= 1.0 {
		fmt.Fprintf(os.Stderr, "FPS: %d\n", fpsCounter)
		fpsCounter = 0
		fpsLastPrint = now
	}
}
