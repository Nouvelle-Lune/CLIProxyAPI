package executor

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestNormalizeClaudeThinkingForToolUse_NoMessages(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro"}`)
	result := normalizeClaudeThinkingForToolUse(body)
	if string(result) != string(body) {
		t.Fatalf("expected unchanged body, got %s", string(result))
	}
}

func TestNormalizeClaudeThinkingForToolUse_EmptyMessages(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[]}`)
	result := normalizeClaudeThinkingForToolUse(body)
	if string(result) != string(body) {
		t.Fatalf("expected unchanged body, got %s", string(result))
	}
}

func TestNormalizeClaudeThinkingForToolUse_NoToolUse(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":[{"type":"text","text":"Hello"}]},
		{"role":"assistant","content":[{"type":"text","text":"Hi there"}]}
	]}`)
	result := normalizeClaudeThinkingForToolUse(body)
	// Should not add thinking when there are no tool_use
	thinking := gjson.GetBytes(result, "messages.1.content.0.type")
	if thinking.Exists() && thinking.String() == "thinking" {
		t.Fatalf("expected no thinking block, got one")
	}
}

func TestNormalizeClaudeThinkingForToolUse_WithToolUseNoThinking(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":[{"type":"text","text":"Hello"}]},
		{"role":"assistant","content":[
			{"type":"text","text":"Let me search."},
			{"type":"tool_use","id":"call_1","name":"WebSearch","input":{"query":"test"}}
		]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"result"}]}
	]}`)
	result := normalizeClaudeThinkingForToolUse(body)
	// Should add thinking block for assistant message with tool_use
	thinking := gjson.GetBytes(result, "messages.1.content.0.type")
	if !thinking.Exists() || thinking.String() != "thinking" {
		t.Fatalf("expected thinking block to be added, got %s", thinking.String())
	}
	thinkingText := gjson.GetBytes(result, "messages.1.content.0.thinking")
	if thinkingText.String() != "[reasoning unavailable]" {
		t.Fatalf("expected '[reasoning unavailable]', got %s", thinkingText.String())
	}
}

func TestNormalizeClaudeThinkingForToolUse_WithToolUseAndThinking(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":[{"type":"text","text":"Hello"}]},
		{"role":"assistant","content":[
			{"type":"thinking","thinking":"My reasoning"},
			{"type":"text","text":"Let me search."},
			{"type":"tool_use","id":"call_1","name":"WebSearch","input":{"query":"test"}}
		]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"result"}]}
	]}`)
	result := normalizeClaudeThinkingForToolUse(body)
	// Should not change existing thinking
	thinking := gjson.GetBytes(result, "messages.1.content.0.type")
	if !thinking.Exists() || thinking.String() != "thinking" {
		t.Fatalf("expected thinking block to exist, got %s", thinking.String())
	}
	thinkingText := gjson.GetBytes(result, "messages.1.content.0.thinking")
	if thinkingText.String() != "My reasoning" {
		t.Fatalf("expected 'My reasoning', got %s", thinkingText.String())
	}
}

func TestNormalizeClaudeThinkingForToolUse_MultipleAssistantMessages(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":[{"type":"text","text":"Hello"}]},
		{"role":"assistant","content":[
			{"type":"thinking","thinking":"First reasoning"},
			{"type":"text","text":"Response 1"}
		]},
		{"role":"user","content":[{"type":"text","text":"Continue"}]},
		{"role":"assistant","content":[
			{"type":"text","text":"Let me search."},
			{"type":"tool_use","id":"call_1","name":"WebSearch","input":{"query":"test"}}
		]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"result"}]}
	]}`)
	result := normalizeClaudeThinkingForToolUse(body)
	// Should use latest thinking from previous assistant message
	thinking := gjson.GetBytes(result, "messages.3.content.0.type")
	if !thinking.Exists() || thinking.String() != "thinking" {
		t.Fatalf("expected thinking block to be added, got %s", thinking.String())
	}
	thinkingText := gjson.GetBytes(result, "messages.3.content.0.thinking")
	if thinkingText.String() != "First reasoning" {
		t.Fatalf("expected 'First reasoning', got %s", thinkingText.String())
	}
}

func TestNormalizeClaudeThinkingForToolUse_EmptyThinkingWithToolUse(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":[{"type":"text","text":"Hello"}]},
		{"role":"assistant","content":[
			{"type":"thinking","thinking":""},
			{"type":"text","text":"Let me search."},
			{"type":"tool_use","id":"call_1","name":"WebSearch","input":{"query":"test"}}
		]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"result"}]}
	]}`)
	result := normalizeClaudeThinkingForToolUse(body)
	// Empty thinking should be treated as no thinking
	thinking := gjson.GetBytes(result, "messages.1.content.0.type")
	if !thinking.Exists() || thinking.String() != "thinking" {
		t.Fatalf("expected thinking block to be added, got %s", thinking.String())
	}
	thinkingText := gjson.GetBytes(result, "messages.1.content.0.thinking")
	if thinkingText.String() != "[reasoning unavailable]" {
		t.Fatalf("expected '[reasoning unavailable]', got %s", thinkingText.String())
	}
}

func TestNormalizeClaudeThinkingForToolUse_UserMessageWithToolResult(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":[{"type":"text","text":"Hello"}]},
		{"role":"assistant","content":[
			{"type":"text","text":"Let me search."},
			{"type":"tool_use","id":"call_1","name":"WebSearch","input":{"query":"test"}}
		]},
		{"role":"user","content":[
			{"type":"tool_result","tool_use_id":"call_1","content":"result"},
			{"type":"text","text":"What now?"}
		]}
	]}`)
	result := normalizeClaudeThinkingForToolUse(body)
	// Should only patch assistant messages, not user messages
	thinking := gjson.GetBytes(result, "messages.1.content.0.type")
	if !thinking.Exists() || thinking.String() != "thinking" {
		t.Fatalf("expected thinking block to be added to assistant message, got %s", thinking.String())
	}
}
