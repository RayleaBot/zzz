package app

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSimulationReferenceProbabilityVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/simulation-probability-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Game     string `json:"game"`
		Kind     string `json:"kind"`
		Five     int    `json:"five"`
		Weekly   int    `json:"weekly"`
		Expected int    `json:"expected"`
	}
	if json.Unmarshal(raw, &cases) != nil {
		t.Fatal("bad vectors")
	}
	for _, c := range cases {
		if actual := simulationProbability(c.Game, c.Kind, c.Five, c.Weekly); actual != c.Expected {
			t.Fatalf("%s/%s five=%d week=%d: %d != %d", c.Game, c.Kind, c.Five, c.Weekly, actual, c.Expected)
		}
	}
}
func TestSimulationQuotaReplayResetAndBusinessDay(t *testing.T) {
	s := &SimulationStore{Game: "genshin", Deck: testGame(t, "genshin").Data.Simulation, Directory: t.TempDir(), Random: func(n int) int { return n - 1 }}
	now := time.Date(2026, 9, 19, 19, 59, 0, 0, time.UTC)
	if _, err := s.Action("a", "simulation.configure", map[string]any{"daily_limit": 10}, now); err != nil {
		t.Fatal(err)
	}
	batch, err := s.Action("a", "simulation.draw", map[string]any{"count": 10, "request_ref": "first"}, now)
	if err != nil || batch["daily_remaining"] != 0 {
		t.Fatal(err)
	}
	restarted := &SimulationStore{Game: s.Game, Deck: s.Deck, Directory: s.Directory, Random: s.Random}
	again, err := restarted.Action("a", "simulation.draw", map[string]any{"count": 10, "request_ref": "first"}, now)
	if err != nil || again["total"] != int64(10) {
		t.Fatal("restart replay duplicated draws", err)
	}
	if _, err = s.Action("a", "simulation.draw", map[string]any{"count": 1, "request_ref": "first"}, now); err == nil || PublicError(err).Code != "plugin.game_simulation_conflict" {
		t.Fatal("mismatched replay accepted", err)
	}
	if _, err = s.Action("a", "simulation.reset", map[string]any{"confirm": true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Action("a", "simulation.draw", map[string]any{"count": 1, "request_ref": "new"}, now); err == nil || PublicError(err).Code != "plugin.game_simulation_quota" {
		t.Fatal("reset bypassed quota", err)
	}
	if result, err := s.Action("a", "simulation.draw", map[string]any{"count": 10, "request_ref": "next-day"}, now.Add(time.Minute)); err != nil || result["daily_used"] != 10 {
		t.Fatal("04:00 reset failed", err)
	}
	if result, err := s.Action("b", "simulation.status", nil, now); err != nil || result["total"] != int64(0) {
		t.Fatal("scope shared simulated inventory")
	}
}
func TestSimulationConcurrentAdmissionDoesNotPartiallySpend(t *testing.T) {
	s := &SimulationStore{Game: "starrail", Deck: testGame(t, "starrail").Data.Simulation, Directory: t.TempDir(), Random: func(n int) int { return n - 1 }}
	now := time.Now()
	_, _ = s.Action("a", "simulation.configure", map[string]any{"daily_limit": 10}, now)
	var wg sync.WaitGroup
	var completed atomic.Int32
	for i := range 20 {
		wg.Go(func() {
			if _, err := s.Action("a", "simulation.draw", map[string]any{"count": 1, "request_ref": strconv.Itoa(i)}, now); err == nil {
				completed.Add(1)
			}
		})
	}
	wg.Wait()
	if completed.Load() != 10 {
		t.Fatal("quota raced", completed.Load())
	}
	status, err := s.Action("a", "simulation.status", nil, now)
	if err != nil || len(status["history"].([]SimulationDraw)) != 10 {
		t.Fatal("partial or missing batch", err)
	}
}
func TestSimulationFateAndPoolStateStayIndependent(t *testing.T) {
	deck := *testGame(t, "genshin").Data.Simulation
	state := newSimulationState(deck)
	s := &SimulationStore{Game: "genshin", Deck: &deck, Random: func(n int) int { return n - 1 }}
	selection := state.Selections["weapon"]
	selection.FateLimit = 1
	if err := s.selectPool(&state, selection); err != nil {
		t.Fatal(err)
	}
	state.Pools["weapon"].Five = 80
	first := s.drawOne(&state, selection, time.Now())
	if first.Rarity != 5 {
		t.Fatal("reference hard guarantee not applied")
	}
	state.Pools["weapon"].Fate = 1
	state.Pools["weapon"].UpFive = true
	state.Pools["weapon"].Five = 80
	second := s.drawOne(&state, selection, time.Now())
	if !second.Fate || second.Name != selection.FateTarget || state.Pools["weapon"].Fate != 0 || state.Pools["weapon"].UpFive {
		t.Fatal("fate guarantee did not consume pending guarantee")
	}
	if state.Pools["character"].Five != 0 || state.Pools["standard"].Five != 0 {
		t.Fatal("pity shared across pool types")
	}
	state.Pools["weapon"].Fate = 1
	selection.FateTarget = ""
	if err := s.selectPool(&state, selection); err != nil || state.Pools["weapon"].Fate != 0 {
		t.Fatal("cancel fate retained points")
	}
}

// Yunzai's 定轨 steps through the pool's five-star weapons and cancels past
// the last one.
func TestSimulationFateStepsLikeYunzai(t *testing.T) {
	weapons := []string{"护摩之杖", "天空之刃"}
	for current, want := range map[string]string{"": "护摩之杖", "护摩之杖": "天空之刃", "天空之刃": "", "旧武器": "护摩之杖"} {
		if got := simulationNextFate(weapons, current); got != want {
			t.Errorf("next after %q = %q, want %q", current, got, want)
		}
	}
}

func TestSimulationLimitTextFollowsUpstream(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	yesterday := now.Add(-20 * time.Hour).UnixMilli()
	history := []SimulationDraw{
		{Name: "迪卢克", Rarity: 5, Kind: "character", Interval: 90, TimeMS: yesterday},
		{Name: "雷电将军", Rarity: 5, Kind: "character", Interval: 73, TimeMS: now.UnixMilli()},
		{Name: "弹弓", Rarity: 3, Kind: "standard", TimeMS: now.UnixMilli()},
		{Name: "护摩之杖", Rarity: 5, Kind: "weapon", Interval: 60, TimeMS: now.UnixMilli()},
		{Name: "黑缨枪", Rarity: 3, Kind: "weapon", TimeMS: now.UnixMilli()},
	}
	status := map[string]any{"history": history, "weekly_top": 3}
	// Yunzai counts the weapon pool apart and lists every five-star today.
	if got := simulationLimitText("genshin", "一个很长的群名片名字", "character", status, now); got != "一个很长的...\n今日五星：雷电将军(73)\n护摩之杖(60)\n本周：3个五星" {
		t.Errorf("genshin = %q", got)
	}
	status = map[string]any{"history": history[2:3], "weekly_top": 0}
	if got := simulationLimitText("genshin", "旅行者", "standard", status, now); got != "旅行者\n今日已抽，累计1抽无五星" {
		t.Errorf("genshin without five-stars = %q", got)
	}
	status = map[string]any{"history": history, "weekly_top": 3}
	if got := simulationLimitText("starrail", "开拓者", "weapon", status, now); got != "今日抽已抽2抽，其中1个五星，明日再来吧" {
		t.Errorf("starrail = %q", got)
	}
}
