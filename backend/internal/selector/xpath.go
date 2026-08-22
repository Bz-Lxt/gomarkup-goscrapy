package selector

import (
	"fmt"
	"strings"

	"goscrapy/internal/domtree"
)

func AbsoluteXPath(n *domtree.Node) string {
	if n == nil {
		return ""
	}
	if n.IDAttr != "" && isSafeIdent(n.IDAttr) {
		return fmt.Sprintf("//*[@id='%s']", n.IDAttr)
	}
	segs := make([]string, 0, 8)
	cur := n
	for cur != nil && cur.Tag != "" && cur.Tag != "html" {
		seg := cur.Tag
		if cur.IDAttr != "" && isSafeIdent(cur.IDAttr) {
			segs = append(segs, fmt.Sprintf("*[@id='%s']", cur.IDAttr))
			break
		}
		same := 0
		idx := 1
		if cur.Parent != nil {
			for _, s := range cur.Parent.Children {
				if s.Tag == cur.Tag {
					same++
					if s == cur {
						idx = same
					}
				}
			}
			if same > 1 {
				seg = fmt.Sprintf("%s[%d]", cur.Tag, idx)
			}
		}
		segs = append(segs, seg)
		cur = cur.Parent
	}
	for i, j := 0, len(segs)-1; i < j; i, j = i+1, j-1 {
		segs[i], segs[j] = segs[j], segs[i]
	}
	return "/" + strings.Join(segs, "/")
}

func CSSToRoughXPath(css string) string {
	css = strings.TrimSpace(css)
	if strings.HasPrefix(css, "#") && isSafeIdent(css[1:]) {
		return fmt.Sprintf("//*[@id='%s']", css[1:])
	}
	if strings.HasPrefix(css, ".") && isSafeIdent(css[1:]) {
		return fmt.Sprintf("//*[contains(concat(' ', normalize-space(@class), ' '), ' %s ')]", css[1:])
	}
	return "//" + css
}
