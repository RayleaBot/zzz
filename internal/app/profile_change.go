package app

import (
	"cmp"
	"context"
	"encoding/json"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/reference"
)

// 面板换装 follows miao's ProfileChange: a word such as
// “雷神换90级5精护摩换绝缘4换天赋10 10 10” names a character and the parts
// to change, and the changed panel is calculated from the kept panel.

// changeSource is another character's kept panel a part comes from,
// optionally of another UID.
type changeSource struct{ CharacterID, UID string }

// PanelChange is a 面板换装 word as miao's matchMsg reads it.
type PanelChange struct {
	CharacterID string
	UID         string
	// Artifacts takes every piece, WeaponFrom the weapon and Pieces single
	// pieces by slot from other characters.
	Artifacts  *changeSource
	WeaponFrom *changeSource
	Pieces     map[int]changeSource
	// Scanned are pieces ark's OCR read from screenshots, by slot; a piece
	// named in the word takes precedence.
	Scanned map[int]PanelEquipment
	// Weapon names a weapon, a level or a refinement; an empty ID keeps the
	// weapon.
	Weapon *struct {
		ID                string
		Level, Refinement int
	}
	// Sets are the sets the pieces change to, by miao's slot groups.
	Sets    []string
	Level   int
	Cons    *int
	Talents map[string]int
	Trees   []string
	// Swap is the character whose panel wears the gear; Element picks the
	// Traveler's.
	Swap, Element string
}

var (
	changeWord      = regexp.MustCompile(`[变改换]`)
	changeUID       = regexp.MustCompile(`uid ?:? ?`)
	changeMain      = regexp.MustCompile(`^#*(\d{9,10})?(.+?)(详细|详情|面板|面版|圣遗物|伤害[1-7]?)?\s*(\d{9,10})?[变换改](.+)`)
	changeUIDs      = regexp.MustCompile(`\d{9,10}`)
	changeWeapon    = regexp.MustCompile(`^(?:等?级?([1-9][0-9])?级?)?\s*(?:([1-5一二三四五满])(精炼?|叠影?)|(精炼?|叠影?)([1-5一二三四五]))?\s*(?:等?级?([1-9][0-9])?级?)?\s*(.*)$`)
	changeCons      = regexp.MustCompile(`([0-6零一二三四五六满])(命|魂|星魂)`)
	changeTalentsGS = regexp.MustCompile(`(?:天赋|技能|行迹)((?:[1][0-5]|[1-9])[ ,]?)((?:[1][0-5]|[1-9])[ ,]?)([1][0-5]|[1-9])`)
	changeTalentsSR = regexp.MustCompile(`(?:天赋|技能|行迹)((?:[1][0-5]|[1-9])[ ,]?)((?:[1][0-5]|[1-9])[ ,]?)((?:[1][0-5]|[1-9])[ ,]?)([1][0-5]|[1-9])`)
	changeLevel     = regexp.MustCompile(`等级(?:^|[^0-9])(100|95|[1-9]|[1-8][0-9]|90)(?:[^0-9]|$)|(?:^|[^0-9])(100|95|[1-9]|[1-8][0-9]|90)级`)
)

// changeKeys are miao's words for the parts of a panel, by part.
var changeKeys = map[string][][2]string{
	"genshin": {{"artis", "圣遗物"}, {"arti1", "花,生之花"}, {"arti2", "毛,羽,羽毛,死之羽"}, {"arti3", "沙,沙漏,表,时之沙"}, {"arti4", "杯,杯子,空之杯"}, {"arti5", "头,冠,理之冠,礼冠,帽子,帽"}, {"weapon", "武器"}},
	"starrail": {{"artis", "圣遗物,遗器"}, {"arti1", "头,帽子,头部"}, {"arti2", "手,手套,手部"}, {"arti3", "衣,衣服,甲,躯干"}, {"arti4", "鞋,靴,鞋子,靴子,脚,脚部"},
		{"arti5", "球,位面球"}, {"arti6", "绳,线,链接绳,连接绳"}, {"weapon", "武器,光锥"}},
}

