package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/looplj/axonhub/llm/transformer/openai"
)

var errUnsupportedImageProvider = errors.New("unsupported image channel provider")

const (
	maxImageCount           = 16
	maxImageFileSize        = 50 * 1024 * 1024
	maxImageRequestCount    = 16
	maxImageIntegerFieldLen = 64
)

var allowedImageContentTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/gif":  {},
	"image/webp": {},
}

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

// validateRawImageRequest checks multipart scalar fields before the dependency
// transformer silently drops malformed integer values.
func validateRawImageRequest(format llm.APIFormat, request *httpclient.Request) error {
	if request == nil || format == llm.APIFormatOpenAIImageGeneration {
		return nil
	}

	mediaType, params, err := mime.ParseMediaType(request.Headers.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(strings.ToLower(mediaType), "multipart/") || params["boundary"] == "" {
		return nil
	}

	reader := multipart.NewReader(bytes.NewReader(request.Body), params["boundary"])
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid multipart image request: %w", err)
		}
		if part.FileName() != "" {
			continue
		}

		field := part.FormName()
		if field != "n" && field != "partial_images" && field != "output_compression" {
			continue
		}
		value, err := io.ReadAll(io.LimitReader(part, maxImageIntegerFieldLen+1))
		if err != nil {
			return fmt.Errorf("failed to read image parameter %q: %w", field, err)
		}
		valueText := strings.TrimSpace(string(value))
		if valueText == "" {
			continue
		}
		if len(value) > maxImageIntegerFieldLen {
			return fmt.Errorf("image parameter %q is too long", field)
		}
		parsed, err := strconv.ParseInt(valueText, 10, 64)
		if err != nil {
			return fmt.Errorf("image parameter %q must be an integer", field)
		}
		switch field {
		case "n":
			if parsed < 1 || parsed > maxImageRequestCount {
				return fmt.Errorf("image parameter n must be between 1 and %d", maxImageRequestCount)
			}
		case "partial_images", "output_compression":
			if parsed < 0 {
				return fmt.Errorf("image parameter %q must not be negative", field)
			}
			if field == "output_compression" && parsed > 100 {
				return fmt.Errorf("image parameter output_compression must be between 0 and 100")
			}
		}
	}
}

// validateImageRequest applies the same decoded image limits to JSON and
// multipart requests. Multipart parsing in the dependency enforces its own
// limits, while this also closes the JSON path gap.
func validateImageRequest(image *llm.ImageRequest) error {
	if image == nil {
		return errors.New("image request is required")
	}
	if image.N != nil && (*image.N < 1 || *image.N > maxImageRequestCount) {
		return fmt.Errorf("image parameter n must be between 1 and %d", maxImageRequestCount)
	}
	if image.PartialImages != nil && *image.PartialImages < 0 {
		return errors.New("image parameter partial_images must not be negative")
	}
	if image.OutputCompression != nil && (*image.OutputCompression < 0 || *image.OutputCompression > 100) {
		return errors.New("image parameter output_compression must be between 0 and 100")
	}
	if len(image.Images) > maxImageCount {
		return fmt.Errorf("too many images: maximum is %d", maxImageCount)
	}
	for _, data := range image.Images {
		if err := validateImageBytes(data); err != nil {
			return fmt.Errorf("invalid image: %w", err)
		}
	}
	if len(image.Mask) > 0 {
		if err := validateImageBytes(image.Mask); err != nil {
			return fmt.Errorf("invalid mask: %w", err)
		}
	}
	return nil
}

func validateImageBytes(data []byte) error {
	if len(data) == 0 {
		return errors.New("image is empty")
	}
	if len(data) > maxImageFileSize {
		return fmt.Errorf("image is too large: maximum is %d bytes", maxImageFileSize)
	}
	if _, ok := allowedImageContentTypes[http.DetectContentType(data)]; !ok {
		return errors.New("unsupported image type")
	}
	return nil
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
