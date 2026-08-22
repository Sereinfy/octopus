package relay

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestBuildOutboundUsesOpenAIImagesForResponsesImageRequest(t *testing.T) {
	request := &llm.Request{
		Model:       "gpt-image-1",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageGeneration,
		Image: &llm.ImageRequest{
			Prompt: "draw a blue square",
		},
	}
	channel := &model.Channel{
		Type:    model.ChannelProviderOpenAIResponses,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}

	outbound, passthrough, err := buildOutbound(*channel, request.APIFormat)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/images/generations" {
		t.Fatalf("unexpected image URL: %s", providerRequest.URL)
	}
}

func TestBuildOutboundUsesOpenAIImagesForResponsesImageVariation(t *testing.T) {
	request := &llm.Request{
		Model:       "dall-e-2",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageVariation,
		Image: &llm.ImageRequest{
			Images: [][]byte{[]byte("image")},
		},
	}
	channel := &model.Channel{
		Type:    model.ChannelProviderOpenAIResponses,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}

	outbound, passthrough, err := buildOutbound(*channel, request.APIFormat)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/images/variations" {
		t.Fatalf("unexpected variation URL: %s", providerRequest.URL)
	}
}

func TestBuildOutboundKeepsResponsesForChat(t *testing.T) {
	request := &llm.Request{
		Model:       "gpt-4o",
		RequestType: llm.RequestTypeChat,
		APIFormat:   llm.APIFormatOpenAIChatCompletion,
		Messages: []llm.Message{{
			Role:    "user",
			Content: llm.MessageContent{Content: stringPtr("hello")},
		}},
	}
	channel := &model.Channel{
		Type:    model.ChannelProviderOpenAIResponses,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}

	outbound, passthrough, err := buildOutbound(*channel, request.APIFormat)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("Chat request targeting Responses must use conversion")
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/responses" {
		t.Fatalf("unexpected chat URL: %s", providerRequest.URL)
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
	request := &llm.Request{
		Model:       "gemini-2.5-flash-image",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageGeneration,
		Image:       &llm.ImageRequest{Prompt: "draw a blue square"},
	}
	channel := model.Channel{
		Type:    model.ChannelProviderGemini,
		BaseURL: "https://example.com",
		Key:     "test-key",
	}

	outbound, passthrough, err := buildOutbound(channel, request.APIFormat)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}
	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if !strings.Contains(providerRequest.URL, "generateContent") {
		t.Fatalf("unexpected Gemini image URL: %s", providerRequest.URL)
	}
}

func TestBuildOutboundSupportsVolcengineImageRequest(t *testing.T) {
	request := &llm.Request{
		Model:       "doubao-seedream",
		RequestType: llm.RequestTypeImage,
		APIFormat:   llm.APIFormatOpenAIImageGeneration,
		Image:       &llm.ImageRequest{Prompt: "draw a blue square"},
	}
	channel := model.Channel{
		Type:    model.ChannelProviderVolcengine,
		BaseURL: "https://example.com",
		Key:     "test-key",
	}

	outbound, passthrough, err := buildOutbound(channel, request.APIFormat)
	if err != nil {
		t.Fatalf("buildOutbound returned error: %v", err)
	}
	if passthrough {
		t.Fatal("image requests must not use passthrough")
	}
	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v3/images/generations" {
		t.Fatalf("unexpected Volcengine image URL: %s", providerRequest.URL)
	}
}

func TestConversionMiddlewareValidatesImageResponseUsingClientFormat(t *testing.T) {
	middleware := &conversionMiddleware{
		format:       llm.APIFormatOpenAIChatCompletion,
		clientFormat: llm.APIFormatOpenAIImageGeneration,
	}

	if _, err := middleware.OnOutboundLlmResponse(t.Context(), &llm.Response{Image: &llm.ImageResponse{}}); err == nil {
		t.Fatal("expected empty image response to fail validation")
	}
	if _, err := middleware.OnOutboundLlmResponse(t.Context(), &llm.Response{
		Image: &llm.ImageResponse{Data: []llm.ImageData{{B64JSON: "image"}}},
	}); err != nil {
		t.Fatalf("valid image response failed validation: %v", err)
	}
}

func TestApplyChannelConfigSkipsParamOverrideForMultipart(t *testing.T) {
	override := `{"size":"512x512"}`
	channel := &model.Channel{
		ParamOverride: &override,
	}
	request := &httpclient.Request{
		Headers: http.Header{
			"Content-Type": []string{"multipart/form-data; boundary=test"},
		},
		Body: []byte("multipart body"),
	}

	if err := applyChannelConfig(*channel, request); err != nil {
		t.Fatalf("applyChannelConfig returned error: %v", err)
	}
	if string(request.Body) != "multipart body" {
		t.Fatalf("multipart body was modified: %q", request.Body)
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestImageLogsRedactBinaryFields(t *testing.T) {
	protocol := imageProtocol(llm.APIFormatOpenAIImageGeneration, "/images/generations", nil)
	request := &httpclient.Request{
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Body:    []byte(`{"prompt":"cat","image":"data:image/png;base64,secret"}`),
	}
	requestLog := requestBodyForLog(protocol, request)
	if strings.Contains(requestLog, "secret") {
		t.Fatalf("request log contains image data: %s", requestLog)
	}

	responseLog := responseBodyForLog(protocol, []byte(`{"created":1,"data":[{"b64_json":"secret"}]}`))
	if strings.Contains(responseLog, "secret") {
		t.Fatalf("response log contains image data: %s", responseLog)
	}
}