// changeDigits reads a count written in digits or Chinese numerals.
func changeDigits(value string) int {
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}
	return map[string]int{"零": 0, "一": 1, "二": 2, "三": 3, "四": 4, "五": 5, "六": 6, "满": 6}[value]
}

// setNames maps every set name, abbreviation and alias to the set.
func setNames(catalog Catalog) map[string]string {
	names := map[string]string{}
	for _, set := range catalog.ArtifactSets {
		names[set] = set
	}
	for set, abbr := range catalog.SetAbbrs {
		names[abbr] = set
	}
	for set, aliases := range catalog.SetAliases {
		for _, alias := range aliases {
			names[alias] = set
		}
	}
	return names
}

// parsePanelChange reads a 面板换装 word; false when it names no character
// or no change.
func parsePanelChange(game Game, catalog Catalog, aliases map[string]string, text string) (PanelChange, bool) {
	if !changeWord.MatchString(text) {
		return PanelChange{}, false
	}
	msg := strings.ReplaceAll(changeUID.ReplaceAllString(strings.ToLower(text), ""), "星铁", "")
	main := changeMain.FindStringSubmatch(msg)
	if main == nil || main[2] == "" {
		return PanelChange{}, false
	}
	character, ok := catalog.Resolve(strings.TrimSpace(changeUIDs.ReplaceAllString(main[2], "")), "character", aliases)
	if !ok {
		return PanelChange{}, false
	}
	change := PanelChange{CharacterID: character.ID, UID: cmp.Or(main[1], main[4]), Pieces: map[int]changeSource{}}
	keys := map[string]string{}
	words := []string{}
	for _, item := range changeKeys[game.ID] {
		for _, word := range strings.Split(item[1], ",") {
			keys[word] = item[0]
			words = append(words, regexp.QuoteMeta(word))
		}
	}
	keyWord := regexp.MustCompile(`^(\d{9,10})?\s*(.+?)\s*(\d{9,10})?\s*((?:` + strings.Join(words, "|") + `|\+)+)$`)
	sets := setNames(catalog)
	setWords := []string{}
	for name := range sets {
		setWords = append(setWords, regexp.QuoteMeta(name))
	}
	// Longer names first, as miao sorts them.
	slices.SortFunc(setWords, func(a, b string) int { return cmp.Or(len(b)-len(a), strings.Compare(a, b)) })
	setGroup := "(" + strings.Join(setWords, "|") + ")"
	setWord := regexp.MustCompile(`^` + setGroup + `套?[2,4]?\+?` + setGroup + `?套?[2,4]?\+?` + setGroup + `?套?[2,4]?$`)
	changed := false
	for _, txt := range strings.Split(strings.NewReplacer("变", "换", "改", "换").Replace(main[5]), "换") {
		txt = strings.TrimSpace(txt)
		if txt == "" {
			continue
		}
		// Parts of another character: 刻晴圣遗物, 100000001胡桃武器+花.
		if key := keyWord.FindStringSubmatch(txt); key != nil && key[4] != "" {
			if source, ok := catalog.Resolve(strings.TrimSpace(key[2]), "character", aliases); ok {
				from := changeSource{CharacterID: source.ID, UID: cmp.Or(key[1], key[3])}
				for _, word := range strings.Split(key[4], "+") {
					switch part := keys[strings.TrimSpace(word)]; {
					case part == "artis":
						change.Artifacts = &from
					case part == "weapon":
						change.WeaponFrom = &from
					case strings.HasPrefix(part, "arti"):
						slot, _ := strconv.Atoi(strings.TrimPrefix(part, "arti"))
						change.Pieces[slot] = from
					}
				}
				changed = true
			} else if len([]rune(key[4])) > 2 {
				continue
			}
		}
		// Sets: 绝缘4, 魔女2+宗室2, 星铁 with a planar set.
		if match := setWord.FindStringSubmatch(txt); match != nil && sets[match[1]] != "" {
			if game.ID == "genshin" {
				change.Sets = []string{sets[match[1]], cmp.Or(sets[match[2]], sets[match[1]])}
			} else {
				chosen := make([]string, 3)
				for _, name := range match[1:4] {
					set := sets[name]
					if set == "" {
						continue
					}
					if pieces := catalog.ArtifactPieces[set]; len(pieces) > 0 && pieces[0] != "" {
						chosen[map[bool]int{true: 1, false: 0}[chosen[0] != ""]] = set
					} else {
						chosen[2] = set
					}
				}
				if chosen[0] != "" && chosen[1] == "" {
					chosen[1] = chosen[0]
				}
				change.Sets = chosen
			}
			changed = true
			continue
		}
		// Weapons: 护摩, 90级5精护摩, 满精武器, 雷神专武.
		if match := changeWeapon.FindStringSubmatch(txt); match != nil && match[7] != "" {
			name := strings.TrimSpace(match[7])
			if strings.Contains(name, "专武") {
				owner := character
				if other, ok := catalog.Resolve(strings.ReplaceAll(name, "专武", ""), "character", aliases); ok {
					owner = other
				}
				name = owner.Name + "专武"
			}
			weapon, found := catalog.Resolve(name, "weapon", aliases)
			if found || name == "武器" {
				affix := changeDigits(cmp.Or(match[2], match[5]))
				if cmp.Or(match[2], match[5]) == "满" {
					affix = 5
				}
				level, _ := strconv.Atoi(cmp.Or(match[1], match[6]))
				if found || affix > 0 || level > 0 {
					change.Weapon = &struct {
						ID                string
						Level, Refinement int
					}{ID: map[bool]string{true: weapon.ID}[found], Level: level, Refinement: affix}
					changed = true
				}
				continue
			}
		}
		// The character: 满命, 满行迹, 天赋10 10 10, 90级, another character.
		if cons := changeCons.FindStringSubmatch(txt); cons != nil {
			value := max(0, min(6, changeDigits(cons[1])))
			change.Cons = &value
			txt = strings.Replace(txt, cons[0], "", 1)
			changed = true
		}
		if game.ID == "starrail" && strings.Contains(txt, "满行迹") {
			change.Trees = []string{"101", "102", "103", "201", "202", "203", "204", "205", "206", "207", "208", "209", "210", "301", "302"}
			txt = strings.Replace(txt, "满行迹", "", 1)
			changed = true
		}
		talents, order := changeTalentsGS, "aeq"
		if game.ID == "starrail" {
			talents, order = changeTalentsSR, "aetq"
		}
		if match := talents.FindStringSubmatch(txt); match != nil {
			change.Talents = map[string]int{}
			for index, key := range order {
				level, _ := strconv.Atoi(strings.Trim(match[index+1], " ,"))
				change.Talents[string(key)] = max(1, level)
			}
			txt = strings.Replace(txt, match[0], "", 1)
			changed = true
		}
		if match := changeLevel.FindStringSubmatch(txt); match != nil {
			change.Level, _ = strconv.Atoi(cmp.Or(match[1], match[2]))
			txt = strings.Replace(txt, match[0], "", 1)
			changed = true
		}
		if txt = strings.TrimSpace(txt); txt != "" {
			if character.ID == "10000005" || character.ID == "10000007" {
				txt = strings.Replace(txt, "元素", "主", 1)
			}
			if other, ok := catalog.Resolve(txt, "character", aliases); ok {
				change.Swap, change.Element = other.ID, other.Element
				changed = true
			}
		}
	}
	return change, changed
}

