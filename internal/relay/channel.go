package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/auth"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/anthropic"
	"github.com/looplj/axonhub/llm/transformer/openai"
	"github.com/looplj/axonhub/llm/transformer/openai/responses"
	"github.com/tidwall/sjson"
)

// buildOutbound 在渠道授权支持的协议内选出本轮上游协议, 构造对应的出站转换器, 并返回选中的协议和能否同协议透传。
// 地址由渠道的协议路径字段与地址拼接, 凭据取自目标绑定的渠道凭据。
// want 是客户端请求使用的协议, 由调用方按入站格式定出; 选中的协议随请求状态推给界面, 故一并返回。
func buildOutbound(channel model.Channel, grant model.ChannelGrant, channelKey model.ChannelKey, want model.Protocol, format llm.APIFormat) (transformer.Outbound, model.Protocol, bool, error) {
	if isImageFormat(format) {
		// Images reuse the Chat grant bit by design, but always use the image endpoint and conversion path.
		if grant.Protocols&model.ProtocolOpenAIChatCompletion == 0 {
			return nil, 0, false, errNoCompatibleProtocol
		}
		key := auth.NewStaticKeyProvider(channelKey.Key)
		endpoint := imageEndpointPath(channel.OpenAIChatCompletionPath, format)
		outbound, err := openai.NewOutboundTransformerWithConfig(&openai.Config{PlatformType: openai.PlatformOpenAI, BaseURL: channel.BaseURL, EndpointPath: endpoint, APIKeyProvider: key})
		return outbound, model.ProtocolOpenAIChatCompletion, false, err
	}

	protocol, passthrough := want, grant.Protocols&want != 0
	if !passthrough {
		protocol = 0
		switch {
		case grant.Protocols&model.ProtocolAnthropicMessage != 0:
			protocol = model.ProtocolAnthropicMessage
		case grant.Protocols&model.ProtocolOpenAIResponse != 0:
			protocol = model.ProtocolOpenAIResponse
		case grant.Protocols&model.ProtocolOpenAIChatCompletion != 0:
			protocol = model.ProtocolOpenAIChatCompletion
		}
	}

	key := auth.NewStaticKeyProvider(channelKey.Key)
	switch protocol {
	case model.ProtocolOpenAIChatCompletion:
		outbound, err := openai.NewOutboundTransformerWithConfig(&openai.Config{PlatformType: openai.PlatformOpenAI, BaseURL: channel.BaseURL, EndpointPath: channel.OpenAIChatCompletionPath, APIKeyProvider: key})
		return outbound, protocol, passthrough, err
	case model.ProtocolOpenAIResponse:
		outbound, err := responses.NewOutboundTransformerWithConfig(&responses.Config{BaseURL: channel.BaseURL, EndpointPath: channel.OpenAIResponsePath, APIKeyProvider: key})
		return outbound, protocol, passthrough, err
	case model.ProtocolAnthropicMessage:
		outbound, err := anthropic.NewOutboundTransformerWithConfig(&anthropic.Config{Type: anthropic.PlatformDirect, BaseURL: channel.BaseURL, EndpointPath: channel.AnthropicMessagePath, APIKeyProvider: key})
		return outbound, protocol, passthrough, err
	default:
		return nil, 0, false, fmt.Errorf("channel grant %d supports no known protocol: %d", grant.ID, grant.Protocols)
	}
}

// applyChannelConfig 按渠道配置覆盖上游请求的参数并追加自定义 Header; model 与 stream 由转发流程决定, 不允许覆盖。
func applyChannelConfig(channel model.Channel, request *httpclient.Request) error {
	imageMultipart := request.RequestType == llm.RequestTypeImage.String() && !strings.Contains(strings.ToLower(request.Headers.Get("Content-Type")), "application/json")
	if channel.ParamOverride != "" {
		var overrides map[string]json.RawMessage
		if err := json.Unmarshal([]byte(channel.ParamOverride), &overrides); err != nil {
			return fmt.Errorf("invalid channel parameter override: %w", err)
		}
		if imageMultipart {
			// Multipart bodies must be rebuilt structurally; applying sjson would corrupt the boundary.
			goto customHeaders
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

customHeaders:
	// 转换器已经写入的认证等敏感 Header 不允许被自定义配置覆盖。
	for _, header := range channel.CustomHeader {
		key := http.CanonicalHeaderKey(header.HeaderKey)
		if isManagedUpstreamHeader(key) {
			continue
		}
		request.Headers.Set(key, header.HeaderValue)
	}
	return nil
}

func imageEndpointPath(chatPath string, format llm.APIFormat) string {
	suffix := "/images/generations"
	if format == llm.APIFormatOpenAIImageEdit {
		suffix = "/images/edits"
	}
	chatPath = strings.TrimRight(strings.TrimSpace(chatPath), "/")
	if strings.HasSuffix(chatPath, "/chat/completions") {
		return strings.TrimSuffix(chatPath, "/chat/completions") + suffix
	}
	if chatPath == "" {
		return "/v1" + suffix
	}
	parent := path.Dir(chatPath)
	if path.Base(parent) == "chat" {
		parent = path.Dir(parent)
	}
	if parent == "." || parent == "/" {
		parent = "/v1"
	}
	return strings.TrimRight(parent, "/") + suffix
}

func isManagedUpstreamHeader(key string) bool {
	switch http.CanonicalHeaderKey(key) {
	case "Content-Type", "Content-Length", "Transfer-Encoding", "Accept", "Authorization":
		return true
	default:
		return httpclient.IsSensitiveHeader(key)
	}
}
