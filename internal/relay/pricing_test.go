package relay

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
)

func TestImageResolutionBucketUsesNearestArea(t *testing.T) {
	tests := []struct {
		size string
		want string
	}{
		{size: "1024x1024", want: "1K"},
		{size: "2048x2048", want: "2K"},
		{size: "4096x2160", want: "4K"},
		{size: "auto", want: ""},
	}
	for _, test := range tests {
		if got := imageResolutionBucket(test.size); got != test.want {
			t.Fatalf("imageResolutionBucket(%q) = %q, want %q", test.size, got, test.want)
		}
	}
}

func TestImagePricingFallsBackToHighestConfiguredRate(t *testing.T) {
	channel := model.Channel{Image1K: 0.1, Image2K: 0.3, Image4K: 0.2}
	label, rate, ok := imagePricingForRequest(channel, &llm.ImageRequest{Size: "1024x1024"})
	if !ok || label != "1K" || rate != 0.1 {
		t.Fatalf("recognized pricing = (%q, %v, %v)", label, rate, ok)
	}

	label, rate, ok = imagePricingForRequest(channel, &llm.ImageRequest{Size: "2048x2048"})
	if !ok || label != "2K" || rate != 0.3 {
		t.Fatalf("recognized pricing = (%q, %v, %v)", label, rate, ok)
	}

	label, rate, ok = imagePricingForRequest(channel, &llm.ImageRequest{Size: "auto"})
	if !ok || label != "2K" || rate != 0.3 {
		t.Fatalf("fallback pricing = (%q, %v, %v)", label, rate, ok)
	}
}

func TestImageRequestCountDefaultsToOne(t *testing.T) {
	if got := imageRequestCount(&llm.ImageRequest{}); got != 1 {
		t.Fatalf("default image count = %d, want 1", got)
	}
	n := int64(3)
	if got := imageRequestCount(&llm.ImageRequest{N: &n}); got != 3 {
		t.Fatalf("image count = %d, want 3", got)
	}
	tooMany := int64(maxImageRequestCount + 1)
	if got := imageRequestCount(&llm.ImageRequest{N: &tooMany}); got != maxImageRequestCount {
		t.Fatalf("protected image count = %d, want %d", got, maxImageRequestCount)
	}
}

func TestPricedMetricsUsesImageChargeForRoundStats(t *testing.T) {
	n := int64(3)
	metrics := pricedMetrics(
		model.Channel{Image2K: 0.4},
		"",
		nil,
		&llm.ImageRequest{Size: "2048x2048", N: &n},
	)
	if metrics.InputCost != 0 || abs(metrics.OutputCost-1.2) > 1e-9 {
		t.Fatalf("image round metrics = (%v, %v), want (0, 1.2)", metrics.InputCost, metrics.OutputCost)
	}
}

func TestChargeImageMarksMixedRetryPricing(t *testing.T) {
	image := &llm.ImageRequest{Size: "1024x1024"}
	request := &RequestState{}
	request.chargeImage(model.Channel{Image1K: 0.1}, image)
	request.chargeImage(model.Channel{Image1K: 0.3}, image)
	if request.PricingLabel != "mixed" {
		t.Fatalf("mixed image pricing label = %q, want mixed", request.PricingLabel)
	}
	if abs(request.PricingValue-0.2) > 1e-9 || abs(request.Cost-0.4) > 1e-9 {
		t.Fatalf("mixed image pricing = (%v, %v), want (0.2, 0.4)", request.PricingValue, request.Cost)
	}
}
