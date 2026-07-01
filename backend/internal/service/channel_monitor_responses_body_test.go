package service

import (
	"encoding/json"
	"testing"
)

func TestBuildRequestBody_OpenAIResponsesDefaultInputIsList(t *testing.T) {
	body, err := buildRequestBody(providerOpenAIResponsesAdapter, MonitorProviderOpenAI, MonitorAPIModeResponses, "gpt-test", "return 2", &CheckOptions{APIMode: MonitorAPIModeResponses})
	if err != nil {
		t.Fatalf("build request body: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	input, ok := got["input"].([]any)
	if !ok {
		t.Fatalf("responses input must be a list for OAuth upstream compatibility, got %T: %#v", got["input"], got["input"])
	}
	if len(input) != 1 {
		t.Fatalf("expected one input item, got %#v", input)
	}
}
