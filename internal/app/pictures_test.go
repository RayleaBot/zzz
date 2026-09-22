package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-zzz/internal/artwork"
)

func writeArtwork(t *testing.T, root, name, content string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPicturesReadDownloadedLibraries(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}, Game: Game{Prefix: "#", Pictures: Pictures{
		Photos: []PictureSource{{Source: "miao-plugin", Paths: []string{"resources/character-img/{name}"}}},
		Atlas:  []PictureSource{{Source: "genshin-atlas", Index: "path.json"}, {Source: "xiaoyao-plus", Paths: []string{"wuqi_tujian/{name}.png"}}},
	}}}
	if photos := a.characterPhotos(Entry{Name: "七七"}); len(photos) != 0 {
		t.Fatal("photos before a download", photos)
	}
	if hint := a.pictureHint(a.Game.Pictures.Atlas); !strings.Contains(hint, "#素材更新 genshin-atlas xiaoyao-plus") {
		t.Fatal(hint)
	}
	writeArtwork(t, root, "miao-plugin/resources/character-img/空/01.jpg", "a")
	writeArtwork(t, root, "miao-plugin/resources/character-img/荧/01.webp", "b")
	writeArtwork(t, root, "miao-plugin/resources/character-img/荧/notes.txt", "c")
	// The Traveler's photos are both twins', as miao's.
	if photos := a.characterPhotos(Entry{Name: "旅行者"}); len(photos) != 2 || photos[1].Path != "resources/character-img/荧/01.webp" {
		t.Fatal(photos)
	}
	// Atlas searches path.json's modules in the file's order.
	writeArtwork(t, root, "genshin-atlas/path.json", `{"weapon":{"护摩之杖":"/weapon/护摩之杖.png"},"card":{"护摩之杖":"/card/护摩之杖.png"}}`)
	writeArtwork(t, root, "genshin-atlas/weapon/护摩之杖.png", "d")
	writeArtwork(t, root, "genshin-atlas/card/护摩之杖.png", "e")
	writeArtwork(t, root, "xiaoyao-plus/wuqi_tujian/天空之刃.png", "f")
	if file, ok := a.atlasPicture([]string{"护摩", "护摩之杖"}); !ok || file != (artworkFile{"genshin-atlas", "weapon/护摩之杖.png"}) {
		t.Fatal(file, ok)
	}
	if file, ok := a.atlasPicture([]string{"天空之刃"}); !ok || file.Source != "xiaoyao-plus" {
		t.Fatal(file, ok)
	}
	if _, ok := a.atlasPicture([]string{"不存在"}); ok {
		t.Fatal("found a missing name")
	}
	if hint := a.pictureHint(a.Game.Pictures.Atlas); hint != "" {
		t.Fatal(hint)
	}
}

func TestStaticPicturesFindUpstreamNames(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}}
	writeArtwork(t, root, "starrail-plugin/resources/srsr/开拓者•存护.jpg", "a")
	writeArtwork(t, root, "starrail-plugin/resources/srsr/sy/1.jpg", "b")
	panel := StaticPicture{Source: "starrail-plugin", Character: true, Parts: []StaticPart{{Path: "resources/srsr/{name}.jpg"}}}
	// Upstream names the Trailblazer 开拓者•存护, an alias of 穹·存护.
	trailblazer := Entry{Name: "穹·存护", Aliases: []string{"存护开拓者", "开拓者·存护"}}
	if parts, ok := a.staticParts(panel, &trailblazer); !ok || len(parts) != 1 || parts[0][0].Type != "image" {
		t.Fatal(parts, ok)
	}
	if _, ok := a.staticParts(panel, &Entry{Name: "黄泉"}); ok {
		t.Fatal("found a character without an image")
	}
	curve := StaticPicture{Source: "starrail-plugin", Parts: []StaticPart{{Text: "说明"}, {Text: "暴击", Path: "resources/srsr/sy/1.jpg"}}}
	if parts, ok := a.staticParts(curve, nil); !ok || len(parts) != 2 || len(parts[1]) != 2 || parts[1][0].Data["text"] != "暴击" {
		t.Fatal(parts, ok)
	}
}
