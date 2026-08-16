package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

var reTableSepLoose = regexp.MustCompile(`^\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)+\|?$`)

// hasMarkdownTable reports whether md contains a genuine GFM Markdown table.
// Fenced code blocks are skipped so code containing pipes does not false-trigger.
func hasMarkdownTable(md string) bool {
	inCodeBlock := false
	previousLine := ""
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			previousLine = ""
			continue
		}
		if inCodeBlock {
			continue
		}
		if trimmed == "" {
			previousLine = ""
			continue
		}
		if reTableSepLoose.MatchString(trimmed) && tableColumnCount(previousLine) > 1 {
			return true
		}
		previousLine = trimmed
	}
	return false
}

func tableColumnCount(line string) int {
	return len(splitTableCells(line))
}

func splitTableCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	trailingPipeEscaped := false
	for i := len(line) - 2; i >= 0 && line[i] == '\\'; i-- {
		trailingPipeEscaped = !trailingPipeEscaped
	}
	if strings.HasSuffix(line, "|") && !trailingPipeEscaped {
		line = strings.TrimSuffix(line, "|")
	}
	if line == "" {
		return nil
	}

	var cells []string
	var cell strings.Builder
	escaped := false
	for _, r := range line {
		switch r {
		case '|':
			if escaped {
				cell.WriteRune(r)
				escaped = false
			} else {
				cells = append(cells, strings.TrimSpace(cell.String()))
				cell.Reset()
			}
		case '\\':
			if escaped {
				cell.WriteRune('\\')
				escaped = false
			} else {
				escaped = true
			}
		default:
			if escaped {
				cell.WriteRune('\\')
				escaped = false
			}
			cell.WriteRune(r)
		}
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells
}

func extractReplyContext(rctx any) (chatID int64, threadID int, messageID int, ok bool) {
	if rctx == nil {
		return 0, 0, 0, false
	}
	v := reflect.ValueOf(rctx)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0, 0, 0, false
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		name := t.Field(i).Name
		field := v.Field(i)
		switch name {
		case "chatID", "ChatID":
			chatID = field.Int()
		case "threadID", "ThreadID":
			threadID = int(field.Int())
		case "messageID", "MessageID":
			messageID = int(field.Int())
		}
	}
	return chatID, threadID, messageID, true
}

type richMessagePayload struct {
	ChatID          int64               `json:"chat_id"`
	MessageThreadID int                 `json:"message_thread_id,omitempty"`
	RichMessage     richMessageContent  `json:"rich_message"`
	ReplyParameters *richMessageReplyTo `json:"reply_parameters,omitempty"`
}

type richMessageContent struct {
	Markdown string `json:"markdown"`
}

type richMessageReplyTo struct {
	MessageID int `json:"message_id"`
}

type telegramAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// sendRichMessageDirect dispatches a native Telegram table / rich message via Bot API.
func sendRichMessageDirect(ctx context.Context, client *http.Client, token string, chatID int64, threadID int, replyMsgID int, markdown string) error {
	if token == "" || client == nil {
		return fmt.Errorf("nexus: missing telegram token or client")
	}

	payload := richMessagePayload{
		ChatID:          chatID,
		MessageThreadID: threadID,
		RichMessage: richMessageContent{
			Markdown: markdown,
		},
	}
	if replyMsgID > 0 {
		payload.ReplyParameters = &richMessageReplyTo{MessageID: replyMsgID}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("nexus: marshal rich message payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendRichMessage", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("nexus: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("nexus: sendRichMessage request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var apiResp telegramAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("nexus: sendRichMessage status %d: %s", resp.StatusCode, string(respBody))
		}
		return nil
	}

	if !apiResp.OK {
		return fmt.Errorf("nexus: sendRichMessage telegram error: %s", apiResp.Description)
	}

	return nil
}
