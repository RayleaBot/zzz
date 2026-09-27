package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/gacha"
)

// A signal search link sent in a private chat is read as ZZZ-Plugin's
// gachaLog reads it: its authkey fetches the records from the official
// signal search history, and the UID comes from the records, so no account
// is needed. As ZZZ-Plugin's default, links are not taken in groups. The
// message's event moves to the background and fetches the whole history;
// the authkey is held in memory only meanwhile.

const (
	gachaLinkCN = "https://public-operation-common.mihoyo.com/common/gacha_record/api/getGachaLog"
	gachaLinkOS = "https://public-operation-common-sg.hoyoverse.com/common/gacha_record/api/getGachaLog"
	// gachaLinkGap is the wait between two pages of a link.
	gachaLinkGap = 300 * time.Millisecond
	// gachaLinkGroup is ZZZ-Plugin's reply to a link in a group.
	gachaLinkGroup = "当前群聊未开启链接刷新抽卡记录功能，请私聊发送"
)

// gachaLinkPools are ZZZ-Plugin's channels in its order: the request pool,
// its base type, which is also the archive pool, and its name.
var gachaLinkPools = [][3]string{{"3001", "3", "音擎频段"}, {"13001", "103", "音擎回响"}, {"2001", "2", "独家频段"}, {"12001", "102", "独家重映"}, {"1001", "1", "常驻频段"}, {"5001", "5", "邦布频段"}}

type gachaLink struct {
	key, region, biz string
}

// parseGachaLink reads authkey, region and game_biz from a signal search
// link, as ZZZ-Plugin's getQueryVariable; links of the other games are left
// to their plugins. ok is false when the text holds no Zenless link.
func parseGachaLink(text string) (link gachaLink, ok bool, err error) {
	if !strings.Contains(text, "authkey=") || !strings.Contains(text, "/nap/") && !strings.Contains(text, "nap_") {
		return gachaLink{}, false, nil
	}
	text = strings.TrimSpace(text)
	if _, after, found := strings.Cut(text, "?"); found {
		text = after
	}
	text, _, _ = strings.Cut(text, "#")
	params, _ := url.ParseQuery(text)
	link = gachaLink{key: params.Get("authkey"), region: params.Get("region"), biz: params.Get("game_biz")}
	if len(link.key) < 8 || len(link.key) > 16384 || strings.ContainsAny(link.key, " \r\n\t") {
		return gachaLink{}, true, gameError("gacha_link_invalid", "抽卡链接格式错误，请重新发起%抽卡链接")
	}
	if link.region == "" {
		link.region = "prod_gf_cn"
	}
	if link.biz == "" {
		link.biz = "nap_cn"
	}
	return link, true, nil
}

// gachaLinkPage reads one page of a channel with the link's authkey, in
// ZZZ-Plugin's getZZZGachaLink request shape.
func (a *App) gachaLinkPage(ctx context.Context, link gachaLink, pool, base, endID string, page int) (map[string]any, error) {
	endpoint := gachaLinkCN
	if link.biz == "nap_global" {
		endpoint = gachaLinkOS
	}
	query := url.Values{"authkey_ver": {"1"}, "sign_type": {"2"}, "auth_appid": {"webview_gacha"}, "init_log_gacha_type": {pool}, "init_log_gacha_base_type": {base}, "gacha_id": {"2c1f5692fdfbb733a08733f9eb69d32aed1d37"}, "timestamp": {strconv.FormatInt(time.Now().Unix(), 10)}, "lang": {"zh-cn"}, "device_type": {"mobile"}, "plat_type": {"ios"}, "region": {link.region}, "authkey": {link.key}, "game_biz": {link.biz}, "gacha_type": {pool}, "real_gacha_type": {base}, "page": {strconv.Itoa(page)}, "size": {"20"}, "end_id": {endID}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := a.LinkHTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, gameError("gacha_link_unavailable", "抽卡记录获取失败，请稍后重试")
	}
	defer response.Body.Close()
	var envelope struct {
		RetCode int            `json:"retcode"`
		Data    map[string]any `json:"data"`
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil || response.StatusCode != http.StatusOK || json.Unmarshal(body, &envelope) != nil || envelope.RetCode != 0 || envelope.Data == nil {
		return nil, gameError("gacha_link_invalid", "未查询到uid，请检查链接是否正确")
	}
	return envelope.Data, nil
}

// gachaLinkMessage answers a message holding a signal search link; handled
// is false for any other message.
func (a *App) gachaLinkMessage(ctx context.Context, event *rayleabot.EventContext) (handled bool, err error) {
	link, ok, err := parseGachaLink(event.Event.Message.PlainText)
	if !ok {
		return false, nil
	}
	if event.Event.Target.Type == "group" {
		config, configErr := a.Groups.Config(groupScope(event))
		if configErr == nil && config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
		return true, event.SendText(gachaLinkGroup)
	}
	if err != nil {
		return true, event.SendText(friendlyError(err))
	}
	if detached, err := detachChat(ctx, event); !detached {
		return true, err
	}
	notice(ctx, event, "抽卡链接解析成功，正在查询抽卡记录，可能耗费一段时间，请勿重复发送")
	uid := ""
	// The UID is that of the first record of any channel.
	for _, pool := range gachaLinkPools {
		data, err := a.gachaLinkPage(ctx, link, pool[0], pool[1], "0", 1)
		if err != nil {
			break
		}
		if list := asList(data["list"]); len(list) > 0 {
			uid = asText(asObject(list[0])["uid"])
			if region := asText(data["region"]); region != "" {
				link.region = region
			}
			break
		}
	}
	if uid == "" {
		return true, event.SendText("未查询到uid，请检查链接是否正确")
	}
	before := gachaPoolCounts(a.archiveOrEmpty(uid, link.region))
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{Link: true}, uid, link.region, false)
	if err != nil {
		return true, event.SendText(friendlyError(syncError(err)))
	}
	bases := map[string]string{}
	for _, pool := range gachaLinkPools {
		bases[pool[0]] = pool[1]
	}
	info, err = a.runSync(ctx, info, gachaLinkGap, func(ctx context.Context, pool, endID string, page int) (gacha.RemotePage, error) {
		data, err := a.gachaLinkPage(ctx, link, pool, bases[pool], endID, page)
		if err != nil {
			return gacha.RemotePage{}, err
		}
		return gacha.ParsePage(uid, link.region, pool, endID, data)
	}, nil)
	if err != nil {
		return true, event.SendText(friendlyError(syncError(err)))
	}
	return true, event.SendText(a.gachaReply(before, *info.Result))
}

