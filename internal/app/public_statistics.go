package app

import (
	"context"
	"net/url"
	"time"
)

func (c PublicContentClient) estimate(ctx context.Context) (map[string]any, error) {
	keyword := "菲林统计汇总"
	data, err := c.get(ctx, "https://bbs-api.miyoushe.com/painter/api/user_instant/search/list?uid=137101761&size=20&offset=0&sort_type=2&keyword="+url.QueryEscape(keyword), nil)
	if err != nil {
		return nil, err
	}
	rows := []PublicPost{}
	for _, v := range asList(data["list"]) {
		m := asObject(v)
		if post := asObject(m["post"]); post != nil {
			m = post
		}
		p, err := normalizePublicPost(m, false)
		if err != nil {
			return nil, err
		}
		rows = append(rows, p)
	}
	return map[string]any{"items": rows, "more": false, "source": "米游社作者137101761", "fetched_at_ms": time.Now().UnixMilli(), "note": "内容为作者的预估或盘点，请核对原文版本与发布时间。"}, nil
}
