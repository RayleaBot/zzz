package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CloudInput struct {
	localPanel      json.RawMessage
	exchangeData    string
	ImageURL        string                `json:"image_url,omitempty"`
	Slot            int                   `json:"slot,omitempty"`
	Forge           bool                  `json:"forge,omitempty"`
	CharacterIDs    []string              `json:"character_ids,omitempty"`
	UIDs            []string              `json:"uids,omitempty"`
	Query           string                `json:"query,omitempty"`
	Refresh         bool                  `json:"refresh,omitempty"`
	Version         string                `json:"version,omitempty"`
	AdvancedFilters []CloudAdvancedFilter `json:"advanced_filters,omitempty"`
	Authenticated   bool                  `json:"authenticated,omitempty"`
	// proxy sends an authorized request through the accounts plugin, which
	// holds the ark token; anonymous requests go to ark directly.
	proxy       cloudProxy
	Mode        string        `json:"mode"`
	UID         string        `json:"uid"`
	CharacterID string        `json:"character_id"`
	Consent     bool          `json:"consent"`
	Sort        string        `json:"sort,omitempty"`
	Limit       int           `json:"limit,omitempty"`
	Filters     []CloudFilter `json:"filters,omitempty"`
	Ref         string        `json:"ref,omitempty"`
	Index       *int          `json:"index,omitempty"`
	owner       *Subject
}
type CloudJob struct {
	mode          string
	authenticated bool
	CloudResult
	Ref         string `json:"ref"`
	State       string `json:"state"`
	Code        string `json:"code,omitempty"`
	Message     string `json:"message,omitempty"`
	CreatedAtMS int64  `json:"created_at_ms"`
	cancel      context.CancelFunc
	done        chan struct{}
	game        string
	characterID string
	owner       *Subject
}

type CloudResult struct {
	Exchange *CloudExchangeInfo `json:"exchange,omitempty"`
	exchange json.RawMessage
	OCR      *CloudOCR     `json:"ocr,omitempty"`
	View     *View         `json:"view,omitempty"`
	Ranking  *CloudRanking `json:"ranking,omitempty"`
	Panels   []CloudPanel  `json:"panels,omitempty"`
	// Stats are a distribution's scores at ark's fixed ranking percentiles.
	Stats   *CloudStats `json:"stats,omitempty"`
	queryID string
}

// cloudPercentiles are the ranking percentiles of ark's rank/specific
// scores, as ark-plugin labels them.
var cloudPercentiles = []int{1, 5, 10, 20, 30, 50, 70, 90, 95, 99}

// CloudStats is a character's damage distribution: the calculation's name,
// the UIDs counted and the score at each of cloudPercentiles.
type CloudStats struct {
	Title  string    `json:"title"`
	Total  string    `json:"total"`
	Scores []float64 `json:"scores"`
}

// cloudStats reads a rank/specific answer that gives one score per ranking
// percentile.
func cloudStats(data map[string]any) *CloudStats {
	values, ok := data["scores"].([]any)
	if !ok || len(values) != len(cloudPercentiles) {
		return nil
	}
	stats := &CloudStats{Title: plainGameText(asText(data["name"])), Total: cloudNumber(data["total"]), Scores: []float64{}}
	for _, value := range values {
		score, err := strconv.ParseFloat(cloudNumber(value), 64)
		if err != nil || math.IsNaN(score) || math.IsInf(score, 0) {
			return nil
		}
		stats.Scores = append(stats.Scores, score)
	}
	return stats
}

type CloudClient struct {
	mu     sync.Mutex
	HTTP   HTTPDoer
	jobs   map[string]*CloudJob
	closed bool
}

