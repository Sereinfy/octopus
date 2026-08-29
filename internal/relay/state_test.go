package relay

import (
	"testing"

	"github.com/looplj/axonhub/llm"
)

func TestRequestStateKeepsRoundHistoryWithLatestFirst(t *testing.T) {
	request := &RequestState{}
	request.startRound(func() {}, "first-channel", "first-model")
	request.finishRound("first failure")
	request.startRound(func() {}, "second-channel", "second-model")
	mu.Lock()
	request.updateRoundLocked(StatusSuccess, false, "")
	mu.Unlock()

	if len(request.Rounds) != 2 {
		t.Fatalf("round history length = %d, want 2", len(request.Rounds))
	}
	if got := request.Rounds[0]; got.Round != 2 || got.Channel != "second-channel" || got.Status != StatusSuccess || got.Error != "" {
		t.Fatalf("latest round = %+v, want successful round 2", got)
	}
	if got := request.Rounds[1]; got.Round != 1 || got.Status != StatusFailed || got.Error != "first failure" {
		t.Fatalf("previous round = %+v, want failed round 1", got)
	}
}

func TestRequestStateRoundHistoryRecordsFinalFailure(t *testing.T) {
	request := &RequestState{}
	request.startRound(func() {}, "channel", "model")
	request.finishRound("final failure")

	if len(request.Rounds) != 1 || request.Rounds[0].Status != StatusFailed || request.Rounds[0].Error != "final failure" {
		t.Fatalf("round history = %+v, want one failed round", request.Rounds)
	}
}

func TestUsageMetricsExtractsCacheTokensWithoutPrice(t *testing.T) {
	usage := &llm.Usage{
		PromptTokens:     100,
		CompletionTokens: 25,
		PromptTokensDetails: &llm.PromptTokensDetails{
			CachedTokens:      40,
			WriteCachedTokens: 5,
		},
	}

	metrics := usageMetrics("unknown-test-model", usage)
	if metrics.InputToken != 100 || metrics.OutputToken != 25 {
		t.Fatalf("unexpected base tokens: %+v", metrics)
	}
	if metrics.CacheReadToken != 40 || metrics.CacheWriteToken != 5 {
		t.Fatalf("cache token details were not preserved: %+v", metrics)
	}
}
