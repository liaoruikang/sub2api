package handler

import (
	"testing"
)

func TestParseSeedanceUsageTotalTokens(t *testing.T) {
	usage := parseSeedanceUsage([]byte(`{"data":{"usage":{"total_tokens":87300}}}`))
	if usage.InputTokens != 87300 || usage.OutputTokens != 0 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestParseSeedanceUsagePromptCompletionTokens(t *testing.T) {
	usage := parseSeedanceUsage([]byte(`{"task":{"usage":{"prompt_tokens":72000,"completion_tokens":15300}}}`))
	if usage.InputTokens != 72000 || usage.OutputTokens != 15300 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestParseSeedanceUsageCompletedTaskResponse(t *testing.T) {
	usage := parseSeedanceUsage([]byte(`{"task":{"status":"completed","usage":{"total_tokens":87300,"completion_tokens":87300}}}`))
	if usage.InputTokens != 0 || usage.OutputTokens != 87300 {
		t.Fatalf("unexpected completed task usage: %+v", usage)
	}
}
