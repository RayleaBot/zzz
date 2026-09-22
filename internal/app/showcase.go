package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ShowcaseSource is the public showcase service a game refreshes panels from
// without an account, the one ZZZ-Plugin uses by default: Enka. The plugin
// reads the service's answer into panels.
type ShowcaseSource struct {
	Name  string
	URL   func(uid string) string
	Parse ShowcaseParser
}

// ShowcaseParser reads a showcase answer. Panels whose game data is missing
// are left out; ErrShowcaseEmpty means the player shows no character with
// details.
type ShowcaseParser func(ctx context.Context, game Game, catalog Catalog, raw []byte) (ShowcaseProfile, error)

// ShowcaseProfile is a player's showcase: the panels, and how long the
// service asks callers to wait before asking again.
type ShowcaseProfile struct {
	Nickname string
	Level    int
	Panels   []CharacterPanel
	TTL      time.Duration
}

// ErrShowcaseEmpty is the answer of a player who shows no character.
var ErrShowcaseEmpty = errors.New("showcase is empty")

// ShowcaseFailure is a showcase service that could not be read: Status is
// the HTTP status, zero when the service was unreachable.
type ShowcaseFailure struct{ Status int }

func (f *ShowcaseFailure) Error() string { return "showcase service failed: " + strconv.Itoa(f.Status) }

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type showcaseFlight struct {
	done chan struct{}
	raw  []byte
	err  error
}
type showcaseCached struct {
	raw     []byte
	expires time.Time
}

// ShowcaseClient reads one game's showcase service. Answers are kept for the
// time the service asks for, concurrent requests for a UID share one call, and
// a rate-limited service is left alone for a minute.
type ShowcaseClient struct {
	mu        sync.Mutex
	HTTP      HTTPDoer
	cache     map[string]showcaseCached
	flights   map[string]*showcaseFlight
	rateUntil time.Time
}

var uidPattern = regexp.MustCompile(`^[0-9]{6,12}$`)

func (c *ShowcaseClient) Fetch(ctx context.Context, source ShowcaseSource, game Game, catalog Catalog, uid string) (ShowcaseProfile, error) {
	if !uidPattern.MatchString(uid) {
		return ShowcaseProfile{}, gameError("input_invalid", "UID 格式不正确。")
	}
	if source.URL == nil || source.Parse == nil {
		return ShowcaseProfile{}, gameError("showcase_unavailable", "此游戏没有公开展柜服务。")
	}
	c.mu.Lock()
	if c.cache == nil {
		c.cache = map[string]showcaseCached{}
		c.flights = map[string]*showcaseFlight{}
	}
	if cached, ok := c.cache[uid]; ok && time.Now().Before(cached.expires) {
		c.mu.Unlock()
		return source.Parse(ctx, game, catalog, cached.raw)
	}
	if f := c.flights[uid]; f != nil {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return ShowcaseProfile{}, ctx.Err()
		case <-f.done:
		}
		if f.err != nil {
			return ShowcaseProfile{}, f.err
		}
		return source.Parse(ctx, game, catalog, f.raw)
	}
	if time.Now().Before(c.rateUntil) {
		c.mu.Unlock()
		return ShowcaseProfile{}, &ShowcaseFailure{Status: http.StatusTooManyRequests}
	}
	f := &showcaseFlight{done: make(chan struct{})}
	c.flights[uid] = f
	c.mu.Unlock()
	f.raw, f.err = c.fetch(ctx, source.URL(uid))
	var profile ShowcaseProfile
	if f.err == nil {
		profile, f.err = source.Parse(ctx, game, catalog, f.raw)
	}
	c.mu.Lock()
	delete(c.flights, uid)
	if f.err == nil {
		if len(c.cache) >= 128 {
			var oldest string
			for key, value := range c.cache {
				if oldest == "" || value.expires.Before(c.cache[oldest].expires) {
					oldest = key
				}
			}
			delete(c.cache, oldest)
		}
		c.cache[uid] = showcaseCached{raw: f.raw, expires: time.Now().Add(min(max(profile.TTL, time.Minute), 24*time.Hour))}
	}
	var failure *ShowcaseFailure
	if errors.As(f.err, &failure) && failure.Status == http.StatusTooManyRequests {
		c.rateUntil = time.Now().Add(time.Minute)
	}
	close(f.done)
	c.mu.Unlock()
	return profile, f.err
}

func (c *ShowcaseClient) fetch(ctx context.Context, address string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, &ShowcaseFailure{}
	}
	req.Header.Set("User-Agent", "RayleaBot-GamePlugins/0.1.0")
	req.Header.Set("Accept", "application/json")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &ShowcaseFailure{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &ShowcaseFailure{Status: resp.StatusCode}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
	if err != nil || len(raw) > 4*1024*1024 || !json.Valid(raw) {
		return nil, &ShowcaseFailure{Status: resp.StatusCode}
	}
	return raw, nil
}

func asObject(value any) map[string]any { result, _ := value.(map[string]any); return result }
func asList(value any) []any            { result, _ := value.([]any); return result }
func asText(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	}
	return ""
}
func number(value any) int { n, _ := strconv.Atoi(asText(value)); return n }
func gameError(code, message string) *rayleabot.ActionError {
	return &rayleabot.ActionError{Code: "plugin.game_" + code, Message: message}
}
