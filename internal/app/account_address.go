package app

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func accountOnline(protocol, botID string, bots []rayleabot.Bot) bool {
	return botID != "" && slices.ContainsFunc(bots, func(bot rayleabot.Bot) bool {
		return bot.SourceProtocol == protocol && bot.ID == botID
	})
}

func contentRef(game string, owner Subject, groupID string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{owner.SourceProtocol, owner.BotID, groupID}, "\x00")))
	return "game.content." + game + "." + hex.EncodeToString(sum[:16])
}
