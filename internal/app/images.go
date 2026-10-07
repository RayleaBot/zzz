package app

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/zzz/internal/artwork"
	"github.com/RayleaBot/zzz/internal/gacha"
)

// Image is a reply drawn with one of the game plugin's own templates, laid out
// after the upstream plugin's image for the same command.
type Image struct {
	Template  string
	Data      map[string]any
	Resources []rayleabot.RenderImageResource
}

// ImageContext is what an image builder may use besides the query result.
type ImageContext struct {
	Game    Game
	Catalog Catalog
	Artwork *artwork.Store
	Now     time.Time
	// Query runs another official query for the same role, for images that
	// upstream builds from several requests. It is nil outside a query.
	Query func(operation string, input map[string]any) (QueryResult, error)
	// Input is what the command asked the query for, as in schedule_type for
	// 上期. It is nil outside a query.
	Input map[string]any
	// Word is the command word as sent, for images that read choices the
	// query does not take, as 上期危局.
	Word string
	// RankShown is whether the requester's UID shows in the group rankings
	// the query feeds, for the note upstream ends a record with; nil outside a
	// group or for a query that feeds none.
	RankShown *bool
	// Score runs the pinned upstream scoring on a panel, for images that show
	// many characters' equipment scores, as 练度统计. It is nil when the game
	// has no scoring.
	Score func(CharacterPanel) (CharacterPanel, error)
	// SavedPanels reads the panels kept for a UID, as 练度统计 reads
	// upstream's saved panels.
	SavedPanels func(uid string) []CharacterPanel

	ctx context.Context
}

// PanelImage is one character's panel with what the upstream panel image
// shows beside it. Panel carries its score detail when scoring succeeded;
// Damage is the card's calculation, without damages when none could be
// calculated.
type PanelImage struct {
	Panel  CharacterPanel
	UID    string
	Damage DamageResult
	// Portrait is a custom picture of the character, as a path in the plugin
	// data directory, shown instead of the default portrait; "" for none.
	Portrait string
}

// PanelImageBuilder draws a single-character panel with the plugin's
// template, or returns false to keep the generic summary card.
type PanelImageBuilder func(ImageContext, PanelImage) (Image, bool)

// GachaImage is a gacha archive with the command word that asked for it;
// upstream picks the pool from the word, as in 武器记录.
type GachaImage struct {
	UID     string
	Role    Role
	Word    string
	Archive gacha.Archive
}

// GachaImageBuilder draws a gacha record reply with the plugin's template, or
// returns false to keep the generic summary card.
type GachaImageBuilder func(ImageContext, GachaImage) (Image, bool)

// EntryImage is what a reference page draws on: the command's ID, the
// command word as sent (with the levels a 技能 page reads after it) and the
// catalog entry it names.
type EntryImage struct {
	Command, Word string
	Entry         Entry
}

// EntryImageBuilder draws a catalog entry's page with the plugin's template,
// or returns false to keep the entry in text.
type EntryImageBuilder func(ImageContext, EntryImage) (Image, bool)

// entryImage draws an entry with the plugin's reference pages, if any.
func (a *App) entryImage(ctx context.Context, command, word string, entry Entry) *Image {
	if a.entryPage == nil {
		return nil
	}
	image, ok := a.entryPage(a.imageContext(ctx), EntryImage{Command: command, Word: word, Entry: entry})
	if !ok {
		return nil
	}
	return &image
}

var (
	talentLevels    = regexp.MustCompile(`(?:天赋|技能)[0-9A-Za-z.]*$`)
	skillWord       = regexp.MustCompile(`(?:天赋|技能)(.*)$`)
	skillLevelSplit = regexp.MustCompile(`\.|\s+`)
)

// talentWords splits a talent-wiki command into the agent's name and the text
// its page reads. ZZZ-Plugin reads a 技能 or 天赋 page's levels to the end of
// the message, split by dots or spaces, so levels after the word belong to
// the page, as in "艾莲技能12 12 10".
func talentWords(word string, args []string) (name, text string) {
	if len(args) > 1 && talentLevels.MatchString(word) {
		return args[0], strings.Join(append([]string{word}, args[1:]...), " ")
	}
	return strings.Join(args, " "), word
}

