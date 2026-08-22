package relay

import (
	"errors"
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/transformer"
)

var errUnsupportedTarget = errors.New("unsupported relay target")

// newOutboundForRequest keeps normal channel routing unchanged while applying
// the OpenAI Images compatibility rule to image requests.
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
