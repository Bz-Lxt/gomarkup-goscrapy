package parser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"

	"goscrapy/internal/model"
	"goscrapy/internal/urlx"
)

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

type PageExtract struct {
	Items []model.JSONMap
	Links []string
}

func (e *Engine) Extract(rule *model.Rule, pageURL string, body []byte) (*PageExtract, error) {
	if rule == nil {
		return nil, fmt.Errorf("nil rule")
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	root, _ := htmlquery.Parse(bytes.NewReader(body))
	out := &PageExtract{Items: []model.JSONMap{}, Links: []string{}}

	itemSel := strings.TrimSpace(rule.ItemSelector)
	var scopes []Scope
	if itemSel == "" {
		scopes = []Scope{{HTML: string(body), Doc: doc, Node: root, Base: pageURL}}
	} else {
		doc.Find(itemSel).Each(func(_ int, s *goquery.Selection) {
			htmlStr, _ := goquery.OuterHtml(s)
			subDoc, err := goquery.NewDocumentFromReader(strings.NewReader(wrapFragment(htmlStr)))
			if err != nil {
				return
			}
			var node *html.Node
			if s.Length() > 0 {
				node = s.Get(0)
			}
			scopes = append(scopes, Scope{HTML: htmlStr, Doc: subDoc, Node: node, Base: pageURL})
		})
	}

	for _, sc := range scopes {
		item := model.JSONMap{}
		for _, f := range rule.Fields {
			f = f.Normalize()
			if err := f.Valid(); err != nil {
				continue
			}
			val, err := ExtractorFor(f.Kind).Extract(sc, f)
			if err != nil {
				item[f.Name] = ""
				item[f.Name+"_error"] = err.Error()
				continue
			}
			item[f.Name] = val
		}
		if len(item) > 0 {
			out.Items = append(out.Items, item)
		}
	}

	linkSel := strings.TrimSpace(rule.LinkSelector)
	if linkSel != "" {
		seen := map[string]struct{}{}
		doc.Find(linkSel).Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			abs := urlx.Resolve(pageURL, href)
			if abs == "" || !urlx.IsHTTP(abs) {
				return
			}
			if _, ok := seen[abs]; ok {
				return
			}
			seen[abs] = struct{}{}
			out.Links = append(out.Links, abs)
		})
	}
	return out, nil
}

func wrapFragment(frag string) string {
	if strings.Contains(strings.ToLower(frag), "<html") {
		return frag
	}
	return "<html><body>" + frag + "</body></html>"
}

func (e *Engine) Preview(rule *model.Rule, pageURL, html string) (*PageExtract, error) {
	return e.Extract(rule, pageURL, []byte(html))
}
