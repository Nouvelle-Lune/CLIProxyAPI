package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertGeminiCLIResponseToClaude_StreamTerminalChunkKeepsToolUseStopReason(t *testing.T) {
	var param any
	ctx := context.Background()
	originalRequest := []byte(`{"tools":[{"name":"read_file"}]}`)

	chunks := [][]byte{
		[]byte(`{"response":{"candidates":[{"content":{"parts":[{"functionCall":{"name":"read_file","args":{"path":"README.md"}}}]}}]}}`),
		[]byte(`{"response":{"candidates":[{"finishReason":"STOP","content":{"parts":[]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2}}}`),
	}

	var outputs [][]byte
	for _, chunk := range chunks {
		outputs = append(outputs, ConvertGeminiCLIResponseToClaude(ctx, "", originalRequest, nil, chunk, &param)...)
	}

	stopReason := ""
	for _, out := range outputs {
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := gjson.Parse(strings.TrimPrefix(line, "data: "))
			if data.Get("type").String() == "message_delta" {
				stopReason = data.Get("delta.stop_reason").String()
			}
		}
	}
	if stopReason != "tool_use" {
		t.Fatalf("stop_reason = %q, want tool_use. Outputs=%q", stopReason, outputs)
	}
}
