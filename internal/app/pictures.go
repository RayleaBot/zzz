package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 图鉴 reads the upstream images an administrator downloaded the way Atlas
// reads its libraries. Each game declares where in game.json.

// Pictures are where a game's downloaded chat images are.
type Pictures struct {
	// Atlas are the 图鉴 libraries, tried in order: an Atlas repository's
	// path.json index.
	Atlas []PictureSource `json:"atlas"`
}

type PictureSource struct {
	Source string `json:"source"`
	Index  string `json:"index,omitempty"`
	// Rules are Atlas's rules for the library's modules (its
	// rule_default/<module>.yaml), config for the others.
	Rules map[string]AtlasRule `json:"rules,omitempty"`
}

// AtlasRule is which words an Atlas module answers: Condition 0 any word, 1
// a word after the prefix, 2 one with a Pick word, 3 either, 4 both, 5 one
// with a Pick word, the prefix optional; the Pick words are removed before
// the name is looked up. Commands always come after the prefix.
type AtlasRule struct {
	Condition int      `json:"condition"`
	Pick      []string `json:"pick"`
}

// config is the rule of the modules without their own: the library's
// config, else Atlas's default of 图鉴 or the prefix.
func (s PictureSource) config() AtlasRule {
	if rule, ok := s.Rules["config"]; ok {
		return rule
	}
	return AtlasRule{Condition: 3, Pick: []string{"图鉴"}}
}

// pick is Atlas's PickRule for a word that came after the prefix: the name
// it leaves, false when the rule does not answer the word.
func (r AtlasRule) pick(word string) (string, bool) {
	words := r.Pick
	if len(words) == 0 {
		words = []string{"图鉴"}
	}
	// Atlas joins the pick words into one expression.
	pick, err := regexp.Compile("(" + strings.Join(words, "|") + ")")
	if err != nil {
		return "", false
	}
	switch r.Condition {
	case 0, 1:
		return strings.TrimSpace(word), true
	case 3:
		return strings.TrimSpace(pick.ReplaceAllString(word, "")), true
	case 2, 4, 5:
		if pick.MatchString(word) {
			return strings.TrimSpace(pick.ReplaceAllString(word, "")), true
		}
	}
	return "", false
}

// artworkFile is a downloaded file of an artwork source.
type artworkFile struct{ Source, Path string }

var pictureExtensions = []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}

// atlasIndexes keeps each Atlas path.json, read again when the file changes.
type atlasIndexes struct {
	mu    sync.Mutex
	items map[string]atlasIndex
}

type atlasIndex struct {
	modified time.Time
	// modules are in the file's order, which Atlas searches in.
	modules []atlasModule
}

// atlasModule maps the keys of a path.json module to image paths, and the
// aliases in the library's othername/<module>.yaml to keys, which is how
// Atlas turns a name into a key; Star Rail's keys are IDs.
type atlasModule struct {
	name    string
	paths   map[string]string
	aliases map[string]string
}

func (c *atlasIndexes) read(root string, source PictureSource) []atlasModule {
	file := filepath.Join(root, source.Source, filepath.FromSlash(source.Index))
	info, err := os.Stat(file)
	if err != nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.items[source.Source]; ok && cached.modified.Equal(info.ModTime()) {
		return cached.modules
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	modules := []atlasModule{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil
	}
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			return nil
		}
		module := atlasModule{name: name.(string), paths: map[string]string{}}
		if decoder.Decode(&module.paths) != nil {
			return nil
		}
		if aliases, err := os.ReadFile(filepath.Join(root, source.Source, "othername", module.name+".yaml")); err == nil {
			module.aliases = atlasAliases(aliases)
		}
		modules = append(modules, module)
	}
	if c.items == nil {
		c.items = map[string]atlasIndex{}
	}
	c.items[source.Source] = atlasIndex{modified: info.ModTime(), modules: modules}
	return modules
}

// atlasAliases reads an Atlas othername file: top-level keys, each followed
// by a list of its names. An alias listed under several keys belongs to the
// first, as Atlas finds it.
func atlasAliases(raw []byte) map[string]string {
	unquote := func(value string) string {
		if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') && value[len(value)-1] == value[0] {
			return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
		}
		return value
	}
	aliases := map[string]string{}
	key := ""
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "" || strings.HasPrefix(trimmed, "#"):
		case strings.HasPrefix(trimmed, "- "):
			if alias := unquote(strings.TrimSpace(trimmed[2:])); key != "" && alias != "" {
				if _, taken := aliases[alias]; !taken {
					aliases[alias] = key
				}
			}
		case line[0] != ' ' && strings.HasSuffix(trimmed, ":"):
			key = unquote(strings.TrimSuffix(trimmed, ":"))
		}
	}
	return aliases
}

// atlasPicture is Atlas's search for a command word: each downloaded
// library's modules in path.json order, the module's rule deciding whether
// it answers the word and what name is left; the name is looked up among the
// module's aliases, then as the character or item the plugin's aliases name,
// as Atlas borrows ZZZ-Plugin's alias file.
func (a *App) atlasPicture(word string, aliases map[string]string) (artworkFile, bool) {
	for _, source := range a.Game.Pictures.Atlas {
		if source.Index == "" || !a.Artwork.Ready(source.Source) {
			continue
		}
		for _, module := range a.atlases.read(a.Artwork.Root, source) {
			rule, ok := source.Rules[module.name]
			if !ok {
				rule = source.config()
			}
			name, ok := rule.pick(word)
			if !ok || name == "" {
				continue
			}
			names := []string{name}
			if entry, _, found := a.aliasOwner(name, aliases); found && entry.Name != name {
				names = append(names, entry.Name)
			}
			for _, name := range names {
				key, aliased := module.aliases[name]
				if !aliased {
					key = name
				}
				// Some modules also index the library's alias files.
				if file, ok := module.paths[key]; ok && slices.Contains(pictureExtensions, strings.ToLower(path.Ext(file))) {
					if _, found := a.Artwork.File(source.Source, strings.TrimPrefix(file, "/")); found {
						return artworkFile{source.Source, strings.TrimPrefix(file, "/")}, true
					}
				}
			}
		}
	}
	return artworkFile{}, false
}

// pictureHint tells how to download the sources a reply found nothing in.
func (a *App) pictureHint(sources []PictureSource) string {
	missing := []string{}
	for _, source := range sources {
		if !a.Artwork.Ready(source.Source) && !slices.Contains(missing, source.Source) {
			missing = append(missing, source.Source)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "管理员可发送“" + a.Game.Prefix + "素材更新 " + strings.Join(missing, " ") + "”下载图片素材。"
}

// rememberImage keeps the image just sent for 原图.
func (a *App) rememberImage(event *rayleabot.EventContext, ref string) error {
	return a.Interactions.edit(chatOwner(event), func(p *InteractionProfile) error {
		p.LastImage = ref
		p.LastImageMS = time.Now().UnixMilli()
		p.LastImageTargetType = event.Event.Target.Type
		p.LastImageTargetID = event.Event.Target.ID
		return nil
	})
}

// sendArtwork sends a downloaded image.
func (a *App) sendArtwork(event *rayleabot.EventContext, file artworkFile) error {
	data, err := a.Artwork.Open(file.Source, file.Path)
	if err != nil {
		return event.SendText("图片素材读取失败，请重新下载素材。")
	}
	if err = a.rememberImage(event, "artwork:"+file.Source+"/"+file.Path); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(data)))
}