// changeDefaultWeapons are miao's weapons for a Genshin Impact panel whose
// weapon cannot be used, by weapon type.
var changeDefaultWeapons = map[string]string{"bow": "西风猎弓", "catalyst": "西风秘典", "claymore": "西风大剑", "polearm": "西风长枪", "sword": "西风剑"}

// changedPanel is miao's getProfile: the character's kept panel with the
// changes applied, its properties calculated again from the parts.
func (a *App) changedPanel(ctx context.Context, uid string, change PanelChange) (CharacterPanel, error) {
	kept := func(from changeSource) (CharacterPanel, bool) {
		saved, err := a.Profiles.Read(cmp.Or(from.UID, uid))
		if err != nil {
			return CharacterPanel{}, false
		}
		item, ok := saved.Panels[cmp.Or(from.CharacterID, change.CharacterID)]
		return item.panel(), ok
	}
	source, hasSource := kept(changeSource{})
	id := cmp.Or(change.Swap, change.CharacterID)
	element := change.Element
	if element == "" && hasSource && id == source.ID {
		element = source.Element
	}
	if element == "" {
		entry, _ := a.Catalog.Get(id)
		element = entry.Element
	}
	record, err := findReferenceCharacter(a.Game.Calc, a.Game.ID, CharacterPanel{ID: id, Element: element})
	if err != nil {
		return CharacterPanel{}, err
	}
	gs := a.Game.ID == "genshin"
	panel := CharacterPanel{ID: id, Level: cmp.Or(change.Level, source.Level, 90), Element: record.Element, Source: "change", RankKnown: true, WeaponKnown: true, EquipmentKnown: true,
		Stats: []PanelStat{}, Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}}
	if entry, ok := a.Catalog.Get(id); ok {
		panel.Name = entry.Name
	}
	if hasSource && panel.Level == source.Level {
		panel.Promote = source.Promote
	}
	if change.Cons != nil {
		panel.Rank = *change.Cons
	} else if hasSource {
		panel.Rank = source.Rank
	}
	// The weapon: the one named, another character's, or the kept one; in
	// Genshin Impact one of the wrong type becomes miao's default.
	weaponSource := source.Weapon
	if change.WeaponFrom != nil {
		if other, ok := kept(*change.WeaponFrom); ok {
			weaponSource = other.Weapon
		}
	}
	if weaponSource == nil {
		weaponSource = &PanelEquipment{}
	}
	weapons := map[string]reference.Weapon{}
	for _, weapon := range a.Game.Calc.Metadata().Weapons {
		weapons[weapon.ID] = weapon
		if weapon.Name == changeDefaultWeapons[record.WeaponType] && gs {
			weapons["default"] = weapon
		}
	}
	weaponID := weaponSource.ID
	if change.Weapon != nil && change.Weapon.ID != "" {
		weaponID = change.Weapon.ID
	}
	if weapon, ok := weapons[weaponID]; gs && (!ok || weapon.Type != record.WeaponType) {
		weaponID = weapons["default"].ID
	}
	weapon := PanelEquipment{ID: weaponID, Name: weapons[weaponID].Name, Main: []PanelStat{}, Sub: []PanelStat{}, Complete: true}
	maxLevel := 90
	if !gs {
		maxLevel = 80
	}
	requested := 0
	if change.Weapon != nil {
		requested = change.Weapon.Level
	}
	weapon.Level = min(maxLevel, cmp.Or(requested, weaponSource.Level, maxLevel))
	if weapon.Level == weaponSource.Level {
		weapon.Promote = weaponSource.Promote
	}
	star, _ := a.Catalog.Get(weaponID)
	sourceStar, _ := a.Catalog.Get(weaponSource.ID)
	weapon.Rarity = strconv.Itoa(star.Rarity)
	weapon.Refinement = cmp.Or(weaponSource.Refinement, 5)
	if star.Rarity == 5 && sourceStar.Rarity != 5 {
		weapon.Refinement = 1
	}
	if change.Weapon != nil && change.Weapon.Refinement > 0 {
		weapon.Refinement = change.Weapon.Refinement
	}
	weapon.Refinement = min(5, weapon.Refinement)
	// Talents: the ones named, or the kept panel's without its bonuses, plus
	// this panel's constellation or eidolon bonuses.
	talents := map[string]int{}
	if change.Talents != nil {
		talents = change.Talents
	} else {
		original := map[string]int{"a": 9, "e": 9, "q": 9}
		if !gs {
			original = map[string]int{"a": 6, "e": 8, "t": 8, "q": 8}
		}
		if hasSource {
			sourceRecord, err := findReferenceCharacter(a.Game.Calc, a.Game.ID, source)
			if err == nil {
				original = map[string]int{}
				for key, level := range PanelTalents(a.Game.ID, source, sourceRecord) {
					original[key] = level.Original
				}
			}
		}
		for key, level := range original {
			talents[key] = level + talentBonus(a.Game.ID, record, key, panel.Rank)
		}
	}
	// Traces: 满行迹, or the kept panel's.
	trees := []string{}
	if change.Trees != nil {
		for _, suffix := range change.Trees {
			trees = append(trees, record.ID+suffix)
		}
	} else if hasSource {
		for _, skill := range source.Skills {
			if skill.Active && skill.PointType != 2 && skill.ID != "" {
				trees = append(trees, RecordSkillID(a.Game.ID, source, record, skill))
			}
		}
	}
	// Artifacts: another character's, then single pieces, then the sets.
	pieces := map[int]PanelEquipment{}
	artifacts := source
	if change.Artifacts != nil {
		artifacts, _ = kept(*change.Artifacts)
	}
	for _, piece := range artifacts.Equipment {
		pieces[piece.Slot] = piece
	}
	slots, groups := 5, "00111"
	if !gs {
		slots, groups = 6, "001122"
	}
	for slot := 1; slot <= slots; slot++ {
		if from, ok := change.Pieces[slot]; ok {
			if other, ok := kept(from); ok {
				for _, piece := range other.Equipment {
					if piece.Slot == slot {
						pieces[slot] = piece
					}
				}
			}
		} else if piece, ok := change.Scanned[slot]; ok {
			pieces[slot] = piece
		}
		piece, ok := pieces[slot]
		group := int(groups[slot-1] - '0')
		if ok && group < len(change.Sets) && change.Sets[group] != "" {
			piece.SetName = change.Sets[group]
			if names := a.Catalog.ArtifactPieces[piece.SetName]; slot <= len(names) {
				piece.Name = names[slot-1]
			}
			pieces[slot] = piece
		}
		if ok {
			panel.Equipment = append(panel.Equipment, pieces[slot])
		}
	}
	return a.computedPanel(ctx, record, panel, weapon, talents, trees)
}

