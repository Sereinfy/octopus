package relay

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestImageEndpointPath(t *testing.T) {
	tests := []struct {
		chatPath string
		format   llm.APIFormat
		want     string
	}{
		{"/v1/chat/completions", llm.APIFormatOpenAIImageGeneration, "/v1/images/generations"},
		{"/chat/completions", llm.APIFormatOpenAIImageEdit, "/images/edits"},
		{"/api/v3/custom", llm.APIFormatOpenAIImageGeneration, "/api/v3/images/generations"},
		{"", llm.APIFormatOpenAIImageEdit, "/v1/images/edits"},
	}
	for _, test := range tests {
		if got := imageEndpointPath(test.chatPath, test.format); got != test.want {
			t.Fatalf("imageEndpointPath(%q, %q) = %q, want %q", test.chatPath, test.format, got, test.want)
		}
	}
}

func TestRequestBodyForStateOmitsImageData(t *testing.T) {
	raw := &httpclient.Request{
		Body: []byte(`{"model":"gpt-image-1","image":"data:image/png;base64,AAAA"}`),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}
	body := requestBodyForState(raw, llm.APIFormatOpenAIImageGeneration)
	if strings.Contains(body, "AAAA") {
		t.Fatalf("request state contains image data: %s", body)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("request state is not valid JSON: %v", err)
	}
	if decoded["model"] != "gpt-image-1" {
		t.Fatalf("request state lost model: %v", decoded["model"])
	}
}
