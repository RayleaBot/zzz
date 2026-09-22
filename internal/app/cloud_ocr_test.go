package app

import (
	"context"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"strings"
	"testing"
)

func TestCloudOCRURLAndGearProjection(t *testing.T) {
	q := CloudInput{Mode: "ocr", Consent: true, ImageURL: "https://images.example.com/gear.png", Slot: 1, Forge: true}
	route, body, err := cloudRequest("genshin", q)
	if err != nil || route != "ocr/profilechange/gs" || body["forge"] != true {
		t.Fatal(route, body, err)
	}
	for _, url := range []string{"http://example.com/a.png", "https://127.0.0.1/a.png", "https://user:pass@example.com/a.png", "https://example.com/a.png?authkey=secret", "https://localhost/a.png"} {
		q.ImageURL = url
		if _, _, err = cloudRequest("genshin", q); err == nil {
			t.Fatal("invalid image URL accepted")
		}
	}
	data := cloudObject(t, `{"retcode":100,"data":[{"data":{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204]}},{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204,501204],"token":"not-for-output"}]}`)
	result, err := projectCloud(testGame(t, "genshin"), q, data)
	if err != nil || result.OCR == nil || result.OCR.Replacement == nil || result.OCR.NeedsIdentity {
		t.Fatal(result, err)
	}
	if len(result.OCR.Replacement.Main) != 1 || result.OCR.Replacement.Main[0].Value != "4778.55" || !strings.Contains(result.View.Text(), "7.78%") {
		t.Fatal(result)
	}
	if result.OCR.replacement["token"] != nil {
		t.Fatal("unknown OCR metadata persisted")
	}
	raw := asObject(asList(data["data"])[1])
	delete(raw, "name")
	result, err = projectCloud(testGame(t, "genshin"), q, data)
	if err != nil || !result.OCR.NeedsIdentity {
		t.Fatal("missing identity was invented", err)
	}
}

type ocrCaller struct {
	deny    bool
	queries int
}

func (c *ocrCaller) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	if c.deny {
		return gameError("role_missing", "revoked")
	}
	c.queries++
	data := map[string]any{"list": []any{map[string]any{"id": 10000046, "name": "胡桃", "element": "Pyro", "level": 90, "promote_level": 6, "actived_constellation_num": 0, "weapon": map[string]any{"id": 13501, "name": "护摩之杖", "level": 90, "promote_level": 6, "affix_level": 1}, "relics": []any{}, "skills": []any{map[string]any{"skill_type": 1, "level": 10}, map[string]any{"skill_type": 1, "level": 10}, map[string]any{"skill_type": 1, "level": 10}}, "properties": []any{map[string]any{"property_type": 2000, "base": "15552.31", "final": "30000"}, map[string]any{"property_type": 2001, "base": "714.5", "final": "2000"}, map[string]any{"property_type": 2002, "base": "876.15", "final": "1000"}, map[string]any{"property_type": 20, "final": "50%"}, map[string]any{"property_type": 22, "final": "100%"}, map[string]any{"property_type": 40, "final": "46.6%"}, map[string]any{"property_type": 28, "final": "100"}, map[string]any{"property_type": 23, "final": "100%"}}}}}
	return decodeObject(QueryResult{Role: Role{Game: "genshin", Ref: "role", UID: "100000001"}, Data: data}, out)
}
func TestCloudOCRUsesFreshAuthorizedPanelAndProducesCandidate(t *testing.T) {
	raw := cloudObject(t, `{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204,501204]}`)
	a := App{Game: testGame(t, "genshin")}
	a.Cloud.jobs = map[string]*CloudJob{"ocr": {Ref: "ocr", State: "completed", game: "genshin", CloudResult: CloudResult{OCR: &CloudOCR{Slot: 1, replacement: raw}}}}
	caller := &ocrCaller{}
	client := AccountsClient{Game: "genshin", Caller: caller}
	input := map[string]any{"ref": "ocr", "account_ref": "account", "role_ref": "role", "character_id": "10000046"}
	result, err := a.compareCloudOCR(t.Context(), client, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	build := result["build"].(BuildResult)
	if caller.queries != 1 || build.Candidate == nil || len(build.Candidate.Results) == 0 || !strings.Contains(result["view"].(View).Text(), "→") {
		t.Fatal("OCR comparison lacks candidate")
	}
	changed := false
	for i, before := range build.Baseline.Results {
		after := build.Candidate.Results[i]
		if before.Expected != nil && after.Expected != nil && *before.Expected != *after.Expected {
			changed = true
		}
	}
	if !changed {
		t.Fatal("replacement gear did not affect calculation")
	}
	caller.deny = true
	if _, err = a.compareCloudOCR(t.Context(), client, input); err == nil {
		t.Fatal("revoked account reused panel")
	}
}
