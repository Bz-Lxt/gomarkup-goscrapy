package renderer

import (
	"strings"

	"goscrapy/internal/domtree"
	"goscrapy/internal/model"
)

const (
	ViewportW = 1440
	ViewportH = 900
)

func AssignBoxes(tree *domtree.Tree) {
	if tree == nil || tree.Root == nil {
		return
	}
	tree.Width = ViewportW
	tree.Height = ViewportH
	y := 16.0
	assign(tree.Root, 24, &y, ViewportW-48)
}

func assign(n *domtree.Node, x float64, y *float64, maxW float64) {
	if n == nil {
		return
	}
	tag := n.Tag
	h := estimateHeight(n)
	w := maxW
	if tag == "article" || hasClass(n, "product-card") {
		w = 280
		h = 160
	}
	if tag == "h1" {
		h = 40
	}
	if tag == "h2" || hasClass(n, "title") {
		h = 32
		w = minf(w, 280)
	}
	if hasClass(n, "price") || hasClass(n, "sku") {
		h = 22
		w = minf(w, 160)
	}
	if tag == "a" {
		h = 24
		w = minf(w, 200)
	}
	if tag == "header" {
		h = 56
		w = ViewportW
		x = 0
	}
	n.Box = model.Box{X: x, Y: *y, W: w, H: h}
	if isBlock(n) {
		*y += h + 8
	}
	cx := x + 8
	cy := n.Box.Y + 8
	rowY := cy
	col := 0
	for _, c := range n.Children {
		if hasClass(n, "grid") || hasClass(n, "product-card") && c.Tag == "article" {
			// handled below
		}
		if hasClass(n, "grid") {
			cx = 24 + float64(col%4)*300
			if col > 0 && col%4 == 0 {
				rowY += 180
			}
			cy = rowY
			assignAt(c, cx, cy, 280)
			col++
			continue
		}
		assign(c, x+12, y, maxW-16)
	}
	if hasClass(n, "grid") {
		*y = rowY + 180
	}
}

func assignAt(n *domtree.Node, x, y, w float64) {
	if n == nil {
		return
	}
	n.Box = model.Box{X: x, Y: y, W: w, H: estimateHeight(n)}
	if hasClass(n, "product-card") || n.Tag == "article" {
		n.Box.H = 160
	}
	inner := y + 12
	for _, c := range n.Children {
		c.Box = model.Box{X: x + 12, Y: inner, W: w - 24, H: estimateHeight(c)}
		if c.Tag == "h2" || hasClass(c, "title") {
			c.Box.H = 32
		}
		inner += c.Box.H + 6
		assignAtChildren(c)
	}
}

func assignAtChildren(n *domtree.Node) {
	for _, c := range n.Children {
		c.Box = model.Box{X: n.Box.X, Y: n.Box.Y, W: n.Box.W, H: estimateHeight(c)}
	}
}

func estimateHeight(n *domtree.Node) float64 {
	if n == nil {
		return 18
	}
	switch n.Tag {
	case "h1":
		return 40
	case "h2":
		return 32
	case "p":
		return 28
	case "header":
		return 56
	case "article":
		return 160
	case "section":
		return 40
	default:
		if n.OwnText != "" {
			return 22
		}
		return 18
	}
}

func isBlock(n *domtree.Node) bool {
	switch n.Tag {
	case "div", "section", "article", "header", "main", "h1", "h2", "p", "ul", "li":
		return true
	}
	return hasClass(n, "product-card") || hasClass(n, "grid")
}

func hasClass(n *domtree.Node, name string) bool {
	return n != nil && n.HasClass(name)
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func VisibleNodes(tree *domtree.Tree) []model.SnapshotNode {
	out := make([]model.SnapshotNode, 0)
	if tree == nil {
		return out
	}
	for _, n := range tree.Visible() {
		if n.Box.W <= 0 || n.Box.H <= 0 {
			continue
		}
		if n.Tag == "body" || n.Tag == "html" {
			continue
		}
		text := n.OwnText
		if text == "" {
			text = n.Text
		}
		if len(text) > 80 {
			text = text[:80]
		}
		if strings.TrimSpace(text) == "" && n.Tag != "article" && n.Tag != "section" && n.Tag != "img" {
			continue
		}
		out = append(out, model.SnapshotNode{
			NodeID: n.ID,
			Tag:    n.Tag,
			Text:   text,
			Box:    n.Box,
		})
	}
	return out
}
