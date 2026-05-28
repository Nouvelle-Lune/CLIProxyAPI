package executor

import (
	"testing"

	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	"github.com/tidwall/gjson"
)

func resetClaudeThinkingReplayStore() {
	deepSeekClaudeThinkingReplay = &claudeThinkingReplayStore{items: make(map[string]string)}
}

func TestShouldReplayClaudeThinkingForDeepSeek_ScopesToDeepSeekThinking(t *testing.T) {
	body := []byte(`{"thinking":{"type":"adaptive"}}`)
	if !shouldReplayClaudeThinkingForDeepSeek("claude-opus-4-7", "https://api.deepseek.com/anthropic", body) {
		t.Fatalf("expected DeepSeek Anthropic endpoint to enable replay")
	}
	if !shouldReplayClaudeThinkingForDeepSeek("deepseek/deepseek-v4-pro", "http://127.0.0.1:8317", body) {
		t.Fatalf("expected DeepSeek model alias to enable replay")
	}
	if shouldReplayClaudeThinkingForDeepSeek("claude-opus-4-7", "https://api.anthropic.com", body) {
		t.Fatalf("expected Anthropic endpoint to skip replay")
	}
	if shouldReplayClaudeThinkingForDeepSeek("deepseek/deepseek-v4-pro", "http://127.0.0.1:8317", []byte(`{}`)) {
		t.Fatalf("expected requests without thinking mode to skip replay")
	}
}

func TestReplayClaudeThinkingForToolUse_NoCachedThinkingLeavesBodyUnchanged(t *testing.T) {
	resetClaudeThinkingReplayStore()
	body := []byte(`{"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"ToolSearch","input":{"query":"x"}}]}]}`)
	result, err := replayClaudeThinkingForToolUse(body, "scope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(body) {
		t.Fatalf("expected unchanged body without cached thinking, got %s", result)
	}
}

