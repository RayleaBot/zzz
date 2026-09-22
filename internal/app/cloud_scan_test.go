package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestScanGearReadsSlotsOfAReforge(t *testing.T) {
	a := App{Game: testGame(t, "genshin")}
	a.Cloud.HTTP = cloudDoer(func(req *http.Request) (*http.Response, error) {
		var body map[string]any
		if req.URL.String() != "https://ark.ivny.cn/ocr/profilechange/gs" || json.NewDecoder(req.Body).Decode(&body) != nil || body["forge"] != true {
			t.Fatal("unexpected OCR request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":100,"data":[{"type":"arti1","data":{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204,501204]}},{"type":"arti1","data":{"star":5,"level":20,"mainId":13001,"attrIds":[501204,501224]}}]}`))}, nil
	})
	gears, err := a.scanGear(t.Context(), "https://multimedia.nt.qq.com.cn/download?rkey=1", true)
	if err != nil || len(gears) != 2 || gears[0].Slot != 1 || gears[1].Slot != 1 {
		t.Fatal(gears, err)
	}
	if _, err = a.scanGear(t.Context(), "http://gchat.qpic.cn/a.png", true); err == nil {
		t.Fatal("plain HTTP image accepted")
	}
}

func TestReforgeMatchesKeptPieceBySubstats(t *testing.T) {
	game := testGame(t, "genshin")
	// Without a slot in the answer, the piece's name gives it.
	read := readGear(game, scannedGear{Gear: cloudObject(t, `{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204,501204]}`)})
	if read == nil || read.Slot != 1 {
		t.Fatal(read)
	}
	kept := PanelEquipment{Slot: 1, Sub: []PanelStat{{Key: "cpct", Value: "7.7%"}}}
	if !sameGear("genshin", kept, *read) {
		t.Fatal("the game's shown value did not match")
	}
	for _, other := range []PanelStat{{Key: "cpct", Value: "7.0%"}, {Key: "cdmg", Value: "7.7%"}} {
		if sameGear("genshin", PanelEquipment{Slot: 1, Sub: []PanelStat{other}}, *read) {
			t.Fatal("another piece matched", other)
		}
	}
	// Star Rail's speed may be a few rolls apart, as ark-plugin allows.
	relic := PanelEquipment{Sub: []PanelStat{{Key: "speed", Value: "4.6"}, {Key: "cpct", Value: "5.8%"}}}
	if !sameGear("starrail", PanelEquipment{Sub: []PanelStat{{Key: "speed", Value: "6"}, {Key: "cpct", Value: "5.8%"}}}, relic) {
		t.Fatal("speed within ark-plugin's range did not match")
	}
	if sameGear("starrail", PanelEquipment{Sub: []PanelStat{{Key: "speed", Value: "7"}, {Key: "cpct", Value: "5.8%"}}}, relic) {
		t.Fatal("distant speed matched")
	}
}

func TestDistributionKeepsArkPercentileScores(t *testing.T) {
	data := cloudObject(t, `{"retcode":100,"data":{"name":"重击伤害","total":1234,"scores":[98000,90000,85000.5,80000,76000,70000,62000,50000,40000,30000]}}`)
	result, err := projectCloud(testGame(t, "genshin"), CloudInput{Mode: "distribution", CharacterID: "10000046"}, data)
	if err != nil || result.Stats == nil || result.Stats.Title != "重击伤害" || result.Stats.Total != "1234" || len(result.Stats.Scores) != 10 || result.Stats.Scores[2] != 85000.5 {
		t.Fatal(result.Stats, err)
	}
	if text := result.View.Text(); !strings.Contains(text, "TOP 1%：98000") || !strings.Contains(text, "TOP 99%：30000") {
		t.Fatal(text)
	}
}
