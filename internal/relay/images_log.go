package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

// requestBodyForLog 避免将图片字节和 base64 数据写入 Relay 日志。
func requestBodyForLog(format llm.APIFormat, request *httpclient.Request) string {
	if request == nil || !isImageFormat(format) {
		if request == nil {
			return ""
		}
		return string(request.Body)
	}

	contentType := request.Headers.Get("Content-Type")
	if strings.Contains(strings.ToLower(contentType), "multipart/") {
		if body, ok := multipartImageRequestForLog(contentType, request.Body); ok {
			return body
		}
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

// responseBodyForLog 保持旧调用方的脱敏行为; 成功终态使用 responseBodyForLogStatus。
func responseBodyForLog(format llm.APIFormat, body []byte) string {
	return responseBodyForLogStatus(format, body, false)
}

// responseBodyForLogStatus 避免将生成图片的 base64 数据写入 Relay 日志。
// success 仅在最终响应已完整交付客户端时为 true。
func responseBodyForLogStatus(format llm.APIFormat, body []byte, success bool) string {
	if !isImageFormat(format) || len(body) == 0 {
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
	marker := "[redacted]"
	if success {
		marker = "success"
	}
	for i := range payload.Data {
		if payload.Data[i].URL != "" {
			payload.Data[i].URL = redactImageURL(payload.Data[i].URL)
		}
		if payload.Data[i].B64JSON != "" {
			payload.Data[i].B64JSON = marker
		}
	}
	redacted, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"content_type":"application/json","body_bytes":%d}`, len(body))
	}
	return string(redacted)
}

func redactImageURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "data" {
		return "[redacted]"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

// multipartImageRequestForLog extracts only safe multipart metadata. Binary parts
// are intentionally drained by the multipart reader but never copied to the log.
func multipartImageRequestForLog(contentType string, body []byte) (string, bool) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(strings.ToLower(mediaType), "multipart/") || params["boundary"] == "" {
		return "", false
	}

	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	filenames := make([]string, 0, 2)
	referenceCount := 0
	maskFilename := ""
	maskPresent := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false
		}

		fieldName := part.FormName()
		filename := safeImageFilename(part.FileName())
		isImagePart := filename != "" || strings.HasPrefix(strings.ToLower(part.Header.Get("Content-Type")), "image/")
		switch fieldName {
		case "image", "image[]":
			if isImagePart {
				referenceCount++
				if filename != "" {
					filenames = append(filenames, filename)
				}
			}
		case "mask":
			if isImagePart {
				maskPresent = true
				if filename != "" {
					maskFilename = filename
				}
			}
		}
	}

	payload := map[string]any{
		"content_type":          contentType,
		"reference_image_count": referenceCount,
	}
	if len(filenames) > 0 {
		payload["reference_images"] = filenames
	}
	if maskFilename != "" {
		payload["mask"] = maskFilename
	} else if maskPresent {
		payload["mask_present"] = true
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", false
	}
	return string(encoded), true
}

func safeImageFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = filepath.Base(filename)
	filename = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, filename)
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." || filename == "/" {
		return ""
	}
	if len(filename) > 256 {
		filename = filename[:256]
	}
	return filename
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
