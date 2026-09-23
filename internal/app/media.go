package app

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/RayleaBot/plugin-zzz/internal/localdata"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const mediaMaxBytes = 4 * 1024 * 1024

var mediaRefPattern = regexp.MustCompile(`^[A-Z2-7]{26}$`)
var mediaKinds = []string{"character", "weapon", "food", "enemy", "domain", "artifact", "material", "guide", "birthday", "photo", "other"}

// mediaLabels name the categories as the image library page does.
var mediaLabels = map[string]string{"character": "角色图鉴", "weapon": "装备", "food": "食物", "enemy": "怪物", "domain": "秘境", "artifact": "套装", "material": "材料", "guide": "攻略", "birthday": "生日", "photo": "角色照片", "other": "其他"}

type MediaEntry struct {
	Ref       string `json:"ref"`
	Revision  uint64 `json:"revision"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	CatalogID string `json:"catalog_id"`
	Source    string `json:"source"`
	License   string `json:"license"`
	MIME      string `json:"mime"`
	Bytes     int    `json:"bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	CreatedMS int64  `json:"created_ms"`
}
type mediaFile struct {
	Entry MediaEntry `json:"entry"`
	Data  []byte     `json:"data"`
}
type mediaUpload struct {
	entry   MediaEntry
	size    int
	data    []byte
	expires time.Time
	timer   *time.Timer
}
type MediaStore struct {
	mu        sync.Mutex
	Directory string
	uploads   map[string]*mediaUpload
	closed    bool
}

