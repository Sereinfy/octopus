package relay

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestImageLogsRedactBinaryFields(t *testing.T) {
	request := &httpclient.Request{
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Body:    []byte(`{"prompt":"cat","image":"data:image/png;base64,secret"}`),
	}
	requestLog := requestBodyForLog(llm.APIFormatOpenAIImageGeneration, request)
	if strings.Contains(requestLog, "secret") {
		t.Fatalf("request log contains image data: %s", requestLog)
	}

	responseLog := responseBodyForLog(llm.APIFormatOpenAIImageGeneration, []byte(`{"created":1,"data":[{"b64_json":"secret"}]}`))
	if strings.Contains(responseLog, "secret") {
		t.Fatalf("response log contains image data: %s", responseLog)
	}
	if !strings.Contains(responseLog, "[redacted]") {
		t.Fatalf("response log lost redaction marker: %s", responseLog)
	}

	successLog := responseBodyForLogStatus(llm.APIFormatOpenAIImageGeneration, []byte(`{"created":1,"data":[{"b64_json":"secret"}]}`), true)
	if strings.Contains(successLog, "secret") || !strings.Contains(successLog, `"b64_json":"success"`) {
		t.Fatalf("unexpected successful response log: %s", successLog)
	}
}

func TestMultipartImageRequestLogKeepsFilenamesAndCount(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	addImagePart := func(field, filename string) {
		part, err := writer.CreateFormFile(field, filename)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte("image-bytes"))
	}
	addImagePart("image[]", `C:\\private\\input.png`)
	addImagePart("image[]", "reference.jpg")
	mask, err := writer.CreateFormFile("mask", "mask.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = mask.Write([]byte("mask-bytes"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	contentType := writer.FormDataContentType()
	logBody, ok := multipartImageRequestForLog(contentType, body.Bytes())
	if !ok {
		t.Fatal("multipart log parsing failed")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(logBody), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["reference_image_count"] != float64(2) {
		t.Fatalf("unexpected reference image count: %v", payload["reference_image_count"])
	}
	if !strings.Contains(logBody, `"input.png"`) || !strings.Contains(logBody, `"reference.jpg"`) || !strings.Contains(logBody, `"mask":"mask.png"`) {
		t.Fatalf("unexpected multipart log: %s", logBody)
	}
	if strings.Contains(logBody, "image-bytes") || strings.Contains(logBody, "mask-bytes") {
		t.Fatalf("multipart log contains binary data: %s", logBody)
	}
}

func TestMultipartImageRequestLogFallsBackToCount(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreatePart(textprotoMIMEHeader(`form-data; name="image"`, "image/png"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("image-bytes"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	logBody, ok := multipartImageRequestForLog(writer.FormDataContentType(), body.Bytes())
	if !ok || !strings.Contains(logBody, `"reference_image_count":1`) {
		t.Fatalf("unexpected fallback log: %s", logBody)
	}
	if strings.Contains(logBody, "image-bytes") {
		t.Fatalf("fallback log contains binary data: %s", logBody)
	}
}

func textprotoMIMEHeader(disposition, contentType string) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Disposition": []string{disposition},
		"Content-Type":        []string{contentType},
	}
}
