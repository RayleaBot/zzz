package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func cloudObject(t *testing.T, s string) map[string]any {
	t.Helper()
	var v map[string]any
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
func waitCloud(t *testing.T, c *CloudClient, j CloudJob) CloudJob {
	t.Helper()
	for range 500 {
		v, err := c.Poll(j.Ref, false)
		if err != nil {
			t.Fatal(err)
		}
		if v.State != "running" {
			return v
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("cloud job did not complete")
	return CloudJob{}
}
func TestCloudCustomFiltersAndProjection(t *testing.T) {
	for op, rule := range map[string]int{">=": 0, "=": 1, "<=": 2, ">": 3, "<": 4, "!=": 5} {
		route, body, err := cloudRequest("starrail", CloudInput{Mode: "custom", Consent: true, CharacterID: "1001", Sort: "mark_score", Limit: 7, Filters: []CloudFilter{{op, 0}}})
		if err != nil || route != "rank/custom" || body["game"] != "sr" {
			t.Fatal(route, err)
		}
		d := body["data"].(map[string]any)
		f := d["filter"].([]any)[0].(map[string]any)
		if f["rule"] != rule || f["type"] != "cons" || d["nums"] != 7 {
			t.Fatal(body)
		}
	}
	for _, q := range []CloudInput{{Sort: "sql"}, {Limit: 51}, {Filters: []CloudFilter{{"=", 7}}}, {Filters: []CloudFilter{{"or", 0}}}, {Filters: []CloudFilter{{"=", 0}, {"=", 0}, {"=", 0}}}} {
		q.Mode = "custom"
		q.Consent = true
		q.CharacterID = "1001"
		if _, _, err := cloudRequest("starrail", q); err == nil {
			t.Fatal("invalid filter accepted")
		}
	}
	r, err := projectCloud(testGame(t, "genshin"), CloudInput{Mode: "custom"}, cloudObject(t, `{"retcode":0,"data":{"rows":[{"uid":"100000001","cons":0,"dmg_avg":0,"mark_score":99.5,"cookie":"sentinel-private"}],"query_id":"private-query-ref"},"token":"sentinel-private"}`))
	if err != nil || len(r.Ranking.Rows) != 1 || r.Ranking.Rows[0].Cons != "0" || r.Ranking.Rows[0].Damage != "0" {
		t.Fatal(r, err)
	}
	raw, _ := json.Marshal(r)
	if strings.Contains(string(raw), "sentinel-private") || strings.Contains(string(raw), "private-query-ref") {
		t.Fatal("nonbusiness fields exposed")
	}
	r, err = projectCloud(Game{}, CloudInput{Mode: "custom"}, cloudObject(t, `{"retcode":0,"data":{"rows":[]}}`))
	if err != nil || len(r.Ranking.Rows) != 0 {
		t.Fatal(r, err)
	}
}
func TestCloudRankPanelUsesRetainedRowAndExpires(t *testing.T) {
	client := CloudClient{HTTP: cloudDoer(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Cookie") != "" || req.Header.Get("Authorization") != "" {
			t.Error("credential sent")
		}
		var body map[string]any
		_ = json.NewDecoder(req.Body).Decode(&body)
		response := `{"retcode":0,"data":{"rows":[{"uid":"100000001"}],"query_id":"upstream-ref"}}`
		if req.URL.Path == "/rank/custom/specific" {
			if body["query_id"] != "upstream-ref" || body["index"] != float64(0) || body["uid"] != nil {
				t.Error("incorrect detail request")
			}
			response = `{"retcode":0,"data":{"info":{"uid":"100000001","avatars":{"10000046":{"id":10000046,"level":90,"cons":0}}}}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	defer client.Close()
	j, err := client.Start(testGame(t, "genshin"), CloudInput{Mode: "custom", CharacterID: "10000046", Consent: true})
	if err != nil {
		t.Fatal(err)
	}
	j = waitCloud(t, &client, j)
	if j.State != "completed" {
		t.Fatal(j)
	}
	zero := 0
	q := CloudInput{Mode: "rank_panel", Ref: j.Ref, Index: &zero, UID: "999999999", Consent: true}
	if _, err = client.Start(testGame(t, "starrail"), q); err == nil {
		t.Fatal("cross game detail accepted")
	}
	child, err := client.Start(testGame(t, "genshin"), q)
	if err != nil {
		t.Fatal(err)
	}
	child = waitCloud(t, &client, child)
	if child.State != "completed" || len(child.Panels) != 1 {
		t.Fatal(child)
	}
	if _, err = client.Poll(j.Ref, false); err != nil {
		t.Fatal("reading child removed rank")
	}
	client.mu.Lock()
	client.jobs[j.Ref].CreatedAtMS = time.Now().Add(-6 * time.Minute).UnixMilli()
	client.mu.Unlock()
	if _, err = client.Start(testGame(t, "genshin"), q); err == nil {
		t.Fatal("expired rank used")
	}
}
func TestCloudPanelIdentityMissingAndGear(t *testing.T) {
	d := cloudObject(t, `{"retcode":100,"data":{"uid":"100000001","cookie":"not-for-ui","avatars":{"10000046":{"id":10000046,"cons":0,"talent":{"a":1,"e":{"level":2}},"artis":{"1":{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204,501204]}}}}}}`)
	r, err := projectCloud(testGame(t, "genshin"), CloudInput{Mode: "panel", UID: "100000001"}, d)
	if err != nil || len(r.Panels) != 1 {
		t.Fatal(r, err)
	}
	s := r.Panels[0].View.Text()
	if !strings.Contains(s, "4778.55") || !strings.Contains(s, "7.78%") || strings.Contains(s, "等级：0") || strings.Contains(s, "not-for-ui") {
		t.Fatal(s)
	}
	if _, err = projectCloud(testGame(t, "genshin"), CloudInput{Mode: "panel", UID: "100000002"}, d); err == nil {
		t.Fatal("wrong uid accepted")
	}
	gear := asObject(asObject(asObject(asObject(d["data"])["avatars"])["10000046"])["artis"])
	asObject(gear["1"])["attrIds"] = []any{"unknown"}
	r, err = projectCloud(testGame(t, "genshin"), CloudInput{Mode: "panel", UID: "100000001"}, d)
	if err != nil || !strings.Contains(r.Panels[0].View.Text(), "未知档位") || strings.Contains(r.Panels[0].View.Text(), "7.78%") {
		t.Fatal(r, err)
	}
	sr := cloudObject(t, `{"retcode":100,"data":{"uid":"100000001","avatars":[{"id":1001,"artis":{"1":{"id":61011,"level":15,"mainId":1,"attrIds":["1,2,2"]}}}]}}`)
	r, err = projectCloud(testGame(t, "starrail"), CloudInput{Mode: "panel", UID: "100000001"}, sr)
	if err != nil || !strings.Contains(r.Panels[0].View.Text(), "云无留迹的过客") || strings.Contains(r.Panels[0].View.Text(), "未输出") {
		t.Fatal(r, err)
	}
}

func TestCloudAnonymousPanelPreservesPrivacyAndSelectedCharacter(t *testing.T) {
	d := cloudObject(t, `{"retcode":0,"data":{"info":{"uid":"anonymous","avatars":[{"id":10000046,"cons":0},{"id":10000016,"cons":6}]}}}`)
	r, err := projectCloud(testGame(t, "genshin"), CloudInput{Mode: "rank_panel", CharacterID: "10000046"}, d)
	if err != nil || len(r.Panels) != 1 || r.Panels[0].ID != "10000046" || !strings.Contains(r.View.Subtitle, "服务未公开") {
		t.Fatal(r, err)
	}
	if _, err = projectCloud(testGame(t, "genshin"), CloudInput{Mode: "rank_panel", CharacterID: "10000099"}, d); err == nil {
		t.Fatal("different character accepted")
	}
	if _, err = projectCloud(testGame(t, "genshin"), CloudInput{Mode: "panel", UID: "100000001"}, d); err == nil {
		t.Fatal("anonymous transfer accepted as owned target")
	}
}

func TestCloudVerificationScopeConsentAndNoAutomaticRetry(t *testing.T) {
	owner := Subject{SourceProtocol: "onebot11", SourceAdapter: "test", BotID: "10000", ActorID: "10001"}
	input := CloudInput{Mode: "verify_bind", UID: "100000001", Consent: true}
	if _, _, err := cloudRequest("genshin", input); err == nil {
		t.Fatal("management impersonation accepted")
	}
	input.owner = &owner
	route, body, err := cloudRequest("genshin", input)
	if err != nil || route != "verify" || body["qq"] != owner.ActorID || body["uid"] != input.UID {
		t.Fatal(body, err)
	}
	input.Mode = "verify_code"
	_, body, err = cloudRequest("genshin", input)
	if err != nil || body["qq"] != nil {
		t.Fatal("get code sent QQ")
	}
	started := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	c := CloudClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		calls++
		close(started)
		<-release
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":100,"data":{"verifyCode":"synthetic-123"},"token":"not-for-ui"}`))}, nil
	})}
	defer c.Close()
	j, err := c.Start(testGame(t, "genshin"), input)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if _, err = c.Start(testGame(t, "genshin"), input); err == nil {
		t.Fatal("concurrent verification accepted")
	}
	if _, err = c.Poll(j.Ref, false); err == nil {
		t.Fatal("private result exposed to management")
	}
	if _, err = c.Poll(j.Ref, true); err == nil {
		t.Fatal("management canceled private result")
	}
	other := owner
	other.BotID = "20000"
	if _, err = c.pollVerification("genshin", other, false); err == nil {
		t.Fatal("cross bot read accepted")
	}
	close(release)
	for range 500 {
		j, err = c.pollVerification("genshin", owner, false)
		if err != nil {
			t.Fatal(err)
		}
		if j.State != "running" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if j.State != "completed" || !strings.Contains(j.View.Text(), "synthetic-123") || strings.Contains(j.View.Text(), "not-for-ui") || calls != 1 {
		t.Fatal(j, err)
	}
	if _, err = c.pollVerification("starrail", owner, false); err == nil {
		t.Fatal("cross game read accepted")
	}
	if _, err = c.pollVerification("genshin", owner, true); err != nil {
		t.Fatal(err)
	}
	if _, err = c.pollVerification("genshin", owner, false); err == nil {
		t.Fatal("canceled verification retained")
	}
	for _, code := range []string{"302", "305", "306", "-1"} {
		_, err := cloudVerifyResult(Game{}, CloudInput{Mode: "verify_bind"}, map[string]any{"retcode": code, "message": "private-upstream"})
		if err == nil || strings.Contains(err.Error(), "private-upstream") {
			t.Fatal(err)
		}
	}
}

