package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
)

// Query ranks follow ZZZ-Plugin's group rankings: a member who queries one of
// the game's challenge records in a group joins that group's ranking for it,
// the record is kept under the UID, and the ranking draws every current
// member's latest record. Members hide or show themselves per ranking.

// QueryRankType is one ranking in game.json: the ID members show or hide
// themselves under, its name, the builder's page, the operation whose queries
// feed it, the pattern that names it in commands, and the reply when nothing
// ranks ({prefix} is the reply prefix), and the reply while group rankings
// are off. Rankings sharing an ID share the choice, as upstream's 危局 and
// 绝境 do; the first that matches a word wins.
type QueryRankType struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Page      string `json:"page"`
	Operation string `json:"operation"`
	Words     string `json:"words"`
	Empty     string `json:"empty"`
	Closed    string `json:"closed"`
}

// QueryRankRecord is a UID's latest record for a ranking's operation: the
// member who queried it, the role and the official data. Summary is the
// builder's one-line result for text replies.
type QueryRankRecord struct {
	UID       string         `json:"uid"`
	ActorID   string         `json:"actor_id"`
	Role      Role           `json:"role"`
	Data      map[string]any `json:"data"`
	SavedAtMS int64          `json:"saved_at_ms"`
	Avatar    string         `json:"-"`
	Summary   string         `json:"-"`
}

// QueryRankMember is a UID in a group's ranking and whether its member hid it.
type QueryRankMember struct {
	ActorID string `json:"actor_id"`
	Hidden  bool   `json:"hidden,omitempty"`
}

// QueryRankImage is what a query ranking draws on: the ranking and the
// records of the group's visible members.
type QueryRankImage struct {
	Type    QueryRankType
	Records []QueryRankRecord
}

// QueryRankImageBuilder filters, orders and draws a ranking as upstream does.
// It returns the ranked records, each with its Summary, for the text reply,
// and false when none ranks.
type QueryRankImageBuilder func(ImageContext, QueryRankImage) (Image, []QueryRankRecord, bool)

// QueryRankStore keeps each UID's latest record per operation, as upstream
// keeps one per UID for every group.
type QueryRankStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *QueryRankStore) file(operation, uid string) string {
	sum := sha256.Sum256([]byte(operation + "\x00" + uid))
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}

func (s *QueryRankStore) Save(operation string, record QueryRankRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return localdata.Write(s.file(operation, record.UID), record)
}

func (s *QueryRankStore) Read(operation, uid string) (QueryRankRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var record QueryRankRecord
	if err := localdata.Read(s.file(operation, uid), &record); err != nil || record.UID != uid {
		return QueryRankRecord{}, false
	}
	return record, true
}

