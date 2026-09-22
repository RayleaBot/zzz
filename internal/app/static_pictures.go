package app

import (
	"cmp"
	"context"
	"encoding/base64"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// StaticPicture is a command upstream answers with images it ships, such as
// StarRail-plugin's 参考面板 and 收益曲线: its parts in order, each an
// optional caption and an image of Source, {name} standing for the character
// the word names. Forward names the forwarded message upstream sends them
// in; Missing is the reply when the character has no image.
type StaticPicture struct {
	Source    string       `json:"source"`
	Character bool         `json:"character,omitempty"`
	Forward   string       `json:"forward,omitempty"`
	Missing   string       `json:"missing,omitempty"`
	Parts     []StaticPart `json:"parts"`
}

type StaticPart struct {
	Text string `json:"text,omitempty"`
	Path string `json:"path,omitempty"`
}

// staticCommand sends a static picture command's parts.
func (a *App) staticCommand(ctx context.Context, event *rayleabot.EventContext, static StaticPicture, args []string) error {
	var entry *Entry
	if static.Character {
		found, ok := a.Catalog.Resolve(strings.Join(args, ""), "character", a.aliasMap(event))
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		entry = &found
	}
	if !a.Artwork.Ready(static.Source) {
		return event.SendText(strings.TrimSpace("暂无图片素材。" + a.pictureHint([]PictureSource{{Source: static.Source}})))
	}
	parts, ok := a.staticParts(static, entry)
	if !ok {
		missing := cmp.Or(static.Missing, "暂无[{name}]的图片")
		return event.SendText(strings.ReplaceAll(missing, "{name}", entry.Name))
	}
	if len(parts) == 0 {
		return event.SendText("图片素材不完整，请管理员重新下载素材。")
	}
	if static.Forward != "" {
		return a.sendForward(ctx, event, parts)
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, slices.Concat(parts...)...)
}

// staticParts reads a static picture's parts, false when the character has
// no image of its own.
func (a *App) staticParts(static StaticPicture, entry *Entry) ([][]rayleabot.Segment, bool) {
	names := []string{""}
	if entry != nil {
		// Upstream names files by its own names, such as 开拓者•存护 for
		// 穹·存护, so each name of the character is tried, also with •.
		names = []string{}
		for _, name := range append([]string{entry.Name}, entry.Aliases...) {
			names = append(names, name, strings.ReplaceAll(name, "·", "•"))
		}
		names = slices.Compact(names)
	}
	parts := [][]rayleabot.Segment{}
	for _, part := range static.Parts {
		segments := []rayleabot.Segment{}
		if part.Text != "" {
			segments = append(segments, rayleabot.Text(part.Text))
		}
		if part.Path != "" {
			var data []byte
			for _, name := range names {
				if file, err := a.Artwork.Open(static.Source, strings.ReplaceAll(part.Path, "{name}", name)); err == nil {
					data = file
					break
				}
			}
			if data == nil && strings.Contains(part.Path, "{name}") {
				return nil, false
			}
			if data != nil {
				segments = append(segments, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(data)))
			}
		}
		if len(segments) > 0 {
			parts = append(parts, segments)
		}
	}
	return parts, true
}

// sendForward sends parts as one forwarded message, as upstream's
// makeForwardMsg, or as one ordinary message where forwarding is not
// available.
func (a *App) sendForward(ctx context.Context, event *rayleabot.EventContext, parts [][]rayleabot.Segment) error {
	if event.Event.SourceProtocol == "onebot11" {
		messages := []rayleabot.ForwardMessage{}
		for _, part := range parts {
			content := []any{}
			for _, segment := range part {
				content = append(content, map[string]any{"type": segment.Type, "data": segment.Data})
			}
			messages = append(messages, rayleabot.ForwardMessage{"type": "node", "data": map[string]any{"user_id": event.Bot.ID, "nickname": cmp.Or(event.Bot.Nickname, a.Game.Name), "content": content}})
		}
		if _, err := event.Actions().MessageForwardSend(ctx, rayleabot.MessageForwardSendRequest{TargetType: rayleabot.ConversationType(event.Event.Target.Type), TargetID: event.Event.Target.ID, Messages: messages}); err == nil {
			return event.Result(map[string]any{"handled": true})
		}
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, slices.Concat(parts...)...)
}
