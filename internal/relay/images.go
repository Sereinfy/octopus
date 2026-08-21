package relay

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/openai"
)

// HandleImageGenerations 处理 OpenAI 图片生成请求。
func HandleImageGenerations(c *gin.Context) {
	(&execution{ctx: c, protocol: imageProtocol(
		llm.APIFormatOpenAIImageGeneration,
		"/images/generations",
		openai.NewImageGenerationInboundTransformer(),
	)}).execute()
}

// HandleImageEdits 处理 OpenAI 图片编辑请求。
func HandleImageEdits(c *gin.Context) {
	(&execution{ctx: c, protocol: imageProtocol(
		llm.APIFormatOpenAIImageEdit,
		"/images/edits",
		openai.NewImageEditInboundTransformer(),
	)}).execute()
}

// HandleImageVariations 处理 OpenAI 图片变体请求。
func HandleImageVariations(c *gin.Context) {
	(&execution{ctx: c, protocol: imageProtocol(
		llm.APIFormatOpenAIImageVariation,
		"/images/variations",
		openai.NewImageVariationInboundTransformer(),
	)}).execute()
}

func imageProtocol(format llm.APIFormat, route string, inbound transformer.Inbound) *relayProtocol {
	return &relayProtocol{
		format:   format,
		route:    route,
		authType: httpclient.AuthTypeBearer,
		inbound:  inbound,
	}
}

// newOutboundForRequest 保持普通请求的渠道路由不变。
// 图片请求统一使用 OpenAI Images 转换器，即使渠道的普通请求使用 /responses。
func newOutboundForRequest(request *llm.Request, channel *model.Channel) (transformer.Outbound, error) {
	provider := channel.Type
	if request != nil && request.RequestType == llm.RequestTypeImage {
		switch channel.Type {
		case model.ChannelProviderOpenAI, model.ChannelProviderOpenAIResponses:
			provider = model.ChannelProviderOpenAI
		default:
			return nil, fmt.Errorf("%w: image requests do not support channel provider %s", errUnsupportedTarget, channel.Type)
		}
	}
	return newOutbound(provider, channel.BaseURL, channel.Key)
}

func isImageProtocol(protocol *relayProtocol) bool {
	if protocol == nil {
		return false
	}
	return protocol.format == llm.APIFormatOpenAIImageGeneration ||
		protocol.format == llm.APIFormatOpenAIImageEdit ||
		protocol.format == llm.APIFormatOpenAIImageVariation
}

func imageResponseHasData(response *llm.Response) bool {
	return response != nil && response.Image != nil && len(response.Image.Data) > 0
}