// computedPanel fills in a panel's properties, weapon stats, skills and
// constellations from its parts, as miao's Attr does: the character's level,
// promotion and rank, the weapon, the talent levels with their bonuses, the
// traces and the equipment already on the panel.
func (a *App) computedPanel(ctx context.Context, record reference.Character, panel CharacterPanel, weapon PanelEquipment, talents map[string]int, trees []string) (CharacterPanel, error) {
	gear, err := buildGearSet(a.Game.ID, panel)
	if err != nil {
		return CharacterPanel{}, err
	}
	input := map[string]any{"level": panel.Level, "promote": panel.Promote, "rank": panel.Rank, "talents": talents, "trees": trees, "equipment": gear,
		"weapon": BuildWeapon{ID: weapon.ID, Level: weapon.Level, Promote: weapon.Promote, Refinement: weapon.Refinement}}
	raw, err := a.Game.Calc.Change(ctx, record, input)
	var result struct {
		Promote       int                `json:"promote"`
		WeaponPromote int                `json:"weapon_promote"`
		Attributes    map[string]float64 `json:"attributes"`
		WeaponAttrs   map[string]any     `json:"weapon_attrs"`
	}
	if err != nil || json.Unmarshal(raw, &result) != nil {
		return CharacterPanel{}, buildUnavailable("reference_calculation")
	}
	panel.Promote, weapon.Promote = &result.Promote, &result.WeaponPromote
	panel.Stats = changedStats(a.Game.ID, record.Element, result.Attributes)
	weapon.Main, weapon.Sub = changedWeaponStats(a.Game.ID, result.WeaponAttrs)
	if weapon.ID != "" {
		panel.Weapon = &weapon
	}
	panel.Skills = changedSkills(a.Game.ID, record, talents, trees, panel.Rank)
	constellations := asObject(record.Data["cons"])
	for index := 1; index <= 6; index++ {
		panel.Ranks = append(panel.Ranks, PanelSkill{Name: asText(asObject(constellations[strconv.Itoa(index)])["name"]), Active: index <= panel.Rank})
	}
	return panel, nil
}

