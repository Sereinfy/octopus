package relay

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
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

func TestImageAttemptBillableClassifiesKnownHTTPRejections(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "success", err: nil, want: true},
		{name: "bad request", err: &httpclient.Error{StatusCode: http.StatusBadRequest}, want: false},
		{name: "rate limit", err: &httpclient.Error{StatusCode: http.StatusTooManyRequests}, want: false},
		{name: "server error", err: &httpclient.Error{StatusCode: http.StatusInternalServerError}, want: true},
		{name: "timeout", err: context.DeadlineExceeded, want: true},
		{name: "unknown", err: errors.New("connection reset by peer"), want: true},
		{name: "formatted bad request", err: errors.New("HTTP error 422"), want: false},
	}
	for _, test := range tests {
		if got := imageAttemptBillable(test.err); got != test.want {
			t.Fatalf("%s: imageAttemptBillable() = %v, want %v", test.name, got, test.want)
		}
	}
}

func TestRecordImageChargeSkipsNonBillableAttempt(t *testing.T) {
	image := &llm.ImageRequest{Size: "1024x1024"}
	request := &RequestState{}
	request.recordImageCharge(model.Channel{Image1K: 0.1}, image, false)
	request.recordImageCharge(model.Channel{Image1K: 0.1}, image, true)
	if abs(request.Cost) > 1e-9 || abs(request.imageCost-0.1) > 1e-9 {
		t.Fatalf("image cost after settlement = (cost=%v, imageCost=%v), want (0, 0.1)", request.Cost, request.imageCost)
	}
	if request.PricingCount != 1 {
		t.Fatalf("pricing count = %d, want 1", request.PricingCount)
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
	if abs(request.PricingValue-0.2) > 1e-9 || abs(request.imageCost-0.4) > 1e-9 {
		t.Fatalf("mixed image pricing = (%v, %v), want (0.2, 0.4)", request.PricingValue, request.imageCost)
	}
}
