package relay

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestNewOutboundForRequestUsesOpenAIImagesForResponsesChannel(t *testing.T) {
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

	outbound, err := newOutboundForRequest(request, channel)
	if err != nil {
		t.Fatalf("newOutboundForRequest returned error: %v", err)
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/images/generations" {
		t.Fatalf("unexpected image URL: %s", providerRequest.URL)
	}
}

func TestNewOutboundForRequestUsesOpenAIImageVariations(t *testing.T) {
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

	outbound, err := newOutboundForRequest(request, channel)
	if err != nil {
		t.Fatalf("newOutboundForRequest returned error: %v", err)
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/images/variations" {
		t.Fatalf("unexpected variation URL: %s", providerRequest.URL)
	}
}

func TestNewOutboundForRequestKeepsResponsesForChat(t *testing.T) {
	request := &llm.Request{
		Model:       "gpt-4o",
		RequestType: llm.RequestTypeChat,
		APIFormat:   llm.APIFormatOpenAIChatCompletion,
	}
	channel := &model.Channel{
		Type:    model.ChannelProviderOpenAIResponses,
		BaseURL: "https://example.com/v1",
		Key:     "test-key",
	}

	outbound, err := newOutboundForRequest(request, channel)
	if err != nil {
		t.Fatalf("newOutboundForRequest returned error: %v", err)
	}

	providerRequest, err := outbound.TransformRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if providerRequest.URL != "https://example.com/v1/responses" {
		t.Fatalf("unexpected chat URL: %s", providerRequest.URL)
	}
}

func TestApplyChannelOptionsSkipsParamOverrideForMultipart(t *testing.T) {
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

	if err := applyChannelOptions(channel, request); err != nil {
		t.Fatalf("applyChannelOptions returned error: %v", err)
	}
	if string(request.Body) != "multipart body" {
		t.Fatalf("multipart body was modified: %q", request.Body)
	}
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
