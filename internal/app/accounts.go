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
	return result, nil
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
