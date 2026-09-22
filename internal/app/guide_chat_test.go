package app

import (
	"testing"
)

func guidePost(t *testing.T, raw string) map[string]any {
	t.Helper()
	return jsonObject(t, raw)
}

func TestGuidePostImageSkipsTagsAndGIFs(t *testing.T) {
	// The largest file, the later one on a tie.
	post := guidePost(t, `{"post":{"subject":"星见雅攻略"},"image_list":[{"url":"https://upload-bbs.miyoushe.com/a.png","size":"10"},{"url":"https://upload-bbs.miyoushe.com/b.png","size":"30"},{"url":"https://upload-bbs.miyoushe.com/c.png","size":"30"}]}`)
	if got := guidePostImage("星见雅", post); got != "https://upload-bbs.miyoushe.com/c.png" {
		t.Fatal(got)
	}
	if got := guidePostImage("艾莲", post); got != "" {
		t.Fatal("another character's post matched")
	}
	// 【…本…】 tags and GIFs are ignored.
	post = guidePost(t, `{"post":{"subject":"【2.0版本】星见雅攻略"},"image_list":[{"url":"https://upload-bbs.miyoushe.com/a.gif","size":"99","format":"gif"},{"url":"https://upload-bbs.miyoushe.com/b.png","size":"5","format":"png"}]}`)
	if got := guidePostImage("星见雅", post); got != "https://upload-bbs.miyoushe.com/b.png" {
		t.Fatal(got)
	}
	post = guidePost(t, `{"post":{"subject":"【本期深渊】星见雅"},"image_list":[{"url":"https://upload-bbs.miyoushe.com/b.png","size":"5"}]}`)
	if got := guidePostImage("本期", post); got != "" {
		t.Fatal("a book tag matched")
	}
}
