package nexus

import (
	"strings"
	"testing"
)

func TestExtractCopyButtons_Specimens(t *testing.T) {
	opts := DefaultCopyPolicyOptions()
	opts.Enabled = true

	// Specimen 1: Commit SHA
	shaContent := "Fixed in commit 3ec6050a1c74b030ce59686add81f42302a2613c."
	buttons := ExtractCopyButtons(shaContent, opts)
	if len(buttons) != 1 || len(buttons[0]) != 1 {
		t.Fatalf("expected 1 button for SHA, got %v", buttons)
	}
	if buttons[0][0].Text != "Copy SHA" || buttons[0][0].Data != "copy:3ec6050a1c74b030ce59686add81f42302a2613c" {
		t.Fatalf("SHA button mismatch: %+v", buttons[0][0])
	}

	// Specimen 2: WIN
	winContent := "Recorded under WIN 001-006-20260816-003 in workshop."
	buttons = ExtractCopyButtons(winContent, opts)
	if len(buttons) != 1 || len(buttons[0]) != 1 {
		t.Fatalf("expected 1 button for WIN, got %v", buttons)
	}
	if buttons[0][0].Text != "Copy WIN" || buttons[0][0].Data != "copy:001-006-20260816-003" {
		t.Fatalf("WIN button mismatch: %+v", buttons[0][0])
	}

	// Specimen 3: Path
	pathContent := "Check artifact Rooms/workshop/workbench/review/WI-0022-cc-connect-telegram-consecutive-aggregation-seam.md for details."
	buttons = ExtractCopyButtons(pathContent, opts)
	if len(buttons) != 1 || len(buttons[0]) != 1 {
		t.Fatalf("expected 1 button for Path, got %v", buttons)
	}
	if buttons[0][0].Text != "Copy path" || buttons[0][0].Data != "copy:Rooms/workshop/workbench/review/WI-0022-cc-connect-telegram-consecutive-aggregation-seam.md" {
		t.Fatalf("Path button mismatch: %+v", buttons[0][0])
	}

	// Specimen 4: Short Command
	cmdContent := "Run `git status --short` to inspect workspace."
	buttons = ExtractCopyButtons(cmdContent, opts)
	if len(buttons) != 1 || len(buttons[0]) != 1 {
		t.Fatalf("expected 1 button for command, got %v", buttons)
	}
	if buttons[0][0].Text != "Copy command" || buttons[0][0].Data != "copy:git status --short" {
		t.Fatalf("Command button mismatch: %+v", buttons[0][0])
	}
}

func TestExtractCopyButtons_Deduplication(t *testing.T) {
	opts := DefaultCopyPolicyOptions()
	opts.Enabled = true
	content := "Commit 3ec6050a1c74b030ce59686add81f42302a2613c was rebased as 3ec6050a1c74b030ce59686add81f42302a2613c."
	buttons := ExtractCopyButtons(content, opts)

	totalButtons := 0
	for _, row := range buttons {
		totalButtons += len(row)
	}
	if totalButtons != 1 {
		t.Fatalf("expected exactly 1 deduplicated button, got %d", totalButtons)
	}
}

func TestExtractCopyButtons_NormalProse_NoSpam(t *testing.T) {
	opts := DefaultCopyPolicyOptions()
	opts.Enabled = true
	content := "This is a normal conversational response without any git hashes, work item numbers, paths, or terminal commands. Just plain text."
	buttons := ExtractCopyButtons(content, opts)
	if len(buttons) != 0 {
		t.Fatalf("expected 0 buttons for ordinary prose, got %v", buttons)
	}
}

func TestExtractCopyButtons_CappingAndRows(t *testing.T) {
	opts := DefaultCopyPolicyOptions()
	opts.Enabled = true
	opts.MaxButtons = 4
	opts.ButtonsPerRow = 2

	content := `
	WIN: 001-006-20260816-001
	WIN: 001-006-20260816-002
	WIN: 001-006-20260816-003
	WIN: 001-006-20260816-004
	WIN: 001-006-20260816-005
	`
	buttons := ExtractCopyButtons(content, opts)
	if len(buttons) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(buttons))
	}
	if len(buttons[0]) != 2 || len(buttons[1]) != 2 {
		t.Fatalf("expected 2 buttons per row, got row 0: %d, row 1: %d", len(buttons[0]), len(buttons[1]))
	}
}

func TestExtractCopyButtons_OverLimitLengthOmitted(t *testing.T) {
	opts := DefaultCopyPolicyOptions()
	opts.Enabled = true
	opts.MaxCopyTextLen = 50

	longPath := "Rooms/" + strings.Repeat("subfolder/", 10) + "file.md"
	content := "File: " + longPath
	buttons := ExtractCopyButtons(content, opts)
	if len(buttons) != 0 {
		t.Fatalf("expected over-limit target to be omitted, got %v", buttons)
	}
}

func TestExtractCopyButtons_DefaultDisabled(t *testing.T) {
	opts := DefaultCopyPolicyOptions()
	if opts.Enabled {
		t.Fatalf("DefaultCopyPolicyOptions should be disabled by default")
	}
	content := "Commit 3ec6050a1c74b030ce59686add81f42302a2613c under WIN 001-006-20260816-003"
	buttons := ExtractCopyButtons(content, opts)
	if len(buttons) != 0 {
		t.Fatalf("expected 0 buttons when disabled by default, got %v", buttons)
	}
}
