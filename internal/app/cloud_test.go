package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type cloudDoer func(*http.Request) (*http.Response, error)

func (f cloudDoer) Do(r *http.Request) (*http.Response, error) { return f(r) }
func TestCloudRequiresConsentAndUsesOnlyFixedPublicParameters(t *testing.T) {
	if _, _, err := cloudRequest("genshin", CloudInput{Mode: "usage"}); err == nil {
		t.Fatal("cloud enabled without consent")
	}
	client := CloudClient{HTTP: cloudDoer(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://ark.ivny.cn/rank/data" || req.Header.Get("Cookie") != "" || req.Header.Get("Authorization") != "" {
			t.Fatal("unexpected cloud credential or endpoint")
		}
		var body map[string]any
		if json.NewDecoder(req.Body).Decode(&body) != nil || body["uid"] != "100000001" || body["id"] != float64(1001) || body["qq"] != nil {
			t.Fatal("unexpected cloud payload")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":100,"rank":12,"score":99.5,"token":"not-rendered"}`))}, nil
	})}
	defer client.Close()
	job, err := client.Start(Game{ID: "genshin"}, CloudInput{Mode: "rank", UID: "100000001", CharacterID: "1001", Consent: true})
	if err != nil {
		t.Fatal(err)
	}
	for range 100 {
		job, err = client.Poll(job.Ref, false)
		if err != nil {
			t.Fatal(err)
		}
		if job.State != "running" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if job.State != "completed" || job.View == nil || strings.Contains(job.View.Text(), "not-rendered") {
		t.Fatal(job.State)
	}
}
func TestCloudCancellationDiscardsLateResults(t *testing.T) {
	started := make(chan struct{})
	client := CloudClient{HTTP: cloudDoer(func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, context.Canceled
	})}
	defer client.Close()
	job, err := client.Start(Game{ID: "genshin"}, CloudInput{Mode: "usage", Consent: true})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if job, err = client.Poll(job.Ref, true); err != nil || job.State != "canceled" {
		t.Fatal(err)
	}
	if _, err = client.Poll(job.Ref, false); err == nil {
		t.Fatal("canceled result retained")
	}
}

func TestCloudDistributionAcceptsCurrentMultipleScoreLists(t *testing.T) {
	client := CloudClient{HTTP: cloudDoer(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"retcode":100,"data":{"name":"方案一","total":10,"scores":["1","2"]}},{"retcode":100,"data":{"name":"方案二","total":20,"scores":["3"]}}]`))}, nil
	})}
	view, err := client.fetch(t.Context(), Game{ID: "genshin"}, CloudInput{Mode: "distribution"}, "rank/specific", map[string]any{})
	if err != nil || len(view.Sections) != 4 {
		t.Fatal(view, err)
	}
}
