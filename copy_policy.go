package nexus

import (
	"regexp"
	"strings"

	"github.com/chenhg5/cc-connect/core"
)

var (
	// WIN: 001-006-20260816-003
	reWIN = regexp.MustCompile(`\b\d{3}-\d{3}-\d{8}-\d{3}\b`)

	// Hex SHA candidate pattern (7 to 40 characters)
	reHexCandidate = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)

	// File / World path patterns
	reFilePath = regexp.MustCompile(`(?:\b(?:Rooms|post-office|Party|foundry|cmd|core|platform|agent|config|daemon|extensions|internal|web)\/[A-Za-z0-9_.\-\/]+\b|[A-Za-z0-9_.\-\/]+\.(?:md|go|json|yaml|yml|py|ts|js|toml|sql|sh)\b|[A-Za-z]:\\[A-Za-z0-9_.\-\\\/]+)`)

	// Short command patterns inside backticks
	reInlineCode = regexp.MustCompile("`([^`\n]+)`")

	// Common CLI prefixes for command recognition
	cmdPrefixes = []string{
		"git ", "go ", "python ", "python3 ", "npm ", "pnpm ", "npx ", "cargo ", "curl ", "docker ", "kubectl ", "gh ", "agy ",
	}
)

type CopyPolicyOptions struct {
	MaxButtons      int
	ButtonsPerRow   int
	MaxCopyTextLen  int
	EnableSHA       bool
	EnableWIN       bool
	EnablePath      bool
	EnableCommand   bool
}

func DefaultCopyPolicyOptions() CopyPolicyOptions {
	return CopyPolicyOptions{
		MaxButtons:     4,
		ButtonsPerRow:  2,
		MaxCopyTextLen: 256,
		EnableSHA:      true,
		EnableWIN:      true,
		EnablePath:     true,
		EnableCommand:  true,
	}
}

type copyCandidate struct {
	label string
	text  string
}

// ExtractCopyButtons analyzes message content and returns structured rows of CopyText ButtonOptions.
func ExtractCopyButtons(content string, opts CopyPolicyOptions) [][]core.ButtonOption {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	if opts.MaxButtons <= 0 {
		opts.MaxButtons = 4
	}
	if opts.ButtonsPerRow <= 0 {
		opts.ButtonsPerRow = 2
	}
	if opts.MaxCopyTextLen <= 0 {
		opts.MaxCopyTextLen = 256
	}

	var candidates []copyCandidate
	seen := make(map[string]bool)

	addCandidate := func(label, text string) {
		text = strings.TrimSpace(text)
		// Clean outer quotes or backticks if any
		text = strings.Trim(text, "`\"'")
		if text == "" || len(text) > opts.MaxCopyTextLen {
			return
		}
		if seen[text] {
			return
		}
		seen[text] = true
		candidates = append(candidates, copyCandidate{label: label, text: text})
	}

	// 1. Extract WINs
	if opts.EnableWIN {
		for _, m := range reWIN.FindAllString(content, -1) {
			addCandidate("Copy WIN", m)
		}
	}

	// 2. Extract Git SHAs
	if opts.EnableSHA {
		for _, m := range reHexCandidate.FindAllString(content, -1) {
			if isHexSHA(m) {
				addCandidate("Copy SHA", m)
			}
		}
	}

	// 3. Extract Command invocations from backticks
	if opts.EnableCommand {
		for _, match := range reInlineCode.FindAllStringSubmatch(content, -1) {
			if len(match) > 1 {
				cmdText := strings.TrimSpace(match[1])
				for _, prefix := range cmdPrefixes {
					if strings.HasPrefix(cmdText, prefix) {
						addCandidate("Copy command", cmdText)
						break
					}
				}
			}
		}
	}

	// 4. Extract Paths
	if opts.EnablePath {
		for _, m := range reFilePath.FindAllString(content, -1) {
			// Exclude if it looks like a URL
			if strings.HasPrefix(m, "http://") || strings.HasPrefix(m, "https://") {
				continue
			}
			addCandidate("Copy path", m)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Bound to MaxButtons
	if len(candidates) > opts.MaxButtons {
		candidates = candidates[:opts.MaxButtons]
	}

	// Group into rows
	var rows [][]core.ButtonOption
	var currentRow []core.ButtonOption

	for _, c := range candidates {
		currentRow = append(currentRow, core.ButtonOption{
			Text:     c.label,
			CopyText: c.text,
		})
		if len(currentRow) == opts.ButtonsPerRow {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	return rows
}

func isHexSHA(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	hasLetter := false
	for _, c := range s {
		if c >= 'a' && c <= 'f' {
			hasLetter = true
		} else if c < '0' || c > '9' {
			return false
		}
	}
	// 40-char hashes are always valid SHAs; shorter hashes must contain at least one hex letter
	return len(s) == 40 || hasLetter
}