// talentBonus is the levels a constellation or eidolon adds to a talent, as
// miao counts them from talentCons.
func talentBonus(game string, record reference.Character, key string, rank int) int {
	step := map[string]int{"a": 1, "e": 2, "q": 2, "t": 2, "me": 1, "mt": 1, "xe": 1}[key]
	if game == "genshin" {
		step = 3
	}
	bonus := 0
	switch need := asObject(record.Data["talentCons"])[key].(type) {
	case float64:
		if need > 0 && float64(rank) >= need {
			bonus = step
		}
	case []any:
		for _, level := range need {
			if value, ok := level.(float64); ok && float64(rank) >= value {
				bonus += step
			}
		}
	}
	return bonus
}

// changedStats writes calculated properties as the official panel lists them.
func changedStats(game, element string, attrs map[string]float64) []PanelStat {
	flat := func(v float64) string { return strconv.FormatFloat(math.Round(v), 'f', 0, 64) }
	percent := func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" }
	stat := func(id, key string, value, base float64, pct bool) PanelStat {
		if pct {
			return PanelStat{ID: id, Key: key, Value: percent(value), Base: percent(base), Added: percent(value - base)}
		}
		return PanelStat{ID: id, Key: key, Value: flat(value), Base: flat(base), Added: flat(value - base)}
	}
	if game == "genshin" {
		stats := []PanelStat{stat("2000", "", attrs["hp"], attrs["hpBase"], false), stat("2001", "", attrs["atk"], attrs["atkBase"], false), stat("2002", "", attrs["def"], attrs["defBase"], false),
			stat("28", "mastery", attrs["mastery"], 0, false), stat("20", "cpct", attrs["cpct"], 5, true), stat("22", "cdmg", attrs["cdmg"], 50, true),
			stat("23", "recharge", attrs["recharge"], 100, true), stat("26", "heal", attrs["heal"], 0, true), stat("30", "phy", attrs["phy"], 0, true)}
		for key, id := range map[string]string{"pyro": "40", "electro": "41", "hydro": "42", "dendro": "43", "anemo": "44", "geo": "45", "cryo": "46"} {
			value := 0.0
			if key == canonicalBuildElement(element) {
				value = attrs["dmg"]
			}
			stats = append(stats, stat(id, key, value, 0, true))
		}
		return stats
	}
	speed := stat("4", "", attrs["speed"], attrs["speedBase"], false)
	speed.Value = strconv.FormatFloat(math.Floor(attrs["speed"]*10+1e-6)/10, 'f', -1, 64)
	// miao counts Star Rail energy regeneration without its base 100%.
	stats := []PanelStat{stat("1", "", attrs["hp"], attrs["hpBase"], false), stat("2", "", attrs["atk"], attrs["atkBase"], false), stat("3", "", attrs["def"], attrs["defBase"], false), speed,
		stat("5", "", attrs["cpct"], 5, true), stat("6", "", attrs["cdmg"], 50, true), stat("7", "", attrs["heal"], 0, true), stat("9", "", attrs["recharge"]+100, 100, true),
		stat("10", "", attrs["effPct"], 0, true), stat("11", "", attrs["effDef"], 0, true), stat("58", "", attrs["stance"], 0, true)}
	for key, id := range map[string]string{"physical": "12", "fire": "14", "ice": "16", "lightning": "18", "wind": "20", "quantum": "22", "imaginary": "24"} {
		value := 0.0
		if normalizedElement(element) == key || canonicalBuildElement(element) == canonicalBuildElement(key) {
			value = attrs["dmg"]
		}
		stats = append(stats, stat(id, "", value, 0, true))
	}
	slices.SortFunc(stats, func(a, b PanelStat) int { x, _ := strconv.Atoi(a.ID); y, _ := strconv.Atoi(b.ID); return x - y })
	return stats
}

