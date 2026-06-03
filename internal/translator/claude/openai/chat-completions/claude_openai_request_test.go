package chat_completions

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIRequestToClaude_ToolResultTextAndBase64Image(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{
				"role": "assistant",
				"content": "",
				"tool_calls": [
					{
						"id": "call_1",
						"type": "function",
						"function": {
							"name": "do_work",
							"arguments": "{\"a\":1}"
						}
					}
				]
			},
			{
				"role": "tool",
				"tool_call_id": "call_1",
				"content": [
					{"type": "text", "text": "tool ok"},
					{
						"type": "image_url",
						"image_url": {
							"url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg=="
						}
					}
				]
			}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)
	messages := resultJSON.Get("messages").Array()

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d. Messages: %s", len(messages), resultJSON.Get("messages").Raw)
	}

	toolResult := messages[1].Get("content.0")
	if got := toolResult.Get("type").String(); got != "tool_result" {
		t.Fatalf("Expected content[0].type %q, got %q", "tool_result", got)
	}
	if got := toolResult.Get("tool_use_id").String(); got != "call_1" {
		t.Fatalf("Expected tool_use_id %q, got %q", "call_1", got)
	}

	toolContent := toolResult.Get("content")
	if !toolContent.IsArray() {
		t.Fatalf("Expected tool_result content array, got %s", toolContent.Raw)
	}
	if got := toolContent.Get("0.type").String(); got != "text" {
		t.Fatalf("Expected first tool_result part type %q, got %q", "text", got)
	}
	if got := toolContent.Get("0.text").String(); got != "tool ok" {
		t.Fatalf("Expected first tool_result part text %q, got %q", "tool ok", got)
	}
	if got := toolContent.Get("1.type").String(); got != "image" {
		t.Fatalf("Expected second tool_result part type %q, got %q", "image", got)
	}
	if got := toolContent.Get("1.source.type").String(); got != "base64" {
		t.Fatalf("Expected image source type %q, got %q", "base64", got)
	}
	if got := toolContent.Get("1.source.media_type").String(); got != "image/png" {
		t.Fatalf("Expected image media type %q, got %q", "image/png", got)
	}
	if got := toolContent.Get("1.source.data").String(); got != "iVBORw0KGgoAAAANSUhEUg==" {
		t.Fatalf("Unexpected base64 image data: %q", got)
	}
}

func TestConvertOpenAIRequestToClaude_ToolResultURLImageOnly(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{
				"role": "assistant",
				"content": "",
				"tool_calls": [
					{
						"id": "call_1",
						"type": "function",
						"function": {
							"name": "do_work",
							"arguments": "{\"a\":1}"
						}
					}
				]
			},
			{
				"role": "tool",
				"tool_call_id": "call_1",
				"content": [
					{
						"type": "image_url",
						"image_url": {
							"url": "https://example.com/tool.png"
						}
					}
				]
			}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)
	messages := resultJSON.Get("messages").Array()

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d. Messages: %s", len(messages), resultJSON.Get("messages").Raw)
	}

	toolContent := messages[1].Get("content.0.content")
	if !toolContent.IsArray() {
		t.Fatalf("Expected tool_result content array, got %s", toolContent.Raw)
	}
	if got := toolContent.Get("0.type").String(); got != "image" {
		t.Fatalf("Expected tool_result part type %q, got %q", "image", got)
	}
	if got := toolContent.Get("0.source.type").String(); got != "url" {
		t.Fatalf("Expected image source type %q, got %q", "url", got)
	}
	if got := toolContent.Get("0.source.url").String(); got != "https://example.com/tool.png" {
		t.Fatalf("Unexpected image URL: %q", got)
	}
}

func TestConvertOpenAIRequestToClaude_SystemRoleBecomesTopLevelSystem(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Hello"}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)

	system := resultJSON.Get("system")
	if !system.IsArray() {
		t.Fatalf("Expected top-level system array, got %s", system.Raw)
	}
	if len(system.Array()) != 1 {
		t.Fatalf("Expected 1 system block, got %d. System: %s", len(system.Array()), system.Raw)
	}
	if got := system.Get("0.type").String(); got != "text" {
		t.Fatalf("Expected system block type %q, got %q", "text", got)
	}
	if got := system.Get("0.text").String(); got != "You are a helpful assistant." {
		t.Fatalf("Expected system text %q, got %q", "You are a helpful assistant.", got)
	}

	messages := resultJSON.Get("messages").Array()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 non-system message, got %d. Messages: %s", len(messages), resultJSON.Get("messages").Raw)
	}
	if got := messages[0].Get("role").String(); got != "user" {
		t.Fatalf("Expected remaining message role %q, got %q", "user", got)
	}
	if got := messages[0].Get("content.0.text").String(); got != "Hello" {
		t.Fatalf("Expected user text %q, got %q", "Hello", got)
	}
}

