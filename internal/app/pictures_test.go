package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RayleaBot/zzz/internal/artwork"
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
	a := App{Artwork: &artwork.Store{Root: root}, Game: Game{Prefix: "%", Pictures: Pictures{
		Atlas: []PictureSource{{Source: "zzz-atlas", Index: "path.json"}},
	}}}
	if hint := a.pictureHint(a.Game.Pictures.Atlas); !strings.Contains(hint, "%素材更新 zzz-atlas") {
		t.Fatal(hint)
	}
	// Atlas searches path.json's modules in the file's order.
	writeArtwork(t, root, "zzz-atlas/path.json", `{"weapon":{"深海访客":"/weapon/深海访客.png"},"guide":{"深海访客":"/guide/深海访客.png"}}`)
	writeArtwork(t, root, "zzz-atlas/weapon/深海访客.png", "a")
	writeArtwork(t, root, "zzz-atlas/guide/深海访客.png", "b")
	if file, ok := a.atlasPicture("深海访客图鉴", nil); !ok || file != (artworkFile{"zzz-atlas", "weapon/深海访客.png"}) {
		t.Fatal(file, ok)
	}
	if _, ok := a.atlasPicture("不存在图鉴", nil); ok {
		t.Fatal("found a missing name")
	}
	if hint := a.pictureHint(a.Game.Pictures.Atlas); hint != "" {
		t.Fatal(hint)
	}
}

func TestAtlasFindsPicturesByTheLibraryAliases(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}, Game: Game{Pictures: Pictures{Atlas: []PictureSource{{Source: "atlas", Index: "path.json",
		Rules: map[string]AtlasRule{"guide for role": {Condition: 5, Pick: []string{"攻略"}}}}}}}}
	writeArtwork(t, root, "atlas/guide/1001.png", "guide")
	if _, ok := a.atlasPicture("role图鉴", nil); ok {
		t.Error("sent an alias file")
	}
	// Star Rail's library keys pictures by ID; othername lists each key's
	// names.
	writeArtwork(t, root, "atlas/path.json", `{"guide for role":{"三月七":"/guide/1001.png"},"othername":{"role":"/othername/role.yaml"},"role":{"1001":"/role/1001.png","1002":"/role/1002.png"}}`)
	writeArtwork(t, root, "atlas/othername/role.yaml", "# roles\n'1001':\n  - 三月七\n  - mar7th\n\"1002\":\n- 丹恒\n- 三月七\n")
	writeArtwork(t, root, "atlas/role/1001.png", "a")
	writeArtwork(t, root, "atlas/role/1002.png", "b")
	// The guide module answers 攻略, not 图鉴.
	for name, want := range map[string]string{"三月七": "role/1001.png", "mar7th": "role/1001.png", "丹恒": "role/1002.png", "1002": "role/1002.png"} {
		if file, ok := a.atlasPicture(name+"图鉴", nil); !ok || file.Path != want {
			t.Errorf("%s: %+v %v", name, file, ok)
		}
	}
	if file, ok := a.atlasPicture("三月七攻略", nil); !ok || file.Path != "guide/1001.png" {
		t.Errorf("guide: %+v %v", file, ok)
	}
}

func TestAtlasRulesPickTheModule(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}, Game: Game{Pictures: Pictures{Atlas: []PictureSource{{Source: "zzz-atlas", Index: "path.json",
		Rules: map[string]AtlasRule{"config": {Condition: 3, Pick: []string{"图鉴"}}, "material for role": {Condition: 4, Pick: []string{"突破", "材料", "素材", "培养"}}}}}}}}
	writeArtwork(t, root, "zzz-atlas/path.json", `{"role":{"艾莲":"/role/艾莲.png"},"material for role":{"艾莲":"/material for role/艾莲.png"}}`)
	writeArtwork(t, root, "zzz-atlas/role/艾莲.png", "a")
	writeArtwork(t, root, "zzz-atlas/material for role/艾莲.png", "b")
	// Any word after the prefix reaches the modules with condition 3; the
	// materials module needs its own words.
	for word, want := range map[string]string{"艾莲突破": "material for role/艾莲.png", "艾莲图鉴": "role/艾莲.png", "艾莲": "role/艾莲.png"} {
		if file, ok := a.atlasPicture(word, nil); !ok || file.Path != want {
			t.Errorf("%s: %+v %v", word, file, ok)
		}
	}
	if _, ok := a.atlasPicture("不存在突破", nil); ok {
		t.Fatal("found a missing name")
	}
}
