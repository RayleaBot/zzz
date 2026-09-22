package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"slices"
	"sync"
	"time"
)

type ContentJob struct {
	Ref     string         `json:"ref"`
	State   string         `json:"state"`
	Result  map[string]any `json:"result,omitempty"`
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
	created time.Time
	cancel  context.CancelFunc
}
type ContentJobs struct {
	mu     sync.Mutex
	items  map[string]*ContentJob
	closed bool
}

func (s *ContentJobs) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	for _, j := range s.items {
		j.cancel()
	}
	s.items = nil
}
func (s *ContentJobs) Start(run func(context.Context) (map[string]any, error)) (ContentJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ContentJob{}, gameError("public_unavailable", "公开资料查询已停止。")
	}
	if s.items == nil {
		s.items = map[string]*ContentJob{}
	}
	for ref, j := range s.items {
		if j.State == "canceled" || time.Since(j.created) > 10*time.Minute {
			j.cancel()
			delete(s.items, ref)
		}
	}
	if len(s.items) >= 8 {
		return ContentJob{}, gameError("public_limit", "公开查询任务已满，请完成或关闭已有任务。")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	j := &ContentJob{Ref: rand.Text(), State: "running", created: time.Now(), cancel: cancel}
	s.items[j.Ref] = j
	expiry := time.AfterFunc(10*time.Minute, func() { s.mu.Lock(); defer s.mu.Unlock(); cancel(); delete(s.items, j.Ref) })
	j.cancel = func() { cancel(); expiry.Stop() }
	go func() {
		defer cancel()
		result, err := run(ctx)
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || j.State != "running" {
			return
		}
		if err != nil {
			e := PublicError(err)
			j.State = "failed"
			j.Code = e.Code
			j.Message = e.Message
			return
		}
		raw, e := json.Marshal(result)
		if e != nil || len(raw) > 512*1024 {
			j.State = "failed"
			j.Code = "plugin.game_public_invalid"
			j.Message = "公开资料超过显示上限，请缩小范围。"
			return
		}
		j.Result = result
		j.State = "completed"
	}()
	return *j, nil
}
func (s *ContentJobs) Poll(ref string, cancel bool) (ContentJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.items[ref]
	if j == nil {
		return ContentJob{}, gameError("public_missing", "公开资料任务已失效，请重新查询。")
	}
	if cancel {
		j.cancel()
		j.State = "canceled"
		j.Result = nil
	}
	return *j, nil
}
func (a *App) contentAction(action string, input map[string]any) (map[string]any, error) {
	if action == "content.poll" || action == "content.cancel" {
		job, err := a.ContentJobs.Poll(asText(input["ref"]), action == "content.cancel")
		return map[string]any{"job": job}, err
	}
	var q ContentQuery
	if decodeObject(input, &q) != nil || action != "content.start" || !slices.Contains([]string{"stats", "estimate", "guides", "codes", "news", "search", "post", "calendar"}, q.Kind) {
		return nil, gameError("operation_denied", "公开资料查询类型不存在。")
	}
	job, err := a.ContentJobs.Start(func(ctx context.Context) (map[string]any, error) { return a.fetchContent(ctx, q) })
	return map[string]any{"job": job}, err
}

// fetchContent runs one anonymous public query. Chat commands wait for it
// within the event deadline; the management page runs it as a job.
func (a *App) fetchContent(ctx context.Context, q ContentQuery) (map[string]any, error) {
	switch q.Kind {
	case "stats":
		return a.publicStatistics(ctx, q)
	case "estimate":
		return a.Content.estimate(ctx, a.Game.ID)
	case "guides":
		return a.Content.guides(ctx, a.Game.ID, q)
	case "codes":
		return a.Content.codes(ctx, a.Game.ID)
	case "calendar":
		return a.Content.calendar(ctx, a.Game.ID)
	default:
		return a.Content.posts(ctx, a.Game.ID, q)
	}
}
