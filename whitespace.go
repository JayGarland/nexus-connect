package nexus

import (
	"strings"
)

// NormalizeExcessiveNewlines collapses 3 or more consecutive newlines (\n{3,})
// outside fenced code blocks to \n\n, while preserving code block contents
// (delimited by ``` or ~~~) untouched.
func NormalizeExcessiveNewlines(s string) string {
	if len(s) == 0 {
		return s
	}

	// Normalize CRLF to LF
	s = strings.ReplaceAll(s, "\r\n", "\n")

	var sb strings.Builder
	sb.Grow(len(s))

	inCodeBlock := false
	var fenceChar byte
	fenceLen := 0

	lines := strings.Split(s, "\n")
	consecutiveEmptyLines := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inCodeBlock {
			sb.WriteString(line)
			if i < len(lines)-1 {
				sb.WriteByte('\n')
			}
			if isClosingFence(trimmed, fenceChar, fenceLen) {
				inCodeBlock = false
				fenceChar = 0
				fenceLen = 0
			}
			continue
		}

		if char, flen, ok := isOpeningFence(trimmed); ok {
			inCodeBlock = true
			fenceChar = char
			fenceLen = flen
			consecutiveEmptyLines = 0
			sb.WriteString(line)
			if i < len(lines)-1 {
				sb.WriteByte('\n')
			}
			continue
		}

		if len(trimmed) == 0 {
			consecutiveEmptyLines++
			// Keep at most 1 empty line (corresponding to a standard \n\n paragraph break)
			if consecutiveEmptyLines <= 1 {
				sb.WriteString(line)
				if i < len(lines)-1 {
					sb.WriteByte('\n')
				}
			}
			continue
		}

		consecutiveEmptyLines = 0
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func isOpeningFence(trimmed string) (byte, int, bool) {
	if len(trimmed) < 3 {
		return 0, 0, false
	}
	char := trimmed[0]
	if char != '`' && char != '~' {
		return 0, 0, false
	}
	count := 0
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == char {
			count++
		} else {
			break
		}
	}
	if count < 3 {
		return 0, 0, false
	}
	return char, count, true
}

func isClosingFence(trimmed string, expectedChar byte, minLen int) bool {
	if len(trimmed) < minLen {
		return false
	}
	count := 0
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == expectedChar {
			count++
		} else {
			return false
		}
	}
	return count >= minLen
}
