package relay

import (
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