func (s *MediaStore) file(ref string) string { return filepath.Join(s.Directory, ref+".json") }
func (s *MediaStore) read(ref string) (mediaFile, error) {
	var v mediaFile
	if !mediaRefPattern.MatchString(ref) {
		return v, gameError("media_missing", "素材编号无效。")
	}
	if err := localdata.Read(s.file(ref), &v); err != nil {
		return v, err
	}
	if v.Entry.Ref != ref || len(v.Data) != v.Entry.Bytes {
		return v, gameError("media_missing", "素材不存在或文件不完整。")
	}
	return v, nil
}
func (s *MediaStore) metadata(ref string) (MediaEntry, error) {
	var entry MediaEntry
	f, err := os.Open(s.file(ref))
	if err != nil {
		return entry, err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 16384))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return entry, gameError("media_invalid", "素材元数据无效。")
	}
	key, err := d.Token()
	if err != nil || key != "entry" {
		return entry, gameError("media_invalid", "素材元数据顺序无效。")
	}
	if err = d.Decode(&entry); err != nil {
		return entry, err
	}
	if entry.Ref != ref || entry.Bytes < 12 || entry.Bytes > mediaMaxBytes {
		return entry, gameError("media_invalid", "素材元数据无效。")
	}
	return entry, nil
}
func (s *MediaStore) entries() ([]MediaEntry, error) {
	files, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return []MediaEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []MediaEntry{}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") || !mediaRefPattern.MatchString(strings.TrimSuffix(f.Name(), ".json")) {
			continue
		}
		entry, err := s.metadata(strings.TrimSuffix(f.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	slices.SortFunc(out, func(a, b MediaEntry) int {
		if a.CreatedMS > b.CreatedMS {
			return -1
		}
		if a.CreatedMS < b.CreatedMS {
			return 1
		}
		return strings.Compare(a.Ref, b.Ref)
	})
	return out, nil
}
func (s *MediaStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	for _, v := range s.uploads {
		v.timer.Stop()
	}
	s.uploads = nil
}
func mediaInfo(raw []byte) (string, int, int, error) {
	return imageInfo(raw, mediaMaxBytes)
}

// imageInfo checks a picture of at most limit bytes and returns its MIME
// type and size.
func imageInfo(raw []byte, limit int) (string, int, int, error) {
	if len(raw) < 12 || len(raw) > limit {
		return "", 0, 0, gameError("media_invalid", "图片大小无效。")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	width, height := config.Width, config.Height
	if err != nil && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "WEBP" {
		width, height, err = webpDimensions(raw)
		format = "webp"
	}
	if err != nil || !slices.Contains([]string{"png", "jpeg", "gif", "webp"}, format) || width < 1 || height < 1 || int64(width)*int64(height) > 24000000 {
		return "", 0, 0, gameError("media_invalid", "仅支持有效 PNG/JPEG/GIF/WebP，最大2400万像素。")
	}
	return "image/" + format, width, height, nil
}

// Read RIFF dimensions only. Original pixels are decoded by the displaying client.
func webpDimensions(raw []byte) (int, int, error) {
	bad := fmt.Errorf("invalid webp container")
	if len(raw) < 20 || uint64(binary.LittleEndian.Uint32(raw[4:8]))+8 != uint64(len(raw)) {
		return 0, 0, bad
	}
	w, h := 0, 0
	pixels := false
	u24 := func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 }
	for at := 12; at < len(raw); {
		if at+8 > len(raw) {
			return 0, 0, bad
		}
		n := int(binary.LittleEndian.Uint32(raw[at+4 : at+8]))
		end := at + 8 + n
		if n < 0 || end > len(raw) || end+(n&1) > len(raw) {
			return 0, 0, bad
		}
		p := raw[at+8 : end]
		switch string(raw[at : at+4]) {
		case "VP8X":
			if at != 12 || len(p) != 10 {
				return 0, 0, bad
			}
			w, h = u24(p[4:7])+1, u24(p[7:10])+1
		case "VP8 ":
			if len(p) < 10 || p[0]&1 != 0 || !bytes.Equal(p[3:6], []byte{0x9d, 0x01, 0x2a}) {
				return 0, 0, bad
			}
			if w == 0 {
				w, h = int(binary.LittleEndian.Uint16(p[6:8])&0x3fff), int(binary.LittleEndian.Uint16(p[8:10])&0x3fff)
			}
			pixels = true
		case "VP8L":
			if len(p) < 5 || p[0] != 0x2f {
				return 0, 0, bad
			}
			n := binary.LittleEndian.Uint32(p[1:5])
			if n>>29 != 0 {
				return 0, 0, bad
			}
			if w == 0 {
				w, h = int(n&0x3fff)+1, int((n>>14)&0x3fff)+1
			}
			pixels = true
		case "ANMF":
			if len(p) < 24 || w == 0 {
				return 0, 0, bad
			}
			pixels = true
		}
		if n&1 != 0 && raw[end] != 0 {
			return 0, 0, bad
		}
		at = end + (n & 1)
	}
	if !pixels || w == 0 || h == 0 {
		return 0, 0, bad
	}
	return w, h, nil
}
func validateMediaEntry(entry MediaEntry, catalog Catalog) error {
	if len([]rune(strings.TrimSpace(entry.Title))) < 1 || len([]rune(entry.Title)) > 120 || !slices.Contains(mediaKinds, entry.Category) || len([]rune(entry.Source)) > 512 || len([]rune(strings.TrimSpace(entry.License))) < 1 || len([]rune(entry.License)) > 256 {
		return gameError("input_invalid", "请填写有效素材名称、分类、来源及使用许可。")
	}
	if entry.CatalogID != "" {
		if _, ok := catalog.Get(entry.CatalogID); !ok {
			return gameError("input_invalid", "关联资料条目不存在。")
		}
	}
	return nil
}
func (a *App) mediaAction(action string, input map[string]any) (map[string]any, error) {
	s := a.Media
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, gameError("media_missing", "图库已停止。")
	}
	if action == "media.upload.start" {
		var q struct {
			MediaEntry
			Size   int  `json:"size"`
			Rights bool `json:"rights_confirmed"`
		}
		if decodeObject(input, &q) != nil || !q.Rights || q.Size < 12 || q.Size > mediaMaxBytes {
			return nil, gameError("input_invalid", "请选择不超过4 MiB的图片并确认使用许可。")
		}
		if err := validateMediaEntry(q.MediaEntry, a.Catalog); err != nil {
			return nil, err
		}
		if s.uploads == nil {
			s.uploads = map[string]*mediaUpload{}
		}
		if len(s.uploads) >= 8 {
			return nil, gameError("media_limit", "上传会话已满，请完成或取消其他上传。")
		}
		ref := rand.Text()
		entry := MediaEntry{Ref: ref, Title: q.Title, Category: q.Category, CatalogID: q.CatalogID, Source: q.Source, License: q.License, Revision: 1, CreatedMS: time.Now().UnixMilli()}
		v := &mediaUpload{entry: entry, size: q.Size, expires: time.Now().Add(10 * time.Minute)}
		v.timer = time.AfterFunc(10*time.Minute, func() { s.mu.Lock(); defer s.mu.Unlock(); delete(s.uploads, ref) })
		s.uploads[ref] = v
		return map[string]any{"ref": ref, "offset": 0}, nil
	}
	if strings.HasPrefix(action, "media.upload.") {
		ref := asText(input["ref"])
		v := s.uploads[ref]
		if v == nil {
			if action == "media.upload.finish" && mediaRefPattern.MatchString(ref) {
				if entry, err := s.metadata(ref); err == nil {
					return map[string]any{"entry": entry}, nil
				}
			}
			return nil, gameError("media_missing", "上传已失效，请重新开始。")
		}
		if action == "media.upload.cancel" {
			v.timer.Stop()
			delete(s.uploads, ref)
			return map[string]any{"canceled": true}, nil
		}
		if time.Now().After(v.expires) {
			v.timer.Stop()
			delete(s.uploads, ref)
			return nil, gameError("media_missing", "上传已过期。")
		}
		if action == "media.upload.append" {
			var q struct {
				Offset int    `json:"offset"`
				Data   string `json:"data"`
			}
			if decodeObject(input, &q) != nil || q.Offset < 0 || len(q.Data) > 65536 {
				return nil, gameError("input_invalid", "图片分块无效。")
			}
			raw, err := base64.StdEncoding.DecodeString(q.Data)
			if err != nil || len(raw) == 0 || len(raw) > 49152 || q.Offset+len(raw) > v.size {
				return nil, gameError("input_invalid", "图片分块无效。")
			}
			if q.Offset < len(v.data) {
				if q.Offset+len(raw) > len(v.data) || !bytes.Equal(v.data[q.Offset:q.Offset+len(raw)], raw) {
					return nil, gameError("media_sequence", "重复图片分块内容不一致。")
				}
			} else if q.Offset != len(v.data) {
				return nil, gameError("media_sequence", "图片上传偏移不连续。")
			} else {
				v.data = append(v.data, raw...)
			}
			return map[string]any{"offset": len(v.data)}, nil
		}
		if action != "media.upload.finish" || len(v.data) != v.size {
			return nil, gameError("media_sequence", "图片还未上传完整。")
		}
		mime, w, h, err := mediaInfo(v.data)
		if err != nil {
			return nil, err
		}
		entries, err := s.entries()
		if err != nil {
			return nil, err
		}
		total := v.size
		for _, e := range entries {
			total += e.Bytes
		}
		if len(entries) >= 2000 || total > 512*1024*1024 {
			return nil, gameError("media_limit", "图库超过2000项或512 MiB上限，请清理后再上传。")
		}
		v.entry.MIME = mime
		v.entry.Width = w
		v.entry.Height = h
		v.entry.Bytes = v.size
		if err = localdata.Write(s.file(ref), mediaFile{v.entry, v.data}); err != nil {
			return nil, err
		}
		v.timer.Stop()
		delete(s.uploads, ref)
		return map[string]any{"entry": v.entry}, nil
	}
	if action == "media.list" {
		var q struct {
			Query     string `json:"query"`
			Category  string `json:"category"`
			CatalogID string `json:"catalog_id"`
			Offset    int    `json:"offset"`
		}
		if decodeObject(input, &q) != nil || q.Offset < 0 || len(q.Query) > 512 {
			return nil, gameError("input_invalid", "图库筛选无效。")
		}
		all, err := s.entries()
		if err != nil {
			return nil, err
		}
		items := []MediaEntry{}
		totalBytes := 0
		for _, e := range all {
			totalBytes += e.Bytes
			if q.Category != "" && e.Category != q.Category || q.CatalogID != "" && e.CatalogID != q.CatalogID || q.Query != "" && !strings.Contains(strings.ToLower(e.Title+" "+e.Source+" "+e.CatalogID), strings.ToLower(q.Query)) {
				continue
			}
			items = append(items, e)
		}
		start, end := min(q.Offset, len(items)), min(q.Offset+50, len(items))
		var next *int
		if end < len(items) {
			next = &end
		}
		return map[string]any{"items": items[start:end], "total": len(items), "next_offset": next, "total_bytes": totalBytes, "all_count": len(all), "categories": mediaKinds}, nil
	}
	ref := asText(input["ref"])
	v, err := s.read(ref)
	if err != nil {
		return nil, err
	}
	switch action {
	case "media.read":
		var q struct {
			Offset int `json:"offset"`
		}
		if decodeObject(input, &q) != nil || q.Offset < 0 || q.Offset > len(v.Data) {
			return nil, gameError("input_invalid", "原图偏移无效。")
		}
		end := min(q.Offset+49152, len(v.Data))
		return map[string]any{"entry": v.Entry, "data": base64.StdEncoding.EncodeToString(v.Data[q.Offset:end]), "offset": end, "more": end < len(v.Data)}, nil
	case "media.update", "media.remove":
		var q struct {
			MediaEntry
			Confirm bool `json:"confirm"`
		}
		if decodeObject(input, &q) != nil || q.Revision != v.Entry.Revision {
			return nil, gameError("media_changed", "素材已变化，请刷新。")
		}
		if action == "media.remove" {
			if !q.Confirm {
				return nil, gameError("input_invalid", "请确认删除此素材。")
			}
			if err = os.Remove(s.file(ref)); err != nil {
				return nil, err
			}
			return map[string]any{"removed": true}, nil
		}
		if err = validateMediaEntry(q.MediaEntry, a.Catalog); err != nil {
			return nil, err
		}
		v.Entry.Title = q.Title
		v.Entry.Category = q.Category
		v.Entry.CatalogID = q.CatalogID
		v.Entry.Source = q.Source
		v.Entry.License = q.License
		v.Entry.Revision++
		if err = localdata.Write(s.file(ref), v); err != nil {
			return nil, err
		}
		return map[string]any{"entry": v.Entry}, nil
	}
	return nil, gameError("operation_denied", "图库操作不存在。")
}
