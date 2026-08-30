package relay

import (
	"context"
	"testing"
	"time"

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

func TestCancelRequestStopsRetryWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	request := newRequestState("test-model", "{}", 0, cancel)
	defer func() {
		mu.Lock()
		delete(requests, request.ID)
		mu.Unlock()
	}()

	done := make(chan bool, 1)
	go func() {
		done <- request.wait(ctx, 60)
	}()

	CancelRequest(request.ID)

	select {
	case waited := <-done:
		if waited {
			t.Fatal("request wait continued after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("request wait did not stop after cancellation")
	}

	mu.Lock()
	status := request.Status
	mu.Unlock()
	if status != StatusCanceled {
		t.Fatalf("request status = %q, want %q", status, StatusCanceled)
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
