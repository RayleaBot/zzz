package app

import (
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestQueryRankKeepsRecordsAndMembersChoices(t *testing.T) {
	directory := t.TempDir()
	a := &App{Game: Game{ID: "zzz", QueryRanks: []QueryRankType{
		{ID: "DEADLY", Page: "deadly-hard", Operation: "zzz.deadly", Words: "绝境"},
		{ID: "DEADLY", Page: "deadly", Operation: "zzz.deadly", Words: "危局"},
		{ID: "ABYSS", Page: "abyss", Operation: "zzz.challenge", Words: "式舆|深渊"},
	}}, Groups: &GroupStore{Directory: directory + "/groups"}, QueryRanks: &QueryRankStore{Directory: directory + "/ranks"}}
	event := &rayleabot.EventContext{Event: rayleabot.Event{EventType: "message.group", SourceProtocol: "onebot11", SourceAdapter: "a", Actor: rayleabot.Actor{ID: "10001"}, Target: rayleabot.Target{Type: "group", ID: "g"}}, Bot: rayleabot.Bot{ID: "bot"}}
	result := QueryResult{Role: Role{UID: "10000001", Nickname: "绳匠"}, Data: map[string]any{"has_data": true}}
	a.recordQueryRank(event, "zzz.deadly", result)
	scope := groupScope(event)
	// A member's choice to hide survives the next query.
	_ = a.Groups.Update(scope, func(data *GroupData) error {
		member := data.QueryRanks["DEADLY"]["10000001"]
		member.Hidden = true
		data.QueryRanks["DEADLY"]["10000001"] = member
		return nil
	})
	a.recordQueryRank(event, "zzz.deadly", result)
	data, _ := a.Groups.Read(scope)
	if member := data.QueryRanks["DEADLY"]["10000001"]; !member.Hidden || member.ActorID != "10001" || len(data.QueryRanks) != 1 {
		t.Fatalf("members = %+v", data.QueryRanks)
	}
	record, ok := a.QueryRanks.Read("zzz.deadly", "10000001")
	if !ok || record.Role.Nickname != "绳匠" || record.Data["has_data"] != true || record.ActorID != "10001" {
		t.Fatalf("record = %+v", record)
	}
	// Unlisted operations keep nothing.
	a.recordQueryRank(event, "zzz.note", result)
	if _, ok := a.QueryRanks.Read("zzz.note", "10000001"); ok {
		t.Error("unranked operation recorded")
	}
	// While group rankings are off, as upstream, the member still joins but
	// no record is kept.
	event.Config = map[string]any{"group_rank_enabled": false}
	a.recordQueryRank(event, "zzz.challenge", result)
	if _, ok := a.QueryRanks.Read("zzz.challenge", "10000001"); ok {
		t.Error("record kept while group rankings are off")
	}
	if data, _ := a.Groups.Read(scope); data.QueryRanks["ABYSS"]["10000001"].ActorID != "10001" {
		t.Error("member did not join while group rankings are off")
	}
	for word, page := range map[string]string{"危局绝境排名": "deadly-hard", "危局排名": "deadly", "深渊排名": "abyss", "重置深渊排名": "abyss"} {
		if rank, ok := a.queryRankType(word); !ok || rank.Page != page {
			t.Errorf("queryRankType(%s) = %+v", word, rank)
		}
	}
}
