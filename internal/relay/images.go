package relay

import (
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
)

func HandleImageGenerations(c *gin.Context) {
	Forward(llm.APIFormatOpenAIImageGeneration)(c)
}

func HandleImageEdits(c *gin.Context) {
	Forward(llm.APIFormatOpenAIImageEdit)(c)
}

func HandleImageVariations(c *gin.Context) {
	Forward(llm.APIFormatOpenAIImageVariation)(c)
}
