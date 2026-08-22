package parser

import (
	"os"
	"path/filepath"
	"testing"

	"goscrapy/internal/model"
)

func sampleHTML(t *testing.T) []byte {
	t.Helper()
	candidates := []string{
		filepath.Join("testdata", "sample_list.html"),
		filepath.Join("..", "..", "testdata", "sample_list.html"),
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			return b
		}
	}
	t.Fatal("sample_list.html not found")
	return nil
}

func TestExtractTitlePrice(t *testing.T) {
	html := sampleHTML(t)
	rule := &model.Rule{
		ItemSelector: ".product-card",
		LinkSelector: "a.product-link",
		Fields: model.FieldList{
			{Name: "title", Kind: "css", Expr: ".title", Attr: "text"},
			{Name: "price", Kind: "css", Expr: ".price", Attr: "text"},
			{Name: "sku", Kind: "regex", Expr: `SKU-(\d+)`, Attr: "text"},
		},
	}
	out, err := NewEngine().Extract(rule, "http://mock-target/list.html", html)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Items) < 10 {
		t.Fatalf("want >=10 items, got %d", len(out.Items))
	}
	for i, it := range out.Items {
		title, _ := it["title"].(string)
		price, _ := it["price"].(string)
		if title == "" || price == "" {
			t.Fatalf("item %d missing title/price: %+v", i, it)
		}
	}
	if len(out.Links) < 10 {
		t.Fatalf("want >=10 links, got %d", len(out.Links))
	}
}

func TestXPathExtractor(t *testing.T) {
	html := sampleHTML(t)
	rule := &model.Rule{
		ItemSelector: ".product-card",
		Fields: model.FieldList{
			{Name: "title", Kind: "xpath", Expr: ".//h2", Attr: "text"},
		},
	}
	out, err := NewEngine().Extract(rule, "http://mock-target/list.html", html)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Items) < 10 {
		t.Fatalf("xpath items=%d", len(out.Items))
	}
}
