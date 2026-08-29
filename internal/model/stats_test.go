package model

import "testing"

func TestStatsMetricsAddIncludesCacheTokens(t *testing.T) {
	var total StatsMetrics
	total.Add(StatsMetrics{
		InputToken:      10,
		OutputToken:     20,
		CacheReadToken:  3,
		CacheWriteToken: 4,
		RequestSuccess:  1,
	})
	total.Add(StatsMetrics{CacheReadToken: 2, CacheWriteToken: 1, RequestFailed: 1})

	if total.InputToken != 10 || total.OutputToken != 20 || total.CacheReadToken != 5 || total.CacheWriteToken != 5 {
		t.Fatalf("unexpected token totals: %+v", total)
	}
	if total.RequestSuccess != 1 || total.RequestFailed != 1 {
		t.Fatalf("unexpected request totals: %+v", total)
	}
}
