package executor

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const maxClaudeThinkingReplayEntries = 2048

type claudeThinkingReplayStore struct {
	mu    sync.Mutex
	items map[string]string
	order []string
}

var deepSeekClaudeThinkingReplay = &claudeThinkingReplayStore{items: make(map[string]string)}

func shouldReplayClaudeThinkingForDeepSeek(baseModel, baseURL string, body []byte) bool {
	providerKey := strings.ToLower(strings.TrimSpace(baseModel + " " + baseURL))
	if !strings.Contains(providerKey, "deepseek") {
		return false
	}

	thinkingType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "thinking.type").String()))
	return thinkingType == "enabled" || thinkingType == "adaptive" || thinkingType == "auto"
}

func claudeThinkingReplayScope(auth *cliproxyauth.Auth, opts cliproxyexecutor.Options, baseModel, baseURL string) string {
	authID := ""
	if auth != nil {
		authID = auth.ID
	}
	sessionID := strings.TrimSpace(opts.Headers.Get("X-Claude-Code-Session-Id"))
	return strings.Join([]string{sessionID, authID, strings.TrimSpace(baseModel), strings.TrimSpace(baseURL)}, "\x00")
}

func replayClaudeThinkingForToolUse(body []byte, scope string) ([]byte, error) {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return body, nil
	}

	out := body
	for msgIdx, msg := range messages.Array() {
		if strings.TrimSpace(msg.Get("role").String()) != "assistant" {
			continue
		}

		content := msg.Get("content")
		if !content.Exists() || !content.IsArray() {
			continue
		}

		firstThinkingIdx := -1
		hasNonEmptyThinking := false
		firstToolUseID := ""
		partIdx := 0
		content.ForEach(func(_, part gjson.Result) bool {
			defer func() { partIdx++ }()
			switch part.Get("type").String() {
			case "thinking":
				if firstThinkingIdx == -1 {
					firstThinkingIdx = partIdx
				}
				if strings.TrimSpace(part.Get("thinking").String()) != "" {
					hasNonEmptyThinking = true
				}
			case "tool_use":
				if firstToolUseID == "" {
					firstToolUseID = strings.TrimSpace(part.Get("id").String())
				}
			}
			return true
		})
		if hasNonEmptyThinking || firstToolUseID == "" {
			continue
		}

		thinkingBlock, ok := deepSeekClaudeThinkingReplay.get(scope, firstToolUseID)
		if !ok {
			continue
		}

		var err error
		if firstThinkingIdx >= 0 {
			path := fmt.Sprintf("messages.%d.content.%d", msgIdx, firstThinkingIdx)
			out, err = sjson.SetRawBytes(out, path, []byte(thinkingBlock))
			if err != nil {
				return body, fmt.Errorf("claude executor: failed to replay thinking block: %w", err)
			}
			continue
		}

		newContent := []byte(`[]`)
		newContent, err = sjson.SetRawBytes(newContent, "-1", []byte(thinkingBlock))
		if err != nil {
			return body, fmt.Errorf("claude executor: failed to build replayed thinking content: %w", err)
		}
		content.ForEach(func(_, part gjson.Result) bool {
			if err != nil {
				return false
			}
			newContent, err = sjson.SetRawBytes(newContent, "-1", []byte(part.Raw))
			return err == nil
		})
		if err != nil {
			return body, fmt.Errorf("claude executor: failed to preserve assistant content while replaying thinking: %w", err)
		}

		path := fmt.Sprintf("messages.%d.content", msgIdx)
		out, err = sjson.SetRawBytes(out, path, newContent)
		if err != nil {
			return body, fmt.Errorf("claude executor: failed to set replayed thinking content: %w", err)
		}
	}

	return out, nil
}

func rememberClaudeThinkingForToolUseFromResponse(body []byte, scope string) {
	if gjson.ValidBytes(body) {
		rememberClaudeThinkingForToolUseFromMessage(gjson.ParseBytes(body), scope)
		return
	}

	recorder := newClaudeThinkingStreamReplayRecorder(scope)
	for _, line := range bytes.Split(body, []byte("\n")) {
		recorder.consumeLine(line)
	}
}

