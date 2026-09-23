package app

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"
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
func TestMediaAtomicChunksOriginalBytesEditAndRemoval(t *testing.T) {
	a := App{Media: &MediaStore{Directory: t.TempDir()}}
	defer a.Close()
	raw := pngFixture(t)
	begin, err := a.mediaAction("media.upload.start", map[string]any{"title": "合成图片", "category": "photo", "license": "test fixture", "size": len(raw), "rights_confirmed": true})
	if err != nil {
		t.Fatal(err)
	}
	ref := begin["ref"]
	if _, err = a.mediaAction("media.upload.finish", map[string]any{"ref": ref}); err == nil {
		t.Fatal("partial saved")
	}
	part := map[string]any{"ref": ref, "offset": 0, "data": base64.StdEncoding.EncodeToString(raw[:20])}
	if _, err = a.mediaAction("media.upload.append", part); err != nil {
		t.Fatal(err)
	}
	if _, err = a.mediaAction("media.upload.append", part); err != nil {
		t.Fatal("consistent replay rejected", err)
	}
	part["data"] = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 20))
	if _, err = a.mediaAction("media.upload.append", part); err == nil {
		t.Fatal("conflicting replay accepted")
	}
	if _, err = a.mediaAction("media.upload.append", map[string]any{"ref": ref, "offset": 20, "data": base64.StdEncoding.EncodeToString(raw[20:])}); err != nil {
		t.Fatal(err)
	}
	saved, err := a.mediaAction("media.upload.finish", map[string]any{"ref": ref})
	if err != nil {
		t.Fatal(err)
	}
	entry := saved["entry"].(MediaEntry)
	if entry.Width != 3 || entry.Height != 2 || entry.MIME != "image/png" {
		t.Fatal(entry)
	}
	read, err := a.mediaAction("media.read", map[string]any{"ref": ref})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := base64.StdEncoding.DecodeString(read["data"].(string))
	if !bytes.Equal(raw, data) {
		t.Fatal("original changed")
	}
	listed, err := a.mediaAction("media.list", nil)
	if err != nil || listed["all_count"] != 1 {
		t.Fatal(listed, err)
	}
	if _, err = a.mediaAction("media.remove", map[string]any{"ref": ref, "revision": 0, "confirm": true}); err == nil {
		t.Fatal("stale remove")
	}
	if _, err = a.mediaAction("media.update", map[string]any{"ref": ref, "revision": 1, "title": "修改标题", "category": "guide", "license": "fixture"}); err != nil {
		t.Fatal(err)
	}
	if _, err = a.mediaAction("media.remove", map[string]any{"ref": ref, "revision": 2, "confirm": true}); err != nil {
		t.Fatal(err)
	}
	if _, err = a.mediaAction("media.read", map[string]any{"ref": "../outside"}); err == nil {
		t.Fatal("path accepted")
	}
	if _, err = a.mediaAction("media.read", map[string]any{"ref": ref}); err == nil {
		t.Fatal("removed image read")
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
	mime, w, h, err := mediaInfo(v)
	if err != nil || mime != "image/webp" || w != 300 || h != 200 {
		t.Fatal(mime, w, h, err)
	}
	v[4]++
	if _, _, _, err = mediaInfo(v); err == nil {
		t.Fatal("inconsistent RIFF accepted")
	}
	if _, _, _, err = mediaInfo([]byte("<svg width=10 height=10/>")); err == nil {
		t.Fatal("unsupported image accepted")
	}
}
