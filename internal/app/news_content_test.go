package app

import (
	"strings"
	"testing"
)

func TestNewsContentKeepsOnlyStyledMarkupAndResources(t *testing.T) {
	pictures := []string{"https://upload-bbs.miyoushe.com/a.png", "https://upload-bbs.miyoushe.com/b.png"}
	emoticons := map[string]string{"派蒙-开心": "https://bbs-static.miyoushe.com/e.png"}
	requested := []string{}
	resource := func(address string) string {
		if !strings.HasPrefix(address, "https://") {
			return ""
		}
		requested = append(requested, address)
		return "url-" + string(rune('0'+len(requested)-1))
	}
	content := `<div class="ql-image"><div class="ql-image-box"><img src="https://upload-bbs.miyoushe.com/a.png?x=1"></div></div>` +
		`<div class="ql-image"><div class="ql-image-box"><img style="opacity: 0;"></div><div class="ql-image-mask">超链图片</div></div>` +
		`<p onclick="x()" class="ql-align-center bad\"class"><span style="color: rgb(47, 50, 56); background-image: url(https://evil.example/x.png); font-size: 1em">a &lt;b&gt; _(派蒙-开心)_(未知)</span></p>` +
		`<script>alert(1)</script><iframe src="https://evil.example"></iframe><a href="https://evil.example">链接</a><custom>保留</custom><br>`
	got := newsContent(content, pictures, emoticons, resource)
	want := `<div class="ql-image"><div class="ql-image-box"><img data-render-resource="url-0"></div></div>` +
		`<div class="ql-image"><div class="ql-image-box"><img data-render-resource="url-1"></div><div class="ql-image-mask">超链图片</div></div>` +
		`<p class="ql-align-center"><span style="color: rgb(47, 50, 56); font-size: 1em">a &lt;b&gt; <img class="emoticon-image" data-render-resource="url-2"></span></p>` +
		`<a>链接</a>保留<br>`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	// The linked picture takes the structured content's second picture.
	if len(requested) != 3 || requested[1] != pictures[1] {
		t.Fatal(requested)
	}
}

func TestNewsStructuredPicturesFollowImageInserts(t *testing.T) {
	structured := `[{"insert":"文字"},{"insert":{"image":"https://upload-bbs.miyoushe.com/a.png"}},{"insert":{"divider":"line_1"}},{"insert":{"image":"https://upload-bbs.miyoushe.com/b.png"}}]`
	if got := newsStructuredPictures(structured); len(got) != 2 || got[1] != "https://upload-bbs.miyoushe.com/b.png" {
		t.Fatal(got)
	}
	if got := newsStructuredPictures(""); len(got) != 0 {
		t.Fatal(got)
	}
}
