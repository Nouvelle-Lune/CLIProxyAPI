package executor

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestNormalizeDeepSeekReasoningContent_NoMessages(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro"}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(body) {
		t.Fatalf("expected unchanged body, got %s", string(result))
	}
}

func TestNormalizeDeepSeekReasoningContent_EmptyMessages(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(body) {
		t.Fatalf("expected unchanged body, got %s", string(result))
	}
}

func TestNormalizeDeepSeekReasoningContent_NoToolCalls(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"Hi there"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not add reasoning_content when there are no tool_calls
	reasoning := gjson.GetBytes(result, "messages.1.reasoning_content")
	if reasoning.Exists() {
		t.Fatalf("expected no reasoning_content, got %s", reasoning.String())
	}
}

func TestNormalizeDeepSeekReasoningContent_WithToolCallsNoReasoning(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"result"},
		{"role":"user","content":"Continue"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should add reasoning_content for assistant message with tool_calls
	reasoning := gjson.GetBytes(result, "messages.1.reasoning_content")
	if !reasoning.Exists() {
		t.Fatalf("expected reasoning_content to be added")
	}
	if reasoning.String() != "[reasoning unavailable]" {
		t.Fatalf("expected '[reasoning unavailable]', got %s", reasoning.String())
	}
}

func TestNormalizeDeepSeekReasoningContent_WithToolCallsAndReasoning(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"","reasoning_content":"My reasoning","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"result"},
		{"role":"user","content":"Continue"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not change existing reasoning_content
	reasoning := gjson.GetBytes(result, "messages.1.reasoning_content")
	if !reasoning.Exists() {
		t.Fatalf("expected reasoning_content to exist")
	}
	if reasoning.String() != "My reasoning" {
		t.Fatalf("expected 'My reasoning', got %s", reasoning.String())
	}
}

func TestNormalizeDeepSeekReasoningContent_MultipleAssistantMessages(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"First response","reasoning_content":"First reasoning"},
		{"role":"user","content":"Continue"},
		{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"result"},
		{"role":"user","content":"What now?"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use latest reasoning from previous assistant message
	reasoning := gjson.GetBytes(result, "messages.3.reasoning_content")
	if !reasoning.Exists() {
		t.Fatalf("expected reasoning_content to be added")
	}
	if reasoning.String() != "First reasoning" {
		t.Fatalf("expected 'First reasoning', got %s", reasoning.String())
	}
}

func TestNormalizeDeepSeekReasoningContent_EmptyReasoningInToolCalls(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"","reasoning_content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"result"},
		{"role":"user","content":"Continue"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should add reasoning_content when existing one is empty
	reasoning := gjson.GetBytes(result, "messages.1.reasoning_content")
	if !reasoning.Exists() {
		t.Fatalf("expected reasoning_content to be added")
	}
	if reasoning.String() != "[reasoning unavailable]" {
		t.Fatalf("expected '[reasoning unavailable]', got %s", reasoning.String())
	}
}

func TestNormalizeDeepSeekReasoningContent_WhitespaceReasoningInToolCalls(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"","reasoning_content":"   ","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"result"},
		{"role":"user","content":"Continue"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should add reasoning_content when existing one is whitespace-only
	reasoning := gjson.GetBytes(result, "messages.1.reasoning_content")
	if !reasoning.Exists() {
		t.Fatalf("expected reasoning_content to be added")
	}
	if reasoning.String() != "[reasoning unavailable]" {
		t.Fatalf("expected '[reasoning unavailable]', got %s", reasoning.String())
	}
}

func TestNormalizeDeepSeekReasoningContent_MultipleToolCalls(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"","reasoning_content":"First reasoning","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test1","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"result1"},
		{"role":"assistant","content":"","reasoning_content":"Second reasoning","tool_calls":[{"id":"call_2","type":"function","function":{"name":"test2","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_2","content":"result2"},
		{"role":"user","content":"Continue"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both assistant messages should retain their reasoning_content
	reasoning1 := gjson.GetBytes(result, "messages.1.reasoning_content")
	reasoning2 := gjson.GetBytes(result, "messages.3.reasoning_content")
	if !reasoning1.Exists() || reasoning1.String() != "First reasoning" {
		t.Fatalf("expected 'First reasoning', got %s", reasoning1.String())
	}
	if !reasoning2.Exists() || reasoning2.String() != "Second reasoning" {
		t.Fatalf("expected 'Second reasoning', got %s", reasoning2.String())
	}
}

func TestNormalizeDeepSeekReasoningContent_NoToolCallsInAssistant(t *testing.T) {
	body := []byte(`{"model":"deepseek-v4-pro","messages":[
		{"role":"user","content":"Hello"},
		{"role":"assistant","content":"Response without tool calls"},
		{"role":"user","content":"Continue"}
	]}`)
	result, err := normalizeDeepSeekReasoningContent(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not add reasoning_content when there are no tool_calls
	reasoning := gjson.GetBytes(result, "messages.1.reasoning_content")
	if reasoning.Exists() {
		t.Fatalf("expected no reasoning_content, got %s", reasoning.String())
	}
}