// changedWeaponStats writes a weapon's calculated base and bonus.
func changedWeaponStats(game string, attrs map[string]any) ([]PanelStat, []PanelStat) {
	if attrs == nil {
		return []PanelStat{}, []PanelStat{}
	}
	number := func(value any) float64 { n, _ := value.(float64); return n }
	if game != "genshin" {
		main := []PanelStat{}
		for _, key := range []string{"hp", "atk", "def"} {
			main = append(main, PanelStat{ID: key, Key: key, Value: strconv.FormatFloat(math.Floor(number(attrs[key])+1e-6), 'f', 0, 64)})
		}
		return main, []PanelStat{}
	}
	main := []PanelStat{{ID: "4", Name: "基础攻击力", Value: strconv.FormatFloat(math.Round(number(attrs["atkBase"])), 'f', 0, 64)}}
	bonus := asObject(attrs["attr"])
	key := strings.TrimSuffix(asText(bonus["key"]), "Pct")
	if key == "" {
		return main, []PanelStat{}
	}
	value := number(bonus["value"])
	text := strconv.FormatFloat(value, 'f', 1, 64) + "%"
	if key == "mastery" {
		text = strconv.FormatFloat(math.Round(value), 'f', 0, 64)
	}
	return main, []PanelStat{{ID: key, Key: key, Value: text}}
}

