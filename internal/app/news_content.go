package app

import (
	"encoding/json"
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A 米游社 post's content is the editor's HTML. The Yunzai 原神插件 pastes it
// into html/mysNews as is; the renderer here may reach the network, so the
// content is rebuilt from the tags and classes the page styles, with images
// read only through render resources.

var (
	newsKeptTags = []string{"p", "div", "span", "br", "strong", "b", "em", "i", "u", "s", "del", "sub", "sup", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "blockquote", "pre", "code", "hr", "table", "thead", "tbody", "tr", "th", "td", "a", "label", "section", "figure", "figcaption"}
	// newsDroppedTags lose their content too; other unknown tags keep it.
	newsDroppedTags   = []string{"script", "style", "iframe", "frame", "object", "embed", "noscript", "template", "svg", "math", "video", "audio", "canvas", "form", "input", "button", "select", "textarea", "link", "meta", "title", "head"}
	newsStyles        = []string{"color", "background-color", "text-align", "font-weight", "font-style", "text-decoration", "font-size"}
	newsStyleToken    = `(?:#[0-9a-fA-F]{3,8}|rgba?\([0-9., %]+\)|[a-zA-Z-]+|-?[0-9.]+(?:px|em|rem|%)?)`
	newsStyleValue    = regexp.MustCompile(`^` + newsStyleToken + `(?:\s+` + newsStyleToken + `)*$`)
	newsClassName     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	newsEmoticonToken = regexp.MustCompile(`_\([^)]*\)`)
)

// newsContent rebuilds a post's content for the page: images of the post
// are resource IDs from resource, "" leaving them out. Linked pictures carry
// no address in the HTML, so an empty one in an image block takes the
// structured content's picture of the same block. _(name) becomes that
// emoticon, or nothing, as upstream.
func newsContent(content string, pictures []string, emoticons map[string]string, resource func(url string) string) string {
	nodes, err := nethtml.ParseFragment(strings.NewReader(content), &nethtml.Node{Type: nethtml.ElementNode, DataAtom: atom.Div, Data: "div"})
	if err != nil {
		return ""
	}
	var out strings.Builder
	block := 0
	var write func(*nethtml.Node, bool)
	write = func(node *nethtml.Node, inImageBox bool) {
		switch node.Type {
		case nethtml.TextNode:
			last := 0
			for _, at := range newsEmoticonToken.FindAllStringIndex(node.Data, -1) {
				out.WriteString(html.EscapeString(node.Data[last:at[0]]))
				if icon := emoticons[node.Data[at[0]+2:at[1]-1]]; icon != "" {
					if id := resource(icon); id != "" {
						out.WriteString(`<img class="emoticon-image" data-render-resource="` + id + `">`)
					}
				}
				last = at[1]
			}
			out.WriteString(html.EscapeString(node.Data[last:]))
			return
		case nethtml.ElementNode:
		default:
			return
		}
		tag := node.Data
		if slices.Contains(newsDroppedTags, tag) {
			return
		}
		class, style, src := "", "", ""
		for _, attr := range node.Attr {
			switch attr.Key {
			case "class":
				class = newsClass(attr.Val)
			case "style":
				style = newsStyle(attr.Val)
			case "src":
				src = attr.Val
			}
		}
		if tag == "img" {
			if inImageBox && src == "" && block <= len(pictures) {
				src = pictures[block-1]
			}
			if id := resource(src); id != "" {
				out.WriteString("<img")
				if class != "" {
					out.WriteString(` class="` + class + `"`)
				}
				out.WriteString(` data-render-resource="` + id + `">`)
			}
			return
		}
		kept := slices.Contains(newsKeptTags, tag)
		if kept {
			out.WriteString("<" + tag)
			if class != "" {
				out.WriteString(` class="` + class + `"`)
			}
			if style != "" {
				out.WriteString(` style="` + style + `"`)
			}
			out.WriteString(">")
			if tag == "br" || tag == "hr" {
				return
			}
		}
		if slices.Contains(strings.Fields(class), "ql-image-box") {
			inImageBox = true
			block++
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			write(child, inImageBox)
		}
		if kept {
			out.WriteString("</" + tag + ">")
		}
	}
	for _, node := range nodes {
		write(node, false)
	}
	return out.String()
}

func newsClass(value string) string {
	names := []string{}
	for _, name := range strings.Fields(value) {
		if newsClassName.MatchString(name) {
			names = append(names, name)
		}
	}
	return strings.Join(names, " ")
}

func newsStyle(value string) string {
	rules := []string{}
	for _, rule := range strings.Split(value, ";") {
		name, value, ok := strings.Cut(rule, ":")
		name, value = strings.ToLower(strings.TrimSpace(name)), strings.TrimSpace(value)
		if ok && slices.Contains(newsStyles, name) && newsStyleValue.MatchString(value) {
			rules = append(rules, name+": "+value)
		}
	}
	return strings.Join(rules, "; ")
}

// newsStructuredPictures are the pictures of a post's structured content in
// order, one for each image block.
func newsStructuredPictures(structured string) []string {
	var ops []any
	if json.Unmarshal([]byte(structured), &ops) != nil {
		return nil
	}
	pictures := []string{}
	for _, op := range ops {
		if picture := asText(asObject(asObject(op)["insert"])["image"]); picture != "" {
			pictures = append(pictures, picture)
		}
	}
	return pictures
}

// newsQRCode draws the post's link as html/mysNews's qrcode.js does: 130
// pixels, error correction level H, no quiet zone.
func newsQRCode(link string) string {
	code, err := qrcode.New(link, qrcode.High)
	if err != nil {
		return ""
	}
	code.DisableBorder = true
	bitmap := code.Bitmap()
	size := len(bitmap)
	var path strings.Builder
	for y, row := range bitmap {
		for x, dark := range row {
			if dark {
				path.WriteString("M" + strconv.Itoa(x) + " " + strconv.Itoa(y) + "h1v1h-1z")
			}
		}
	}
	return `<svg xmlns="http://www.w3.org/2000/svg" width="130" height="130" viewBox="0 0 ` + strconv.Itoa(size) + " " + strconv.Itoa(size) + `" shape-rendering="crispEdges"><rect width="100%" height="100%" fill="#ffffff"/><path fill="#000000" d="` + path.String() + `"/></svg>`
}
