package service

import (
	"strings"
	"testing"
)

func TestExtractOpenAIResponsesText_FromSSEOutputDelta(t *testing.T) {
	body := []byte(strings.Join([]string{
		`event: response.created`,
		`data: {"type":"response.created","response":{"id":"resp_1"}}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"4"}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"2"}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"output":[]}}`,
		``,
	}, "\n"))

	if got := extractOpenAIResponsesText(body); got != "42" {
		t.Fatalf("expected SSE deltas to be aggregated, got %q", got)
	}
}
