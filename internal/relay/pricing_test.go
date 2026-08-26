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
}