func (c *CloudClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	for _, j := range c.jobs {
		j.cancel()
	}
	c.jobs = nil
}
func cloudRequest(game string, input CloudInput) (string, map[string]any, error) {
	if game != "genshin" && game != "starrail" || !input.Consent {
		return "", nil, gameError("cloud_consent_required", "请确认将查询参数发送给 ark 云服务。")
	}
	if input.Authenticated && input.proxy == nil {
		return "", nil, gameError("cloud_credential_missing", "授权查询需要在账号插件管理页配置 ark 令牌。")
	}
	body := map[string]any{"version": "0.1.0"}
	if input.Mode == "private_panel" {
		return privateCloudPanelRequest(game, input)
	}
	if strings.HasPrefix(input.Mode, "verify_") {
		return cloudVerifyRequest(game, input)
	}
	if input.Mode == "exchange_upload" || input.Mode == "exchange_download" {
		return cloudExchangeRequest(game, input)
	}
	if input.Mode == "ocr" {
		return cloudOCRRequest(game, input)
	}
	if route, body, handled, err := extendedCloudRequest(game, input); handled {
		return route, body, err
	}
	if input.Mode == "usage" {
		return "auth/usage", body, nil
	}
	if input.Mode == "panel" {
		if !uidPattern.MatchString(input.UID) {
			return "", nil, gameError("input_invalid", "请输入公开游戏 UID。")
		}
		return "panel/download", map[string]any{"version": "0.1.0", "uid": input.UID, "type": cloudGame(game)}, nil
	}
	id, err := strconv.Atoi(input.CharacterID)
	if err != nil || id < 1 || id > 1000000000 {
		return "", nil, gameError("input_invalid", "请选择有效角色 ID。")
	}
	body["id"] = id
	switch input.Mode {
	case "custom":
		return customCloudRequest(game, input, id)
	case "rank":
		if !uidPattern.MatchString(input.UID) {
			return "", nil, gameError("input_invalid", "请输入公开游戏 UID。")
		}
		body["uid"] = input.UID
		if len(input.localPanel) > 0 {
			body["uid"] = "999999999"
			body["data"] = input.localPanel
		}
		if !slices.Contains([]string{"", "dmg", "mark", "all"}, input.Query) {
			return "", nil, gameError("input_invalid", "请选择伤害、装备或两类排名。")
		}
		body["query"] = input.Query
		if input.Query == "" {
			body["query"] = "dmg"
		}
		body["update"] = 0
		if input.Refresh && len(input.localPanel) == 0 {
			body["update"] = 1
		}
		return "rank/data", body, nil
	case "distribution":
		body["percent"] = 0
		return "rank/specific", body, nil
	}
	return "", nil, gameError("operation_denied", "云查询类型不存在。")
}
func (c *CloudClient) Start(game Game, input CloudInput) (CloudJob, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return CloudJob{}, gameError("cloud_unavailable", "云查询已停止。")
	}
	for ref, j := range c.jobs {
		if time.Since(time.UnixMilli(j.CreatedAtMS)) > 5*time.Minute {
			j.cancel()
			delete(c.jobs, ref)
		}
	}
	var route string
	var body map[string]any
	var err error
	if input.Mode == "rank_panel" {
		route, body, input, err = c.rankPanelRequest(game.ID, input)
	} else {
		route, body, err = cloudRequest(game.ID, input)
	}
	if err != nil {
		return CloudJob{}, err
	}
	if input.owner != nil {
		for ref, j := range c.jobs {
			if j.owner != nil && *j.owner == *input.owner && j.game == game.ID {
				if j.State == "running" {
					return CloudJob{}, gameError("cloud_limit", "已有本人私聊云请求正在处理，请先查询进度。")
				}
				j.cancel()
				delete(c.jobs, ref)
			}
		}
	}
	if len(c.jobs) >= 8 {
		oldest := ""
		var oldestMS int64
		for ref, j := range c.jobs {
			if j.State == "canceled" && (oldest == "" || j.CreatedAtMS < oldestMS) {
				oldest, oldestMS = ref, j.CreatedAtMS
			}
		}
		if oldest != "" {
			c.jobs[oldest].cancel()
			delete(c.jobs, oldest)
		}
	}
	if len(c.jobs) >= 8 {
		return CloudJob{}, gameError("cloud_limit", "云查询任务已达上限，请完成或取消已有任务。")
	}
	if c.jobs == nil {
		c.jobs = map[string]*CloudJob{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	job := &CloudJob{mode: input.Mode, authenticated: input.Authenticated, Ref: rand.Text(), State: "running", CreatedAtMS: time.Now().UnixMilli(), cancel: cancel, game: game.ID, characterID: input.CharacterID}
	if input.owner != nil {
		owner := *input.owner
		job.owner = &owner
	}
	c.jobs[job.Ref] = job
	expiry := time.AfterFunc(5*time.Minute, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.jobs[job.Ref] == job {
			cancel()
			delete(c.jobs, job.Ref)
		}
	})
	job.cancel = func() { cancel(); expiry.Stop() }
	done := make(chan struct{})
	job.done = done
	go func() {
		defer close(done)
		defer cancel()
		result, err := c.fetchResult(ctx, game, input, route, body)
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.closed || job.State != "running" {
			return
		}
		if err != nil {
			failure := PublicError(err)
			job.State = "failed"
			job.Code = failure.Code
			job.Message = failure.Message
		} else {
			job.State = "completed"
			job.CloudResult = result
		}
	}()
	return *job, nil
}

