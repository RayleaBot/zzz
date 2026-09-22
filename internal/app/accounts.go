package app

import (
	"context"
	"errors"
	"slices"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type Subject struct {
	SourceProtocol string `json:"source_protocol"`
	SourceAdapter  string `json:"source_adapter"`
	BotID          string `json:"bot_id"`
	ActorID        string `json:"actor_id"`
}
type Role struct {
	Ref      string `json:"ref"`
	Game     string `json:"game"`
	GameBiz  string `json:"game_biz"`
	UID      string `json:"uid"`
	Region   string `json:"region"`
	Nickname string `json:"nickname"`
	Level    int    `json:"level"`
}
type Account struct {
	CloudConfigured []string `json:"cloud_configured"`
	Ref             string   `json:"ref"`
	Owner           Subject  `json:"owner"`
	AccountLabel    string   `json:"account_label"`
	Roles           []Role   `json:"roles"`
	Status          string   `json:"status"`
}
type Selection struct {
	AccountRef string `json:"account_ref"`
	RoleRef    string `json:"role_ref"`
}
type Accounts struct {
	Items       []Account             `json:"items"`
	Defaults    map[string]Selection  `json:"default_roles"`
	UIDBindings map[string]UIDBinding `json:"uid_bindings"`
	NextPage    *int                  `json:"next_page"`
}
type QueryResult struct {
	Operation   string         `json:"operation"`
	Role        Role           `json:"role"`
	Data        map[string]any `json:"data"`
	FetchedAtMS int64          `json:"fetched_at_ms"`
}

type ServiceCaller interface {
	CallService(context.Context, rayleabot.ServiceCallRequest, any) error
}
type AccountsClient struct {
	Caller   ServiceCaller
	Provider string
	Game     string
}

func (c AccountsClient) call(ctx context.Context, method string, params map[string]any, out any) error {
	provider := c.Provider
	if provider == "" {
		provider = "raylea.mihoyo-accounts"
	}
	return c.Caller.CallService(ctx, rayleabot.ServiceCallRequest{TargetPluginID: provider, Service: "accounts", ServiceVersion: 1, Method: method, Params: params}, out)
}
func (c AccountsClient) List(ctx context.Context, page int) (Accounts, error) {
	result := Accounts{}
	err := c.call(ctx, "list", map[string]any{"page": page}, &result)
	for index := range result.Items {
		result.Items[index].Roles = slices.DeleteFunc(result.Items[index].Roles, func(role Role) bool { return role.Game != c.Game })
	}
	return result, err
}
func (c AccountsClient) Select(ctx context.Context, choice Selection) error {
	return c.call(ctx, "select", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef}, nil)
}

func (c AccountsClient) PublicProfile(ctx context.Context, uid, region string) (QueryResult, error) {
	result := QueryResult{}
	err := c.call(ctx, "public.execute", map[string]any{"game": c.Game, "uid": uid, "region": region}, &result)
	return result, err
}
func (c AccountsClient) Execute(ctx context.Context, choice Selection, operation string, input map[string]any) (QueryResult, error) {
	result := QueryResult{}
	if input == nil {
		input = map[string]any{}
	}
	err := c.call(ctx, "execute", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "operation": operation, "input": input}, &result)
	if err != nil {
		return result, err
	}
	return result, shapeQueryResult(operation, input, &result)
}

// shapeQueryResult does the business shaping the accounts plugin leaves to
// the game: the official Star Rail panel lists every character and the rogue
// endpoint returns both periods, so each keeps only what was asked for.
func shapeQueryResult(operation string, input map[string]any, result *QueryResult) error {
	switch operation {
	case "starrail.character":
		ids, ok := input["character_ids"].([]any)
		if !ok {
			return nil
		}
		avatars, ok := result.Data["avatar_list"].([]any)
		if !ok {
			return gameError("upstream_invalid", "官方返回的角色列表格式无效。")
		}
		wanted := map[string]bool{}
		for _, id := range ids {
			wanted[asText(id)] = true
		}
		kept := []any{}
		for _, raw := range avatars {
			if wanted[asText(asObject(raw)["id"])] {
				kept = append(kept, raw)
			}
		}
		result.Data["avatar_list"] = kept
	case "starrail.rogue":
		switch asText(input["schedule_type"]) {
		case "1":
			delete(result.Data, "last_record")
		case "2":
			delete(result.Data, "current_record")
		}
	}
	return nil
}

// Ark sends an authorized ark request through the accounts plugin, which keeps
// the ark token; the response comes back without it.
func (c AccountsClient) Ark(ctx context.Context, route string, body map[string]any) (any, error) {
	var result struct {
		Data any `json:"data"`
	}
	err := c.call(ctx, "ark.request", map[string]any{"route": route, "body": body}, &result)
	return result.Data, err
}

func (c AccountsClient) ExecuteConfirmed(ctx context.Context, choice Selection, operation string, input map[string]any) (QueryResult, error) {
	result := QueryResult{}
	if input == nil {
		input = map[string]any{}
	}
	err := c.call(ctx, "execute", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "operation": operation, "input": input, "write_confirmed": true}, &result)
	return result, err
}
func Choose(accounts Accounts, game, uid string) (Selection, Role, error) {
	var choices []Selection
	var roles []Role
	for _, account := range accounts.Items {
		for _, role := range account.Roles {
			if role.Game == game && (uid == "" || role.UID == uid) {
				choices = append(choices, Selection{AccountRef: account.Ref, RoleRef: role.Ref})
				roles = append(roles, role)
			}
		}
	}
	if uid == "" {
		if choice, ok := accounts.Defaults[game]; ok {
			for i, candidate := range choices {
				if candidate == choice {
					return choice, roles[i], nil
				}
			}
		}
	}
	if len(choices) == 1 {
		return choices[0], roles[0], nil
	}
	if len(choices) == 0 {
		return Selection{}, Role{}, &rayleabot.ActionError{Code: "plugin.game_role_missing", Message: "未找到可用角色，请先扫码并授权此游戏。"}
	}
	return Selection{}, Role{}, &rayleabot.ActionError{Code: "plugin.game_role_ambiguous", Message: "存在多个角色，请指定 UID 或选择默认角色。"}
}

func PublicError(err error) *rayleabot.ActionError {
	var failure *rayleabot.ActionError
	if errors.As(err, &failure) {
		return failure
	}
	return &rayleabot.ActionError{Code: "plugin.game_operation_failed", Message: "操作未完成，请稍后重试。"}
}
