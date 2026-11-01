package ai

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	_ "golang.org/x/image/webp"
	provider "tangled.org/arabica.social/arabica/internal/ai"
)

// Image is transient request data, never a record or a remote resource.
type Image struct {
	MIMEType string `json:"mime_type"`
	Data     string `json:"data"`
}

const maxImageBytes = 1024 * 1024

func validateImage(img Image) error {
	formats := map[string]string{"image/jpeg": "jpeg", "image/png": "png", "image/webp": "webp"}
	format, ok := formats[img.MIMEType]
	if !ok || len(img.Data) == 0 || len(img.Data) > base64.StdEncoding.EncodedLen(maxImageBytes) || strings.ContainsAny(img.Data, "\r\n") {
		return ErrInvalidRequest
	}
	data, err := base64.StdEncoding.Strict().DecodeString(img.Data)
	if err != nil || len(data) > maxImageBytes {
		return ErrInvalidRequest
	}
	cfg, actual, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || actual != format || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 2048 || cfg.Height > 2048 || cfg.Width*cfg.Height > 2048*2048 {
		return ErrInvalidRequest
	}
	decoded, actual, err := image.Decode(bytes.NewReader(data))
	if err != nil || actual != format || decoded.Bounds().Dx() != cfg.Width || decoded.Bounds().Dy() != cfg.Height {
		return ErrInvalidRequest
	}
	return nil
}

func imageConversation(contextJSON string, messages []Message) []provider.Message {
	out := []provider.Message{{Role: "system", Content: policy}, {Role: "user", Content: contextJSON}}
	for _, m := range messages {
		next := provider.Message{Role: m.Role, Content: m.Content}
		if m.ImagesUnavailable > 0 {
			next.Content += fmt.Sprintf("\n[%d earlier image(s) unavailable; ask for reattachment if needed.]", m.ImagesUnavailable)
		}
		if len(m.Images) > 0 {
			if m.Content != "" {
				next.Parts = append(next.Parts, provider.ContentPart{Type: "text", Text: m.Content})
			}
			for _, img := range m.Images {
				next.Parts = append(next.Parts, provider.ContentPart{Type: "image_url", ImageURL: &provider.ImageURL{URL: "data:" + img.MIMEType + ";base64," + img.Data}})
			}
		}
		out = append(out, next)
	}
	return out
}
