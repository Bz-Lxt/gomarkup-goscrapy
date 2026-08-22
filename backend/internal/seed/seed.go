package seed

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"goscrapy/internal/logger"
	"goscrapy/internal/model"
	"goscrapy/internal/store"
)

const DemoRuleName = "mock-shop-list"

func Ensure(ctx context.Context, repos *store.Repos, mockTarget string) error {
	if repos == nil || repos.Rules == nil {
		return nil
	}
	if mockTarget == "" {
		mockTarget = "http://mock-target"
	}
	existing, err := repos.Rules.GetByName(ctx, DemoRuleName)
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	robots := true
	rule := &model.Rule{
		Name:          DemoRuleName,
		StartURL:      mockTarget + "/list.html",
		ItemSelector:  ".product-card",
		LinkSelector:  "a.product-link",
		RespectRobots: robots,
		QPS:           2,
		Version:       1,
		Fields: model.FieldList{
			{Name: "title", Kind: model.KindCSS, Expr: ".title", Attr: "text"},
			{Name: "price", Kind: model.KindCSS, Expr: ".price", Attr: "text"},
			{Name: "sku", Kind: model.KindRegex, Expr: `SKU-(\d+)`, Attr: "text"},
		},
	}
	if err := repos.Rules.Create(ctx, rule); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil
		}
		return err
	}
	logger.Named("seed").Info("inserted demo rule", zap.Int64("id", rule.ID), zap.String("name", rule.Name))
	return nil
}
