package relay

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/auth"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/doubao"
	"github.com/looplj/axonhub/llm/transformer/gemini"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
	"github.com/tidwall/sjson"
)

// buildOutbound 按渠道协议构造出站转换器, 并判断客户端请求能否直接透传。
func buildOutbound(channel model.Channel, format llm.APIFormat) (transformer.Outbound, bool, error) {
	provider := channel.Type
	if isImageFormat(format) {
		switch provider {
		case model.ChannelProviderOpenAI, model.ChannelProviderGemini, model.ChannelProviderVolcengine:
			// These providers expose image-capable outbound transformers.
		case model.ChannelProviderOpenAIResponses:
			// Keep the established compatibility rule: Responses image requests
			// use the OpenAI Images outbound.
			provider = model.ChannelProviderOpenAI
		default:
			return nil, false, fmt.Errorf("unsupported channel provider for image request: %s", channel.Type)
		}
	}
	outbound, err := newOutbound(provider, channel.BaseURL, channel.Key)
	if err != nil {
		return nil, false, err
	}
	passthrough := !isImageFormat(format) && ((provider == model.ChannelProviderOpenAI && format == llm.APIFormatOpenAIChatCompletion) ||
		(provider == model.ChannelProviderOpenAIResponses && format == llm.APIFormatOpenAIResponse) ||
		(provider == model.ChannelProviderAnthropic && format == llm.APIFormatAnthropicMessage))
	return outbound, passthrough, nil
}

func newOutbound(provider model.ChannelProvider, baseURL, apiKey string) (transformer.Outbound, error) {
	key := auth.NewStaticKeyProvider(apiKey)
	switch provider {
	case model.ChannelProviderOpenAI:
		return openai.NewOutboundTransformerWithConfig(&openai.Config{PlatformType: openai.PlatformOpenAI, BaseURL: baseURL, APIKeyProvider: key})
	case model.ChannelProviderOpenAIResponses:
		return responses.NewOutboundTransformerWithConfig(&responses.Config{BaseURL: baseURL, APIKeyProvider: key})
	case model.ChannelProviderAnthropic:
		return anthropic.NewOutboundTransformerWithConfig(&anthropic.Config{Type: anthropic.PlatformDirect, BaseURL: baseURL, APIKeyProvider: key})
	case model.ChannelProviderGemini:
		return gemini.NewOutboundTransformerWithConfig(gemini.Config{BaseURL: baseURL, APIKeyProvider: key})
	case model.ChannelProviderVolcengine:
		return doubao.NewOutboundTransformerWithConfig(&doubao.Config{BaseURL: baseURL, APIKeyProvider: key})
	default:
		return nil, fmt.Errorf("unsupported channel provider: %s", provider)
	}
}

func isImageFormat(format llm.APIFormat) bool {
	return format == llm.APIFormatOpenAIImageGeneration ||
		format == llm.APIFormatOpenAIImageEdit ||
		format == llm.APIFormatOpenAIImageVariation
}

// applyChannelConfig 按渠道配置覆盖上游请求的参数并追加自定义 Header; model 与 stream 由转发流程决定, 不允许覆盖。
func applyChannelConfig(channel model.Channel, request *httpclient.Request) error {
	if channel.ParamOverride != nil && *channel.ParamOverride != "" && !strings.Contains(strings.ToLower(request.Headers.Get("Content-Type")), "multipart/") {
		var overrides map[string]json.RawMessage
		if err := json.Unmarshal([]byte(*channel.ParamOverride), &overrides); err != nil {
			return fmt.Errorf("invalid channel parameter override: %w", err)
		}
		body := request.Body
		// 覆盖键可能自带点号或冒号, 转义后再作为 sjson 路径使用, 避免被解析成嵌套路径。
		escape := strings.NewReplacer("\\", "\\\\", ".", "\\.", ":", "\\:")
		for key, value := range overrides {
			if key == "model" || key == "stream" {
				continue
			}
			next, err := sjson.SetRawBytes(body, ":"+escape.Replace(key), value)
			if err != nil {
				return fmt.Errorf("apply channel parameter %q: %w", key, err)
			}
			body = next
		}
		request.Body = body
		if len(request.JSONBody) > 0 {
			request.JSONBody = slices.Clone(body)
		}
	}

	// 转换器已经写入的认证等敏感 Header 不允许被自定义配置覆盖。
	for _, header := range channel.CustomHeader {
		if request.Headers.Get(header.HeaderKey) != "" && httpclient.IsSensitiveHeader(header.HeaderKey) {
			continue
		}
		request.Headers.Set(header.HeaderKey, header.HeaderValue)
	}
	return nil
}
