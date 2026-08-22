package domtree

import (
	"bytes"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"goscrapy/internal/model"
)

type Node struct {
	ID       int
	Tag      string
	IDAttr   string
	Classes  []string
	Attrs    map[string]string
	Text     string
	OwnText  string
	Box      model.Box
	Parent   *Node
	Children []*Node
	Index    int // sibling index among element siblings
	Depth    int
	Raw      *html.Node
}

type Tree struct {
	Root   *Node
	ByID   map[int]*Node
	HTML   string
	Width  int
	Height int
}

func Parse(raw []byte) (*Tree, error) {
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	t := &Tree{ByID: map[int]*Node{}, HTML: string(raw), Width: 1440, Height: 900}
	var next int
	var walk func(n *html.Node, parent *Node, depth int)
	walk = func(n *html.Node, parent *Node, depth int) {
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, parent, depth)
			}
			return
		}
		tag := strings.ToLower(n.Data)
		if tag == "script" || tag == "style" || tag == "noscript" || tag == "svg" {
			return
		}
		next++
		node := &Node{
			ID:      next,
			Tag:     tag,
			Attrs:   map[string]string{},
			Parent:  parent,
			Depth:   depth,
			Raw:     n,
			OwnText: ownText(n),
		}
		for _, a := range n.Attr {
			k := strings.ToLower(a.Key)
			node.Attrs[k] = a.Val
			if k == "id" {
				node.IDAttr = a.Val
			}
			if k == "class" {
				node.Classes = splitClass(a.Val)
			}
		}
		if parent != nil {
			node.Index = len(parent.Children)
			parent.Children = append(parent.Children, node)
		} else if t.Root == nil {
			t.Root = node
		}
		t.ByID[node.ID] = node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, node, depth+1)
		}
		node.Text = strings.Join(strings.Fields(innerText(n)), " ")
		if len(node.Text) > 80 {
			node.Text = node.Text[:80]
		}
	}
	walk(doc, nil, 0)
	return t, nil
}

func ParseQuery(raw []byte) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(bytes.NewReader(raw))
}

func splitClass(s string) []string {
	parts := strings.Fields(s)
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func ownText(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func innerText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func (n *Node) Attr(key string) string {
	if n == nil || n.Attrs == nil {
		return ""
	}
	return n.Attrs[strings.ToLower(key)]
}

func (n *Node) HasClass(name string) bool {
	for _, c := range n.Classes {
		if c == name {
			return true
		}
	}
	return false
}

func (n *Node) DataAttrs() []string {
	out := make([]string, 0)
	for k := range n.Attrs {
		if strings.HasPrefix(k, "data-") && n.Attrs[k] != "" {
			out = append(out, k)
		}
	}
	return out
}

func (t *Tree) Visible() []*Node {
	out := make([]*Node, 0, len(t.ByID))
	for id := 1; id <= len(t.ByID); id++ {
		n := t.ByID[id]
		if n == nil {
			continue
		}
		if n.Tag == "html" || n.Tag == "head" || n.Tag == "meta" || n.Tag == "link" || n.Tag == "title" {
			continue
		}
		out = append(out, n)
	}
	return out
}
