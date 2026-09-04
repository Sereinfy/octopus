package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestSendConvertedImageGenerationUsesChatGrantAndImageEndpoint(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode upstream body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`)
	}))
	defer server.Close()

	channel := model.Channel{ChannelConfig: model.ChannelConfig{
		BaseURL:                  server.URL,
		OpenAIChatCompletionPath: "/v1/chat/completions",
	}}
	outbound, target, passthrough, err := buildOutbound(channel, model.ChannelGrant{Protocols: model.ProtocolOpenAIChatCompletion}, model.ChannelKey{ChannelKeyConfig: model.ChannelKeyConfig{Key: "upstream-key"}}, model.ProtocolOpenAIChatCompletion, llm.APIFormatOpenAIImageGeneration)
	if err != nil {
		t.Fatalf("buildOutbound: %v", err)
	}
	if target != model.ProtocolOpenAIChatCompletion || passthrough {
		t.Fatalf("image target = (%v, passthrough=%v), want chat conversion", target, passthrough)
	}

	raw := &httpclient.Request{
		Method:  http.MethodPost,
		URL:     "http://relay.test/v1/images/generations",
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    []byte(`{"model":"client-model","prompt":"draw a lighthouse"}`),
	}
	result, err := sendConverted(context.Background(), llm.APIFormatOpenAIImageGeneration, raw, channel, outbound, false, "provider-image-model")
	if err != nil {
		t.Fatalf("sendConverted: %v", err)
	}
	if gotPath != "/v1/images/generations" {
		t.Fatalf("upstream path = %q, want /v1/images/generations", gotPath)
	}
	if gotAuth != "Bearer upstream-key" {
		t.Fatalf("upstream auth = %q, want bearer key", gotAuth)
	}
	if gotBody["model"] != "provider-image-model" {
		t.Fatalf("upstream model = %v, want provider-image-model", gotBody["model"])
	}
	if len(result.body) == 0 {
		t.Fatal("image response body is empty")
	}
}

func TestSendConvertedImageEditPreservesMultipartPayload(t *testing.T) {
	var gotPath string
	var gotImage, gotMask []byte
	var gotModel, gotPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Errorf("parse upstream multipart: %v", err)
			return
		}
		gotModel = r.FormValue("model")
		gotPrompt = r.FormValue("prompt")
		if file, _, err := r.FormFile("image"); err == nil {
			gotImage, _ = io.ReadAll(file)
			file.Close()
		}
		if file, _, err := r.FormFile("mask"); err == nil {
			gotMask, _ = io.ReadAll(file)
			file.Close()
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"created":1,"data":[{"url":"https://images.test/result.png"}]}`)
	}))
	defer server.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "client-model"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("prompt", "remove the clouds"); err != nil {
		t.Fatal(err)
	}
	imageHeader := make(textproto.MIMEHeader)
	imageHeader.Set("Content-Disposition", `form-data; name="image"; filename="source.png"`)
	imageHeader.Set("Content-Type", "image/png")
	imagePart, err := writer.CreatePart(imageHeader)
	if err != nil {
		t.Fatal(err)
	}
	imageBytes := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 's', 'o', 'u', 'r', 'c', 'e'}
	if _, err := imagePart.Write(imageBytes); err != nil {
		t.Fatal(err)
	}
	maskHeader := make(textproto.MIMEHeader)
	maskHeader.Set("Content-Disposition", `form-data; name="mask"; filename="mask.png"`)
	maskHeader.Set("Content-Type", "image/png")
	maskPart, err := writer.CreatePart(maskHeader)
	if err != nil {
		t.Fatal(err)
	}
	maskBytes := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 'm', 'a', 's', 'k'}
	if _, err := maskPart.Write(maskBytes); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	channel := model.Channel{ChannelConfig: model.ChannelConfig{
		BaseURL:                  server.URL,
		OpenAIChatCompletionPath: "/v1/chat/completions",
	}}
	outbound, _, _, err := buildOutbound(channel, model.ChannelGrant{Protocols: model.ProtocolOpenAIChatCompletion}, model.ChannelKey{ChannelKeyConfig: model.ChannelKeyConfig{Key: "upstream-key"}}, model.ProtocolOpenAIChatCompletion, llm.APIFormatOpenAIImageEdit)
	if err != nil {
		t.Fatalf("buildOutbound: %v", err)
	}
	raw := &httpclient.Request{
		Method:  http.MethodPost,
		URL:     "http://relay.test/v1/images/edits",
		Headers: http.Header{"Content-Type": {writer.FormDataContentType()}},
		Body:    body.Bytes(),
	}
	result, err := sendConverted(context.Background(), llm.APIFormatOpenAIImageEdit, raw, channel, outbound, false, "provider-image-model")
	if err != nil {
		t.Fatalf("sendConverted: %v", err)
	}
	if gotPath != "/v1/images/edits" {
		t.Fatalf("upstream path = %q, want /v1/images/edits", gotPath)
	}
	if gotModel != "provider-image-model" || gotPrompt != "remove the clouds" {
		t.Fatalf("upstream fields = model %q prompt %q", gotModel, gotPrompt)
	}
	if !bytes.Equal(gotImage, imageBytes) || !bytes.Equal(gotMask, maskBytes) {
		t.Fatalf("multipart files were not preserved: image=%q mask=%q", gotImage, gotMask)
	}
	if !strings.Contains(string(result.body), "result.png") {
		t.Fatalf("unexpected image response: %s", result.body)
	}
}

func TestSendConvertedRejectsEmptyImageResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"created":1,"data":[]}`)
	}))
	defer server.Close()

	channel := model.Channel{ChannelConfig: model.ChannelConfig{BaseURL: server.URL, OpenAIChatCompletionPath: "/v1/chat/completions"}}
	outbound, _, _, err := buildOutbound(channel, model.ChannelGrant{Protocols: model.ProtocolOpenAIChatCompletion}, model.ChannelKey{ChannelKeyConfig: model.ChannelKeyConfig{Key: "key"}}, model.ProtocolOpenAIChatCompletion, llm.APIFormatOpenAIImageGeneration)
	if err != nil {
		t.Fatalf("buildOutbound: %v", err)
	}
	_, err = sendConverted(context.Background(), llm.APIFormatOpenAIImageGeneration, &httpclient.Request{
		Method:  http.MethodPost,
		URL:     "http://relay.test/v1/images/generations",
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    []byte(`{"model":"client-model","prompt":"draw"}`),
	}, channel, outbound, false, "provider-image-model")
	if err == nil || !strings.Contains(err.Error(), "upstream image response is empty") {
		t.Fatalf("empty image response error = %v", err)
	}
}

func TestBuildOutboundImageRequiresChatGrant(t *testing.T) {
	channel := model.Channel{ChannelConfig: model.ChannelConfig{BaseURL: "https://upstream.test", OpenAIChatCompletionPath: "/v1/chat/completions"}}
	_, _, _, err := buildOutbound(channel, model.ChannelGrant{Protocols: model.ProtocolOpenAIResponse}, model.ChannelKey{ChannelKeyConfig: model.ChannelKeyConfig{Key: "key"}}, model.ProtocolOpenAIChatCompletion, llm.APIFormatOpenAIImageGeneration)
	if err != errNoCompatibleProtocol {
		t.Fatalf("missing chat grant error = %v, want %v", err, errNoCompatibleProtocol)
	}
}
