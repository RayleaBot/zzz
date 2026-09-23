package app

import (
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestMentionedUserSkipsTheBot(t *testing.T) {
	at := func(user string) rayleabot.Segment {
		return rayleabot.Segment{Type: "at", Data: map[string]any{"user_id": user}}
	}
	event := &rayleabot.EventContext{Bot: rayleabot.Bot{ID: "10000"}}
	event.Event.Message.Segments = []rayleabot.Segment{at("10000"), rayleabot.Text(" 抽卡分析"), at("20000")}
	if user := mentionedUser(event); user != "20000" {
		t.Fatal(user)
	}
	event.Event.Message.Segments = event.Event.Message.Segments[:2]
	if user := mentionedUser(event); user != "" {
		t.Fatal("the bot was taken for the mentioned user")
	}
}
