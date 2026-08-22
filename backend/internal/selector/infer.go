package selector

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"goscrapy/internal/domtree"
	"goscrapy/internal/model"
)

// Infer builds CSS/XPath candidates for a clicked node.
// Priority: id > data-* > unique class > attribute combo > nth-child path.
func Infer(tree *domtree.Tree, nodeID int) (*model.SelectorData, error) {
	if tree == nil {
		return nil, fmt.Errorf("nil tree")
	}
	n := tree.ByID[nodeID]
	if n == nil {
		return nil, fmt.Errorf("node %d not found", nodeID)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(tree.HTML))
	if err != nil {
		return nil, err
	}
	cands := collectCandidates(n)
	scored := make([]model.SelectorCandidate, 0, len(cands))
	seen := map[string]struct{}{}
	for _, c := range cands {
		key := c.Kind + "|" + c.Expr
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		hits := countHits(doc, c)
		c.Unique = hits == 1
		c.Score = scoreCandidate(c, hits, n)
		scored = append(scored, c)
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return len(scored[i].Expr) < len(scored[j].Expr)
		}
		return scored[i].Score > scored[j].Score
	})
	if len(scored) > 8 {
		scored = scored[:8]
	}
	list := Generalize(tree, n)
	return &model.SelectorData{Candidates: scored, ListRule: list}, nil
}

func collectCandidates(n *domtree.Node) []model.SelectorCandidate {
	out := make([]model.SelectorCandidate, 0, 12)
	if n.IDAttr != "" && isSafeIdent(n.IDAttr) {
		out = append(out, model.SelectorCandidate{Kind: model.KindCSS, Expr: "#" + n.IDAttr})
		out = append(out, model.SelectorCandidate{Kind: model.KindXPath, Expr: fmt.Sprintf("//*[@id='%s']", n.IDAttr)})
	}
	for _, dk := range n.DataAttrs() {
		val := n.Attr(dk)
		if val == "" || strings.ContainsAny(val, "'\"") {
			continue
		}
		out = append(out, model.SelectorCandidate{Kind: model.KindCSS, Expr: fmt.Sprintf("[%s='%s']", dk, val)})
		out = append(out, model.SelectorCandidate{Kind: model.KindXPath, Expr: fmt.Sprintf("//*[@%s='%s']", dk, val)})
	}
	for _, cls := range n.Classes {
		if !isSafeIdent(cls) {
			continue
		}
		out = append(out, model.SelectorCandidate{Kind: model.KindCSS, Expr: n.Tag + "." + cls})
	}
	if parent := n.Parent; parent != nil && parent.IDAttr != "" && isSafeIdent(parent.IDAttr) {
		field := fieldPart(n)
		out = append(out, model.SelectorCandidate{Kind: model.KindCSS, Expr: "#" + parent.IDAttr + " " + field})
		out = append(out, model.SelectorCandidate{Kind: model.KindXPath, Expr: fmt.Sprintf("//*[@id='%s']//%s", parent.IDAttr, n.Tag)})
	}
	if combo := attributeCombo(n); combo != "" {
		out = append(out, model.SelectorCandidate{Kind: model.KindCSS, Expr: combo})
	}
	out = append(out, model.SelectorCandidate{Kind: model.KindCSS, Expr: nthPath(n)})
	out = append(out, model.SelectorCandidate{Kind: model.KindXPath, Expr: xpathNth(n)})
	return out
}

func fieldPart(n *domtree.Node) string {
	if len(n.Classes) > 0 && isSafeIdent(n.Classes[0]) {
		return "." + n.Classes[0]
	}
	return n.Tag
}

func attributeCombo(n *domtree.Node) string {
	parts := []string{n.Tag}
	if len(n.Classes) > 0 && isSafeIdent(n.Classes[0]) {
		parts[0] += "." + n.Classes[0]
	}
	for _, k := range []string{"name", "type", "role", "itemprop"} {
		if v := n.Attr(k); v != "" && !strings.ContainsAny(v, "'\" ") {
			parts = append(parts, fmt.Sprintf("[%s='%s']", k, v))
		}
	}
	if len(parts) == 1 {
		return ""
	}
	return strings.Join(parts, "")
}

func nthPath(n *domtree.Node) string {
	segs := make([]string, 0, 8)
	cur := n
	for cur != nil && cur.Tag != "html" && len(segs) < 8 {
		seg := cur.Tag
		if cur.IDAttr != "" && isSafeIdent(cur.IDAttr) {
			segs = append(segs, "#"+cur.IDAttr)
			break
		}
		if len(cur.Classes) > 0 && isSafeIdent(cur.Classes[0]) {
			seg += "." + cur.Classes[0]
		} else if cur.Parent != nil {
			seg += fmt.Sprintf(":nth-child(%d)", cur.Index+1)
		}
		segs = append(segs, seg)
		cur = cur.Parent
	}
	for i, j := 0, len(segs)-1; i < j; i, j = i+1, j-1 {
		segs[i], segs[j] = segs[j], segs[i]
	}
	return strings.Join(segs, " > ")
}

func xpathNth(n *domtree.Node) string {
	segs := make([]string, 0, 8)
	cur := n
	for cur != nil && cur.Tag != "html" && len(segs) < 8 {
		if cur.IDAttr != "" && isSafeIdent(cur.IDAttr) {
			segs = append(segs, fmt.Sprintf("*[@id='%s']", cur.IDAttr))
			break
		}
		segs = append(segs, fmt.Sprintf("%s[%d]", cur.Tag, cur.Index+1))
		cur = cur.Parent
	}
	for i, j := 0, len(segs)-1; i < j; i, j = i+1, j-1 {
		segs[i], segs[j] = segs[j], segs[i]
	}
	return "//" + strings.Join(segs, "/")
}

func countHits(doc *goquery.Document, c model.SelectorCandidate) int {
	if c.Kind != model.KindCSS {
		return -1
	}
	defer func() { _ = recover() }()
	return doc.Find(c.Expr).Length()
}

func scoreCandidate(c model.SelectorCandidate, hits int, n *domtree.Node) float64 {
	score := 0.4
	if strings.HasPrefix(c.Expr, "#") || strings.Contains(c.Expr, "[@id=") {
		score = 0.98
	} else if strings.Contains(c.Expr, "data-") {
		score = 0.93
	} else if strings.Contains(c.Expr, ".") && !strings.Contains(c.Expr, "nth-child") {
		score = 0.82
	} else if strings.Contains(c.Expr, "[") {
		score = 0.7
	} else {
		score = 0.45
	}
	if hits == 1 {
		score += 0.05
	} else if hits > 1 {
		score -= 0.15
	}
	if c.Kind == model.KindXPath {
		score -= 0.04
	}
	if score > 1 {
		score = 1
	}
	if score < 0.05 {
		score = 0.05
	}
	_ = n
	return score
}

func isSafeIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '-'
		if i > 0 {
			ok = ok || (r >= '0' && r <= '9')
		}
		if !ok {
			return false
		}
	}
	return true
}
