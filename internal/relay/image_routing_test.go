package relay

import (
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
)

func TestBuildOutboundUsesOpenAIImagesForResponsesGeneration(t *testing.T) {
	outbound, passthrough, err := buildOutbound(model.Channel{
		Type:    model.ChannelProviderOpenAIResponses,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}, llm.APIFormatOpenAIImageGeneration)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), &llm.Request{
		Model:       "gpt-image-1",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageGeneration,
		Image:       &llm.ImageRequest{Prompt: "draw a blue square"},
	})
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/images/generations" {
		t.Fatalf("unexpected image URL: %s", providerRequest.URL)
	}
}

func TestBuildOutboundUsesOpenAIImagesForResponsesEdit(t *testing.T) {
	outbound, passthrough, err := buildOutbound(model.Channel{
		Type:    model.ChannelProviderOpenAIResponses,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}, llm.APIFormatOpenAIImageEdit)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), &llm.Request{
		Model:       "gpt-image-1",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageEdit,
		Image: &llm.ImageRequest{
			Prompt: "edit this image",
			Images: [][]byte{[]byte("source-image")},
		},
	})
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/images/edits" {
		t.Fatalf("unexpected edit URL: %s", providerRequest.URL)
	}
}

func TestBuildOutboundUsesOpenAIImagesForResponsesVariation(t *testing.T) {
	outbound, passthrough, err := buildOutbound(model.Channel{
		Type:    model.ChannelProviderOpenAIResponses,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}, llm.APIFormatOpenAIImageVariation)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), &llm.Request{
		Model:       "dall-e-2",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageVariation,
		Image:       &llm.ImageRequest{Images: [][]byte{[]byte("image")}},
	})
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/images/variations" {
		t.Fatalf("unexpected variation URL: %s", providerRequest.URL)
	}
}

func TestBuildOutboundKeepsOpenAIChatPassthrough(t *testing.T) {
	outbound, passthrough, err := buildOutbound(model.Channel{
		Type:    model.ChannelProviderOpenAI,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}, llm.APIFormatOpenAIChatCompletion)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if outbound == nil || !passthrough {
		t.Fatal("OpenAI Chat requests should retain passthrough behavior")
	}
}

func TestBuildOutboundRejectsUnsupportedImageProvider(t *testing.T) {
	if _, _, err := buildOutbound(model.Channel{
		Type:    model.ChannelProviderAnthropic,
		BaseURL: "https://example.com",
		Key:     "test-key",
	}, llm.APIFormatOpenAIImageGeneration); err == nil {
		t.Fatal("expected unsupported image provider error")
	}
}

func TestBuildOutboundSupportsGeminiImageRequest(t *testing.T) {
	outbound, passthrough, err := buildOutbound(model.Channel{
		Type:    model.ChannelProviderGemini,
		BaseURL: "https://example.com",
		Key:     "test-key",
	}, llm.APIFormatOpenAIImageGeneration)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), &llm.Request{
		Model:       "gemini-2.5-flash-image",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageGeneration,
		Image:       &llm.ImageRequest{Prompt: "draw a blue square"},
	})
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if !strings.Contains(providerRequest.URL, "generateContent") {
		t.Fatalf("unexpected Gemini image URL: %s", providerRequest.URL)
	}
}

func TestBuildOutboundSupportsVolcengineImageRequest(t *testing.T) {
	outbound, passthrough, err := buildOutbound(model.Channel{
		Type:    model.ChannelProviderVolcengine,
		BaseURL: "https://example.com",
		Key:     "test-key",
	}, llm.APIFormatOpenAIImageGeneration)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), &llm.Request{
		Model:       "doubao-seedream",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageGeneration,
		Image:       &llm.ImageRequest{Prompt: "draw a blue square"},
	})
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v3/images/generations" {
		t.Fatalf("unexpected Volcengine image URL: %s", providerRequest.URL)
	}
}
