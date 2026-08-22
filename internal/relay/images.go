package relay

import (
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
)

// relayProtocol is retained as a small format descriptor for image log redaction.
// Request execution uses the v0.11 Forward pipeline directly.
type relayProtocol struct {
	format   llm.APIFormat
	route    string
	authType string
	inbound  transformer.Inbound
}

func HandleImageGenerations(c *gin.Context) {
	Forward(llm.APIFormatOpenAIImageGeneration)(c)
}

func HandleImageEdits(c *gin.Context) {
	Forward(llm.APIFormatOpenAIImageEdit)(c)
}

func HandleImageVariations(c *gin.Context) {
	Forward(llm.APIFormatOpenAIImageVariation)(c)
}

func imageProtocol(format llm.APIFormat, route string, inbound transformer.Inbound) *relayProtocol {
	return &relayProtocol{format: format, route: route, authType: httpclient.AuthTypeBearer, inbound: inbound}
}

func isImageProtocol(protocol *relayProtocol) bool {
	return protocol != nil && isImageFormat(protocol.format)
}

func imageResponseHasData(response *llm.Response) bool {
	return response != nil && response.Image != nil && len(response.Image.Data) > 0
}
