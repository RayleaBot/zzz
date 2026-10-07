package app

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RayleaBot/zzz/internal/pluginmeta"
)

// commandSet maps the command word the host delivered back to this plugin's
// declaration. The host matched the word against the same triggers and only
// passes the word on, so the plugin repeats the lookup in the host's order:
// declarations in manifest order, exact names compared as is, patterns with
// MatchString. Handlers dispatch on the command ID, so display names and
// trigger words can follow upstream without touching them.
type commandSet struct {
	commands []declaredCommand
}

type declaredCommand struct {
	id       string
	names    []string
	pattern  *regexp.Regexp
	fallback bool
}

func newCommandSet(manifest pluginmeta.Manifest) (commandSet, error) {
	set := commandSet{}
	for _, command := range manifest.Commands {
		declared := declaredCommand{id: command.ID, fallback: command.Trigger.Fallback}
		switch command.Trigger.Type {
		case "exact":
			declared.names = command.Trigger.Names
		case "pattern":
			expression, err := regexp.Compile(command.Trigger.Pattern)
			if err != nil {
				return set, fmt.Errorf("command %s has an invalid pattern: %w", command.ID, err)
			}
			declared.pattern = expression
		default:
			continue
		}
		set.commands = append(set.commands, declared)
	}
	return set, nil
}

// resolve returns the command ID and arguments for a delivered command word.
// The named groups of a pattern trigger become the leading arguments, so
// "雷神面板 1000" reaches the panel handler with ["雷神", "1000"]. As the host
// does, fallback commands are tried only after every ordinary one.
func (s commandSet) resolve(word string, args []string) (string, []string, bool) {
	if id, leading, ok := s.match(strings.TrimSpace(word), args, false); ok {
		return id, leading, ok
	}
	return s.match(strings.TrimSpace(word), args, true)
}

func (s commandSet) match(word string, args []string, fallback bool) (string, []string, bool) {
	if word == "" {
		return "", args, false
	}
	for _, command := range s.commands {
		if command.fallback != fallback {
			continue
		}
		if command.pattern == nil {
			for _, name := range command.names {
				if strings.TrimSpace(name) == word {
					return command.id, args, true
				}
			}
			continue
		}
		match := command.pattern.FindStringSubmatch(word)
		if match == nil {
			continue
		}
		leading := []string{}
		for i, group := range command.pattern.SubexpNames() {
			if group != "" && match[i] != "" {
				leading = append(leading, match[i])
			}
		}
		return command.id, append(leading, args...), true
	}
	return "", args, false
}

// usage writes a declared usage with the prefix replies use. Manifests start
// usages with the placeholder "#", which the host help menu also replaces.
func (a *App) usage(text string) string {
	if strings.HasPrefix(text, "#") {
		return a.Game.Prefix + text[1:]
	}
	return text
}