func TestConvertOpenAIRequestToClaude_MultipleSystemMessagesMergedIntoTopLevelSystem(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{"role": "system", "content": "Rule 1"},
			{"role": "system", "content": [{"type": "text", "text": "Rule 2"}]},
			{"role": "user", "content": "Hello"}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)

	system := resultJSON.Get("system").Array()
	if len(system) != 2 {
		t.Fatalf("Expected 2 system blocks, got %d. System: %s", len(system), resultJSON.Get("system").Raw)
	}
	if got := system[0].Get("text").String(); got != "Rule 1" {
		t.Fatalf("Expected first system text %q, got %q", "Rule 1", got)
	}
	if got := system[1].Get("text").String(); got != "Rule 2" {
		t.Fatalf("Expected second system text %q, got %q", "Rule 2", got)
	}

	messages := resultJSON.Get("messages").Array()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 non-system message, got %d. Messages: %s", len(messages), resultJSON.Get("messages").Raw)
	}
	if got := messages[0].Get("role").String(); got != "user" {
		t.Fatalf("Expected remaining message role %q, got %q", "user", got)
	}
	if got := messages[0].Get("content.0.text").String(); got != "Hello" {
		t.Fatalf("Expected user text %q, got %q", "Hello", got)
	}
}

func TestConvertOpenAIRequestToClaude_SystemOnlyInputKeepsFallbackUserMessage(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)

	system := resultJSON.Get("system").Array()
	if len(system) != 1 {
		t.Fatalf("Expected 1 system block, got %d. System: %s", len(system), resultJSON.Get("system").Raw)
	}
	if got := system[0].Get("text").String(); got != "You are a helpful assistant." {
		t.Fatalf("Expected system text %q, got %q", "You are a helpful assistant.", got)
	}

	messages := resultJSON.Get("messages").Array()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 fallback message, got %d. Messages: %s", len(messages), resultJSON.Get("messages").Raw)
	}
	if got := messages[0].Get("role").String(); got != "user" {
		t.Fatalf("Expected fallback message role %q, got %q", "user", got)
	}
	if got := messages[0].Get("content.0.type").String(); got != "text" {
		t.Fatalf("Expected fallback content type %q, got %q", "text", got)
	}
	if got := messages[0].Get("content.0.text").String(); got != "" {
		t.Fatalf("Expected fallback text %q, got %q", "", got)
	}
}

func TestConvertOpenAIRequestToClaude_AssistantReasoningContentBecomesThinkingBeforeTextAndToolUse(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{
				"role": "assistant",
				"content": "I will inspect the file.",
				"reasoning_content": "Need to read before editing.",
				"tool_calls": [
					{"id":"call_1","type":"function","function":{"name":"Read","arguments":"{\"file_path\":\"/tmp/a.go\"}"}}
				]
			}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	content := gjson.GetBytes(result, "messages.0.content")

	if got := content.Get("0.type").String(); got != "thinking" {
		t.Fatalf("Expected first content type %q, got %q. Content: %s", "thinking", got, content.Raw)
	}
	if got := content.Get("0.thinking").String(); got != "Need to read before editing." {
		t.Fatalf("Expected thinking text %q, got %q", "Need to read before editing.", got)
	}
	if got := content.Get("1.type").String(); got != "text" {
		t.Fatalf("Expected second content type %q, got %q. Content: %s", "text", got, content.Raw)
	}
	if got := content.Get("1.text").String(); got != "I will inspect the file." {
		t.Fatalf("Expected text content %q, got %q", "I will inspect the file.", got)
	}
	if got := content.Get("2.type").String(); got != "tool_use" {
		t.Fatalf("Expected third content type %q, got %q. Content: %s", "tool_use", got, content.Raw)
	}
	if got := content.Get("2.id").String(); got != "call_1" {
		t.Fatalf("Expected tool_use id %q, got %q", "call_1", got)
	}
}

func TestConvertOpenAIRequestToClaude_AssistantReasoningFallbackKeyBecomesThinking(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{"role":"assistant","content":[{"type":"text","text":"done"}],"reasoning":"legacy reasoning"}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	content := gjson.GetBytes(result, "messages.0.content")

	if got := content.Get("0.type").String(); got != "thinking" {
		t.Fatalf("Expected first content type %q, got %q. Content: %s", "thinking", got, content.Raw)
	}
	if got := content.Get("0.thinking").String(); got != "legacy reasoning" {
		t.Fatalf("Expected thinking text %q, got %q", "legacy reasoning", got)
	}
	if got := content.Get("1.text").String(); got != "done" {
		t.Fatalf("Expected existing text to be preserved, got %q", got)
	}
}

