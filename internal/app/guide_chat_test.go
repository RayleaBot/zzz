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

func TestGuideSettingsKeepAllAndItsCount(t *testing.T) {
	store := &GuideSettings{Path: t.TempDir() + "/guides.json"}
	out, err := store.Manage("guides.configure", map[string]any{"revision": 0, "default_source": "0", "forward_count": 7})
	if err != nil {
		t.Fatal(err)
	}
	config := out["settings"].(GuideConfig)
	if config.Default != "0" || config.forwardCount() != 7 {
		t.Fatalf("settings = %+v", config)
	}
	// The management page saves a source without touching the count.
	out, err = store.Manage("guides.configure", map[string]any{"revision": config.Revision, "default_source": "2"})
	if config = out["settings"].(GuideConfig); err != nil || config.Default != "2" || config.forwardCount() != 7 {
		t.Fatalf("settings = %+v %v", config, err)
	}
	for _, input := range []map[string]any{{"revision": config.Revision, "default_source": "8"}, {"revision": config.Revision, "default_source": "1", "forward_count": 8}} {
		if _, err := store.Manage("guides.configure", input); err == nil {
			t.Errorf("%v accepted", input)
		}
	}
	if (GuideConfig{}).forwardCount() != 4 {
		t.Error("upstream's default count is 4")
	}
}
