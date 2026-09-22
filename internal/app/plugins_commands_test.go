package app

import (
	"encoding/json"
	"maps"
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
		"genshin": {
			{"雷神面板", "character", []string{"雷神"}},
			{"面板", "panel-list", nil},
			{"面板列表100000001", "panel-list", []string{"100000001"}},
			{"更新面板", "panel-refresh", nil},
			{"更新面板100000001", "panel-refresh", []string{"100000001"}},
			{"米游社更新面板", "panel-refresh-account", nil},
			{"删除面板100000001", "panel-delete", []string{"100000001"}},
			{"圣遗物列表", "artifact-list", nil},
			{"今日素材", "daily-material", nil},
			{"雷神换90级5精护摩换绝缘4", "panel-change", nil},
			{"月谕圣牌交换", "role-cards-exchange", nil},
			{"周三材料", "daily-material", nil},
			{"胡桃材料", "materials", []string{"胡桃"}},
			{"胡桃圣遗物", "score", []string{"胡桃"}},
			{"胡桃圣遗物", "score", []string{"胡桃"}},
			{"胡桃伤害2", "build", []string{"胡桃"}},
			{"刻晴养成", "growth", []string{"刻晴"}},
			{"6.7卡池", "calendar", []string{"6.7"}},
			{"卡池", "calendar", nil},
			{"胡桃卡池", "banner-history", []string{"胡桃"}},
			{"上期深渊", "abyss", []string{"上期"}},
			{"原石7月", "monthly", []string{"7"}},
			{"角色3", "profile", nil},
			{"今日五星天赋统计", "talent-stat", nil},
			{"胡桃天赋", "talent-wiki", []string{"胡桃"}},
			{"夜兰命座", "talent-wiki", []string{"夜兰"}},
			{"角色", "characters", nil},
			{"七圣召唤查询卡组3", "tcg_decks", nil},
			{"七圣查询行动牌", "tcg_cards", nil},
			{"原石统计", "monthly-history", nil},
			{"十连武器", "simulation", nil},
			{"武器十连", "simulation", nil},
			{"十连2", "simulation", nil},
			{"单抽", "simulation", nil},
			{"抽卡", "simulation", nil},
			{"抽卡记录", "gacha", nil},
			{"定轨护摩", "simulation-fate", []string{"护摩"}},
			{"绑定100000001", "select", []string{"100000001"}},
			{"绑定uid100000001", "select", []string{"100000001"}},
			{"uid", "accounts", nil},
			{"UID2", "accounts", nil},
			{"删除uid1", "uid-remove", []string{"1"}},
			{"兑换码", "codes", nil},
			{"兑换码使用ABC123", "redeem", []string{"ABC123"}},
			{"米游社原神签到", "community-sign", nil},
			{"米游社七七", "search", []string{"七七"}},
			{"甜甜花在哪里", "map", []string{"甜甜花"}},
			{"尘歌壶模数123456789012", "blueprint", []string{"123456789012"}},
			{"老婆设置心海", "interaction", []string{"老婆", "设置", "心海"}},
			{"老婆照片", "interaction", []string{"老婆", "照片"}},
			{"心海照片", "photo", []string{"心海"}},
			{"心海图鉴", "catalog", []string{"心海"}},
			{"胡桃排名", "rank", nil},
			{"总排名", "cloud-total-rank", nil},
			{"角色排名胡桃100000001", "cloud-character-rank", []string{"胡桃100000001"}},
			{"胡桃排名统计", "cloud-rank-stats", []string{"胡桃"}},
			{"导出面板数据", "cloud-export", nil},
			{"导入面板数据100000001", "cloud-import", []string{"100000001"}},
			{"胡桃圣遗物排行榜", "rank", nil},
			{"群排名", "rank", nil},
			{"最强胡桃", "rank-top", nil},
			{"最高分排行", "rank-top", nil},
			{"重置胡桃排名", "rank-reset", nil},
			{"刷新排名", "rank-refresh", nil},
			{"关闭群排名", "rank-switch", nil},
			{"挑战排行", "challenge-rank", nil},
			{"100000001", "public-profile", []string{"100000001"}},
			{"开启公告推送", "subscribe", []string{"公告"}},
			{"2025年札记统计", "monthly-history", nil},
			{"原神帮助", "help", nil},
			{"刷新天赋", "panel-refresh-account", nil},
			{"强制更新所有天赋", "panel-refresh-account", nil},
			{"月谕卡牌换牌", "role-cards-exchange", nil},
			{"幻想卡片收集", "role-cards", nil},
			{"角色养成", "growth-help", nil},
			{"刻晴养成81", "growth", []string{"刻晴", "81"}},
			{"尘歌壶模数养成", "blueprint", nil},
			{"抽奖记录", "gacha", nil},
			{"武器池记录", "gacha", nil},
			{"角色统计", "gacha-versions", nil},
			{"绑定uid+100000001", "select", []string{"100000001"}},
			{"喵喵更新图像", "artwork", nil},
			{"安卓帮助", "gacha-help", nil},
			{"卡池帮助", "pool-help", nil},
			{"面板帮助", "panel-help", nil},
			{"更换面板帮助", "panel-help", nil},
			{"公告", "news", nil},
			{"原神公告3", "news", []string{"3"}},
			{"公告列表", "news", []string{"列表"}},
			{"官方资讯2", "info", []string{"2"}},
			{"活动列表", "events", []string{"列表"}},
			{"活动日历", "live-calendar", nil},
			{"原石预估", "estimate", nil},
			{"盘点", "estimate", nil},
			{"喵喵别名", "alias-help", nil},
			{"喵喵别名原神设置", "alias-set", nil},
			{"喵喵别名删除", "alias-remove", nil},
			{"喵喵别名列表", "alias-list", nil},
		},
		"starrail": {
			{"希儿面板", "character", []string{"希儿"}},
			{"面板", "panel-list", nil},
			{"更新面板", "panel-refresh", nil},
			{"mys更新面板", "panel-refresh-account", nil},
			{"遗器列表", "artifact-list", nil},
			{"希儿换满命换满行迹", "talent-wiki", []string{"希儿换满命换满"}},
			{"希儿换满命", "panel-change", nil},
			{"圣遗物列表100000001", "artifact-list", []string{"100000001"}},
			{"希儿遗器", "score", []string{"希儿"}},
			{"希儿伤害", "build", []string{"希儿"}},
			{"上期忘却之庭", "challenge", []string{"上期"}},
			{"上期模拟宇宙", "rogue", []string{"上期"}},
			{"差分宇宙", "divergent", nil},
			{"常规演算二", "divergent-normal", nil},
			{"上周差分宇宙记录", "divergent-week", nil},
			{"周期演算", "divergent-cycle", nil},
			{"星琼统计", "monthly-history", nil},
			{"希儿行迹", "talent-wiki", []string{"希儿"}},
			{"希儿遗器排名", "rank", nil},
			{"最强希儿", "rank-top", nil},
			{"今年星琼统计", "monthly-history", nil},
			{"跃迁记录", "gacha", nil},
			{"十连光锥", "simulation", nil},
			{"更新跃迁记录", "gacha-background", nil},
			{"帮助", "help", nil},
			{"刷新行迹", "panel-refresh-account", nil},
			{"角色分析", "gacha", nil},
			{"光锥记录", "gacha", nil},
			{"抽卡统计", "gacha", nil},
			{"版本统计", "gacha-versions", nil},
			{"up统计", "gacha-versions", nil},
			{"更新图像", "artwork", nil},
			{"克拉拉养成75", "growth", []string{"克拉拉", "75"}},
			{"抽卡链接", "gacha-link", nil},
			{"希儿参考面板", "ref-panel", []string{"希儿"}},
			{"参考面板帮助", "ref-panel-help", nil},
			{"攻略", "path-guides", nil},
			{"希儿攻略", "guides", []string{"希儿"}},
			{"深渊攻略", "abyss-guide", nil},
			{"强度榜", "tier-list", nil},
			{"面板帮助", "panel-help", nil},
			{"预估", "estimate", nil},
			{"星琼盘点", "estimate", nil},
			{"公告2", "news", []string{"2"}},
			{"喵喵别名星铁设置", "alias-set", nil},
			{"设置希儿别名", "alias-add", []string{"希儿"}},
			{"删除别名鸭鸭", "alias-remove", []string{"鸭鸭"}},
			{"喵喵别名删除", "alias-remove", nil},
			{"希儿昵称", "aliases", []string{"希儿"}},
			{"喵喵别名帮助", "alias-help", nil},
		},
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
			{"下载全部资源", "artwork", nil},
			{"获取抽卡链接", "gacha-link", nil},
			{"显示排名", "query-rank-switch", nil},
			{"苗圃", "hollow_zero", nil},
			{"抽卡", "gacha", nil},
			{"帮助", "help", nil},
			{"菲林预估", "estimate", nil},
			{"资讯列表", "info", []string{"列表"}},
			{"添加艾莲别名鲨鲨", "alias-set", []string{"艾莲", "鲨鲨"}},
			{"删除别名鲨鲨", "alias-remove", []string{"鲨鲨"}},
			{"艾莲别名", "aliases", []string{"艾莲"}},
		},
	}
	for game, list := range cases {
		manifest, err := pluginmeta.Read(pluginFile(t, game, "info.json"))
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
	for _, game := range []string{"genshin", "starrail", "zzz"} {
		manifest, err := pluginmeta.Read(pluginFile(t, game, "info.json"))
		if err != nil {
			t.Fatal(err)
		}
		var data Game
		if err := json.Unmarshal(pluginFile(t, game, "internal/assets/game.json"), &data); err != nil {
			t.Fatal(err)
		}
		ids := slices.AppendSeq(slices.Collect(maps.Keys(data.Hints)), maps.Keys(data.Pictures.Static))
		for _, id := range ids {
			if !slices.ContainsFunc(manifest.Commands, func(command pluginmeta.Command) bool { return command.ID == id }) {
				t.Errorf("%s answer %s names no command", game, id)
			}
		}
	}
}

// ZZZ-Plugin names the tower seasons 爬塔S1–S4; each word picks its own
// season and nothing else.
func TestShippedTowerRankingsFollowTheirSeason(t *testing.T) {
	a := &App{Game: Game{}}
	if err := json.Unmarshal(pluginFile(t, "zzz", "internal/assets/game.json"), &a.Game); err != nil {
		t.Fatal(err)
	}
	for word, name := range map[string]string{"爬塔S1排名": "爬塔S1", "爬塔s2排名": "爬塔S2", "爬塔S3群排名": "爬塔S3", "爬塔 S4排名": "爬塔S4", "拟真鏖战试炼排名": "爬塔S1", "鏖战试炼：末路排名": "爬塔S2"} {
		if rank, ok := a.queryRankType(word); !ok || rank.Name != name {
			t.Errorf("%s: %+v %v", word, rank, ok)
		}
	}
	if rank, ok := a.queryRankType("爬塔1排名"); ok {
		t.Errorf("a season without S matched %s", rank.Name)
	}
}
