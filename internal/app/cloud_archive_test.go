package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type detailCaller struct {
	role  Role
	deny  bool
	calls int
}

func (c *detailCaller) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	c.calls++
	if c.deny {
		return gameError("role_missing", "revoked")
	}
	if req.Method != "roles" {
		return gameError("operation_denied", "unexpected")
	}
	return decodeObject(map[string]any{"roles": []Role{c.role}}, out)
}

func TestCloudArchiveSanitizesAndPreservesMiaoBusinessFields(t *testing.T) {
	raw := cloudObject(t, `{"uid":"100000001","cookie":"discard-root","avatars":{"10000046":{"id":10000046,"name":"胡桃","cons":0,"_time":"2026-09-01 12:00:00","_source":"mys","stoken":"discard-avatar","weapon":{"name":"护摩之杖","level":90,"affix":1,"cookie_token":"discard-weapon"},"talent":{"a":10,"e":{"level":10,"cookie":"discard-talent"}},"artis":{"1":{"name":"角斗士的留恋","level":20,"mainId":13001,"star":5,"attrIds":[501204],"cookie":"discard-gear"}}}}}`)
	clean, err := cleanCloudPlayer("genshin", "100000001", raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(clean)
	if strings.Contains(string(encoded), "discard-") || !strings.Contains(string(encoded), `"_source":"share"`) || !strings.Contains(string(encoded), `"cons":0`) {
		t.Fatal("credential metadata retained or zero lost")
	}
	if _, err = cleanCloudPlayer("starrail", "100000001", raw); err == nil {
		t.Fatal("wrong game accepted")
	}
	if _, err = cleanCloudPlayer("genshin", "100000002", raw); err == nil {
		t.Fatal("wrong UID accepted")
	}
}
func TestCloudTransferAtomicMergeReplayExportAndRevocation(t *testing.T) {
	a := App{Game: testGame(t, "genshin"), CloudArchive: &CloudArchiveStore{Directory: filepath.Join(t.TempDir(), "exchange")}}
	defer a.Close()
	role := Role{Ref: "role", Game: "genshin", UID: "100000001", Region: "cn_gf01"}
	caller := &detailCaller{role: role}
	client := AccountsClient{Caller: caller, Provider: "provider", Game: "genshin"}
	choice := Selection{"account", "role"}
	input := map[string]any{"account_ref": "account", "role_ref": "role", "uid": role.UID, "count": 2}
	start, err := a.cloudTransferAction(t.Context(), client, "cloud.archive.import.start", input)
	if err != nil {
		t.Fatal(err)
	}
	ref := start["ref"]
	avatar := map[string]any{"id": 10000046, "level": 90, "cookie": "not-saved"}
	appendInput := map[string]any{"ref": ref, "offset": 0, "avatar": avatar}
	if _, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.import.append", appendInput); err != nil {
		t.Fatal(err)
	}
	if _, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.import.append", appendInput); err != nil {
		t.Fatal("replay rejected", err)
	}
	if _, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.import.finish", map[string]any{"ref": ref}); err == nil {
		t.Fatal("partial import committed")
	}
	if old, _ := a.CloudArchive.Read(client.Provider, choice); len(old.Avatars) != 0 {
		t.Fatal("partial import changed archive")
	}
	appendInput["offset"] = 1
	appendInput["avatar"] = map[string]any{"id": 10000016, "level": 80}
	if _, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.import.append", appendInput); err != nil {
		t.Fatal(err)
	}
	if _, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.import.finish", map[string]any{"ref": ref}); err != nil {
		t.Fatal(err)
	}
	export, err := a.cloudTransferAction(t.Context(), client, "cloud.archive.export.start", input)
	if err != nil || export["total"] != 2 {
		t.Fatal(export, err)
	}
	row, err := a.cloudTransferAction(t.Context(), client, "cloud.archive.export.read", map[string]any{"ref": export["ref"], "offset": 0})
	if err != nil || row["avatar"].(map[string]any)["cookie"] != nil {
		t.Fatal(row, err)
	}
	caller.deny = true
	if _, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.export.read", map[string]any{"ref": export["ref"], "offset": 1}); err == nil {
		t.Fatal("revoked role could export")
	}
	caller.deny = false
	pending, err := a.cloudTransferAction(t.Context(), client, "cloud.archive.import.start", map[string]any{"account_ref": "account", "role_ref": "role", "uid": role.UID, "count": 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.import.append", map[string]any{"ref": pending["ref"], "offset": 0, "avatar": avatar})
	if err != nil {
		t.Fatal(err)
	}
	old, _ := a.CloudArchive.Read(client.Provider, choice)
	if err = a.CloudArchive.Update(client.Provider, choice, old.Revision, role, nil, true); err != nil {
		t.Fatal(err)
	}
	if _, err = a.cloudTransferAction(t.Context(), client, "cloud.archive.import.finish", map[string]any{"ref": pending["ref"]}); err == nil {
		t.Fatal("late import resurrected removed archive")
	}
}
func TestCloudExchangeDownloadKeepsRawDataOutOfResultAndUploadConsent(t *testing.T) {
	q := CloudInput{Mode: "exchange_download", UID: "100000001", Consent: true}
	result, err := projectCloud(testGame(t, "genshin"), q, cloudObject(t, `{"retcode":100,"data":{"uid":"100000001","avatars":{"10000046":{"id":10000046,"level":90,"cookie":"not-kept"}}}}`))
	if err != nil || result.Exchange.Characters != 1 || len(result.exchange) == 0 {
		t.Fatal(result, err)
	}
	out, _ := json.Marshal(result)
	if strings.Contains(string(out), "avatars") || strings.Contains(string(out), "not-kept") {
		t.Fatal("raw exchange leaked in public result")
	}
	q.Mode = "exchange_upload"
	if _, _, err = cloudRequest("genshin", q); err == nil {
		t.Fatal("arbitrary caller upload accepted without owned archive")
	}
	q.exchangeData = string(result.exchange)
	if route, body, err := cloudRequest("genshin", q); err != nil || route != "panel/upload" || body["type"] != "gs" {
		t.Fatal(body, err)
	}
	q.Consent = false
	if _, _, err = cloudRequest("genshin", q); err == nil {
		t.Fatal("upload without consent")
	}
}