func TestCanceledCloudRequestsReleaseCapacity(t *testing.T) {
	c := CloudClient{HTTP: cloudDoer(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	defer c.Close()
	for range 12 {
		j, err := c.Start(testGame(t, "genshin"), CloudInput{Mode: "usage", Consent: true})
		if err != nil {
			t.Fatal("canceled requests exhausted capacity", err)
		}
		if _, err = c.Poll(j.Ref, true); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPrivateCloudPanelCannotBeRequestedOrReadFromManagement(t *testing.T) {
	q := CloudInput{Mode: "private_panel", Consent: true, UID: "100000001"}
	if _, _, err := cloudRequest("genshin", q); err == nil {
		t.Fatal("management could supply QQ")
	}
	owner := Subject{SourceProtocol: "onebot11", SourceAdapter: "fixture", BotID: "bot", ActorID: "10001"}
	q.owner = &owner
	route, body, err := cloudRequest("genshin", q)
	if err != nil || route != "panel/data" || body["qq"] != "10001" {
		t.Fatal(body, err)
	}
	c := CloudClient{HTTP: cloudDoer(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":100,"data":{"playerData":{"uid":"100000001","avatars":{"10000046":{"id":10000046,"level":90}}}}}`))}, nil
	})}
	defer c.Close()
	j, err := c.Start(testGame(t, "genshin"), q)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Poll(j.Ref, false); err == nil {
		t.Fatal("management read private panel")
	}
	var result CloudJob
	for range 100 {
		result, err = c.pollPrivatePanel("genshin", owner, false)
		if err != nil {
			t.Fatal(err)
		}
		if result.State != "running" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if result.State != "completed" || result.Exchange == nil || result.Exchange.Characters != 1 {
		t.Fatal(result)
	}
	other := owner
	other.BotID = "other"
	if _, err = c.pollPrivatePanel("genshin", other, false); err == nil {
		t.Fatal("cross bot private data")
	}
	if _, err = c.pollVerification("genshin", owner, false); err == nil {
		t.Fatal("panel reused as verification")
	}
}
