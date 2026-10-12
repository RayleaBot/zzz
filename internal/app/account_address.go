package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Adapter fields are only retained to recover old records without a bot ID.
// Neither identity nor outgoing account addresses include an adapter.
func (s Subject) identity() Subject {
	s.SourceAdapter = ""
	return s
}

func (s Subject) MarshalJSON() ([]byte, error) {
	type record Subject
	if s.BotID != "" {
		s = s.identity()
	}
	return json.Marshal(record(s))
}

func (s GroupScope) identity() GroupScope {
	s.Adapter = ""
	return s
}

func (s GroupScope) MarshalJSON() ([]byte, error) {
	type record GroupScope
	if s.BotID != "" {
		s = s.identity()
	}
	return json.Marshal(record(s))
}

func recoverBot(protocol, adapter, botID string, bots []rayleabot.Bot) string {
	// QQ official openids cannot be recovered from a connection that may now
	// log in as another account.
	if protocol == "onebot11" && botID == "" && adapter != "" {
		for _, bot := range bots {
			if bot.SourceProtocol == protocol && bot.SourceAdapter == adapter {
				return bot.ID
			}
		}
	}
	return botID
}

func (s Subject) recover(bots []rayleabot.Bot) Subject {
	s.BotID = recoverBot(s.SourceProtocol, s.SourceAdapter, s.BotID, bots)
	if s.BotID != "" {
		s = s.identity()
	}
	return s
}

func accountOnline(protocol, botID string, bots []rayleabot.Bot) bool {
	return botID != "" && slices.ContainsFunc(bots, func(bot rayleabot.Bot) bool {
		return bot.SourceProtocol == protocol && bot.ID == botID
	})
}

func contentRef(game string, owner Subject, groupID string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{owner.SourceProtocol, owner.BotID, groupID}, "\x00")))
	return "game.content." + game + "." + hex.EncodeToString(sum[:16])
}
