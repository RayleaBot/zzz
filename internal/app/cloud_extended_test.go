package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCloudExtendedRanksPreserveCorrespondenceAndTypes(t *testing.T) {
	q := CloudInput{Mode: "self_rank", UID: "100000001", CharacterIDs: []string{"1001", "1002"}, Consent: true}
	route, body, err := cloudRequest("starrail", q)
	if err != nil || route != "rank/self" || body["type"] != "sr" {
		t.Fatal(route, body, err)
	}
	result, err := projectCloud(Game{ID: "starrail"}, q, cloudObject(t, `{"retcode":100,"rank":[{"retcode":100,"rank":20,"score":2},{"retcode":404}]}`))
	if err != nil || len(result.View.Sections) != 2 || result.View.Sections[1].Text == "" {
		t.Fatal(result, err)
	}
	if _, err = projectCloud(Game{ID: "starrail"}, q, cloudObject(t, `{"retcode":100,"rank":[{"rank":20}]}`)); err == nil {
		t.Fatal("shifted association accepted")
	}
	q.Mode = "rank"
	q.Query = "all"
	q.CharacterID = "1001"
	result, err = projectCloud(Game{ID: "starrail"}, q, []any{cloudObject(t, `{"retcode":100,"rank":1,"percent":0,"score":100}`), cloudObject(t, `{"retcode":100,"rank":10,"percent":20,"score":50}`)})
	if err != nil || len(result.View.Sections) != 2 || result.View.Sections[1].Title != "装备评分排名" {
		t.Fatal(result, err)
	}
	q = CloudInput{Mode: "group_rank", CharacterID: "10000046", UIDs: []string{"100000001", "100000002"}, Query: "mark", Consent: true}
	route, body, err = cloudRequest("genshin", q)
	if err != nil || route != "rank/group" || body["query"] != "mark" || body["data"] != nil {
		t.Fatal(body, err)
	}
	q.UIDs = append(q.UIDs, q.UIDs[0])
	if _, _, err = cloudRequest("genshin", q); err == nil {
		t.Fatal("duplicate UID accepted")
	}
}
func TestAkashaFixedHostNeverReceivesArkToken(t *testing.T) {
	c := CloudClient{HTTP: doerFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || req.URL.Host != "akasha.cv" || req.Header.Get("Authorization") != "" || req.URL.Query().Get("uids") != "[uid]100000001" || req.URL.Query().Get("version") != "6_0" {
			t.Fatal("invalid Akasha request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"uid":"100000001","index":2,"stygianSeconds":100,"stygianIndex":6,"playerInfo":{"nickname":"测试"}},{"uid":"100000002","index":1}]}`))}, nil
	})}
	q := CloudInput{Mode: "akasha_stygian", UIDs: []string{"100000001"}, Version: "6.0", Consent: true, proxy: func(context.Context, string, map[string]any) (any, error) {
		t.Fatal("Akasha went through the ark proxy")
		return nil, nil
	}}
	result, err := c.akashaStygian(context.Background(), Game{ID: "genshin"}, q)
	if err != nil || len(result.View.Sections) != 1 {
		t.Fatal(result, err)
	}
	q.Authenticated = true
	if _, _, err = cloudRequest("genshin", q); err == nil {
		t.Fatal("ark auth allowed for Akasha")
	}
}
