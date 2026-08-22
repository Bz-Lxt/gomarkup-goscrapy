package model

import (
	"reflect"
	"testing"
)

// TestApplyFieldsNoAliasing verifies that RuleInput.Apply produces a Rule whose
// Fields slice does not share its backing array with the caller's RuleInput.Fields.
// Callers commonly reuse the request object after Apply (filling defaults or
// re-normalizing); such later mutations must not leak into the generated Rule.
func TestApplyFieldsNoAliasing(t *testing.T) {
	in := RuleInput{
		Name:     "demo",
		StartURL: "https://example.com",
		Fields: FieldList{
			{Name: "title", Kind: KindCSS, Expr: ".title", Attr: ""},
			{Name: "price", Kind: KindCSS, Expr: ".price", Attr: ""},
		},
	}
	// Snapshot the original field expressions for later comparison.
	origExprs := make([]string, len(in.Fields))
	for i, f := range in.Fields {
		origExprs[i] = f.Expr
	}

	rule := &Rule{RespectRobots: true, QPS: 2, Version: 1}
	in.Apply(rule)

	// Generated rule must hold normalized field values (Attr defaults to "text").
	wantFields := FieldList{
		{Name: "title", Kind: KindCSS, Expr: ".title", Attr: "text"},
		{Name: "price", Kind: KindCSS, Expr: ".price", Attr: "text"},
	}
	if !reflect.DeepEqual(rule.Fields, wantFields) {
		t.Fatalf("rule.Fields = %+v, want %+v", rule.Fields, wantFields)
	}

	// Now simulate the caller reusing the request object after generation: it
	// mutates the original Fields slice in place. The generated rule must be
	// unaffected.
	for i := range in.Fields {
		in.Fields[i].Expr = "CHANGED-" + in.Fields[i].Expr
		in.Fields[i].Attr = "CHANGED"
	}
	if !reflect.DeepEqual(rule.Fields, wantFields) {
		t.Fatalf("rule.Fields mutated by caller change: got %+v, want %+v", rule.Fields, wantFields)
	}

	// The caller's own slice should of course reflect its own mutation.
	for i, f := range in.Fields {
		if f.Expr != "CHANGED-"+origExprs[i] {
			t.Fatalf("caller field[%d].Expr = %q, want %q", i, f.Expr, "CHANGED-"+origExprs[i])
		}
	}
}

// TestApplyFieldsReusedRequest ensures a second Apply on a different rule with a
// reused/modified request does not retroactively alter the first rule.
func TestApplyFieldsReusedRequest(t *testing.T) {
	in := RuleInput{
		Name:     "demo",
		StartURL: "https://example.com",
		Fields: FieldList{
			{Name: "title", Kind: KindCSS, Expr: ".title"},
		},
	}
	r1 := &Rule{RespectRobots: true, QPS: 2, Version: 1}
	in.Apply(r1)

	// Caller appends more fields to the request and generates a second rule.
	in.Fields = append(in.Fields, FieldSpec{Name: "price", Kind: KindCSS, Expr: ".price"})
	r2 := &Rule{RespectRobots: true, QPS: 2, Version: 1}
	in.Apply(r2)

	if len(r1.Fields) != 1 || r1.Fields[0].Expr != ".title" {
		t.Fatalf("r1.Fields changed after second Apply: %+v", r1.Fields)
	}
	if len(r2.Fields) != 2 {
		t.Fatalf("r2.Fields len = %d, want 2", len(r2.Fields))
	}
	if r2.Fields[1].Expr != ".price" {
		t.Fatalf("r2.Fields[1] = %+v, want expr .price", r2.Fields[1])
	}
}
