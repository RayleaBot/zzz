package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 照片, 老婆 and 图鉴 read the upstream images an administrator downloaded:
// photos the way miao-plugin reads character-img, 图鉴 the way Atlas and
// xiaoyao read their libraries. Each game declares where in game.json.

// Pictures are where a game's downloaded chat images are.
type Pictures struct {
	// Photos are directories of a character's photos; {name} stands for the
	// character.
	Photos []PictureSource `json:"photos"`
	// Atlas are the 图鉴 libraries, tried in order: an Atlas repository's
	// path.json index, or file paths with {name}.
	Atlas []PictureSource `json:"atlas"`
}

type PictureSource struct {
	Source string   `json:"source"`
	Index  string   `json:"index,omitempty"`
	Paths  []string `json:"paths,omitempty"`
}

// artworkFile is a downloaded file of an artwork source.
type artworkFile struct{ Source, Path string }

var pictureExtensions = []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}

// characterPhotos lists a character's downloaded photos.
func (a *App) characterPhotos(entry Entry) []artworkFile {
	names := []string{entry.Name}
	files := []artworkFile{}
	for _, source := range a.Game.Pictures.Photos {
		for _, pattern := range source.Paths {
			for _, name := range names {
				for _, file := range a.Artwork.List(source.Source, strings.ReplaceAll(pattern, "{name}", name)) {
					if slices.Contains(pictureExtensions, strings.ToLower(path.Ext(file))) {
						files = append(files, artworkFile{source.Source, file})
					}
				}
			}
		}
	}
	return files
}

// atlasIndexes keeps each Atlas path.json, read again when the file changes.
type atlasIndexes struct {
	mu    sync.Mutex
	items map[string]atlasIndex
}

type atlasIndex struct {
	modified time.Time
	// modules map names to image paths, in the file's order, which Atlas
	// searches in.
	modules []map[string]string
}

func (c *atlasIndexes) read(root string, source PictureSource) []map[string]string {
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
	modules := []map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil
	}
	for decoder.More() {
		if _, err := decoder.Token(); err != nil {
			return nil
		}
		module := map[string]string{}
		if decoder.Decode(&module) != nil {
			return nil
		}
		modules = append(modules, module)
	}
	if c.items == nil {
		c.items = map[string]atlasIndex{}
	}
	c.items[source.Source] = atlasIndex{modified: info.ModTime(), modules: modules}
	return modules
}

// atlasPicture finds the first downloaded 图鉴 image of any of the names.
func (a *App) atlasPicture(names []string) (artworkFile, bool) {
	for _, source := range a.Game.Pictures.Atlas {
		if !a.Artwork.Ready(source.Source) {
			continue
		}
		if source.Index != "" {
			for _, module := range a.atlases.read(a.Artwork.Root, source) {
				for _, name := range names {
					if file, ok := module[name]; ok {
						if _, found := a.Artwork.File(source.Source, strings.TrimPrefix(file, "/")); found {
							return artworkFile{source.Source, strings.TrimPrefix(file, "/")}, true
						}
					}
				}
			}
		}
		for _, pattern := range source.Paths {
			for _, name := range names {
				file := strings.ReplaceAll(pattern, "{name}", name)
				if _, found := a.Artwork.File(source.Source, file); found {
					return artworkFile{source.Source, file}, true
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
