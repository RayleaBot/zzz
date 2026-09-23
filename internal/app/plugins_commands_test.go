package app

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/pluginmeta"
)

// The shipped manifests follow upstream wording; these cases pin the words
// whose patterns overlap, so a reordered manifest cannot steal a command.
func TestShippedManifestsResolveUpstreamWording(t *testing.T) {
	cases := map[string][]struct {
		word, id string
		args     []string
	}{
		"zzz": {
			{"艾莲面板", "character", []string{"艾莲"}},
			{"面板", "panel-list", nil},
			{"面板列表", "panel-list", nil},
			{"更新面板", "panel-refresh", nil},
			{"更新展柜面板", "panel-refresh", nil},
			{"面板刷新", "panel-refresh", nil},
			{"上期式舆防卫战", "challenge", []string{"上期"}},
			{"艾莲伤害", "build", []string{"艾莲"}},
			{"2.0卡池", "banner-current", []string{"2.0"}},
			{"卡池", "banner-current", nil},
			{"艾莲复刻记录", "banner-history", []string{"艾莲"}},
			{"迷宫记录", "zenkov_detail", nil},
			{"零号空洞", "hollow-zero", nil},
			{"深渊排名", "query-rank", nil},
			{"艾莲技能", "talent-wiki", []string{"艾莲"}},
			{"艾莲技能12.12.10.12.12.6", "talent-wiki", []string{"艾莲"}},
			{"艾莲影画", "talent-wiki", []string{"艾莲"}},
			{"危局绝境排名", "query-rank", nil},
			{"爬塔S2排名", "query-rank", nil},
			{"鏖战试炼：荣耀排名", "query-rank", nil},
			{"爬塔排名", "query-rank-help", nil},
			{"隐藏深渊排名", "query-rank-switch", nil},
			{"下载全部资源", "artwork-all", nil},
			{"删除所有资源", "artwork-delete", nil},
			{"更新素材", "artwork", nil},
			{"获取抽卡链接", "gacha-link-get", nil},
			{"显示排名", "query-rank-switch", nil},
			{"苗圃", "hollow_zero", nil},
			{"抽卡", "gacha", nil},
			{"帮助", "help", nil},
			{"菲林预估", "estimate", nil},
			{"资讯列表", "info", []string{"列表"}},
			{"添加艾莲别名鲨鲨", "alias-set", []string{"艾莲", "鲨鲨"}},
			{"删除别名鲨鲨", "alias-remove", []string{"鲨鲨"}},
			{"艾莲别名", "aliases", []string{"艾莲"}},
			{"开启挑战提醒", "challenge-enable", nil},
			{"关闭挑战提醒", "challenge-stop", nil},
			{"设置式舆阈值6", "challenge-threshold", []string{"式舆", "6"}},
			{"设置防卫战阈值", "challenge-threshold", []string{"防卫战"}},
			{"设置全局危局阈值9", "challenge-global-threshold", []string{"危局", "9"}},
			{"设置个人提醒时间每周六20时10分", "challenge-time", []string{"每周六20时10分"}},
			{"设置全局提醒时间每日20时", "challenge-global-time", []string{"每日20时"}},
			{"关闭全局挑战提醒", "challenge-global-switch", []string{"关闭"}},
			{"个人提醒时间", "challenge-time-status", nil},
			{"上传艾莲面板图", "panel-image-upload", []string{"艾莲"}},
			{"查看艾莲角色图2", "panel-image-list", []string{"艾莲", "2"}},
			{"删除艾莲面板图1,2", "panel-image-remove", []string{"艾莲", "1,2"}},
			{"开启群内式舆排名", "group-rank-switch", nil},
			{"关闭群危局排名", "group-rank-switch", nil},
			{"开启深渊群排名", "query-rank-switch", nil},
			{"重置深渊排名", "query-rank-reset", nil},
			{"清空爬塔S2排名", "query-rank-reset", nil},
			{"设置默认设备", "device-default", nil},
		},
	}
	for game, list := range cases {
		manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
		if err != nil {
			t.Fatal(err)
		}
		set, err := newCommandSet(manifest)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range list {
			id, args, ok := set.resolve(tc.word, nil)
			if len(args) == 0 {
				args = nil
			}
			if !ok || id != tc.id || !reflect.DeepEqual(args, tc.args) {
				t.Errorf("%s %s resolved to %q %v %v, want %s %v", game, tc.word, id, args, ok, tc.id, tc.args)
			}
		}
	}
}

// A hint or static picture answers a command by ID, so each must name one
// the manifest has.
func TestShippedHintsNameManifestCommands(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &data); err != nil {
		t.Fatal(err)
	}
	for id := range data.Hints {
		if !slices.ContainsFunc(manifest.Commands, func(command pluginmeta.Command) bool { return command.ID == id }) {
			t.Errorf("answer %s names no command", id)
		}
	}
}
