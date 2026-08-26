package relay

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

var testPNG = []byte{137, 80, 78, 71, 13, 10, 26, 10}

func TestValidateImageRequestRejectsUnsafeJSONValues(t *testing.T) {
	zero := int64(0)
	tooMany := int64(maxImageRequestCount + 1)
	compression := int64(101)
	partial := int64(-1)

	tests := []struct {
		name  string
		image *llm.ImageRequest
		want  string
	}{
		{name: "nil request", image: nil, want: "image request is required"},
		{name: "zero n", image: &llm.ImageRequest{N: &zero}, want: "between 1 and"},
		{name: "large n", image: &llm.ImageRequest{N: &tooMany}, want: "between 1 and"},
		{name: "negative partial images", image: &llm.ImageRequest{PartialImages: &partial}, want: "partial_images"},
		{name: "large compression", image: &llm.ImageRequest{OutputCompression: &compression}, want: "output_compression"},
		{name: "unsupported content", image: &llm.ImageRequest{Images: [][]byte{[]byte("not an image")}}, want: "unsupported image type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateImageRequest(test.image)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateImageRequest error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestValidateImageRequestEnforcesImageLimits(t *testing.T) {
	images := make([][]byte, maxImageCount+1)
	for i := range images {
		images[i] = testPNG
	}
	if err := validateImageRequest(&llm.ImageRequest{Images: images}); err == nil {
		t.Fatal("expected image count limit error")
	}

	oversized := make([]byte, maxImageFileSize+1)
	copy(oversized, testPNG)
	if err := validateImageRequest(&llm.ImageRequest{Images: [][]byte{oversized}}); err == nil {
		t.Fatal("expected image size limit error")
	}

	if err := validateImageRequest(&llm.ImageRequest{Images: [][]byte{testPNG}, Mask: testPNG}); err != nil {
		t.Fatalf("valid image request rejected: %v", err)
	}
}

func TestValidateRawImageRequestRejectsMalformedMultipartIntegers(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
	}{
		{name: "invalid n", field: "n", value: "abc"},
		{name: "negative partial images", field: "partial_images", value: "-1"},
		{name: "large compression", field: "output_compression", value: "101"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if err := writer.WriteField(test.field, test.value); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			request := &httpclient.Request{
				Headers: http.Header{"Content-Type": []string{writer.FormDataContentType()}},
				Body:    body.Bytes(),
			}
			err := validateRawImageRequest(llm.APIFormatOpenAIImageEdit, request)
			if err == nil {
				t.Fatal("expected malformed multipart integer to be rejected")
			}
		})
	}
}

func TestCachedInboundReusesParsedRequest(t *testing.T) {
	raw := &httpclient.Request{Body: []byte(`{"prompt":"draw"}`)}
	parsed := &llm.Request{Model: "cached"}
	inbound := &cachedInbound{
		Inbound: imageInboundForFormat(llm.APIFormatOpenAIImageGeneration),
		parsed:  parsed,
		raw:     raw,
	}
	got, err := inbound.TransformRequest(t.Context(), raw)
	if err != nil {
		t.Fatalf("TransformRequest returned error: %v", err)
	}
	if got != parsed {
		t.Fatal("cached inbound did not return the original parsed request")
	}
}
