package relay

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
)

var imageSizePattern = regexp.MustCompile(`(?i)^\s*(\d+)\s*[x×]\s*(\d+)`)

// imagePricingForRequest returns the configured fixed charge for one image request.
// When the requested size cannot be classified, the highest configured charge is used.
func imagePricingForRequest(channel model.Channel, image *llm.ImageRequest) (label string, rate float64, ok bool) {
	label = imageResolutionBucket(imageSize(image))
	configured := []struct {
		label string
		rate  float64
	}{
		{label: "1K", rate: channel.Image1K},
		{label: "2K", rate: channel.Image2K},
		{label: "4K", rate: channel.Image4K},
	}
	for _, candidate := range configured {
		if label == candidate.label && candidate.rate > 0 {
			return candidate.label, candidate.rate, true
		}
	}

	for _, candidate := range configured {
		if candidate.rate > rate {
			label, rate = candidate.label, candidate.rate
		}
	}
	return label, rate, rate > 0
}

func imageSize(image *llm.ImageRequest) string {
	if image == nil {
		return ""
	}
	return strings.TrimSpace(image.Size)
}

func imageResolutionBucket(size string) string {
	normalized := strings.ToUpper(strings.TrimSpace(size))
	for _, label := range []string{"1K", "2K", "4K"} {
		if strings.Contains(normalized, label) {
			return label
		}
	}

	matches := imageSizePattern.FindStringSubmatch(normalized)
	if len(matches) != 3 {
		return ""
	}
	width, widthErr := strconv.ParseFloat(matches[1], 64)
	height, heightErr := strconv.ParseFloat(matches[2], 64)
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return ""
	}

	resolution := width
	if height > resolution {
		resolution = height
	}
	closest := "1K"
	closestDistance := abs(resolution - 1024)
	for _, candidate := range []struct {
		label      string
		resolution float64
	}{
		{label: "2K", resolution: 2048},
		{label: "4K", resolution: 4096},
	} {
		if distance := abs(resolution - candidate.resolution); distance < closestDistance {
			closest = candidate.label
			closestDistance = distance
		}
	}
	return closest
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func imageRequestCount(image *llm.ImageRequest) int64 {
	if image == nil || image.N == nil || *image.N <= 0 {
		return 1
	}
	if *image.N > maxImageRequestCount {
		return maxImageRequestCount
	}
	return *image.N
}

// pricedMetrics applies the channel-specific billing rule to one completed relay round.
// Channel and channel-model statistics are recorded per round, so this must use the
// selected channel rather than the request-level pricing state (which is finalized once).
func pricedMetrics(channel model.Channel, modelName string, usage *llm.Usage, image *llm.ImageRequest) model.StatsMetrics {
	metrics := usageMetrics(modelName, usage)
	if image != nil {
		_, rate, ok := imagePricingForRequest(channel, image)
		if ok {
			metrics.InputCost = 0
			metrics.OutputCost = rate * float64(imageRequestCount(image))
		}
		return metrics
	}
	if channel.Multiplier > 0 {
		metrics.InputCost *= channel.Multiplier
		metrics.OutputCost *= channel.Multiplier
	}
	return metrics
}
