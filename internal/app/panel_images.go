package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ZZZ-Plugin's custom panel pictures: 上传艾莲面板图 keeps the pictures of the
// message, or of the message it quotes including a forwarded one, for the
// character; 查看艾莲面板图 lists them five a page with their IDs, and
// 删除艾莲面板图1,2 removes by ID. A panel shows a random one of its
// character's pictures instead of the default portrait.

// panelImageLimit keeps a picture, once encoded, within the host's message
// frame when it is listed.
const panelImageLimit = 5 << 20

// PanelImages keeps the pictures under Directory/<character ID>/, in name
// order, which is the order of their IDs.
type PanelImages struct {
	mu        sync.Mutex
	Directory string
}

// panelImagesDir is where the pictures are, relative to the plugin data
// directory, as render resources name them.
const panelImagesDir = "panel-images"

func (s *PanelImages) folder(id string) (string, error) {
	if _, err := strconv.Atoi(id); err != nil {
		return "", gameError("input_invalid", "角色无效。")
	}
	return filepath.Join(s.Directory, id), nil
}

func (s *PanelImages) list(id string) ([]string, error) {
	folder, err := s.folder(id)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(folder)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, entry := range entries {
		if entry.Type().IsRegular() && !strings.HasPrefix(entry.Name(), ".") && slices.Contains(pictureExtensions, strings.ToLower(filepath.Ext(entry.Name()))) {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

// List names a character's pictures in ID order.
func (s *PanelImages) List(id string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list(id)
}

// Add keeps a picture after its existing ones.
func (s *PanelImages) Add(id string, raw []byte) error {
	mime, _, _, err := imageInfo(raw, panelImageLimit)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	folder, err := s.folder(id)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(folder, 0o700); err != nil {
		return err
	}
	extension := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}[mime]
	// Names sort by the time they were added, so new pictures take the next
	// IDs.
	stamp := time.Now().UnixMilli()
	for {
		file := filepath.Join(folder, strconv.FormatInt(stamp, 10)+extension)
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			return os.WriteFile(file, raw, 0o600)
		}
		stamp++
	}
}

// Remove deletes pictures by ID, reporting the IDs removed and those that
// name no picture.
func (s *PanelImages) Remove(id string, ids []string) (removed, failed []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	names, err := s.list(id)
	if err != nil {
		return nil, nil, err
	}
	folder, _ := s.folder(id)
	for _, text := range ids {
		index, parseErr := strconv.Atoi(text)
		if parseErr != nil || index < 1 || index > len(names) || slices.Contains(removed, text) {
			failed = append(failed, text)
			continue
		}
		if err = os.Remove(filepath.Join(folder, names[index-1])); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, failed, err
		}
		removed = append(removed, text)
	}
	return removed, failed, nil
}

// panelPortrait is the 原图 reference of the portrait a panel image shows:
// a custom picture or a downloaded one; "" when it shows none.
func panelPortrait(image Image) string {
	for _, resource := range image.Resources {
		if resource.ID != "role-icon" {
			continue
		}
		if name, ok := strings.CutPrefix(resource.Path, panelImagesDir+"/"); ok {
			return "panel:" + name
		}
		if name, ok := strings.CutPrefix(resource.Path, "assets/"); ok {
			return "artwork:" + name
		}
	}
	return ""
}

// Read returns a picture's content.
func (s *PanelImages) Read(id, name string) ([]byte, error) {
	folder, err := s.folder(id)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(folder, filepath.Base(name)))
}

// Random returns one of the character's pictures as a path relative to the
// plugin data directory, or "" when it has none.
func (s *PanelImages) Random(id string) string {
	if s == nil {
		return ""
	}
	names, err := s.List(id)
	if err != nil || len(names) == 0 {
		return ""
	}
	return panelImagesDir + "/" + id + "/" + names[rand.IntN(len(names))]
}

var panelImageIDs = regexp.MustCompile(`[,，、\s]+`)

// panelImageCommand answers 上传面板图, 查看面板图 and 删除面板图.
func (a *App) panelImageCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if len(args) == 0 {
		return event.Result(map[string]any{"handled": false})
	}
	entry, ok := a.Catalog.Resolve(args[0], "character", a.aliasMap(event))
	if !ok {
		return event.SendText("未找到对应角色")
	}
	switch command {
	case "panel-image-upload":
		urls := messageImages(ctx, event.Event.Message.Segments, event.Actions())
		if len(urls) == 0 {
			return event.SendText("消息中未找到图片，请将要发送的图片与消息一同发送或引用要添加的图像。")
		}
		success, failed := 0, 0
		for _, link := range urls {
			raw, err := downloadFile(ctx, link, panelImageLimit)
			if err == nil {
				err = a.PanelImages.Add(entry.ID, raw)
			}
			if err != nil {
				failed++
				continue
			}
			success++
		}
		return event.SendText(fmt.Sprintf("成功上传%d张图片，失败%d张图片。", success, failed))
	case "panel-image-remove":
		ids := []string{}
		for _, text := range panelImageIDs.Split(strings.Join(args[1:], ","), -1) {
			if text != "" {
				ids = append(ids, text)
			}
		}
		if len(ids) == 0 {
			return event.SendText("请在命令后写上要删除的图片ID，如“" + a.Game.Prefix + "删除" + entry.Name + "面板图1,2”。")
		}
		removed, failed, err := a.PanelImages.Remove(entry.ID, ids)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		result := "无失败ID"
		if len(failed) > 0 {
			result = "删除失败ID为" + strings.Join(failed, ",")
		}
		return a.sendForward(ctx, event, [][]rayleabot.Segment{
			{rayleabot.Text("成功删除ID为" + strings.Join(removed, ",") + "的图片")},
			{rayleabot.Text(result)},
			{rayleabot.Text("删除后会重新排序ID，若想要再次删除，请重新获取图片列表，否则可能会删除错误的图片。")},
		})
	}
	names, err := a.PanelImages.List(entry.ID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if len(names) == 0 {
		return event.SendText(entry.Name + "暂无自定义面板图")
	}
	const pageSize = 5
	page := 1
	if len(args) > 1 {
		if parsed, err := strconv.Atoi(args[1]); err == nil && parsed > 1 {
			page = parsed
		}
	}
	start := (page - 1) * pageSize
	if start >= len(names) {
		return event.SendText("哪有这么多图片")
	}
	parts := [][]rayleabot.Segment{{rayleabot.Text(fmt.Sprintf("当前页数：%d，总页数：%d，查询指定页数请在指令后面加上页码。", page, (len(names)+pageSize-1)/pageSize))}}
	for index := start; index < min(start+pageSize, len(names)); index++ {
		part := []rayleabot.Segment{rayleabot.Text("ID：" + strconv.Itoa(index+1))}
		if raw, err := a.PanelImages.Read(entry.ID, names[index]); err == nil {
			part = append(part, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(raw)))
		}
		parts = append(parts, part)
	}
	parts = append(parts, []rayleabot.Segment{rayleabot.Text("删除或者添加后会重新排序ID，此时若想删除，请重新获取图片列表，否则可能会删除错误的图片。")})
	return a.sendForward(ctx, event, parts)
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
