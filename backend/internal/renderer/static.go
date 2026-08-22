package renderer

import (
	"context"
	"fmt"

	"goscrapy/internal/domtree"
	"goscrapy/internal/fetcher"
	"goscrapy/internal/model"
)

func CaptureStatic(ctx context.Context, fetch *fetcher.Client, pageURL string) (*Capture, error) {
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
	htmlText := string(body)
	tree, err := domtree.Parse(body)
	if err != nil {
		return nil, err
	}
	AssignBoxes(tree)
	pngBytes, err := RenderPNG(tree)
	if err != nil {
		return nil, err
	}
	// pngBytes is already a self-owned slice freshly allocated by
	// bytes.Buffer.Bytes(). Do NOT rewrite it onto body's backing array
	// (e.g. append(body[:0], pngBytes...)): body is the caller's buffer
	// that may be reset and returned to a pool after CaptureFromHTML
	// returns, which would corrupt the PNG data we hand back.
	nodes := VisibleNodes(tree)
	return &Capture{
		URL:    pageURL,
		Width:  ViewportW,
		Height: ViewportH,
		PNG:    pngBytes,
		HTML:   htmlText,
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
