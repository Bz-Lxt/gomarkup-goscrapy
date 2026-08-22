package selector

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"goscrapy/internal/domtree"
	"goscrapy/internal/model"
)

// Generalize lifts a single-element click into a repeating-list rule.
// It walks ancestors looking for a parent that has many structurally
// similar siblings (e.g. article.product-card).
func Generalize(tree *domtree.Tree, n *domtree.Node) *model.ListRule {
	if n == nil {
		return nil
	}
	best := (*model.ListRule)(nil)
	bestHits := 0
	cur := n
	fieldFrom := n
	for cur != nil && cur.Tag != "html" && cur.Tag != "body" {
		item := repeatingItem(cur)
		if item != nil {
			field := relativeField(item, fieldFrom)
			hits := countListHits(tree.HTML, item, field)
			if hits > bestHits {
				bestHits = hits
				best = &model.ListRule{
					ItemSelector:  itemSelector(item),
					FieldSelector: field,
					HitCount:      hits,
				}
			}
		}
		cur = cur.Parent
	}
	if best == nil {
		return &model.ListRule{ItemSelector: "", FieldSelector: fieldPart(n), HitCount: 1}
	}
	return best
}

func repeatingItem(n *domtree.Node) *domtree.Node {
	if n == nil || n.Parent == nil {
		return nil
	}
	sibs := similarSiblings(n)
	if len(sibs) >= 3 {
		return n
	}
	return nil
}

func similarSiblings(n *domtree.Node) []*domtree.Node {
	if n.Parent == nil {
		return nil
	}
	out := make([]*domtree.Node, 0)
	for _, s := range n.Parent.Children {
		if s.Tag != n.Tag {
			continue
		}
		if signature(s) == signature(n) {
			out = append(out, s)
		}
	}
	return out
}

func signature(n *domtree.Node) string {
	if n == nil {
		return ""
	}
	cls := ""
	if len(n.Classes) > 0 {
		cls = n.Classes[0]
	}
	childSig := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		tag := c.Tag
		if len(c.Classes) > 0 {
			tag += "." + c.Classes[0]
		}
		childSig = append(childSig, tag)
	}
	return n.Tag + "|" + cls + "|" + strings.Join(childSig, ",")
}

func itemSelector(n *domtree.Node) string {
	if n == nil {
		return ""
	}
	if len(n.Classes) > 0 && isSafeIdent(n.Classes[0]) {
		return "." + n.Classes[0]
	}
	if n.IDAttr != "" && isSafeIdent(n.IDAttr) {
		return "#" + n.IDAttr
	}
	return n.Tag
}

func relativeField(item, target *domtree.Node) string {
	if item == nil || target == nil {
		return ""
	}
	if item == target {
		if len(target.Classes) > 0 {
			return "." + target.Classes[0]
		}
		return target.Tag
	}
	// Walk from target up to item, build a short descendant selector.
	if len(target.Classes) > 0 && isSafeIdent(target.Classes[0]) {
		return "." + target.Classes[0]
	}
	cur := target
	for cur != nil && cur != item {
		if len(cur.Classes) > 0 && isSafeIdent(cur.Classes[0]) {
			return "." + cur.Classes[0]
		}
		cur = cur.Parent
	}
	return target.Tag
}

func countListHits(rawHTML string, item *domtree.Node, field string) int {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return 0
	}
	itemSel := itemSelector(item)
	if itemSel == "" {
		return 0
	}
	n := 0
	doc.Find(itemSel).Each(func(_ int, s *goquery.Selection) {
		if field == "" {
			n++
			return
		}
		if s.Find(field).Length() > 0 {
			n++
		}
	})
	return n
}

func CountCSS(rawHTML, expr string) int {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return 0
	}
	defer func() { _ = recover() }()
	return doc.Find(expr).Length()
}
