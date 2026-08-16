package nexus

import (
	"reflect"
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

func TestRegistrationProbe_DeterministicResolution(t *testing.T) {
	// Probe 1: With text_batch_window_ms > 0, CreatePlatform returns DebouncedTelegramPlatform
	pWrapped, err := core.CreatePlatform("telegram", map[string]any{
		"token":               "test-token",
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
	if debounced.window != 500*1000*1000 { // 500ms in nanoseconds
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
	if !debouncedUnbatched.enableRichMessage {
		t.Fatalf("expected enableRichMessage true")
	}
}
