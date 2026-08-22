package model_test

import (
	"testing"

	"goscrapy/internal/model"
)

func TestRuleInputApplyDoesNotRetainFieldBuffer(t *testing.T) {
	input := model.RuleInput{
		Fields: model.FieldList{
			{Name: " title ", Kind: model.KindCSS, Expr: " h1 "},
			{Name: "price", Kind: model.KindCSS, Expr: ".price"},
		},
	}
	var rule model.Rule
	input.Apply(&rule)

	input.Fields[0].Name = "description"
	input.Fields[1].Expr = ".discount"

	if got := rule.Fields[0].Name; got != "title" {
		t.Fatalf("applied field name changed after reusing input: got %q, want %q", got, "title")
	}
	if got := rule.Fields[1].Expr; got != ".price" {
		t.Fatalf("applied selector changed after reusing input: got %q, want %q", got, ".price")
	}
}
