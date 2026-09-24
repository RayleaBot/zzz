package app

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func pngFixture(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.RGBA{255, 100, 10, 255})
	var raw bytes.Buffer
	if err := png.Encode(&raw, img); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes()
}

func TestPanelImagesKeepIDsInUploadOrder(t *testing.T) {
	store := &PanelImages{Directory: t.TempDir()}
	if store.Random("1191") != "" {
		t.Fatal("a character without pictures has one")
	}
	for range 3 {
		if err := store.Add("1191", pngFixture(t)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Add("1191", []byte("not a picture at all")); err == nil {
		t.Fatal("a file that is no picture was kept")
	}
	if err := store.Add("../1191", pngFixture(t)); err == nil {
		t.Fatal("a character outside the store was accepted")
	}
	names, err := store.List("1191")
	if err != nil || len(names) != 3 {
		t.Fatalf("names = %v %v", names, err)
	}
	if path := store.Random("1191"); !strings.HasPrefix(path, "panel-images/1191/") || !strings.HasSuffix(path, ".png") {
		t.Fatalf("random = %s", path)
	}
	// IDs follow the upload order; unknown and repeated IDs fail.
	removed, failed, err := store.Remove("1191", []string{"2", "2", "9", "x"})
	if err != nil || !reflect.DeepEqual(removed, []string{"2"}) || !reflect.DeepEqual(failed, []string{"2", "9", "x"}) {
		t.Fatalf("removed %v failed %v %v", removed, failed, err)
	}
	left, _ := store.List("1191")
	if !reflect.DeepEqual(left, []string{names[0], names[2]}) {
		t.Fatalf("left %v of %v", left, names)
	}
}

func TestPanelPortraitIsTheOriginalImage(t *testing.T) {
	for path, want := range map[string]string{"panel-images/1191/1700000000000.png": "panel:1191/1700000000000.png", "assets/zzzerouid/role/IconRole01.png": "artwork:zzzerouid/role/IconRole01.png"} {
		image := Image{Resources: []rayleabot.RenderImageResource{{ID: "weapon-icon", Path: "assets/zzzerouid/weapon/w.png"}, {ID: "role-icon", Path: path}}}
		if got := panelPortrait(image); got != want {
			t.Errorf("%s = %s", path, got)
		}
	}
	if panelPortrait(Image{}) != "" {
		t.Error("a panel without a portrait has one")
	}
}

func TestWebPDimensionsAndInvalidContainer(t *testing.T) {
	makeWebp := func(kind string, data []byte) []byte {
		n := len(data) + (len(data) & 1)
		raw := make([]byte, 20+n)
		copy(raw, "RIFF")
		binary.LittleEndian.PutUint32(raw[4:8], uint32(len(raw)-8))
		copy(raw[8:12], "WEBP")
		copy(raw[12:16], kind)
		binary.LittleEndian.PutUint32(raw[16:20], uint32(len(data)))
		copy(raw[20:], data)
		return raw
	}
	lossless := []byte{0x2f, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(lossless[1:], uint32(299|(199<<14)))
	v := makeWebp("VP8L", lossless)
	mime, w, h, err := imageInfo(v, panelImageLimit)
	if err != nil || mime != "image/webp" || w != 300 || h != 200 {
		t.Fatal(mime, w, h, err)
	}
	v[4]++
	if _, _, _, err = imageInfo(v, panelImageLimit); err == nil {
		t.Fatal("inconsistent RIFF accepted")
	}
	if _, _, _, err = imageInfo([]byte("<svg width=10 height=10/>"), panelImageLimit); err == nil {
		t.Fatal("unsupported image accepted")
	}
}
