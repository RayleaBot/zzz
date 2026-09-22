package app

import (
	"context"
	"fmt"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
	"testing"
)

type overseasSyncCaller struct {
	role  Role
	calls int
}

func (c *overseasSyncCaller) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	if req.Method == "roles" {
		return decodeObject(map[string]any{"roles": []Role{c.role}}, out)
	}
	if req.Method == "execute" {
		c.calls++
		pool := asText(asObject(req.Params["input"])["gacha_type"])
		return decodeObject(QueryResult{Role: c.role, Data: map[string]any{"list": []any{map[string]any{"id": fmt.Sprint(9007199254740900 + c.calls), "uid": c.role.UID, "gacha_type": pool, "item_id": "1001", "gacha_id": "9001", "rank_type": "4", "count": "1", "time": "2026-09-01 12:00:00", "name": "合成物品", "item_type": "角色"}}, "has_more": false, "next_end_id": "0", "timezone": -5, "lang": "zh-cn"}}, out)
	}
	return gameError("operation_denied", "unsupported")
}
func TestStarRailOverseasSyncAndChinaRestriction(t *testing.T) {
	a := App{Game: Game{ID: "starrail"}, Gacha: &gacha.Store{Directory: t.TempDir(), Game: "starrail"}}
	defer a.Close()
	caller := &overseasSyncCaller{role: Role{Ref: "r", Game: "starrail", UID: "600000001", Region: "prod_official_usa"}}
	client := AccountsClient{Caller: caller, Game: "starrail", Provider: "p"}
	choice := map[string]any{"account_ref": "a", "role_ref": "r"}
	out, err := a.syncAction(t.Context(), client, "gacha.sync.start", choice)
	if err != nil {
		t.Fatal(err)
	}
	sync := out["sync"].(gacha.SyncInfo)
	for sync.State == "running" {
		out, err = a.syncAction(t.Context(), client, "gacha.sync.step", map[string]any{"ref": sync.Ref, "sequence": sync.Sequence})
		if err != nil {
			t.Fatal(err)
		}
		sync = out["sync"].(gacha.SyncInfo)
	}
	archive, err := a.Gacha.Read(caller.role.UID, caller.role.Region)
	if err != nil || archive.Timezone != -5 || len(archive.Records) != 6 {
		t.Fatal(archive, err)
	}
	caller.role.Region = "prod_gf_cn"
	if _, err = a.syncAction(t.Context(), client, "gacha.sync.start", choice); err == nil || PublicError(err).Code != "plugin.game_sync_unavailable" {
		t.Fatal("China sync unexpectedly enabled", err)
	}
}
