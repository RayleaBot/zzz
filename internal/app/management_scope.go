package app

import (
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Public UID archives and plugin-wide resources have no chat owner. Everything
// else must enter through a host-verified account, even without a page ID.
func accountAction(action string) bool {
	switch action {
	case "status", "public.query", "showcase", "help.query", "aliases.validate",
		"catalog.search", "catalog.get", "calendar.query", "banners.query", "codes.query",
		"content.start", "content.poll", "content.cancel",
		"guides.schema", "guides.settings", "guides.configure", "challenge.schema",
		"build.conditions", "build.presets.list", "build.presets.save", "build.presets.remove",
		"history.list", "history.get", "history.remove":
		return false
	}
	if strings.HasPrefix(action, "artwork.") {
		return false
	}
	return !strings.HasPrefix(action, "gacha.") || strings.HasPrefix(action, "gacha.sync.") || strings.HasPrefix(action, "gacha.task.")
}

func managementBot(event *rayleabot.EventContext) (rayleabot.BotAccount, error) {
	bot, ok := event.Event.ManagementBot()
	if !ok {
		return bot, &rayleabot.ActionError{Code: "platform.invalid_request", Message: "请选择机器人账号。"}
	}
	return bot, nil
}

func (s Subject) belongsTo(bot rayleabot.BotAccount) bool {
	return s.SourceProtocol == bot.SourceProtocol && s.BotID == bot.BotID
}

func checkManagementOwner(event *rayleabot.EventContext, owner Subject) error {
	bot, err := managementBot(event)
	if err != nil {
		return err
	}
	if !owner.belongsTo(bot) {
		return gameError("account_mismatch", "记录不属于当前机器人账号。")
	}
	return nil
}

func visibleOwner(event *rayleabot.EventContext, owner Subject) bool {
	if event.Event.EventType != "management.action" {
		return true
	}
	bot, ok := event.Event.ManagementBot()
	return ok && owner.belongsTo(bot)
}

func checkManagementInput(bot rayleabot.BotAccount, input map[string]any) error {
	for _, fields := range []map[string]any{input, asObject(input["scope"]), asObject(input["owner"]), asObject(input["subject"]), asObject(input["bot"])} {
		if protocol := asText(fields["source_protocol"]); protocol != "" && protocol != bot.SourceProtocol {
			return gameError("account_mismatch", "记录不属于当前机器人账号。")
		}
		if id := asText(fields["bot_id"]); id != "" && id != bot.BotID {
			return gameError("account_mismatch", "记录不属于当前机器人账号。")
		}
	}
	return nil
}
