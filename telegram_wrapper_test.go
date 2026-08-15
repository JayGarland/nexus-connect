package nexus

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

type wrapperMockPlatform struct {
	mu          sync.Mutex
	name        string
	handler     core.MessageHandler
	stopped     bool
	dispatches  []string
	imagesSent  []string
	updatesSent []string
}

func (m *wrapperMockPlatform) Name() string { return m.name }
func (m *wrapperMockPlatform) Start(handler core.MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
	return nil
}
func (m *wrapperMockPlatform) Reply(ctx context.Context, replyCtx any, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dispatches = append(m.dispatches, content)
	return nil
}
func (m *wrapperMockPlatform) Send(ctx context.Context, replyCtx any, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dispatches = append(m.dispatches, content)
	return nil
}
func (m *wrapperMockPlatform) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	return nil
}
func (m *wrapperMockPlatform) ProgressStyle() string {
	return "compact"
}
func (m *wrapperMockPlatform) UpdateMessage(ctx context.Context, replyCtx any, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatesSent = append(m.updatesSent, content)
	return nil
}
func (m *wrapperMockPlatform) SendImage(ctx context.Context, replyCtx any, img core.ImageAttachment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.imagesSent = append(m.imagesSent, img.FileName)
	return nil
}

// Simulate incoming message from underlying platform
func (m *wrapperMockPlatform) Emit(msg *core.Message) {
	m.mu.Lock()
	h := m.handler
	m.mu.Unlock()
	if h != nil {
		h(m, msg)
	}
}