// SkillLevels reads the levels a 技能 word may end with, as ZZZ-Plugin's
// skills does: split by dots or spaces, a letter standing for its place in
// the alphabet; basic, dodge, assist, special and chain attack from 1 to 12
// (12 by default) and the core skill from 0 to 6 (6 by default). ok is false
// when a level is out of range.
func SkillLevels(word string) ([6]int, bool) {
	levels := [6]int{12, 12, 12, 12, 12, 6}
	match := skillWord.FindStringSubmatch(word)
	if match == nil || strings.TrimSpace(match[1]) == "" {
		return levels, true
	}
	parts := skillLevelSplit.Split(strings.TrimSpace(match[1]), -1)
	for index, part := range parts {
		if index >= len(levels) {
			break
		}
		level, err := strconv.Atoi(part)
		if err != nil && part != "" {
			level = int(strings.ToUpper(part)[0]) - 64
		}
		levels[index] = level
	}
	for index, level := range levels {
		if index == 5 && (level < 0 || level > 6) || index < 5 && (level < 1 || level > 12) {
			return levels, false
		}
	}
	return levels, true
}

// CalendarImage is what a 日历 image draws on: the command word as sent,
// which may ask for the list layout, and the official announcements.
type CalendarImage struct {
	Word          string
	Announcements Announcements
}

// CalendarImageBuilder draws 日历 with the plugin's template, or returns
// false to keep the announcement list in text.
type CalendarImageBuilder func(ImageContext, CalendarImage) (Image, bool)

// MonthlyStats is what a 统计 image draws on: the role, the command word as
// sent, which may name a year, and every saved month, oldest first.
type MonthlyStats struct {
	Role   Role
	Word   string
	Months []SavedMonth
}

// SavedMonth is one saved monthly report: its YYYY-MM key and the official
// data.
type SavedMonth struct {
	Month string
	Data  map[string]any
}

// MonthlyStatsImageBuilder draws 统计 with the plugin's template, or returns
// false to keep the text totals.
type MonthlyStatsImageBuilder func(ImageContext, MonthlyStats) (Image, bool)

// ImageBuilder draws a query result with the plugin's template for that
// command. It returns false when the result lacks what the template needs, and
// the reply falls back to the generic summary card.
type ImageBuilder func(ImageContext, QueryResult) (Image, bool)

// ArtworkResource returns a path resource for a downloaded upstream file, or false
// while the source has not been downloaded.
func (c ImageContext) ArtworkResource(id, source, name string) (rayleabot.RenderImageResource, bool) {
	if c.Artwork == nil {
		return rayleabot.RenderImageResource{}, false
	}
	path, ok := c.Artwork.File(source, name)
	if !ok {
		return rayleabot.RenderImageResource{}, false
	}
	return rayleabot.RenderImageResource{ID: id, Path: path}, true
}

// FetchArtworkResource is ArtworkResource for a source fetched on demand: the
// file is downloaded the first time a reply needs it.
func (c ImageContext) FetchArtworkResource(id, source, name string) (rayleabot.RenderImageResource, bool) {
	if c.Artwork == nil {
		return rayleabot.RenderImageResource{}, false
	}
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	path, ok := c.Artwork.Fetch(ctx, source, name)
	if !ok {
		return rayleabot.RenderImageResource{}, false
	}
	return rayleabot.RenderImageResource{ID: id, Path: path}, true
}

// FetchURLResource caches an image an official response links to, the way
// upstream's page loads it, from a source whose mirror is "https://". Other
// schemes are left out.
func (c ImageContext) FetchURLResource(id, source, url string) (rayleabot.RenderImageResource, bool) {
	name, found := strings.CutPrefix(url, "https://")
	if !found {
		return rayleabot.RenderImageResource{}, false
	}
	name, _, _ = strings.Cut(name, "?")
	return c.FetchArtworkResource(id, source, name)
}

func (a *App) imageContext(ctx context.Context) ImageContext {
	context := ImageContext{Game: a.Game, Catalog: a.Catalog, Artwork: a.Artwork, Now: time.Now(), ctx: ctx}
	if a.Game.Calc != nil {
		context.Score = func(panel CharacterPanel) (CharacterPanel, error) { return a.scorePanel(ctx, panel) }
	}
	context.SavedPanels = func(uid string) []CharacterPanel {
		saved, _ := a.Profiles.Read(uid)
		panels := []CharacterPanel{}
		for _, item := range saved.Sorted(a.Catalog, nil) {
			panels = append(panels, item.panel())
		}
		return panels
	}
	return context
}