// Wait returns the job once its request has finished or ctx ends.
func (c *CloudClient) Wait(ctx context.Context, job CloudJob) CloudJob {
	select {
	case <-job.done:
	case <-ctx.Done():
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if current := c.jobs[job.Ref]; current != nil {
		return *current
	}
	return job
}

func (c *CloudClient) Poll(ref string, cancel bool) (CloudJob, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pollLocked(ref, cancel, nil)
}
func (c *CloudClient) pollLocked(ref string, cancel bool, owner *Subject) (CloudJob, error) {
	job := c.jobs[ref]
	if job != nil && job.owner != nil && (owner == nil || *owner != *job.owner) {
		return CloudJob{}, gameError("cloud_missing", "云查询任务不存在。")
	}
	if job != nil && time.Since(time.UnixMilli(job.CreatedAtMS)) > 5*time.Minute {
		job.cancel()
		delete(c.jobs, ref)
		job = nil
	}
	if job == nil {
		return CloudJob{}, gameError("cloud_missing", "云查询任务已失效，请重新查询。")
	}
	if cancel {
		job.cancel()
		job.State = "canceled"
		job.CloudResult = CloudResult{}
		out := *job
		delete(c.jobs, ref)
		return out, nil
	}
	return *job, nil
}
func (c *CloudClient) fetch(ctx context.Context, game Game, input CloudInput, route string, body map[string]any) (View, error) {
	result, err := c.fetchResult(ctx, game, input, route, body)
	if err != nil {
		return View{}, err
	}
	return *result.View, nil
}

type cloudProxy func(ctx context.Context, route string, body map[string]any) (any, error)

// request sends an anonymous ark request.
func (c *CloudClient) request(ctx context.Context, route string, body map[string]any) (any, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://ark.ivny.cn/"+route, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RayleaBot-game-plugin/0.1.0")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, gameError("cloud_unavailable", "云服务暂不可用或请求已取消。")
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return nil, gameError("cloud_auth_rejected", "ark 授权被拒绝，请检查令牌及权限。")
	}
	if response.StatusCode == 429 {
		return nil, gameError("cloud_rate_limited", "ark 查询额度已用尽或请求过于频繁，请稍后再试。")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, gameError("cloud_rejected", "云服务拒绝本次查询，请稍后重试。")
	}
	maxBody := 2 * 1024 * 1024
	if route == "panel/download" || route == "panel/data" {
		maxBody = 8 * 1024 * 1024
	}
	raw, err = io.ReadAll(io.LimitReader(response.Body, int64(maxBody)+1))
	if err != nil || len(raw) > maxBody {
		return nil, gameError("cloud_invalid", "云服务响应过大或无法读取。")
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&decoded) != nil || decoder.Decode(new(any)) != io.EOF {
		return nil, gameError("cloud_invalid", "云服务数据格式暂不兼容。")
	}
	return decoded, nil
}
func (c *CloudClient) fetchResult(ctx context.Context, game Game, input CloudInput, route string, body map[string]any) (CloudResult, error) {
	if input.Mode == "akasha_stygian" {
		return c.akashaStygian(ctx, game, input)
	}
	var decoded any
	var err error
	if input.proxy != nil {
		decoded, err = input.proxy(ctx, route, body)
	} else {
		decoded, err = c.request(ctx, route, body)
	}
	if err != nil {
		return CloudResult{}, err
	}
	result, err := projectCloud(game, input, decoded)
	if err == nil {
		raw, e := json.Marshal(result)
		if e != nil || len(raw) > 256*1024 {
			return CloudResult{}, gameError("cloud_invalid", "云查询结果过大，请缩小查询范围。")
		}
	}
	return result, err
}
func projectCloud(game Game, input CloudInput, decoded any) (CloudResult, error) {
	if list, ok := decoded.([]any); ok && input.Mode == "rank" {
		return rankArrayResult(game, input, list)
	}
	if list, ok := decoded.([]any); ok {
		if input.Mode != "distribution" || len(list) == 0 || len(list) > 20 {
			return CloudResult{}, gameError("cloud_invalid", "云服务列表格式暂不兼容。")
		}
		combined := View{Title: game.Name + "角色云统计", Rows: []Row{}, Note: "来源：ark.ivny.cn；同一角色的不同计算条目分别显示。固定服务结果不代表官方结论。"}
		for _, raw := range list {
			item := asObject(raw)
			if asText(item["retcode"]) != "100" {
				return CloudResult{}, gameError("cloud_rejected", "云服务未能提供完整统计条目。")
			}
			view, err := cloudView(game, input, item)
			if err != nil {
				return CloudResult{}, err
			}
			section := Section{Title: view.Subtitle, Rows: view.Rows}
			combined.Sections = append(combined.Sections, section)
			for _, s := range view.Sections {
				s.Title = view.Subtitle + " · " + s.Title
				combined.Sections = append(combined.Sections, s)
			}
		}
		return CloudResult{View: &combined}, nil
	}
	result := asObject(decoded)
	if result == nil {
		return CloudResult{}, gameError("cloud_invalid", "云服务对象格式暂不兼容。")
	}
	if strings.HasPrefix(input.Mode, "verify_") {
		return cloudVerifyResult(game, input, result)
	}
	if err := cloudRetcode(result); err != nil {
		return CloudResult{}, err
	}
	if input.Mode == "private_panel" {
		q := input
		q.Mode = "exchange_download"
		out, err := cloudExchangeResult(game, q, result)
		if out.View != nil {
			out.View.Title = game.Name + "本人长期云面板"
			out.View.Note = "来源：ark.ivny.cn；已按本人 QQ 与 UID 请求，属于云端历史资料。"
		}
		return out, err
	}
	if input.Mode == "exchange_upload" || input.Mode == "exchange_download" {
		return cloudExchangeResult(game, input, result)
	}
	if input.Mode == "ocr" {
		return cloudOCRResult(game, input, result)
	}
	if output, handled, err := extendedCloudResult(game, input, result); handled {
		return output, err
	}
	if input.Mode == "custom" {
		return customCloudResult(game, input, result)
	}
	if input.Mode == "panel" || input.Mode == "rank_panel" {
		return cloudPanelResult(game, input, result)
	}
	view, err := cloudView(game, input, result)
	out := CloudResult{View: &view}
	if input.Mode == "distribution" {
		out.Stats = cloudStats(asObject(result["data"]))
	}
	return out, err
}