// Test Wrapper E2E cases A-D
func TestDebouncedTelegramPlatform_E2EScenarios(t *testing.T) {
	// Case A: Single Message
	t.Run("CaseA_NormalSingleMessage", func(t *testing.T) {
		mock := &wrapperMockPlatform{name: "telegram"}
		wrapper := &DebouncedTelegramPlatform{
			underlying: mock,
			window:     50 * time.Millisecond,
		}

		var mu sync.Mutex
		var turns []*core.Message
		err := wrapper.Start(func(p core.Platform, msg *core.Message) {
			mu.Lock()
			defer mu.Unlock()
			turns = append(turns, msg)
		})
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		mock.Emit(&core.Message{
			SessionKey: "chat_A",
			Platform:   "telegram",
			UserID:     "user_A",
			MessageID:  "1001",
			Content:    "Normal single message",
		})

		time.Sleep(80 * time.Millisecond)

		mu.Lock()
		count := len(turns)
		var content string
		if count > 0 {
			content = turns[0].Content
		}
		mu.Unlock()

		if count != 1 {
			t.Fatalf("expected 1 turn, got %d", count)
		}
		if content != "Normal single message" {
			t.Fatalf("content mismatch: %q", content)
		}
	})

	// Case B: Rapid follow-up (3 messages in quick succession -> 1 combined turn)
	t.Run("CaseB_RapidFollowUp", func(t *testing.T) {
		mock := &wrapperMockPlatform{name: "telegram"}
		wrapper := &DebouncedTelegramPlatform{
			underlying: mock,
			window:     80 * time.Millisecond,
		}

		var mu sync.Mutex
		var turns []*core.Message
		_ = wrapper.Start(func(p core.Platform, msg *core.Message) {
			mu.Lock()
			defer mu.Unlock()
			turns = append(turns, msg)
		})

		mock.Emit(&core.Message{
			SessionKey: "chat_B",
			Platform:   "telegram",
			UserID:     "user_B",
			MessageID:  "2001",
			Content:    "Part 1",
		})
		time.Sleep(20 * time.Millisecond)

		mock.Emit(&core.Message{
			SessionKey: "chat_B",
			Platform:   "telegram",
			UserID:     "user_B",
			MessageID:  "2002",
			Content:    "Part 2",
		})
		time.Sleep(20 * time.Millisecond)

		mock.Emit(&core.Message{
			SessionKey: "chat_B",
			Platform:   "telegram",
			UserID:     "user_B",
			MessageID:  "2003",
			Content:    "Part 3",
		})

		time.Sleep(120 * time.Millisecond)

		mu.Lock()
		count := len(turns)
		var msg *core.Message
		if count > 0 {
			msg = turns[0]
		}
		mu.Unlock()

		if count != 1 {
			t.Fatalf("expected 1 turn, got %d", count)
		}
		if msg.Content != "Part 1\n\nPart 2\n\nPart 3" {
			t.Fatalf("coalesced content mismatch: %q", msg.Content)
		}
		if msg.MessageID != "2003" {
			t.Fatalf("expected latest message ID 2003, got %q", msg.MessageID)
		}
	})

	// Case C: Delayed follow-up (2 turns separated by quiet window)
	t.Run("CaseC_DelayedFollowUp", func(t *testing.T) {
		mock := &wrapperMockPlatform{name: "telegram"}
		wrapper := &DebouncedTelegramPlatform{
			underlying: mock,
			window:     50 * time.Millisecond,
		}

		var mu sync.Mutex
		var turns []*core.Message
		_ = wrapper.Start(func(p core.Platform, msg *core.Message) {
			mu.Lock()
			defer mu.Unlock()
			turns = append(turns, msg)
		})

		mock.Emit(&core.Message{
			SessionKey: "chat_C",
			Platform:   "telegram",
			UserID:     "user_C",
			MessageID:  "3001",
			Content:    "Turn 1 text",
		})

		time.Sleep(80 * time.Millisecond)

		mock.Emit(&core.Message{
			SessionKey: "chat_C",
			Platform:   "telegram",
			UserID:     "user_C",
			MessageID:  "3002",
			Content:    "Turn 2 text",
		})

		time.Sleep(80 * time.Millisecond)

		mu.Lock()
		count := len(turns)
		mu.Unlock()

		if count != 2 {
			t.Fatalf("expected 2 turns, got %d", count)
		}
	})

	// Case D: Split chunk simulation (>4096 chars)
	t.Run("CaseD_LongTextSplit", func(t *testing.T) {
		mock := &wrapperMockPlatform{name: "telegram"}
		wrapper := &DebouncedTelegramPlatform{
			underlying: mock,
			window:     80 * time.Millisecond,
		}

		var mu sync.Mutex
		var turns []*core.Message
		_ = wrapper.Start(func(p core.Platform, msg *core.Message) {
			mu.Lock()
			defer mu.Unlock()
			turns = append(turns, msg)
		})

		chunk1 := strings.Repeat("X", 3500)
		chunk2 := strings.Repeat("Y", 3500)

		mock.Emit(&core.Message{
			SessionKey: "chat_D",
			Platform:   "telegram",
			UserID:     "user_D",
			MessageID:  "4001",
			Content:    chunk1,
		})
		time.Sleep(20 * time.Millisecond)

		mock.Emit(&core.Message{
			SessionKey: "chat_D",
			Platform:   "telegram",
			UserID:     "user_D",
			MessageID:  "4002",
			Content:    chunk2,
		})

		time.Sleep(120 * time.Millisecond)

		mu.Lock()
		count := len(turns)
		var mergedLen int
		if count > 0 {
			mergedLen = len(turns[0].Content)
		}
		mu.Unlock()

		if count != 1 {
			t.Fatalf("expected 1 turn, got %d", count)
		}
		if mergedLen != 7002 {
			t.Fatalf("expected merged length 7002, got %d", mergedLen)
		}
	})

	// Capability interface delegation test
	t.Run("CapabilityInterfaceDelegation", func(t *testing.T) {
		mock := &wrapperMockPlatform{name: "telegram"}
		var wrapper core.Platform = &DebouncedTelegramPlatform{
			underlying: mock,
			window:     50 * time.Millisecond,
		}

		if updater, ok := wrapper.(core.MessageUpdater); ok {
			_ = updater.UpdateMessage(context.Background(), nil, "updated content")
		} else {
			t.Fatalf("wrapper failed MessageUpdater assertion")
		}

		if sender, ok := wrapper.(core.ImageSender); ok {
			_ = sender.SendImage(context.Background(), nil, core.ImageAttachment{FileName: "test.png"})
		} else {
			t.Fatalf("wrapper failed ImageSender assertion")
		}

		if provider, ok := wrapper.(core.ProgressStyleProvider); ok {
			if style := provider.ProgressStyle(); style != "compact" {
				t.Fatalf("unexpected progress style: %q", style)
			}
		} else {
			t.Fatalf("wrapper failed ProgressStyleProvider assertion")
		}

		mock.mu.Lock()
		updateCount := len(mock.updatesSent)
		imageCount := len(mock.imagesSent)
		mock.mu.Unlock()

		if updateCount != 1 || imageCount != 1 {
			t.Fatalf("capability delegation failed: updates=%d images=%d", updateCount, imageCount)
		}
	})
}