func TestConvertOpenAIRequestToClaude_IgnoresUserAndBlankAssistantReasoning(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{"role":"user","content":"hi","reasoning_content":"should not become thinking"},
			{"role":"assistant","content":"hello","reasoning_content":"   "}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)

	if got := resultJSON.Get("messages.0.content.0.type").String(); got != "text" {
		t.Fatalf("Expected user content type %q, got %q", "text", got)
	}
	if got := resultJSON.Get("messages.1.content.0.type").String(); got != "text" {
		t.Fatalf("Expected blank assistant reasoning to be skipped, got first type %q", got)
	}
	if resultJSON.Get("messages.1.content.1").Exists() {
		t.Fatalf("Expected no extra assistant content for blank reasoning: %s", resultJSON.Get("messages.1.content").Raw)
	}
}

func TestConvertOpenAIRequestToClaude_ConsecutiveToolResultsMergedIntoSingleUserMessage(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{"role": "user", "content": "Do three things"},
			{
				"role": "assistant",
				"content": "",
				"tool_calls": [
					{"id": "call_1", "type": "function", "function": {"name": "read", "arguments": "{\"path\":\"/a\"}"}},
					{"id": "call_2", "type": "function", "function": {"name": "read", "arguments": "{\"path\":\"/b\"}"}},
					{"id": "call_3", "type": "function", "function": {"name": "read", "arguments": "{\"path\":\"/c\"}"}}
				]
			},
			{"role": "tool", "tool_call_id": "call_1", "content": "result 1"},
			{"role": "tool", "tool_call_id": "call_2", "content": "result 2"},
			{"role": "tool", "tool_call_id": "call_3", "content": "result 3"}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)
	messages := resultJSON.Get("messages").Array()

	// Should produce 3 messages: user, assistant, user (merged tool results)
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages (user, assistant, merged-tool-results), got %d. Messages: %s",
			len(messages), resultJSON.Get("messages").Raw)
	}

	// Verify the third message is a user message with all 3 tool_results
	toolResultMsg := messages[2]
	if got := toolResultMsg.Get("role").String(); got != "user" {
		t.Fatalf("Expected third message role %q, got %q", "user", got)
	}

	contentArr := toolResultMsg.Get("content").Array()
	if len(contentArr) != 3 {
		t.Fatalf("Expected 3 tool_result blocks in merged user message, got %d. Content: %s",
			len(contentArr), toolResultMsg.Get("content").Raw)
	}

	for i, expected := range []string{"call_1", "call_2", "call_3"} {
		if got := contentArr[i].Get("type").String(); got != "tool_result" {
			t.Fatalf("Expected content[%d].type %q, got %q", i, "tool_result", got)
		}
		if got := contentArr[i].Get("tool_use_id").String(); got != expected {
			t.Fatalf("Expected content[%d].tool_use_id %q, got %q", i, expected, got)
		}
	}

	if got := contentArr[0].Get("content").String(); got != "result 1" {
		t.Fatalf("Expected first tool result content %q, got %q", "result 1", got)
	}
	if got := contentArr[2].Get("content").String(); got != "result 3" {
		t.Fatalf("Expected third tool result content %q, got %q", "result 3", got)
	}
}

func TestConvertOpenAIRequestToClaude_ToolResultAfterUserMessageStaysSeparate(t *testing.T) {
	inputJSON := `{
		"model": "gpt-4.1",
		"messages": [
			{"role": "user", "content": "Hello"},
			{
				"role": "assistant",
				"content": "",
				"tool_calls": [
					{"id": "call_1", "type": "function", "function": {"name": "read", "arguments": "{}"}}
				]
			},
			{"role": "tool", "tool_call_id": "call_1", "content": "file content"},
			{"role": "assistant", "content": "Here's what I found"},
			{"role": "user", "content": "Thanks"}
		]
	}`

	result := ConvertOpenAIRequestToClaude("claude-sonnet-4-5", []byte(inputJSON), false)
	resultJSON := gjson.ParseBytes(result)
	messages := resultJSON.Get("messages").Array()

	// Should produce 5 messages: user, assistant(tool_use), user(tool_result), assistant, user
	if len(messages) != 5 {
		t.Fatalf("Expected 5 messages, got %d. Messages: %s", len(messages), resultJSON.Get("messages").Raw)
	}

	// Roles should alternate correctly
	expectedRoles := []string{"user", "assistant", "user", "assistant", "user"}
	for i, expected := range expectedRoles {
		if got := messages[i].Get("role").String(); got != expected {
			t.Fatalf("Expected messages[%d].role %q, got %q", i, expected, got)
		}
	}

	// The tool_result user message should not be merged with the final "Thanks" user message
	if got := messages[2].Get("content.0.type").String(); got != "tool_result" {
		t.Fatalf("Expected messages[2] to contain tool_result, got %q", got)
	}
	if got := messages[4].Get("content.0.text").String(); got != "Thanks" {
		t.Fatalf("Expected messages[4] text %q, got %q", "Thanks", got)
	}
}
