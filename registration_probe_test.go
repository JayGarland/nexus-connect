package nexus

import (
	"reflect"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func TestRegistrationProbe_DeterministicResolution(t *testing.T) {
	// Probe 1: With text_batch_window_ms > 0, CreatePlatform returns DebouncedTelegramPlatform
	pWrapped, err := core.CreatePlatform("telegram", map[string]any{
		"token":                "test-token",
		"text_batch_window_ms": 500,
	})
	if err != nil {
		t.Fatalf("CreatePlatform with text_batch_window_ms failed: %v", err)
	}

	wrappedType := reflect.TypeOf(pWrapped).String()
	if wrappedType != "*nexus.DebouncedTelegramPlatform" {
		t.Fatalf("expected wrapped platform *nexus.DebouncedTelegramPlatform, got %s", wrappedType)
	}

	debounced, ok := pWrapped.(*DebouncedTelegramPlatform)
	if !ok {
		t.Fatalf("type assertion to *DebouncedTelegramPlatform failed")
	}
	if debounced.window != 500*time.Millisecond {
		t.Fatalf("expected window 500ms, got %v", debounced.window)
	}
	if debounced.underlying == nil {
		t.Fatalf("expected underlying telegram platform to be initialized")
	}

	// Probe 2: With text_batch_window_ms = 0 or omitted, returns DebouncedTelegramPlatform with window=0
	pUnbatched, err := core.CreatePlatform("telegram", map[string]any{
		"token": "test-token",
	})
	if err != nil {
		t.Fatalf("CreatePlatform vanilla failed: %v", err)
	}

	unbatchedType := reflect.TypeOf(pUnbatched).String()
	if unbatchedType != "*nexus.DebouncedTelegramPlatform" {
		t.Fatalf("expected wrapped platform *nexus.DebouncedTelegramPlatform, got %s", unbatchedType)
	}
	debouncedUnbatched, ok := pUnbatched.(*DebouncedTelegramPlatform)
	if !ok {
		t.Fatalf("type assertion to *DebouncedTelegramPlatform failed")
	}
	if debouncedUnbatched.window != 0 {
		t.Fatalf("expected window 0, got %v", debouncedUnbatched.window)
	}
}

func TestRegistrationProbe_FeatureToggleMatrix(t *testing.T) {
	tests := []struct {
		name           string
		opts           map[string]any
		expectedWindow time.Duration
		expectedCopy   bool
		expectedSHA    bool
		expectedWIN    bool
		expectedPath   bool
		expectedCmd    bool
		expectedMax    int
	}{
		{
			name: "Matrix 1: Aggregation OFF / Copy OFF",
			opts: map[string]any{
				"token":                    "test-token",
				"nexus_aggregation_enabled": false,
				"text_batch_window_ms":     1000,
				"nexus_copy_enabled":        false,
			},
			expectedWindow: 0,
			expectedCopy:   false,
			expectedSHA:    true,
			expectedWIN:    true,
			expectedPath:   true,
			expectedCmd:    true,
			expectedMax:    4,
		},
		{
			name: "Matrix 2: Aggregation ON / Copy OFF",
			opts: map[string]any{
				"token":                    "test-token",
				"nexus_aggregation_enabled": true,
				"text_batch_window_ms":     1000,
				"nexus_copy_enabled":        false,
			},
			expectedWindow: 1000 * time.Millisecond,
			expectedCopy:   false,
			expectedSHA:    true,
			expectedWIN:    true,
			expectedPath:   true,
			expectedCmd:    true,
			expectedMax:    4,
		},
		{
			name: "Matrix 3: Aggregation OFF / Copy ON",
			opts: map[string]any{
				"token":                    "test-token",
				"nexus_aggregation_enabled": false,
				"text_batch_window_ms":     1000,
				"nexus_copy_enabled":        true,
			},
			expectedWindow: 0,
			expectedCopy:   true,
			expectedSHA:    true,
			expectedWIN:    true,
			expectedPath:   true,
			expectedCmd:    true,
			expectedMax:    4,
		},
		{
			name: "Matrix 4: Aggregation ON / Copy ON",
			opts: map[string]any{
				"token":                    "test-token",
				"nexus_aggregation_enabled": true,
				"text_batch_window_ms":     1000,
				"nexus_copy_enabled":        true,
			},
			expectedWindow: 1000 * time.Millisecond,
			expectedCopy:   true,
			expectedSHA:    true,
			expectedWIN:    true,
			expectedPath:   true,
			expectedCmd:    true,
			expectedMax:    4,
		},
		{
			name: "Backward Compat: Omitted flags default to ON with text_batch_window_ms",
			opts: map[string]any{
				"token":                "test-token",
				"text_batch_window_ms": 750,
			},
			expectedWindow: 750 * time.Millisecond,
			expectedCopy:   true,
			expectedSHA:    true,
			expectedWIN:    true,
			expectedPath:   true,
			expectedCmd:    true,
			expectedMax:    4,
		},
		{
			name: "Fine-grained copy switches and max buttons",
			opts: map[string]any{
				"token":              "test-token",
				"nexus_copy_enabled": true,
				"nexus_copy_sha":     false,
				"nexus_copy_win":     true,
				"nexus_copy_path":    false,
				"nexus_copy_command": true,
				"max_copy_buttons":   2,
			},
			expectedWindow: 0,
			expectedCopy:   true,
			expectedSHA:    false,
			expectedWIN:    true,
			expectedPath:   false,
			expectedCmd:    true,
			expectedMax:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := core.CreatePlatform("telegram", tt.opts)
			if err != nil {
				t.Fatalf("CreatePlatform failed: %v", err)
			}
			debounced, ok := p.(*DebouncedTelegramPlatform)
			if !ok {
				t.Fatalf("expected *DebouncedTelegramPlatform")
			}
			if debounced.window != tt.expectedWindow {
				t.Errorf("window = %v, want %v", debounced.window, tt.expectedWindow)
			}
			if debounced.copyOpts.Enabled != tt.expectedCopy {
				t.Errorf("copyOpts.Enabled = %v, want %v", debounced.copyOpts.Enabled, tt.expectedCopy)
			}
			if debounced.copyOpts.EnableSHA != tt.expectedSHA {
				t.Errorf("copyOpts.EnableSHA = %v, want %v", debounced.copyOpts.EnableSHA, tt.expectedSHA)
			}
			if debounced.copyOpts.EnableWIN != tt.expectedWIN {
				t.Errorf("copyOpts.EnableWIN = %v, want %v", debounced.copyOpts.EnableWIN, tt.expectedWIN)
			}
			if debounced.copyOpts.EnablePath != tt.expectedPath {
				t.Errorf("copyOpts.EnablePath = %v, want %v", debounced.copyOpts.EnablePath, tt.expectedPath)
			}
			if debounced.copyOpts.EnableCommand != tt.expectedCmd {
				t.Errorf("copyOpts.EnableCommand = %v, want %v", debounced.copyOpts.EnableCommand, tt.expectedCmd)
			}
			if debounced.copyOpts.MaxButtons != tt.expectedMax {
				t.Errorf("copyOpts.MaxButtons = %v, want %v", debounced.copyOpts.MaxButtons, tt.expectedMax)
			}
		})
	}
}
