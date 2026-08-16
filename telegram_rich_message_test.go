package nexus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

func TestHasMarkdownTable(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "standard table",
			in:   "Here is a table:\n| Name | Age |\n| --- | --- |\n| Alice | 30 |",
			want: true,
		},
		{
			name: "table without outer pipes",
			in:   "Name | Age\n--- | ---\nAlice | 30",
			want: true,
		},
		{
			name: "aligned table",
			in:   "| Col A | Col B |\n| :--- | :---: |\n| 1 | 2 |",
			want: true,
		},
		{
			name: "code fence with pipes (should NOT trigger)",
			in:   "```\n| not | a | table |\n| --- | --- | --- |\n```",
			want: false,
		},
		{
			name: "bullet list with pipes",
			in:   "- item 1 | detail\n- item 2 | detail",
			want: false,
		},
		{
			name: "stray delimiter row without header",
			in:   "| --- | --- |",
			want: false,
		},
		{
			name: "plain text",
			in:   "Just regular text without any tables.",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasMarkdownTable(tt.in)
			if got != tt.want {
				t.Errorf("hasMarkdownTable() = %v, want %v for %q", got, tt.want, tt.in)
			}
		})
	}
}

type mockUnderlyingPlatform struct {
	replyCalled atomic.Int32
	sendCalled  atomic.Int32
	lastReply   string
	lastSend    string
}

func (m *mockUnderlyingPlatform) Name() string                                      { return "mock" }
func (m *mockUnderlyingPlatform) Start(handler core.MessageHandler) error           { return nil }
func (m *mockUnderlyingPlatform) Stop() error                                       { return nil }
func (m *mockUnderlyingPlatform) SendImage(context.Context, any, core.ImageAttachment) error {
	return nil
}

func (m *mockUnderlyingPlatform) Reply(ctx context.Context, replyCtx any, content string) error {
	m.replyCalled.Add(1)
	m.lastReply = content
	return nil
}

func (m *mockUnderlyingPlatform) Send(ctx context.Context, replyCtx any, content string) error {
	m.sendCalled.Add(1)
	m.lastSend = content
	return nil
}

type dummyReplyContext struct {
	chatID    int64
	threadID  int
	messageID int
}

func TestDebouncedTelegramPlatform_RichMessageFallback(t *testing.T) {
	mockUnderlying := &mockUnderlyingPlatform{}

	// Create a mock server that simulates Telegram API failing with 400
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(telegramAPIResponse{
			OK:          false,
			Description: "Bad Request: can't parse rich message",
		})
	}))
	defer mockServer.Close()

	u, _ := url.Parse(mockServer.URL)
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(u),
		},
		Timeout: 5 * time.Second,
	}

	platform := &DebouncedTelegramPlatform{
		underlying:        mockUnderlying,
		token:             "dummy-token",
		httpClient:        httpClient,
		enableRichMessage: true,
	}

	ctx := context.Background()
	rctx := dummyReplyContext{chatID: 12345, threadID: 1, messageID: 10}
	tableContent := "| Col1 | Col2 |\n|---|---|\n| A | B |"

	// Reply with table should attempt rich message, fail, and gracefully fall back to underlying.Reply
	err := platform.Reply(ctx, rctx, tableContent)
	if err != nil {
		t.Fatalf("unexpected error on fallback: %v", err)
	}
	if mockUnderlying.replyCalled.Load() != 1 {
		t.Errorf("expected underlying.Reply to be called once on fallback, got %d", mockUnderlying.replyCalled.Load())
	}
	if mockUnderlying.lastReply != tableContent {
		t.Errorf("expected underlying.Reply to receive original content, got %q", mockUnderlying.lastReply)
	}

	// Reply without table should directly call underlying.Reply without attempting rich message
	nonTableContent := "Hello world"
	err = platform.Reply(ctx, rctx, nonTableContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockUnderlying.replyCalled.Load() != 2 {
		t.Errorf("expected underlying.Reply count 2, got %d", mockUnderlying.replyCalled.Load())
	}
}