// featureImage draws a query result with the plugin's own template when the
// plugin registered one for the operation.
func (a *App) featureImage(ctx context.Context, client AccountsClient, choice Selection, operation, word string, input map[string]any, result QueryResult, shown *bool) *Image {
	build := a.images[operation]
	if build == nil {
		return nil
	}
	imageContext := a.imageContext(ctx)
	imageContext.Query = func(operation string, input map[string]any) (QueryResult, error) {
		return client.Execute(ctx, choice, operation, input)
	}
	imageContext.Input, imageContext.Word, imageContext.RankShown = input, word, shown
	image, ok := build(imageContext, result)
	if !ok {
		return nil
	}
	return &image
}

// Text reads an official field as text; service calls decode numbers as
// json.Number. Missing fields read as "".
func Text(value any) string { return asText(value) }

// Int reads an official whole-number field, whether sent as a number or a
// numeric string. Missing or malformed fields read as 0.
func Int(value any) int {
	parsed, _ := strconv.ParseFloat(asText(value), 64)
	return int(parsed)
}

// ImageResources collects an image's render resources, adding each file once
// under the first ID it was asked for.
type ImageResources struct {
	Context ImageContext
	List    []rayleabot.RenderImageResource
	seen    map[string]string
	remote  int
}

// Artwork adds a downloaded upstream file and returns its resource ID, or ""
// while the source has not been downloaded.
func (r *ImageResources) Artwork(id, source, name string) string {
	return r.once(source+"/"+name, func() (rayleabot.RenderImageResource, bool) { return r.Context.ArtworkResource(id, source, name) })
}

// URL adds an image an official response links to, cached on demand from a
// source whose mirror is "https://", under a generated ID; "" when it cannot
// be fetched.
func (r *ImageResources) URL(source string, url any) string {
	id := "url-" + strconv.Itoa(len(r.seen))
	return r.once("url:"+asText(url), func() (rayleabot.RenderImageResource, bool) {
		return r.Context.FetchURLResource(id, source, asText(url))
	})
}

// Remote adds an image the host fetches from its URL, as chat avatars; the
// host fetches at most 16 per image.
func (r *ImageResources) Remote(id, url string) string {
	return r.once("remote:"+url, func() (rayleabot.RenderImageResource, bool) {
		return rayleabot.RenderImageResource{ID: id, URL: url}, url != ""
	})
}

// Prefetch caches the images the official URLs name, several at a time, so
// the URL calls that follow read the cache; a page with many official images
// would otherwise download them one after another on its first draw.
func (r *ImageResources) Prefetch(source string, urls ...any) {
	if r.Context.Artwork == nil {
		return
	}
	ctx := r.Context.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	names := map[string]bool{}
	for _, url := range urls {
		if name, found := strings.CutPrefix(asText(url), "https://"); found {
			name, _, _ = strings.Cut(name, "?")
			names[name] = true
		}
	}
	var group sync.WaitGroup
	slots := make(chan struct{}, 8)
	for name := range names {
		group.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; group.Done() }()
			r.Context.Artwork.Fetch(ctx, source, name)
		}()
	}
	group.Wait()
}

// The host takes at most 512 resources per image, 16 of them fetched from
// URLs; past that, further pictures are left out instead of failing the image.
const (
	maxImageResources  = 512
	maxRemoteResources = 16
)

func (r *ImageResources) once(key string, add func() (rayleabot.RenderImageResource, bool)) string {
	if id, ok := r.seen[key]; ok {
		return id
	}
	if r.seen == nil {
		r.seen = map[string]string{}
	}
	id := ""
	if len(r.List) < maxImageResources {
		if resource, ok := add(); ok && (resource.URL == "" || r.remote < maxRemoteResources) {
			if resource.URL != "" {
				r.remote++
			}
			r.List = append(r.List, resource)
			id = resource.ID
		}
	}
	r.seen[key] = id
	return id
}
