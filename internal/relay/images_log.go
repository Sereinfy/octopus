package relay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/looplj/axonhub/llm/httpclient"
)

// requestBodyForLog 避免将图片字节和 base64 数据写入 Relay 日志。
func requestBodyForLog(protocol *relayProtocol, request *httpclient.Request) string {
	if request == nil || !isImageProtocol(protocol) {
		if request == nil {
			return ""
		}
		return string(request.Body)
	}

	contentType := request.Headers.Get("Content-Type")
	if strings.Contains(strings.ToLower(contentType), "multipart/") {
		return fmt.Sprintf(`{"content_type":%q,"body_bytes":%d}`, contentType, len(request.Body))
	}

	var payload map[string]any
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		return fmt.Sprintf(`{"content_type":%q,"body_bytes":%d}`, contentType, len(request.Body))
	}
	redactImageFields(payload)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"content_type":%q,"body_bytes":%d}`, contentType, len(request.Body))
	}
	return string(body)
}

// responseBodyForLog 避免将生成图片的 base64 数据写入 Relay 日志。
func responseBodyForLog(protocol *relayProtocol, body []byte) string {
	if !isImageProtocol(protocol) || len(body) == 0 {
		return string(body)
	}

	var payload struct {
		Created int64 `json:"created,omitempty"`
		Data    []struct {
			URL           string `json:"url,omitempty"`
			B64JSON       string `json:"b64_json,omitempty"`
			RevisedPrompt string `json:"revised_prompt,omitempty"`
		} `json:"data,omitempty"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Sprintf(`{"content_type":"application/json","body_bytes":%d}`, len(body))
	}
	for i := range payload.Data {
		payload.Data[i].B64JSON = "[redacted]"
	}
	redacted, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"content_type":"application/json","body_bytes":%d}`, len(body))
	}
	return string(redacted)
}

func redactImageFields(payload map[string]any) {
	for _, key := range []string{"image", "mask"} {
		if value, ok := payload[key]; ok {
			switch typed := value.(type) {
			case string:
				payload[key] = fmt.Sprintf("[redacted bytes: %d]", len(typed))
			case []any:
				payload[key] = fmt.Sprintf("[redacted images: %d]", len(typed))
			default:
				payload[key] = "[redacted]"
			}
		}
	}
}