func TestReplayClaudeThinkingForToolUse_UsesRealResponseThinkingAndPreservesToolResult(t *testing.T) {
	resetClaudeThinkingReplayStore()
	scope := "session-1"
	response := []byte(`{"content":[{"type":"thinking","thinking":"real reasoning","signature":"sig-1"},{"type":"text","text":"Searching"},{"type":"tool_use","id":"call_1","name":"ToolSearch","input":{"query":"x"}}]}`)
	rememberClaudeThinkingForToolUseFromResponse(response, scope)

	body := []byte(`{"messages":[
		{"role":"assistant","content":[{"type":"text","text":"Searching"},{"type":"tool_use","id":"call_1","name":"ToolSearch","input":{"query":"x"}}]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":[{"type":"text","tool_name":"ToolSearch"}],"cache_control":{"type":"ephemeral"}},{"type":"text","text":"Tool loaded."}]}
	]}`)
	result, err := replayClaudeThinkingForToolUse(body, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gjson.GetBytes(result, "messages.0.content.0.type").String(); got != "thinking" {
		t.Fatalf("messages.0.content.0.type = %q, want thinking", got)
	}
	if got := gjson.GetBytes(result, "messages.0.content.0.thinking").String(); got != "real reasoning" {
		t.Fatalf("replayed thinking = %q, want real reasoning", got)
	}
	if got := gjson.GetBytes(result, "messages.0.content.0.signature").String(); got != "sig-1" {
		t.Fatalf("replayed signature = %q, want sig-1", got)
	}
	if got := gjson.GetBytes(result, "messages.0.content.1.text").String(); got != "Searching" {
		t.Fatalf("assistant text changed: %q", got)
	}
	if got := gjson.GetBytes(result, "messages.0.content.2.input.query").String(); got != "x" {
		t.Fatalf("tool_use input changed: %q", got)
	}
	if got := gjson.GetBytes(result, "messages.1.content.0.tool_use_id").String(); got != "call_1" {
		t.Fatalf("tool_result id changed: %q", got)
	}
	if got := gjson.GetBytes(result, "messages.1.content.0.cache_control.type").String(); got != "ephemeral" {
		t.Fatalf("tool_result cache_control changed: %q", got)
	}
	if got := gjson.GetBytes(result, "messages.1.content.0.content.0.tool_name").String(); got != "ToolSearch" {
		t.Fatalf("tool_result content changed: %q", got)
	}
}

func TestReplayClaudeThinkingForToolUse_ReplacesEmptyThinkingWithoutDuplication(t *testing.T) {
	resetClaudeThinkingReplayStore()
	scope := "session-1"
	rememberClaudeThinkingForToolUseFromResponse([]byte(`{"content":[{"type":"thinking","thinking":"real reasoning","signature":""},{"type":"tool_use","id":"call_1","name":"ToolSearch","input":{}}]}`), scope)

	body := []byte(`{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"","signature":""},{"type":"tool_use","id":"call_1","name":"ToolSearch","input":{}}]}]}`)
	result, err := replayClaudeThinkingForToolUse(body, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := gjson.GetBytes(result, "messages.0.content").Array()
	if len(content) != 2 {
		t.Fatalf("expected empty thinking to be replaced without duplication, got %d blocks", len(content))
	}
	if got := gjson.GetBytes(result, "messages.0.content.0.thinking").String(); got != "real reasoning" {
		t.Fatalf("thinking = %q, want real reasoning", got)
	}
	if got := gjson.GetBytes(result, "messages.0.content.1.type").String(); got != "tool_use" {
		t.Fatalf("second block type = %q, want tool_use", got)
	}
}

func TestClaudeThinkingStreamReplayRecorder_StoresThinkingForLaterReplay(t *testing.T) {
	resetClaudeThinkingReplayStore()
	scope := "session-1"
	recorder := newClaudeThinkingStreamReplayRecorder(scope)
	for _, line := range [][]byte{
		[]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature":""}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"real "}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"stream reasoning"}}`),
		[]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig-stream"}}`),
		[]byte(`data: {"type":"content_block_stop","index":0}`),
		[]byte(`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"call_1","name":"WebSearch","input":{}}}`),
	} {
		recorder.consumeLine(line)
	}

	body := []byte(`{"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"WebSearch","input":{}}]}]}`)
	result, err := replayClaudeThinkingForToolUse(body, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := gjson.GetBytes(result, "messages.0.content.0.thinking").String(); got != "real stream reasoning" {
		t.Fatalf("thinking = %q, want real stream reasoning", got)
	}
	if got := gjson.GetBytes(result, "messages.0.content.0.signature").String(); got != "sig-stream" {
		t.Fatalf("signature = %q, want sig-stream", got)
	}
}

func TestClaudeThinkingReplayScope_IsolatesSessionAuthModelAndBaseURL(t *testing.T) {
	baseOpts := cliproxyexecutor.Options{Headers: map[string][]string{"X-Claude-Code-Session-Id": {"session-1"}}}
	baseAuth := &cliproxyauth.Auth{ID: "auth-1"}
	baseScope := claudeThinkingReplayScope(baseAuth, baseOpts, "deepseek/deepseek-v4-pro", "http://127.0.0.1:8317")
	if baseScope == "" {
		t.Fatalf("expected non-empty scope")
	}

	cases := []struct {
		name      string
		auth      *cliproxyauth.Auth
		opts      cliproxyexecutor.Options
		baseModel string
		baseURL   string
	}{
		{
			name:      "session",
			auth:      baseAuth,
			opts:      cliproxyexecutor.Options{Headers: map[string][]string{"X-Claude-Code-Session-Id": {"session-2"}}},
			baseModel: "deepseek/deepseek-v4-pro",
			baseURL:   "http://127.0.0.1:8317",
		},
		{
			name:      "auth",
			auth:      &cliproxyauth.Auth{ID: "auth-2"},
			opts:      baseOpts,
			baseModel: "deepseek/deepseek-v4-pro",
			baseURL:   "http://127.0.0.1:8317",
		},
		{
			name:      "model",
			auth:      baseAuth,
			opts:      baseOpts,
			baseModel: "deepseek/deepseek-v4-lite",
			baseURL:   "http://127.0.0.1:8317",
		},
		{
			name:      "baseURL",
			auth:      baseAuth,
			opts:      baseOpts,
			baseModel: "deepseek/deepseek-v4-pro",
			baseURL:   "https://api.deepseek.com/anthropic",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := claudeThinkingReplayScope(tt.auth, tt.opts, tt.baseModel, tt.baseURL)
			if got == baseScope {
				t.Fatalf("expected %s change to produce isolated scope", tt.name)
			}
		})
	}
}
