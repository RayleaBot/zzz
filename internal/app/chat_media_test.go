package app

import (
	"context"
	"reflect"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// chatMessages answers for the chat platform: messages by ID and forwarded
// messages by forward ID.
type chatMessages struct {
	messages, forwards map[string]rayleabot.ActionResult
}

func (c chatMessages) MessageGet(_ context.Context, id string) (rayleabot.ActionResult, error) {
	return c.messages[id], nil
}

func (c chatMessages) MessageForwardGet(_ context.Context, request rayleabot.MessageForwardGetRequest) (rayleabot.ActionResult, error) {
	return c.forwards[request.ForwardID], nil
}

func TestMessageImagesReadQuotedAndForwardedMessages(t *testing.T) {
	image := func(key, url string) map[string]any {
		return map[string]any{"type": "image", "data": map[string]any{key: url}}
	}
	reader := chatMessages{
		messages: map[string]rayleabot.ActionResult{"quoted": {"message": []any{
			image("url", "https://example.com/1.png"),
			map[string]any{"type": "forward", "data": map[string]any{"id": "f1"}},
			map[string]any{"type": "forward", "data": map[string]any{"id": "inline", "content": []any{map[string]any{"message": []any{image("url", "https://example.com/4.png")}}}}},
		}}},
		forwards: map[string]rayleabot.ActionResult{"f1": {"messages": []any{
			map[string]any{"message": []any{image("url", "https://example.com/2.png"), map[string]any{"type": "text", "data": map[string]any{"text": "文字"}}}},
			map[string]any{"content": []any{image("file", "https://example.com/3.png"), image("url", "https://example.com/1.png")}},
		}}},
	}
	quote := []rayleabot.Segment{{Type: "reply", Data: map[string]any{"message_id": "quoted"}}, rayleabot.Text("上传艾莲面板图")}
	want := []string{"https://example.com/1.png", "https://example.com/2.png", "https://example.com/3.png", "https://example.com/4.png"}
	if got := messageImages(t.Context(), quote, reader); !reflect.DeepEqual(got, want) {
		t.Fatalf("quoted images = %v", got)
	}
	// Pictures in the command's own message come first, and alone.
	own := append([]rayleabot.Segment{{Type: "image", Data: map[string]any{"url": "https://example.com/own.png"}}, {Type: "image", Data: map[string]any{"file": "local.png"}}}, quote...)
	if got := messageImages(t.Context(), own, reader); !reflect.DeepEqual(got, []string{"https://example.com/own.png"}) {
		t.Fatalf("own images = %v", got)
	}
}
