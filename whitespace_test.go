package nexus

import (
	"strings"
	"testing"
)

func TestNormalizeExcessiveNewlines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single newline",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "standard markdown paragraph break (2 newlines)",
			input:    "para1\n\npara2",
			expected: "para1\n\npara2",
		},
		{
			name:     "3 newlines outside code block collapsed to 2",
			input:    "para1\n\n\npara2",
			expected: "para1\n\npara2",
		},
		{
			name:     "many newlines outside code block (26 newlines from consecutive tool calls)",
			input:    "the token-flow facts directly." + strings.Repeat("\n", 26) + "Critical confirmation: native",
			expected: "the token-flow facts directly.\n\nCritical confirmation: native",
		},
		{
			name:     "code block with 4 newlines inside - preserved exactly",
			input:    "Before code\n\n\n\n```python\ndef foo():\n\n\n\n    return 42\n```\n\n\n\nAfter code",
			expected: "Before code\n\n```python\ndef foo():\n\n\n\n    return 42\n```\n\nAfter code",
		},
		{
			name:     "tilde code block with internal blank lines - preserved exactly",
			input:    "Start\n\n\n\n~~~json\n{\n\n\n  \"key\": \"value\"\n}\n~~~\n\n\n\nEnd",
			expected: "Start\n\n~~~json\n{\n\n\n  \"key\": \"value\"\n}\n~~~\n\nEnd",
		},
		{
			name:     "streaming in-progress open code block - trailing newlines preserved",
			input:    "Partial stream\n\n\n\n```go\nfunc main() {\n\n\n",
			expected: "Partial stream\n\n```go\nfunc main() {\n\n\n",
		},
		{
			name:     "multiple code blocks interleaved",
			input:    "A\n\n\n\n```\n1\n\n\n2\n```\n\n\n\nB\n\n\n\n```\n3\n\n\n4\n```\n\n\n\nC",
			expected: "A\n\n```\n1\n\n\n2\n```\n\nB\n\n```\n3\n\n\n4\n```\n\nC",
		},
		{
			name:     "CRLF inputs normalized cleanly",
			input:    "Line 1\r\n\r\n\r\n\r\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "list items preserved",
			input:    "- item 1\n- item 2\n\n- item 3",
			expected: "- item 1\n- item 2\n\n- item 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NormalizeExcessiveNewlines(tt.input)
			if actual != tt.expected {
				t.Errorf("\nInput:    %q\nExpected: %q\nActual:   %q", tt.input, tt.expected, actual)
			}
		})
	}
}
