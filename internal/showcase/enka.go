// Package showcase reads ZZZ panels from Enka, the showcase service
// ZZZ-Plugin refreshes from without an account.
package showcase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RayleaBot/zzz/internal/app"
)

// Source is Enka's ZZZ API, ZZZ-Plugin's default enkaApi.
var Source = app.ShowcaseSource{Name: "Enka", URL: func(uid string) string { return "https://enka.network/api/zzz/uid/" + uid }, Parse: Parse}

type enkaAnswer struct {
	TTL        int `json:"ttl"`
	PlayerInfo *struct {
		SocialDetail struct {
			ProfileDetail struct {
				Nickname string `json:"Nickname"`
				Level    int    `json:"Level"`
			} `json:"ProfileDetail"`
		} `json:"SocialDetail"`
		ShowcaseDetail struct {
			AvatarList []any `json:"AvatarList"`
		} `json:"ShowcaseDetail"`
	} `json:"PlayerInfo"`
}

// Parse reads Enka's answer the way ZZZ-Plugin's refreshPanelFromEnka does:
// its Enka2Mys, run from the bundled scripts, turns every showcased agent
// into an official avatar entry, which is then read like the account's.
func Parse(ctx context.Context, game app.Game, catalog app.Catalog, raw []byte) (app.ShowcaseProfile, error) {
	var answer enkaAnswer
	if err := json.Unmarshal(raw, &answer); err != nil || answer.PlayerInfo == nil {
		return app.ShowcaseProfile{}, &app.ShowcaseFailure{Status: 200}
	}
	player := answer.PlayerInfo.SocialDetail.ProfileDetail
	profile := app.ShowcaseProfile{Nickname: player.Nickname, Level: player.Level, TTL: time.Duration(answer.TTL) * time.Second}
	avatars := answer.PlayerInfo.ShowcaseDetail.AvatarList
	if len(avatars) == 0 {
		return profile, app.ErrShowcaseEmpty
	}
	if game.Calc == nil {
		return profile, &app.ShowcaseFailure{Status: 200}
	}
	converted, err := game.Calc.Showcase(ctx, map[string]any{"avatars": avatars})
	var data map[string]any
	if err != nil || json.Unmarshal(converted, &data) != nil {
		return profile, &app.ShowcaseFailure{Status: 200}
	}
	panels := app.NormalizePanels(app.QueryResult{Data: data}, catalog)
	for index := range panels {
		panels[index].Source = "enka"
	}
	profile.Panels = panels
	return profile, nil
}
