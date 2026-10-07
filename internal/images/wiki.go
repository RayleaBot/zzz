package images

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RayleaBot/zzz/internal/app"
)

// nanoka reads ZZZ-Plugin's wiki data source: static.nanoka.cc publishes each
// agent's data under the latest game version its manifest names. Upstream
// checks the manifest every twelve hours, starting from version 3.1; agents'
// data is kept as long here.
var nanoka = struct {
	sync.Mutex
	version string
	checked time.Time
	agents  map[string]nanokaCached
}{version: "3.1", agents: map[string]nanokaCached{}}

type nanokaCached struct {
	agent   nanokaAgent
	fetched time.Time
}

const nanokaHost = "https://static.nanoka.cc"

func nanokaGet(url string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil || response.StatusCode != http.StatusOK {
		return nil
	}
	return body
}

// nanokaAgentData returns an agent's data for the latest version, or false
// when nanoka has none.
func nanokaAgentData(id string) (nanokaAgent, bool) {
	nanoka.Lock()
	defer nanoka.Unlock()
	if time.Since(nanoka.checked) > 12*time.Hour {
		var manifest struct{ ZZZ struct{ Latest string } }
		if json.Unmarshal(nanokaGet(nanokaHost+"/manifest.json"), &manifest) == nil && manifest.ZZZ.Latest != "" {
			nanoka.version = manifest.ZZZ.Latest
		}
		nanoka.checked = time.Now()
	}
	key := nanoka.version + "/" + id
	if cached, ok := nanoka.agents[key]; ok && time.Since(cached.fetched) < 12*time.Hour {
		return cached.agent, true
	}
	var agent nanokaAgent
	if json.Unmarshal(nanokaGet(nanokaHost+"/zzz/"+nanoka.version+"/zh/character/"+id+".json"), &agent) != nil || agent.ID == 0 {
		return nanokaAgent{}, false
	}
	nanoka.agents[key] = nanokaCached{agent, time.Now()}
	return agent, true
}

// nanokaAgent is the part of nanoka's agent data the pages read.
type nanokaAgent struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	PartnerInfo struct {
		FullName    string `json:"full_name"`
		ImpressionF string `json:"impression_f"`
		ImpressionM string `json:"impression_m"`
	} `json:"partner_info"`
	Skill map[string]struct {
		Description []struct {
			Name  string `json:"name"`
			Desc  string `json:"desc"`
			Param []struct {
				Name  string          `json:"name"`
				Desc  string          `json:"desc"`
				Param json.RawMessage `json:"param"`
			} `json:"param"`
		} `json:"description"`
	} `json:"skill"`
	Passive struct {
		Level map[string]struct {
			Level int      `json:"level"`
			Name  []string `json:"name"`
			Desc  []string `json:"desc"`
		} `json:"level"`
	} `json:"passive"`
	Talent map[string]struct {
		Level int    `json:"level"`
		Name  string `json:"name"`
		Desc  string `json:"desc"`
	} `json:"talent"`
}

// nanokaSkillValue is a skill parameter's value.
type nanokaSkillValue struct {
	Main                float64 `json:"main"`
	Growth              float64 `json:"growth"`
	Format              string  `json:"format"`
	StunRatio           float64 `json:"stun_ratio"`
	StunRatioGrowth     float64 `json:"stun_ratio_growth"`
	SpRecovery          float64 `json:"sp_recovery"`
	SpRecoveryGrowth    float64 `json:"sp_recovery_growth"`
	FeverRecovery       float64 `json:"fever_recovery"`
	FeverRecoveryGrowth float64 `json:"fever_recovery_growth"`
	AttributeInfliction float64 `json:"attribute_infliction"`
}

// nanokaFirstValue is the first value of a parameter object in its JSON
// order, which upstream reads as Object.values(param)[0].
func nanokaFirstValue(raw json.RawMessage) (nanokaSkillValue, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') || !decoder.More() {
		return nanokaSkillValue{}, false
	}
	if _, err := decoder.Token(); err != nil {
		return nanokaSkillValue{}, false
	}
	var value nanokaSkillValue
	return value, decoder.Decode(&value) == nil
}

var (
	nanokaIcon  = regexp.MustCompile(`&lt;IconMap:Icon_(\w+)&gt;`)
	nanokaColor = regexp.MustCompile(`&lt;color=#(\w+?)&gt;(.+?)&lt;/color&gt;`)
)

// nanokaRichText is ZZZ-Plugin's parseRichText: skill icons, coloured bold
// terms and one line per line break; anything else prints as text.
func nanokaRichText(text string) string {
	text = html.EscapeString(text)
	text = nanokaIcon.ReplaceAllString(text, `<span class="skill-icon $1"></span>`)
	text = nanokaColor.ReplaceAllString(text, `<span style="color:#$1"><strong>$2</strong></span>`)
	return `<div class="line">` + strings.ReplaceAll(text, "\n", `</div><div class="line">`) + `</div>`
}