// recordQueryRank keeps a queried record and, in a group, adds its UID to the
// group's rankings fed by the operation, keeping a member's earlier choice to
// hide. It reports whether the UID shows in all of those rankings, or nil
// outside a group, for the record image's closing note.
func (a *App) recordQueryRank(event *rayleabot.EventContext, operation string, result QueryResult) *bool {
	types := []string{}
	for _, rank := range a.Game.QueryRanks {
		if rank.Operation == operation && !slices.Contains(types, rank.ID) {
			types = append(types, rank.ID)
		}
	}
	uid := result.Role.UID
	if len(types) == 0 || uid == "" || event.Event.Actor.ID == "" {
		return nil
	}
	// As upstream, records are kept only while group rankings are on; the
	// member still joins the group's rankings.
	if settings(event).GroupRank {
		_ = a.QueryRanks.Save(operation, QueryRankRecord{UID: uid, ActorID: event.Event.Actor.ID, Role: result.Role, Data: result.Data, SavedAtMS: time.Now().UnixMilli()})
	}
	scope := groupScope(event)
	if event.Event.EventType != "message.group" || !scope.valid() {
		return nil
	}
	shown := true
	err := a.Groups.Update(scope, func(data *GroupData) error {
		if data.QueryRanks == nil {
			data.QueryRanks = map[string]map[string]QueryRankMember{}
		}
		for _, id := range types {
			if data.QueryRanks[id] == nil {
				data.QueryRanks[id] = map[string]QueryRankMember{}
			}
			member := data.QueryRanks[id][uid]
			member.ActorID = event.Event.Actor.ID
			data.QueryRanks[id][uid] = member
			shown = shown && !member.Hidden
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return &shown
}

// queryRankType finds the ranking a command word names.
func (a *App) queryRankType(word string) (QueryRankType, bool) {
	for _, rank := range a.Game.QueryRanks {
		if pattern, err := regexp.Compile(rank.Words); err == nil && rank.Words != "" && pattern.MatchString(word) {
			return rank, true
		}
	}
	return QueryRankType{}, false
}

// queryRankCommand answers a query ranking in a group, or with 显示/隐藏
// shows or hides the requester's UID in one ranking or all of them.
func (a *App) queryRankCommand(ctx context.Context, event *rayleabot.EventContext, command string) error {
	if command == "group-rank-switch" {
		// ZZZ-Plugin's switch is global, whatever mode the command names.
		enable := !queryRankHide.MatchString(event.Event.Command())
		if _, err := event.Actions().ConfigWrite(ctx, map[string]any{"group_rank_enabled": enable}); err != nil {
			return event.SendText(friendlyError(err))
		}
		state := "开启"
		if !enable {
			state = "关闭"
		}
		return event.SendText(a.Game.Name + "群内深渊排名功能已设置为: " + state)
	}
	scope := groupScope(event)
	if event.Event.EventType != "message.group" || !scope.valid() {
		return event.SendText("请在群聊中使用该命令！")
	}
	word := event.Event.Command()
	if command == "query-rank-switch" {
		return a.queryRankSwitch(ctx, event, scope, word)
	}
	rank, ok := a.queryRankType(word)
	if !ok || a.queryRankImage == nil {
		return event.Result(map[string]any{"handled": false})
	}
	if command == "query-rank-reset" {
		// ZZZ-Plugin's 重置排名 means to clear the group's ranking of the
		// mode, but deletes a key its members are not kept under, so nothing
		// goes; this clears it. 绝境 shares 危局's ranking, as upstream's.
		if err := a.Groups.Update(scope, func(data *GroupData) error {
			delete(data.QueryRanks, rank.ID)
			return nil
		}); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText("清除" + rank.Name + "排名成功！")
	}
	if !settings(event).GroupRank {
		return event.SendText(rank.Closed)
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	members := data.QueryRanks[rank.ID]
	// Upstream drops the UIDs of members who have left the group.
	if present, ok := a.groupMembers(ctx, event, scope); ok {
		stale := []string{}
		for uid, member := range members {
			if !present[member.ActorID] {
				stale = append(stale, uid)
			}
		}
		if len(stale) > 0 {
			_ = a.Groups.Update(scope, func(data *GroupData) error {
				for _, byUID := range data.QueryRanks {
					for _, uid := range stale {
						delete(byUID, uid)
					}
				}
				return nil
			})
			for _, uid := range stale {
				delete(members, uid)
			}
		}
	}
	records := []QueryRankRecord{}
	for uid, member := range members {
		if member.Hidden {
			continue
		}
		if record, ok := a.QueryRanks.Read(rank.Operation, uid); ok {
			if scope.Protocol == "onebot11" {
				record.Avatar = "https://q1.qlogo.cn/g?b=qq&nk=" + record.ActorID + "&s=100"
			}
			records = append(records, record)
		}
	}
	slices.SortFunc(records, func(x, y QueryRankRecord) int { return strings.Compare(x.UID, y.UID) })
	image, ranked, ok := a.queryRankImage(a.imageContext(ctx), QueryRankImage{Type: rank, Records: records})
	if !ok {
		return event.SendText(strings.ReplaceAll(rank.Empty, "{prefix}", a.Game.Prefix))
	}
	view := View{Title: rank.Name + "排名", Rows: []Row{}}
	for index, record := range ranked {
		view.Rows = append(view.Rows, Row{Label: strconv.Itoa(index+1) + ". " + record.Role.Nickname + " · UID " + record.UID, Value: record.Summary})
	}
	view.Image = &image
	return a.sendView(ctx, event, view)
}

// groupMembers lists the group's members, when the protocol can.
func (a *App) groupMembers(ctx context.Context, event *rayleabot.EventContext, scope GroupScope) (map[string]bool, bool) {
	if scope.Protocol != "onebot11" {
		return nil, false
	}
	result, err := event.Actions().GroupMemberList(ctx, scope.GroupID)
	if err != nil {
		return nil, false
	}
	list, _ := result["data"].([]any)
	if list == nil {
		return nil, false
	}
	present := map[string]bool{}
	for _, item := range list {
		if id := asText(asObject(item)["user_id"]); id != "" {
			present[id] = true
		}
	}
	return present, len(present) > 0
}

var (
	queryRankShow = regexp.MustCompile(`(?i)显示|展示|开启|打开|on|启用|启动`)
	queryRankHide = regexp.MustCompile(`(?i)隐藏|取消显示|关闭|关掉|off|禁用|停止`)
)

// queryRankSwitch is ZZZ-Plugin's 显示/隐藏排名 for the requester's bound UID.
func (a *App) queryRankSwitch(ctx context.Context, event *rayleabot.EventContext, scope GroupScope, word string) error {
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	_, role, err := Choose(listed, a.Game.ID, "")
	if err != nil || role.UID == "" {
		return event.SendText("未绑定UID，请先绑定")
	}
	// 取消显示 hides, so hiding words are checked first.
	hide := queryRankHide.MatchString(word)
	if !hide && !queryRankShow.MatchString(word) {
		return event.SendText("请输入\"显示\"或\"隐藏\"来设置是否显示个人的深渊排名")
	}
	ids, names := []string{}, []string{}
	if rank, ok := a.queryRankType(word); ok {
		ids, names = append(ids, rank.ID), append(names, rank.Name)
	} else {
		for _, rank := range a.Game.QueryRanks {
			if !slices.Contains(ids, rank.ID) {
				ids, names = append(ids, rank.ID), append(names, rank.Name)
			}
		}
	}
	err = a.Groups.Update(scope, func(data *GroupData) error {
		if data.QueryRanks == nil {
			data.QueryRanks = map[string]map[string]QueryRankMember{}
		}
		for _, id := range ids {
			if data.QueryRanks[id] == nil {
				data.QueryRanks[id] = map[string]QueryRankMember{}
			}
			member := data.QueryRanks[id][role.UID]
			member.ActorID, member.Hidden = event.Event.Actor.ID, hide
			data.QueryRanks[id][role.UID] = member
		}
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	state := "显示"
	if hide {
		state = "隐藏"
	}
	return event.SendText(a.Game.Name + " UID: " + role.UID + "，" + strings.Join(names, "、") + "排名功能已设置为: " + state)
}
