package relay

import (
	"net/http"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestConversionMiddlewareValidatesImageResponseUsingClientFormat(t *testing.T) {
	middleware := &conversionMiddleware{
		format:       llm.APIFormatOpenAIResponse,
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
	request := &httpclient.Request{
		Headers: http.Header{"Content-Type": []string{"multipart/form-data; boundary=test"}},
		Body:    []byte("multipart body"),
	}

	if err := applyChannelConfig(model.Channel{ParamOverride: &override}, request); err != nil {
		t.Fatalf("applyChannelConfig returned error: %v", err)
	}
	if string(request.Body) != "multipart body" {
		t.Fatalf("multipart body was modified: %q", request.Body)
	}
}