// cloudRetcode turns ark's refusals into errors.
func cloudRetcode(result map[string]any) error {
	switch asText(result["retcode"]) {
	case "0", "100":
		return nil
	case "401", "403":
		return gameError("cloud_auth_rejected", "ark 授权被拒绝，请检查令牌及权限。")
	case "429":
		return gameError("cloud_rate_limited", "ark 查询额度已用尽或请求过于频繁，请稍后再试。")
	}
	return gameError("cloud_rejected", "云服务暂未提供此查询结果，可能缺少数据、权限或额度。")
}

func cloudView(game Game, input CloudInput, result map[string]any) (View, error) {
	v := View{Title: game.Name + "云排名", Rows: []Row{}, Note: "来源：ark.ivny.cn。排名按该服务收录与计算结果展示，不代表官方结论；本次未发送米游社 CK 或 QQ。"}
	data := asObject(result["data"])
	switch input.Mode {
	case "usage":
		v.Title = "ark 匿名查询额度"
		if input.Authenticated {
			v.Title = "ark 授权查询额度"
			v.Note = "来源：ark.ivny.cn；使用本游戏独立保存的加密令牌，不发送米游社 CK。"
		}
		auth := asObject(data["auth"])
		if n := cloudNumber(auth["permission"]); n != "" {
			label := map[string]string{"0": "普通", "1": "高级"}[n]
			if label == "" {
				label = "权限 " + n
			}
			v.Rows = append(v.Rows, Row{Label: "权限类型", Value: label})
		}
		if n := cloudNumber(auth["limit_normal"]); n != "" {
			v.Rows = append(v.Rows, Row{Label: "自定义排名额度倍率", Value: n + " 倍"})
		}
		quota := asObject(data["quota"])
		for _, kind := range []string{"rank", "custom"} {
			group := asObject(quota[kind])
			keys := []string{}
			for key := range group {
				keys = append(keys, key)
			}
			slices.Sort(keys)
			for _, key := range keys {
				record := asObject(group[key])
				remaining := asText(record["remaining"])
				if remaining == "" {
					remaining = asText(group[key])
				}
				if remaining != "" {
					v.Rows = append(v.Rows, Row{Label: map[string]string{"rank": "全部请求", "custom": "自定义排名"}[kind] + " / " + map[string]string{"minute": "分钟额度", "hour": "小时额度", "day": "每日额度", "normal": "普通请求", "advanced": "高级请求"}[key], Value: remaining})
				}
			}
		}
	case "rank":
		v.Subtitle = "UID " + input.UID + " · 角色 " + input.CharacterID
		if len(input.localPanel) > 0 {
			v.Subtitle = "导入面板 · 角色 " + input.CharacterID
			v.Note = "来源：ark.ivny.cn；按明确上传的本地交换面板计算服务名次，不代表该 UID 的官方实时成绩。"
		}
		rank := asText(result["rank"])
		if rank == "" {
			rank = asText(data["rank"])
		}
		score := asText(result["score"])
		if score == "" {
			score = asText(data["score"])
		}
		if rank != "" {
			v.Rows = append(v.Rows, Row{Label: "服务收录排名", Value: rank})
		}
		if percent := cloudNumber(result["percent"]); percent != "" {
			v.Rows = append(v.Rows, Row{Label: "服务百分位", Value: percent + "%"})
		}
		if score != "" {
			v.Rows = append(v.Rows, Row{Label: "服务伤害评分", Value: score})
		}
	case "distribution":
		v.Title = game.Name + "角色云统计"
		v.Subtitle = plainGameText(asText(data["name"]))
		if total := asText(data["total"]); total != "" {
			v.Rows = append(v.Rows, Row{Label: "收录总量", Value: total})
		}
		scores := data["scores"]
		rows := []Row{}
		if stats := cloudStats(data); stats != nil {
			for i, score := range stats.Scores {
				rows = append(rows, Row{Label: "TOP " + strconv.Itoa(cloudPercentiles[i]) + "%", Value: strconv.FormatFloat(score, 'f', -1, 64)})
			}
			scores = nil
		}
		switch values := scores.(type) {
		case []any:
			for i, value := range values[:min(50, len(values))] {
				text := asText(value)
				if obj := asObject(value); obj != nil {
					text = strings.Join([]string{firstText(obj, "score", "dmg", "value"), firstText(obj, "count", "num", "total")}, " · ")
				}
				if text != "" {
					rows = append(rows, Row{Label: strconv.Itoa(i + 1), Value: text})
				}
			}
		case map[string]any:
			keys := []string{}
			for key := range values {
				keys = append(keys, key)
			}
			slices.Sort(keys)
			for _, key := range keys[:min(50, len(keys))] {
				rows = append(rows, Row{Label: plainGameText(key), Value: asText(values[key])})
			}
		}
		if len(rows) > 0 && scores == nil {
			v.Sections = []Section{{Title: "排名趋势", Rows: rows}}
		} else if len(rows) > 0 {
			v.Sections = []Section{{Title: "服务返回的分布（前 50 项）", Rows: rows}}
		}
	}
	if len(v.Rows) == 0 && len(v.Sections) == 0 {
		return View{}, gameError("cloud_invalid", "云服务未返回当前可识别的排名或额度字段。")
	}
	return v, nil
}
func (a *App) cloudAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if strings.HasPrefix(action, "cloud.archive.import.") || strings.HasPrefix(action, "cloud.archive.export.") {
		return a.cloudTransferAction(ctx, a.accountClient(event), action, input)
	}
	if strings.HasPrefix(action, "cloud.archive.") {
		return a.cloudArchiveAction(ctx, event, action, input)
	}
	if action == "cloud.ocr.compare" {
		return a.compareCloudOCR(ctx, a.accountClient(event), input)
	}
	if action == "cloud.filter_schema" {
		return map[string]any{"fields": cloudFilterSchema(a.Game.ID)}, nil
	}
	if action == "cloud.start" {
		var q CloudInput
		if decodeObject(input, &q) != nil {
			return nil, gameError("input_invalid", "云查询参数无效。")
		}
		if _, err := strconv.Atoi(q.CharacterID); q.CharacterID != "" && err != nil {
			if entry, ok := a.Catalog.Resolve(q.CharacterID, "character", nil); ok {
				q.CharacterID = entry.ID
			}
		}
		return a.startCloudJob(ctx, event, q)
	}
	if action == "cloud.poll" || action == "cloud.cancel" {
		job, err := a.Cloud.Poll(asText(input["ref"]), action == "cloud.cancel")
		return map[string]any{"job": job}, err
	}
	return nil, gameError("operation_denied", "云查询操作不存在。")
}
func (a *App) Close() {
	if a.Media != nil {
		a.Media.Close()
	}
	if a.Artwork != nil {
		a.Artwork.Close()
	}
	a.Cloud.Close()
	a.CloudTransfers.Close()
	a.BillingJobs.Close()
	a.ContentJobs.Close()
}

func (a *App) startCloudJob(ctx context.Context, event *rayleabot.EventContext, q CloudInput) (map[string]any, error) {
	if q.Authenticated {
		client := a.accountClient(event)
		q.proxy = func(_ context.Context, route string, body map[string]any) (any, error) {
			return client.Ark(ctx, route, body)
		}
	}
	job, err := a.Cloud.Start(a.Game, q)
	// The accounts plugin answers only while this event is open, so an
	// authorized query finishes before the event returns.
	if err == nil && q.proxy != nil {
		job = a.Cloud.Wait(ctx, job)
	}
	return map[string]any{"job": job}, err
}
