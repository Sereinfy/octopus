package relay

import (
	"net/http"
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
}
