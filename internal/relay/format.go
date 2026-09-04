package relay

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
)

var errNoCompatibleProtocol = errors.New("channel grant does not support the requested image protocol")

const maxImageRequestBody = 256 << 20

func inboundForFormat(format llm.APIFormat) transformer.Inbound {
	switch format {
	case llm.APIFormatOpenAIImageGeneration:
		return openai.NewImageGenerationInboundTransformer()
	case llm.APIFormatOpenAIImageEdit:
		return openai.NewImageEditInboundTransformer()
	case llm.APIFormatOpenAIResponse:
		return responses.NewInboundTransformer()
	case llm.APIFormatAnthropicMessage:
		return anthropic.NewInboundTransformer()
	default:
		return openai.NewInboundTransformer()
	}
}

func protocolForFormat(format llm.APIFormat) model.Protocol {
	switch format {
	case llm.APIFormatOpenAIImageGeneration:
		return model.ProtocolOpenAIChatCompletion
	case llm.APIFormatOpenAIImageEdit:
		return model.ProtocolOpenAIChatCompletion
	case llm.APIFormatOpenAIResponse:
		return model.ProtocolOpenAIResponse
	case llm.APIFormatAnthropicMessage:
		return model.ProtocolAnthropicMessage
	default:
		return model.ProtocolOpenAIChatCompletion
	}
}

func isImageFormat(format llm.APIFormat) bool {
	return format == llm.APIFormatOpenAIImageGeneration || format == llm.APIFormatOpenAIImageEdit
}

func isNonRetryableImageError(err error) bool {
	var responseErr *llm.ResponseError
	if !errors.As(err, &responseErr) {
		return false
	}
	switch responseErr.StatusCode {
	case 400, 413, 422:
		return true
	default:
		return false
	}
}

func requestBodyForState(raw *httpclient.Request, format llm.APIFormat) string {
	if !isImageFormat(format) {
		return string(raw.Body)
	}
	body := raw.JSONBody
	if len(body) == 0 {
		body = raw.Body
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err == nil {
		for _, key := range []string{"image", "mask"} {
			if _, ok := fields[key]; ok {
				fields[key] = json.RawMessage(`"[image data omitted]"`)
			}
		}
		if sanitized, err := json.Marshal(fields); err == nil {
			return string(sanitized)
		}
	}
	return fmt.Sprintf(`{"content_type":%q,"body_bytes":%d}`, raw.Headers.Get("Content-Type"), len(raw.Body))
}