func rememberClaudeThinkingForToolUseFromMessage(root gjson.Result, scope string) {
	content := root.Get("content")
	if !content.Exists() || !content.IsArray() {
		return
	}

	latestThinking := ""
	content.ForEach(func(_, part gjson.Result) bool {
		switch part.Get("type").String() {
		case "thinking":
			if strings.TrimSpace(part.Get("thinking").String()) != "" {
				latestThinking = part.Raw
			}
		case "tool_use":
			toolUseID := strings.TrimSpace(part.Get("id").String())
			if toolUseID != "" && latestThinking != "" {
				deepSeekClaudeThinkingReplay.put(scope, toolUseID, latestThinking)
			}
		}
		return true
	})
}

type claudeThinkingStreamReplayRecorder struct {
	scope          string
	blocks         map[int]*claudeThinkingStreamBlock
	latestThinking string
}

type claudeThinkingStreamBlock struct {
	blockType string
	thinking  strings.Builder
	signature strings.Builder
}

func newClaudeThinkingStreamReplayRecorder(scope string) *claudeThinkingStreamReplayRecorder {
	return &claudeThinkingStreamReplayRecorder{scope: scope, blocks: make(map[int]*claudeThinkingStreamBlock)}
}

func (r *claudeThinkingStreamReplayRecorder) consumeLine(line []byte) {
	line = bytes.TrimSpace(line)
	if !bytes.HasPrefix(line, []byte("data:")) {
		return
	}
	payload := bytes.TrimSpace(line[len("data:"):])
	if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) || !gjson.ValidBytes(payload) {
		return
	}

	root := gjson.ParseBytes(payload)
	switch root.Get("type").String() {
	case "content_block_start":
		r.consumeContentBlockStart(root)
	case "content_block_delta":
		r.consumeContentBlockDelta(root)
	case "content_block_stop":
		r.consumeContentBlockStop(root)
	}
}

func (r *claudeThinkingStreamReplayRecorder) consumeContentBlockStart(root gjson.Result) {
	idx := int(root.Get("index").Int())
	block := root.Get("content_block")
	blockType := block.Get("type").String()
	switch blockType {
	case "thinking":
		streamBlock := &claudeThinkingStreamBlock{blockType: blockType}
		streamBlock.thinking.WriteString(block.Get("thinking").String())
		streamBlock.signature.WriteString(block.Get("signature").String())
		r.blocks[idx] = streamBlock
	case "tool_use":
		toolUseID := strings.TrimSpace(block.Get("id").String())
		if toolUseID != "" && r.latestThinking != "" {
			deepSeekClaudeThinkingReplay.put(r.scope, toolUseID, r.latestThinking)
		}
	}
}

func (r *claudeThinkingStreamReplayRecorder) consumeContentBlockDelta(root gjson.Result) {
	idx := int(root.Get("index").Int())
	block := r.blocks[idx]
	if block == nil || block.blockType != "thinking" {
		return
	}
	delta := root.Get("delta")
	switch delta.Get("type").String() {
	case "thinking_delta":
		block.thinking.WriteString(delta.Get("thinking").String())
	case "signature_delta":
		block.signature.WriteString(delta.Get("signature").String())
	}
}

func (r *claudeThinkingStreamReplayRecorder) consumeContentBlockStop(root gjson.Result) {
	idx := int(root.Get("index").Int())
	block := r.blocks[idx]
	if block == nil {
		return
	}
	defer delete(r.blocks, idx)

	if block.blockType != "thinking" || strings.TrimSpace(block.thinking.String()) == "" {
		return
	}
	thinkingJSON := []byte(`{"type":"thinking","thinking":"","signature":""}`)
	thinkingJSON, err := sjson.SetBytes(thinkingJSON, "thinking", block.thinking.String())
	if err != nil {
		return
	}
	thinkingJSON, err = sjson.SetBytes(thinkingJSON, "signature", block.signature.String())
	if err != nil {
		return
	}
	r.latestThinking = string(thinkingJSON)
}

func (s *claudeThinkingReplayStore) get(scope, toolUseID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.items[scope+"\x00"+toolUseID]
	return value, ok
}

func (s *claudeThinkingReplayStore) put(scope, toolUseID, thinkingBlock string) {
	if strings.TrimSpace(scope) == "" || strings.TrimSpace(toolUseID) == "" || strings.TrimSpace(thinkingBlock) == "" {
		return
	}

	key := scope + "\x00" + toolUseID
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[key]; !exists {
		s.order = append(s.order, key)
	}
	s.items[key] = thinkingBlock
	for len(s.items) > maxClaudeThinkingReplayEntries && len(s.order) > 0 {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.items, oldest)
	}
}
