package app

import (
	"context"
	"encoding/json"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

func TestGachaFilesRoundTripAndFillRecords(t *testing.T) {
	a := pluginApp(t)
	// A UIGF v4 file keeps long IDs exact, drops non-standard fields, and
	// takes names and ranks from the catalog when a record leaves them out.
	file := `{"info":{"version":"v4.0","authkey":"synthetic"},"hk4e":[{"uid":"100000009","timezone":8,"list":[]}],"nap":[{"uid":1000000001,"timezone":-5,"lang":"zh-cn","list":[
		{"id":1844674407370955101,"gacha_type":"2","item_id":"1091","time":"2026-09-01 08:00:00","cookie_token":"synthetic"}]}]}`
	archives, err := a.parseGachaFile([]byte(file))
	if err != nil || len(archives) != 1 {
		t.Fatal(archives, err)
	}
	record := archives[0].Records[0]
	if archives[0].UID != "1000000001" || archives[0].Timezone != -5 || record.ID != "1844674407370955101" || record.Name != "星见雅" || record.ItemType != "代理人" || record.Rank != "4" {
		t.Fatalf("%+v", archives[0])
	}
	archive := archives[0]
	raw, _ := json.Marshal(a.gachaFile(archive))
	if again, err := a.parseGachaFile(raw); err != nil || len(again) != 1 || again[0].Records[0] != record {
		t.Fatalf("UIGF: %+v %v", again, err)
	}
	// UIGF has no reprise pools, so such an archive is written whole.
	archive.Records = append(archive.Records, gacha.Record{ID: "1844674407370955102", GachaType: "103", ItemID: "14109", Name: "霰落星殿", ItemType: "音擎", Rank: "4", Time: "2026-09-01 08:01:00"})
	raw, _ = json.Marshal(a.gachaFile(archive))
	if again, err := a.parseGachaFile(raw); err != nil || len(again) != 1 || len(again[0].Records) != 2 || again[0].Records[1] != archive.Records[1] {
		t.Fatalf("archive: %s %+v %v", raw, again, err)
	}
	if other, err := a.parseGachaFile([]byte(`{"info":{"uid":"100000001","uigf_version":"v2.2"},"list":[{"id":"1"}]}`)); err != nil || other != nil {
		t.Fatal("another game's file was read", other, err)
	}
	for text, want := range map[string]string{
		`{"info":{}}`: "json文件内容错误：非统一祈愿记录标准",
		`{"info":{"version":"v4.0"},"nap":[{"uid":"1","timezone":8,"list":[{"id":"1","gacha_type":"2","time":"2026-09-01 08:00:00"}]}]}`: "json文件内容错误：缺少必要字段 item_id",
		`not json`: "{name},json格式错误",
	} {
		if _, err := a.parseGachaFile([]byte(text)); friendlyError(err) != want {
			t.Errorf("%s: %v", text, err)
		}
	}
}

func TestGachaImportsGoUnderTheUIDsRegion(t *testing.T) {
	a := pluginApp(t)
	listed := Accounts{Items: []Account{{Roles: []Role{{Game: "zzz", UID: "1000000001", Region: "prod_gf_us"}}}}}
	for uid, want := range map[string]string{"1000000001": "prod_gf_us", "1500000001": "prod_gf_eu", "1300000001": "prod_gf_jp", "1700000001": "prod_gf_sg", "10000001": "prod_gf_cn"} {
		if region := a.archiveRegion(listed, uid); region != want {
			t.Errorf("%s: %s", uid, region)
		}
	}
	if regionTimezone("prod_gf_eu") != 1 || regionTimezone("prod_gf_us") != -5 || regionTimezone("prod_gf_cn") != 8 {
		t.Error("region time zones")
	}
}

func TestGachaImportFindsTheSentFile(t *testing.T) {
	event := &rayleabot.EventContext{}
	event.Event.Message.Segments = []rayleabot.Segment{{Type: "file", Data: map[string]any{"name": "1000000001.json", "url": "https://files.example/abc"}}}
	if file, name, found := messageFile(context.Background(), event); !found || file != "https://files.example/abc" || name != "1000000001.json" {
		t.Fatal(file, name, found)
	}
	event.Event.Message.Segments = nil
	event.Event.Message.PlainText = "导入记录"
	if _, _, found := messageFile(context.Background(), event); found {
		t.Fatal("a message without a file was taken")
	}
}
