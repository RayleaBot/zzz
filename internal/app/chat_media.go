package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Pictures and files taken from chat messages, and forwarded replies.

// messageReader reads the messages a message quotes or forwards.
type messageReader interface {
	MessageGet(context.Context, string) (rayleabot.ActionResult, error)
	MessageForwardGet(context.Context, rayleabot.MessageForwardGetRequest) (rayleabot.ActionResult, error)
}

// messageImages lists the image addresses of the message, or when it has
// none, of the message it quotes and of the forwarded messages that one
// holds, as ZZZ-Plugin's 上传面板图 reads them.
func messageImages(ctx context.Context, segments []rayleabot.Segment, reader messageReader) []string {
	urls := []string{}
	add := func(kind string, data map[string]any) {
		link := asText(data["url"])
		if link == "" {
			link = asText(data["file"])
		}
		if kind == "image" && (strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "http://")) && !slices.Contains(urls, link) {
			urls = append(urls, link)
		}
	}
	for _, segment := range segments {
		add(segment.Type, segment.Data)
	}
	if len(urls) > 0 {
		return urls
	}
	nodes := func(list []any) {
		for _, node := range list {
			content := asList(asObject(node)["message"])
			if content == nil {
				content = asList(asObject(node)["content"])
			}
			if content == nil {
				content = asList(asObject(asObject(node)["data"])["content"])
			}
			for _, item := range content {
				add(asText(asObject(item)["type"]), asObject(asObject(item)["data"]))
			}
		}
	}
	for _, segment := range segments {
		if segment.Type != "reply" || asText(segment.Data["id"]) == "" {
			continue
		}
		quoted, err := reader.MessageGet(ctx, asText(segment.Data["id"]))
		if err != nil {
			continue
		}
		for _, raw := range asList(quoted["message"]) {
			item := asObject(raw)
			kind, data := asText(item["type"]), asObject(item["data"])
			if kind != "forward" {
				add(kind, data)
				continue
			}
			if inline := asList(data["content"]); inline != nil {
				nodes(inline)
			} else if id := asText(data["id"]); id != "" {
				if forwarded, err := reader.MessageForwardGet(ctx, rayleabot.MessageForwardGetRequest{ForwardID: id}); err == nil {
					nodes(asList(forwarded["messages"]))
				}
			}
		}
	}
	return urls
}

// downloadFile reads a file a message links to, up to limit bytes.
func downloadFile(ctx context.Context, link string, limit int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err == nil && len(raw) > limit {
		err = errors.New("file exceeds the size limit")
	}
	return raw, err
}

// forwardBudget keeps one forwarded message within the host's message
// frame.
const forwardBudget = 7 << 20

// sendForward sends parts as forwarded messages, as upstream's
// makeForwardMsg, starting another where one would outgrow the host's
// message frame; where forwarding is not available, each is sent as an
// ordinary message.
func (a *App) sendForward(ctx context.Context, event *rayleabot.EventContext, parts [][]rayleabot.Segment) error {
	batches := [][][]rayleabot.Segment{{}}
	size := 0
	for _, part := range parts {
		bytes := 0
		for _, segment := range part {
			for _, value := range segment.Data {
				bytes += len(asText(value))
			}
		}
		if last := len(batches) - 1; size+bytes > forwardBudget && len(batches[last]) > 0 {
			batches, size = append(batches, nil), 0
		}
		batches[len(batches)-1] = append(batches[len(batches)-1], part)
		size += bytes
	}
	for _, batch := range batches {
		if event.Event.SourceProtocol == "onebot11" {
			messages := []rayleabot.ForwardMessage{}
			for _, part := range batch {
				content := []any{}
				for _, segment := range part {
					content = append(content, map[string]any{"type": segment.Type, "data": segment.Data})
				}
				messages = append(messages, rayleabot.ForwardMessage{"type": "node", "data": map[string]any{"user_id": event.Bot.ID, "nickname": a.Game.Name, "content": content}})
			}
			if _, err := event.Actions().MessageForwardSend(ctx, rayleabot.MessageForwardSendRequest{TargetType: rayleabot.ConversationType(event.Event.Target.Type), TargetID: event.Event.Target.ID, Messages: messages}); err == nil {
				continue
			}
		}
		segments := []rayleabot.Segment{}
		for index, part := range batch {
			if index > 0 {
				segments = append(segments, rayleabot.Text("\n"))
			}
			segments = append(segments, part...)
		}
		_, _ = event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, TargetType: event.Event.Target.Type, TargetID: event.Event.Target.ID, Message: rayleabot.MessageOut{Segments: segments}})
	}
	return event.Result(map[string]any{"handled": true})
}
