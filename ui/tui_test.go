package ui

import (
	"strings"
	"testing"
)

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		bytesPerSec float64
		expected    string
	}{
		{500, "500 B/s"},
		{1024, "1.00 KB/s"},
		{2048, "2.00 KB/s"},
		{1048576, "1.00 MB/s"},
		{1572864, "1.50 MB/s"},
		{1073741824, "1.00 GB/s"},
	}

	for _, tt := range tests {
		result := formatSpeed(tt.bytesPerSec)
		if result != tt.expected {
			t.Errorf("formatSpeed(%v) = %v; want %v", tt.bytesPerSec, result, tt.expected)
		}
	}
}

func TestFormatETA(t *testing.T) {
	tests := []struct {
		progress      float64
		bytesPerMs    float64
		totalBytes    int64
		completedBytes int64
		expected      string
	}{
		{0.0, 1000, 10000, 0, "Calculating..."},
		{0.5, 0, 10000, 5000, "Calculating..."},
		{0.5, 1, 10000, 5000, "5s"}, // 5000 bytes remaining, 1 byte per ms = 5000 ms = 5s
		{1.0, 10, 1000, 1000, "0s"},
	}

	for _, tt := range tests {
		result := formatETA(tt.progress, tt.bytesPerMs, tt.totalBytes, tt.completedBytes)
		if result != tt.expected {
			t.Errorf("formatETA(%v, %v, %v, %v) = %v; want %v", 
				tt.progress, tt.bytesPerMs, tt.totalBytes, tt.completedBytes, result, tt.expected)
		}
	}
}

func TestTUIModelInitialization(t *testing.T) {
	state := &UIState{
		TotalFiles: 100,
		MaxWorkers: 10,
	}
	model := NewTUIModel(state)

	if model.engineState.TotalFiles != 100 {
		t.Errorf("Expected TotalFiles 100, got %d", model.engineState.TotalFiles)
	}

	
	view := model.View()
	if view == "" {
		t.Errorf("View rendered empty string")
	}
	
	if !strings.Contains(view, "Initializing...") {
		t.Errorf("Expected Initializing view when width is 0")
	}
}