// gachaReply is ZZZ-Plugin's reply after the records were read: the channels
// with the records each gained and holds.
func (a *App) gachaReply(before map[string]int, result gacha.ImportResult) string {
	archive, err := a.Gacha.Read(result.UID, result.Region)
	if err != nil {
		return friendlyError(err)
	}
	return gachaLinkSummary(before, gachaPoolCounts(archive))
}

// gachaLinkSummary lists every channel with its new and total records.
func gachaLinkSummary(before, after map[string]int) string {
	lines := []string{"抽卡记录更新成功，共" + strconv.Itoa(len(gachaLinkPools)) + "个卡池"}
	for _, pool := range gachaLinkPools {
		lines = append(lines, pool[2]+"新增"+strconv.Itoa(after[pool[1]]-before[pool[1]])+"条记录，一共"+strconv.Itoa(after[pool[1]])+"条记录")
	}
	return strings.Join(lines, "\n")
}

// gachaLinkCommand is ZZZ-Plugin's 抽卡链接: it asks for the link in private
// chat, and refuses groups as its default configuration does.
func (a *App) gachaLinkCommand(event *rayleabot.EventContext) error {
	if event.Event.Target.Type == "group" {
		return event.SendText(gachaLinkGroup)
	}
	return event.SendText("请发送抽卡链接，发送“取消”即可取消本次抽卡链接刷新")
}

// overseasUID is ZZZ-Plugin's test for an overseas UID.
var overseasUID = regexp.MustCompile(`^1[0-9][0-9]{8}`)

// gachaLinkGet is ZZZ-Plugin's 获取抽卡链接: in a private chat, the account
// plugin makes a link of the user's own mainland role.
func (a *App) gachaLinkGet(ctx context.Context, event *rayleabot.EventContext) error {
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil {
		return event.SendText(a.uidEmptyReply())
	}
	if overseasUID.MatchString(owner.UID) {
		return event.SendText("国际服不支持此功能")
	}
	if event.Event.Target.Type != "private" {
		return event.SendText("请私聊获取抽卡链接")
	}
	var link struct {
		URL string `json:"url"`
	}
	if !owner.Owned || a.accountClient(event).call(ctx, "gacha.link", map[string]any{"account_ref": owner.Choice.AccountRef, "role_ref": owner.Choice.RoleRef}, &link) != nil || link.URL == "" {
		return event.SendText("authKey获取失败，请检查cookie是否过期")
	}
	return event.SendText(link.URL)
}

// gachaPoolCounts counts an archive's records by channel.
func gachaPoolCounts(archive gacha.Archive) map[string]int {
	counts := map[string]int{}
	for _, record := range archive.Records {
		counts[record.GachaType]++
	}
	return counts
}

func (a *App) archiveOrEmpty(uid, region string) gacha.Archive {
	archive, _ := a.Gacha.Read(uid, region)
	return archive
}

// chatArchive is the archive a record command reads: as upstream, the log of
// the UID named or of the UID in use, bound with or without an account.
func (a *App) chatArchive(ctx context.Context, event *rayleabot.EventContext, uid string) (gacha.Archive, Role, error) {
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return gacha.Archive{}, Role{}, err
	}
	role := owner.Role
	if !owner.Owned {
		role = Role{Game: a.Game.ID, UID: owner.UID}
		if summaries, err := a.Gacha.List(); err == nil {
			for _, summary := range summaries {
				if summary.UID == owner.UID {
					role.Region = summary.Region
					break
				}
			}
		}
	}
	archive, err := a.Gacha.Read(role.UID, role.Region)
	if err != nil {
		return gacha.Archive{}, role, gameError("archive_missing", "未查询到抽卡记录，请先发送抽卡链接或"+a.Game.Prefix+"更新抽卡记录")
	}
	return archive, role, nil
}

// syncError gives the sync engine's failures their public codes.
func syncError(err error) error {
	switch {
	case errors.Is(err, gacha.ErrSync):
		return gameError("sync_expired", "同步任务已取消或过期，请重新开始。")
	case errors.Is(err, gacha.ErrConflict):
		return gameError("sync_conflict", "档案已变更或记录冲突，请重新开始同步；现有档案已保留。")
	case errors.Is(err, gacha.ErrInvalid):
		return gameError("sync_invalid", "官方记录结构或分页异常，现有档案已保留。")
	}
	return err
}
