package parser

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"

	"goscrapy/internal/model"
)

type Extractor interface {
	Extract(scope Scope, spec model.FieldSpec) (string, error)
}

type Scope struct {
	HTML string
	Doc  *goquery.Document
	Node *html.Node
	Base string
}

func NewCSS() Extractor   { return cssExtractor{} }
func NewXPath() Extractor { return xpathExtractor{} }
func NewRegex() Extractor { return regexExtractor{} }

type cssExtractor struct{}

func (cssExtractor) Extract(scope Scope, spec model.FieldSpec) (string, error) {
	if scope.Doc == nil {
		return "", nil
	}
	sel := scope.Doc.Find(spec.Expr)
	if sel.Length() == 0 {
		return "", nil
	}
	return attrOfSelection(sel.First(), spec.Attr), nil
}

type xpathExtractor struct{}

func (xpathExtractor) Extract(scope Scope, spec model.FieldSpec) (string, error) {
	root := scope.Node
	if root == nil && scope.Doc != nil {
		root = scope.Doc.Nodes[0]
	}
	if root == nil {
		return "", nil
	}
	n, err := htmlquery.Query(root, spec.Expr)
	if err != nil || n == nil {
		return "", err
	}
	return attrOfNode(n, spec.Attr), nil
}

type regexExtractor struct{}

func (regexExtractor) Extract(scope Scope, spec model.FieldSpec) (string, error) {
	re, err := regexp.Compile(spec.Expr)
	if err != nil {
		return "", err
	}
	src := scope.HTML
	if spec.Attr != "html" && scope.Doc != nil {
		src = strings.TrimSpace(scope.Doc.Text())
	}
	m := re.FindStringSubmatch(src)
	if len(m) == 0 {
		return "", nil
	}
	if len(m) >= 2 {
		return strings.TrimSpace(m[1]), nil
	}
	return strings.TrimSpace(m[0]), nil
}

func attrOfSelection(s *goquery.Selection, attr string) string {
	attr = strings.ToLower(strings.TrimSpace(attr))
	switch attr {
	case "", "text", "innerText":
		return strings.TrimSpace(s.Text())
	case "html", "innerHTML":
		h, _ := s.Html()
		return strings.TrimSpace(h)
	case "href", "src", "content", "value", "title", "alt":
		v, _ := s.Attr(attr)
		return strings.TrimSpace(v)
	default:
		v, ok := s.Attr(attr)
		if ok {
			return strings.TrimSpace(v)
		}
		return strings.TrimSpace(s.Text())
	}
}

func attrOfNode(n *html.Node, attr string) string {
	attr = strings.ToLower(strings.TrimSpace(attr))
	switch attr {
	case "", "text", "innerText":
		return strings.TrimSpace(htmlquery.InnerText(n))
	case "html", "innerHTML":
		return strings.TrimSpace(htmlquery.OutputHTML(n, false))
	default:
		for _, a := range n.Attr {
			if strings.EqualFold(a.Key, attr) {
				return strings.TrimSpace(a.Val)
			}
		}
		return strings.TrimSpace(htmlquery.InnerText(n))
	}
}

func ExtractorFor(kind string) Extractor {
	switch strings.ToLower(kind) {
	case model.KindXPath:
		return NewXPath()
	case model.KindRegex:
		return NewRegex()
	default:
		return NewCSS()
	}
}
