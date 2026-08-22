package selector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goscrapy/internal/domtree"
)

func sampleHTML(t *testing.T) []byte {
	t.Helper()
	for _, p := range []string{
		filepath.Join("testdata", "sample_list.html"),
		filepath.Join("..", "..", "testdata", "sample_list.html"),
	} {
		b, err := os.ReadFile(p)
		if err == nil {
			return b
		}
	}
	t.Fatal("sample_list.html not found")
	return nil
}

func TestInferTitleGeneralizes(t *testing.T) {
	raw := sampleHTML(t)
	tree, err := domtree.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	var title *domtree.Node
	for _, n := range tree.ByID {
		if n.Tag == "h2" && strings.Contains(n.Text, "Aurora Headphones") {
			title = n
			break
		}
	}
	if title == nil {
		t.Fatal("title node not found")
	}
	data, err := Infer(tree, title.ID)
	if err != nil {
		t.Fatal(err)
	}
	if data.ListRule == nil {
		t.Fatal("expected list_rule")
	}
	if data.ListRule.HitCount < 10 {
		t.Fatalf("hit_count=%d want >=10 item=%s field=%s", data.ListRule.HitCount, data.ListRule.ItemSelector, data.ListRule.FieldSelector)
	}
	if data.ListRule.ItemSelector != ".product-card" {
		t.Fatalf("item_selector=%s", data.ListRule.ItemSelector)
	}
	if len(data.Candidates) == 0 {
		t.Fatal("no candidates")
	}
}

func TestPriorityPrefersID(t *testing.T) {
	raw := sampleHTML(t)
	tree, err := domtree.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	var card *domtree.Node
	for _, n := range tree.ByID {
		if n.IDAttr == "p-1" {
			card = n
			break
		}
	}
	if card == nil {
		t.Fatal("card p-1 missing")
	}
	data, err := Infer(tree, card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if data.Candidates[0].Expr != "#p-1" && !strings.Contains(data.Candidates[0].Expr, "#p-1") {
		t.Fatalf("top candidate should prefer id, got %s", data.Candidates[0].Expr)
	}
}
