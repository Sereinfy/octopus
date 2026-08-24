package relay

import (
	"errors"
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/openai"
)

var errUnsupportedImageProvider = errors.New("unsupported image channel provider")

func isImageFormat(format llm.APIFormat) bool {
	return format == llm.APIFormatOpenAIImageGeneration ||
		format == llm.APIFormatOpenAIImageEdit ||
		format == llm.APIFormatOpenAIImageVariation
}

// imageInboundForFormat returns the client-facing OpenAI Images transformer.
func imageInboundForFormat(format llm.APIFormat) transformer.Inbound {
	switch format {
	case llm.APIFormatOpenAIImageGeneration:
		return openai.NewImageGenerationInboundTransformer()
	case llm.APIFormatOpenAIImageEdit:
		return openai.NewImageEditInboundTransformer()
	case llm.APIFormatOpenAIImageVariation:
		return openai.NewImageVariationInboundTransformer()
	default:
		return nil
	}
}

// imageOutboundProvider selects the provider transformer for an OpenAI Images request.
// Responses-compatible channels are backed by OpenAI Images because the intermediary
// does not accept the Responses image_generation tool.
func imageOutboundProvider(provider model.ChannelProvider) (model.ChannelProvider, error) {
	switch provider {
	case model.ChannelProviderOpenAI, model.ChannelProviderGemini, model.ChannelProviderVolcengine:
		return provider, nil
	case model.ChannelProviderOpenAIResponses:
		return model.ChannelProviderOpenAI, nil
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedImageProvider, provider)
	}
}

func validateImageResponse(response *llm.Response) error {
	if response == nil {
		return errors.New("upstream response is empty")
	}
	if response.Image == nil || len(response.Image.Data) == 0 {
		return errors.New("upstream image response is empty")
	}
	return nil
}
