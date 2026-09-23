package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

type PublicContentClient struct{ HTTP HTTPDoer }

var publicHosts = []string{"bbs-api.miyoushe.com", "bbs-api.mihoyo.com", "bbs-api-static.miyoushe.com", "api-takumi.mihoyo.com", "api-takumi-static.mihoyo.com", "hk4e-api.mihoyo.com", "hkrpg-api.mihoyo.com", "hkrpg-api-static.mihoyo.com", "announcement-api.mihoyo.com", "announcement-static.mihoyo.com"}

func (c PublicContentClient) get(ctx context.Context, address string, headers map[string]string) (map[string]any, error) {
	return c.getUpTo(ctx, address, headers, 2*1024*1024)
}

// collection reads a 米游社 collection's posts in full. Some collections the
// upstreams read answer with several megabytes, which get's limit refuses.
func (c PublicContentClient) collection(ctx context.Context, id string) (map[string]any, error) {
	params := url.Values{"gids": {bbsGID}, "order_type": {"2"}, "collection_id": {id}}
	return c.getUpTo(ctx, "https://bbs-api.mihoyo.com/post/wapi/getPostFullInCollection?"+params.Encode(), nil, 16*1024*1024)
}

// getUpTo is get for an answer of up to limit bytes.
func (c PublicContentClient) getUpTo(ctx context.Context, address string, headers map[string]string, limit int64) (map[string]any, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.User != nil || !slices.Contains(publicHosts, u.Host) {
		return nil, gameError("public_endpoint_invalid", "公开资料地址不在允许范围内。")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.miyoushe.com/")
	for key, value := range headers {
		if key != "x-rpc-act_id" || strings.ContainsAny(value, "\r\n") {
			return nil, gameError("public_endpoint_invalid", "公开资料请求参数无效。")
		}
		req.Header.Set(key, value)
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, gameError("public_unavailable", "官方公开资料暂时无法访问。")
	}
	defer res.Body.Close()
	if res.StatusCode == 429 {
		return nil, gameError("public_rate_limited", "公开资料请求过于频繁，请稍后再试。")
	}
	if res.StatusCode != 200 {
		return nil, gameError("public_unavailable", "官方公开资料暂时无法访问。")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, gameError("public_invalid", "公开资料过大或无法读取。")
	}
	var out map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&out) != nil || decoder.Decode(new(any)) != io.EOF || asText(out["retcode"]) != "0" || asObject(out["data"]) == nil {
		return nil, gameError("public_invalid", "官方公开资料格式暂不兼容。")
	}
	return asObject(out["data"]), nil
}

var publicActPattern = regexp.MustCompile(`(?:act_id=|act_id%3[Dd])([A-Za-z0-9]{6,128})`)
var publicCodePattern = regexp.MustCompile(`^[A-Za-z0-9-]{4,64}$`)

func extractPublicActs(value any) []string {
	raw, _ := json.Marshal(value)
	text := string(raw)
	for range 2 {
		decoded, err := url.QueryUnescape(text)
		if err != nil {
			break
		}
		text = decoded
	}
	out := []string{}
	for _, match := range publicActPattern.FindAllStringSubmatch(text, 16) {
		if !slices.Contains(out, match[1]) {
			out = append(out, match[1])
			if len(out) >= 8 {
				break
			}
		}
	}
	return out
}
func (c PublicContentClient) codes(ctx context.Context) (map[string]any, error) {
	gid, author := "8", "152039148"
	nav, err := c.get(ctx, "https://bbs-api.miyoushe.com/apihub/api/home/new?gids="+gid+"&parts=1%2C3%2C4", nil)
	if err != nil {
		return nil, err
	}
	acts := extractPublicActs(nav["navigator"])
	if len(acts) == 0 {
		posts, err := c.get(ctx, "https://bbs-api.mihoyo.com/painter/api/user_instant/list?offset=0&size=20&uid="+author, nil)
		if err != nil {
			return nil, err
		}
		acts = extractPublicActs(posts["list"])
	}
	rows := []map[string]any{}
	title := ""
	var lastErr error
	for _, act := range acts {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		headers := map[string]string{"x-rpc-act_id": act}
		index, err := c.get(ctx, "https://api-takumi.mihoyo.com/event/miyolive/index", headers)
		if err != nil {
			lastErr = err
			continue
		}
		live := asObject(index["live"])
		version := asText(live["code_ver"])
		if version == "" {
			continue
		}
		title = plainGameText(asText(live["title"]))
		q := url.Values{"version": {version}, "time": {asText(time.Now().Unix())}}
		codes, err := c.get(ctx, "https://api-takumi-static.mihoyo.com/event/miyolive/refreshCode?"+q.Encode(), headers)
		if err != nil {
			lastErr = err
			continue
		}
		list, ok := codes["code_list"].([]any)
		if !ok || len(list) > 50 {
			lastErr = gameError("public_invalid", "兑换码列表格式暂不兼容。")
			continue
		}
		seen := map[string]bool{}
		for _, item := range list {
			v := asObject(item)
			code := asText(v["code"])
			if !publicCodePattern.MatchString(code) || seen[code] {
				continue
			}
			seen[code] = true
			row := map[string]any{"code": code, "reward": plainGameText(asText(v["title"])), "expiry_estimated": false}
			if expires := firstText(v, "expire_time", "end_time"); expires != "" {
				row["expires_at"] = calendarTime(expires)
			} else if stamp, ok := challengeNumber(v["to_get_time"]); ok && stamp > 1000000000 {
				date := time.Unix(int64(stamp), 0).In(time.FixedZone("UTC+8", 28800))
				days, hour := 2, 23
				date = date.AddDate(0, 0, days)
				minute, second := 0, 0
				if hour == 23 {
					minute, second = 59, 59
				}
				row["expires_at"] = time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, 0, date.Location()).Format("2006-01-02 15:04:05 UTC+8")
				row["expiry_estimated"] = true
			}
			rows = append(rows, row)
		}
		return map[string]any{"title": title, "items": rows, "source": "official_miyolive", "fetched_at_ms": time.Now().UnixMilli()}, nil
	}
	if len(acts) > 0 && lastErr != nil {
		return nil, lastErr
	}
	return map[string]any{"title": title, "items": rows, "source": "official_miyolive", "fetched_at_ms": time.Now().UnixMilli()}, nil
}
