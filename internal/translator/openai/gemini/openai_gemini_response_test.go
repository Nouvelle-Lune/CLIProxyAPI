package gemini

import (
	"context"
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponseToGemini_StreamPreservesToolCallsWhenContentSharesChunk(t *testing.T) {
	var param any
	ctx := context.Background()

	first := []byte(`data: {"choices":[{"index":0,"delta":{"content":"hello","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"README.md\"}"}}]},"finish_reason":null}]}`)
	firstOut := ConvertOpenAIResponseToGemini(ctx, "", nil, nil, first, &param)
	if len(firstOut) != 1 {
		t.Fatalf("first chunk outputs = %d, want 1", len(firstOut))
	}
	if got := gjson.GetBytes(firstOut[0], "candidates.0.content.parts.0.text").String(); got != "hello" {
		t.Fatalf("first chunk text = %q, want hello. Output: %s", got, firstOut[0])
	}

	finish := []byte(`data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
	finishOut := ConvertOpenAIResponseToGemini(ctx, "", nil, nil, finish, &param)
	if len(finishOut) != 1 {
		t.Fatalf("finish chunk outputs = %d, want 1", len(finishOut))
	}
	if got := gjson.GetBytes(finishOut[0], "candidates.0.content.parts.0.functionCall.name").String(); got != "read_file" {
		t.Fatalf("functionCall.name = %q, want read_file. Output: %s", got, finishOut[0])
	}
	if got := gjson.GetBytes(finishOut[0], "candidates.0.content.parts.0.functionCall.args.path").String(); got != "README.md" {
		t.Fatalf("functionCall.args.path = %q, want README.md. Output: %s", got, finishOut[0])
	}
}

func TestConvertOpenAIResponseToGemini_StreamPreservesContentAndToolCallsWhenRoleSharesChunk(t *testing.T) {
	var param any
	ctx := context.Background()

	first := []byte(`data: {"choices":[{"index":0,"delta":{"role":"assistant","content":"hello","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"README.md\"}"}}]},"finish_reason":null}]}`)
	firstOut := ConvertOpenAIResponseToGemini(ctx, "", nil, nil, first, &param)
	if len(firstOut) != 1 {
		t.Fatalf("first chunk outputs = %d, want 1", len(firstOut))
	}
	if got := gjson.GetBytes(firstOut[0], "candidates.0.content.parts.0.text").String(); got != "hello" {
		t.Fatalf("first chunk text = %q, want hello. Output: %s", got, firstOut[0])
	}

	finish := []byte(`data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
	finishOut := ConvertOpenAIResponseToGemini(ctx, "", nil, nil, finish, &param)
	if len(finishOut) != 1 {
		t.Fatalf("finish chunk outputs = %d, want 1", len(finishOut))
	}
	if got := gjson.GetBytes(finishOut[0], "candidates.0.content.parts.0.functionCall.name").String(); got != "read_file" {
		t.Fatalf("functionCall.name = %q, want read_file. Output: %s", got, finishOut[0])
	}
	if got := gjson.GetBytes(finishOut[0], "candidates.0.content.parts.0.functionCall.args.path").String(); got != "README.md" {
		t.Fatalf("functionCall.args.path = %q, want README.md. Output: %s", got, finishOut[0])
	}
}
