package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	"testing"
)

func testImage(t *testing.T, format string, width, height int) Image {
	t.Helper()
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var err error
	if format == "jpeg" {
		err = jpeg.Encode(&b, img, nil)
	} else {
		err = png.Encode(&b, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return Image{MIMEType: "image/" + format, Data: base64.StdEncoding.EncodeToString(b.Bytes())}
}

func TestImageValidation(t *testing.T) {
	good := testImage(t, "png", 2, 2)
	large := testImage(t, "png", 2049, 1)
	truncated, _ := base64.StdEncoding.DecodeString(good.Data)
	truncated = truncated[:len(truncated)-15] // PNG header remains readable, pixels are incomplete.
	for name, img := range map[string]Image{
		"bad base64": {"image/png", "!"}, "url": {"image/png", "https://example.com/image.png"},
		"svg": {"image/svg+xml", base64.StdEncoding.EncodeToString([]byte("<svg/>"))},
		"gif": {"image/gif", good.Data}, "mismatch": {"image/jpeg", good.Data},
		"dimensions": large, "truncated": {"image/png", base64.StdEncoding.EncodeToString(truncated)},
		"decoded size": {"image/png", base64.StdEncoding.EncodeToString(make([]byte, maxImageBytes+1))},
		"newline":      {"image/png", good.Data + "\n"},
	} {
		t.Run(name, func(t *testing.T) {
			if !errors.Is(validateImage(img), ErrInvalidRequest) {
				t.Fatal("accepted invalid image")
			}
		})
	}
	for _, img := range []Image{good, testImage(t, "jpeg", 2, 2), testImage(t, "png", 2048, 2048), {MIMEType: "image/webp", Data: "UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA"}} {
		if err := validateImage(img); err != nil {
			t.Fatal(err)
		}
	}
}

func TestImageRequestValidation(t *testing.T) {
	img := testImage(t, "png", 2, 2)
	req := validRequest()
	req.Messages = []Message{{Role: "user", Images: []Image{img}}}
	if err := validateRequest(req); err != nil {
		t.Fatal(err)
	}
	req.Messages[0].Images = []Image{img, img, img, img}
	if err := validateRequest(req); err != nil {
		t.Fatal(err)
	}
	req.Messages = append(req.Messages, Message{Role: "user", Images: []Image{img}})
	if !errors.Is(validateRequest(req), ErrInvalidRequest) {
		t.Fatal("accepted five images across conversation")
	}
	req.Messages = []Message{{Role: "assistant", Content: "image", Images: []Image{img}}, {Role: "user", Content: "followup"}}
	if !errors.Is(validateRequest(req), ErrInvalidRequest) {
		t.Fatal("accepted assistant image")
	}
	req.Messages = []Message{{Role: "user"}}
	if !errors.Is(validateRequest(req), ErrInvalidRequest) {
		t.Fatal("accepted empty message")
	}
}

func TestImageChatContextAndFollowup(t *testing.T) {
	img := testImage(t, "png", 2, 2)
	req := validRequest()
	req.Messages = []Message{{Role: "user", Content: "coffee bag", Images: []Image{img}}, {Role: "assistant", Content: "What amount?"}, {Role: "user", Content: "18g"}}
	client := fakeCompleter(func(ctx context.Context, messages []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
		if len(messages) != 5 || len(messages[2].Parts) != 2 || messages[2].Parts[0].Text != "coffee bag" || messages[2].Parts[1].ImageURL.URL != "data:image/png;base64,"+img.Data || messages[3].Role != "assistant" || messages[3].Content != "What amount?" || messages[4].Content != "18g" {
			t.Fatal("lost transcript association/order")
		}
		if strings.Contains(messages[1].Content, img.Data) || strings.Contains(messages[1].Content, "coffee bag") {
			t.Fatal("image/transcript embedded in JSON context")
		}
		return outputClient(validOutput)(ctx, messages, tools, format)
	})
	if _, err := NewService(client).Chat(context.Background(), ownDID, ownLibrary(), req); err != nil {
		t.Fatal(err)
	}
	if len(req.Messages[0].Images) != 1 {
		t.Fatal("mutated request")
	}
}

func TestImageProviderLimitMapping(t *testing.T) {
	req := validRequest()
	req.Messages[0].Images = []Image{testImage(t, "png", 2, 2)}
	client := fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		return provider.Completion{}, provider.ErrRequestTooLarge
	})
	_, err := NewService(client).Chat(context.Background(), ownDID, ownLibrary(), req)
	if !errors.Is(err, ErrProviderRequestTooLarge) {
		t.Fatal(err)
	}
}
