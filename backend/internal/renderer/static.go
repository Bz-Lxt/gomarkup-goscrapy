package renderer

import (
	"context"
	"fmt"

	"goscrapy/internal/domtree"
	"goscrapy/internal/fetcher"
	"goscrapy/internal/model"
)

type staticFetcher interface {
	Fetch(context.Context, string, bool) (*fetcher.Result, error)
}

func CaptureStatic(ctx context.Context, fetch staticFetcher, pageURL string) (*Capture, error) {
	if fetch == nil {
		return nil, fmt.Errorf("fetcher required for static snapshot")
	}
	res, err := fetch.Fetch(ctx, pageURL, false)
	if err != nil {
		return nil, err
	}
	if res.Status > 0 && res.Status >= 400 {
		return nil, fmt.Errorf("fetch status %d", res.Status)
	}
	return CaptureFromHTML(res.FinalURL, res.Body)
}

func CaptureFromHTML(pageURL string, body []byte) (*Capture, error) {
	tree, err := domtree.Parse(body)
	if err != nil {
		return nil, err
	}
	AssignBoxes(tree)
	pngBytes, err := RenderPNG(tree)
	if err != nil {
		return nil, err
	}
	nodes := VisibleNodes(tree)
	return &Capture{
		URL:    pageURL,
		Width:  ViewportW,
		Height: ViewportH,
		PNG:    pngBytes,
		HTML:   string(body),
		Tree:   tree,
		Nodes:  nodes,
		Source: "static",
	}, nil
}

type Capture struct {
	URL    string
	Width  int
	Height int
	PNG    []byte
	HTML   string
	Tree   *domtree.Tree
	Nodes  []model.SnapshotNode
	Source string
}