// changedSkills gives the changed panel's talents and traces the shape the
// panel images and calculations read.
func changedSkills(game string, record reference.Character, talents map[string]int, trees []string, rank int) []PanelSkill {
	skills := []PanelSkill{}
	ids := map[string]string{}
	for id, key := range asObject(record.Data["talentId"]) {
		if _, taken := ids[asText(key)]; !taken {
			ids[asText(key)] = id
		}
	}
	kinds := map[string]string{"a": "普攻", "e": "战技", "q": "终结技", "t": "天赋", "z": "秘技", "xe": "欢愉技", "me": "忆灵技", "mt": "忆灵天赋"}
	keys := []string{"a", "e", "q"}
	if game != "genshin" {
		keys = []string{"a", "e", "q", "t", "xe", "me", "mt"}
	}
	for _, key := range keys {
		level, ok := talents[key]
		if !ok {
			continue
		}
		skill := PanelSkill{HasSkillType: true, SkillType: 1, ID: ids[key], Level: level, ExtraLevel: talentBonus(game, record, key, rank), Active: true, Name: asText(asObject(asObject(record.Data["talent"])[key])["name"])}
		if game != "genshin" {
			skill = PanelSkill{Kind: kinds[key], PointType: 2, ID: ids[key], Level: level, ExtraLevel: talentBonus(game, record, key, rank), Active: true, Name: kinds[key]}
		}
		skills = append(skills, skill)
	}
	for _, tree := range trees {
		skills = append(skills, PanelSkill{PointType: 1, ID: tree, Level: 1, Active: true, Name: "行迹 " + tree})
	}
	return skills
}

// panelChangeCommand answers a 面板换装 word with the changed panel, its
// score and damage, marked as not real data; it does not enter the group
// ranking. As with ark-plugin, attached or quoted screenshots put the pieces
// its OCR reads on the panel. Words it cannot read are left to other plugins.
func (a *App) panelChangeCommand(ctx context.Context, event *rayleabot.EventContext) error {
	text := strings.TrimSpace(event.Event.Command() + " " + strings.Join(event.Event.Args(), " "))
	change, ok := parsePanelChange(a.Game, a.Catalog, a.aliasMap(event), text)
	var images []string
	if change.CharacterID != "" {
		images = messageImages(ctx, event)
	}
	if !ok && len(images) == 0 || a.Game.Calc == nil {
		return event.Result(map[string]any{"handled": false})
	}
	if len(images) > 0 {
		scanned, err := a.scannedPieces(ctx, images)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		change.Scanned = scanned
	}
	owner, err := a.panelOwner(ctx, event, change.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	panel, err := a.changedPanel(ctx, owner.UID, change)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, a.fullPanelView(ctx, event, panel, owner.UID, false, text))
}