// jsNumber prints a number the way JavaScript converts it to text.
func jsNumber(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

// wikiSkills are ZZZ-Plugin's skill slots in page order.
var wikiSkills = [][3]string{{"basic", "普通攻击", "basic"}, {"dodge", "闪避", "dodge"}, {"assist", "支援技", "assist"}, {"special", "特殊技", "special"}, {"chain", "连携技", "chain"}}

var (
	wikiCinemaWord = regexp.MustCompile(`意象影画|意象|影画|命座`)
)

// Entry draws ZZZ-Plugin's agent pages from nanoka's data: 技能 and 天赋 as
// skills, with every skill's descriptions and its values at the levels asked
// for, and the core skill; 影画, 意象 and 命座 as cinema, the six Mindscape
// Cinema levels beside the third Mindscape's art.
func Entry(context app.ImageContext, page app.EntryImage) (app.Image, bool) {
	if page.Command != "talent-wiki" || page.Entry.Kind != "character" {
		return app.Image{}, false
	}
	cinema := wikiCinemaWord.MatchString(page.Word)
	levels, ok := app.SkillLevels(page.Word)
	if !cinema && !ok {
		return app.Image{}, false
	}
	agent, ok := nanokaAgentData(page.Entry.ID)
	if !ok {
		return app.Image{}, false
	}
	resources := newRecordResources(context, commonArtwork)
	id := strconv.Itoa(agent.ID)
	name := agent.Name
	data := map[string]any{"name": name, "full": agent.PartnerInfo.FullName,
		"female": nanokaText(agent.PartnerInfo.ImpressionF), "male": nanokaText(agent.PartnerInfo.ImpressionM),
	}
	// Upstream falls back to the official square avatar after a third-party
	// one; the official one is used here.
	if avatar, ok := context.FetchArtworkResource("avatar", "mys-zzz", "role_square_avatar/role_square_avatar_"+id+".png"); ok {
		resources.List, data["avatar"] = append(resources.List, avatar), "avatar"
	}
	if cinema {
		keys := []string{}
		for key := range agent.Talent {
			keys = append(keys, key)
		}
		slices.SortFunc(keys, func(a, b string) int { return agent.Talent[a].Level - agent.Talent[b].Level })
		talents := []any{}
		for _, key := range keys {
			talent := agent.Talent[key]
			talents = append(talents, map[string]any{"level": talent.Level, "name": talent.Name, "desc": nanokaRichText(talent.Desc)})
		}
		data["talents"] = talents
		if art, ok := context.FetchArtworkResource("cinema", "nanoka", "assets/zzz/Mindscape_"+id+"_3.webp"); ok {
			resources.List, data["image"] = append(resources.List, art), "cinema"
		}
		return app.Image{Template: "cinema", Data: data, Resources: resources.List}, true
	}
	skills := []any{}
	for index, slot := range wikiSkills {
		skill := agent.Skill[slot[0]]
		level := levels[index]
		items, rates := []any{}, []any{}
		for _, description := range skill.Description {
			if description.Param == nil {
				if description.Desc != "" {
					items = append(items, map[string]any{"title": description.Name, "content": nanokaRichText(description.Desc)})
				}
				continue
			}
			rows, details := []any{}, []any{}
			for _, param := range description.Param {
				value, ok := nanokaFirstValue(param.Param)
				if !ok {
					rows = append(rows, map[string]any{"label": param.Name, "value": param.Desc})
					continue
				}
				growth := float64(level - 1)
				final := value.Main + value.Growth*growth
				text := jsNumber(final)
				if value.Format == "%" {
					text = jsNumber(final/100) + "%"
				}
				rows = append(rows, map[string]any{"label": param.Name, "value": text})
				details = append(details, map[string]any{"no": len(details) + 1, "a": jsNumber(final / 100),
					"b": jsNumber((value.StunRatio + value.StunRatioGrowth*growth) / 100),
					"c": jsNumber((value.SpRecovery + value.SpRecoveryGrowth*growth) / 10000),
					"d": jsNumber((value.FeverRecovery + value.FeverRecoveryGrowth*growth) / 10000),
					"e": jsNumber(value.AttributeInfliction / 100), "f": "0"})
			}
			rates = append(rates, map[string]any{"name": description.Name, "rows": rows, "details": details})
		}
		skills = append(skills, map[string]any{"name": slot[1], "icon": slot[2], "level": level, "items": items, "rates": rates})
	}
	data["skills"] = skills
	// The core skill shows the level after the one asked for, as upstream
	// indexes its levels from 0.
	for _, core := range agent.Passive.Level {
		if core.Level != levels[5]+1 {
			continue
		}
		items := []any{}
		for index, desc := range core.Desc {
			title := ""
			if index < len(core.Name) {
				title = core.Name[index]
			}
			items = append(items, map[string]any{"title": title, "content": nanokaRichText(desc)})
		}
		data["core"] = map[string]any{"level": levels[5], "items": items}
	}
	return app.Image{Template: "skills", Data: data, Resources: resources.List}, true
}

// nanokaText prints an impression line, keeping its line breaks.
func nanokaText(text string) string {
	return strings.ReplaceAll(html.EscapeString(text), "\n", "<br>")
}
