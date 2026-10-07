package images

import (
	"github.com/RayleaBot/zzz/internal/app"
)

// uidListArtwork maps the images named in the converted stylesheets (miao's
// common for the layout, the Yunzai 原神插件's html/user/uid-list) to their
// sources and paths.
var uidListArtwork = [][3]string{
	{"Number", "miao-plugin", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "miao-plugin", "resources/common/font/NZBZ.woff"},
	{"YS", "miao-plugin", "resources/common/font/HYWH-65W.woff"},
	{"common-bg-bg-hydro", "miao-plugin", "resources/common/bg/bg-hydro.webp"},
	{"img-icon-check", "yunzai-genshin", "resources/img/icon/check.webp"},
}

// UIDList draws 我的uid the way the Yunzai 原神插件's html/user/uid-list
// does, for 绝区零: each UID with its number and CK or bound tag, the one in
// use marked, and the player's name and level with miao's common face and the 绝区零 banner.
func UIDList(context app.ImageContext, list app.UIDListImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range uidListArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	uids := []any{}
	for index, entry := range list.Entries {
		// Upstream shows every 绝区零 UID with miao's common face and its own
		// banner.
		face := resources.Artwork("face-common", "miao-plugin", "resources/common/item/face.webp")
		banner := resources.Artwork("banner-common", "yunzai-genshin", "resources/ZZZero/img/other/banner.png")
		kind := "reg"
		if entry.Account {
			kind = "ck"
		}
		item := map[string]any{"index": index + 1, "uid": entry.UID, "type": kind, "active": entry.Active, "face": face, "banner": banner}
		if entry.Nickname != "" && entry.Level > 0 {
			item["name"], item["level"] = entry.Nickname, entry.Level
		}
		uids = append(uids, item)
	}
	return app.Image{Template: "uid-list", Data: map[string]any{"mark": context.Game.Prefix, "name": "绝区零", "no_info": "暂无uid信息", "uids": uids}, Resources: resources.List}, true
}
