package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"go.uber.org/zap"

	"goscrapy/internal/domtree"
	"goscrapy/internal/logger"
	"goscrapy/internal/model"
)

type cdpNode struct {
	NodeID int    `json:"node_id"`
	Tag    string `json:"tag"`
	Text   string `json:"text"`
	Box    boxDTO `json:"box"`
	ID     string `json:"id"`
	Class  string `json:"class"`
}

type boxDTO struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

const extractJS = `(() => {
  const nodes = [];
  let id = 1;
  const skip = new Set(['SCRIPT','STYLE','NOSCRIPT','META','LINK','TITLE']);
  const walk = (el) => {
    if (!el || el.nodeType !== 1) return;
    if (skip.has(el.tagName)) return;
    const r = el.getBoundingClientRect();
    if (r.width > 0 && r.height > 0) {
      nodes.push({
        node_id: id,
        tag: el.tagName.toLowerCase(),
        text: (el.innerText || '').replace(/\s+/g,' ').slice(0,80),
        box: {x: r.x, y: r.y, w: r.width, h: r.height},
        id: el.id || '',
        class: el.className || ''
      });
      el.setAttribute('data-goscrapy-nid', String(id));
      id++;
    }
    for (const c of el.children) walk(c);
  };
  walk(document.body);
  return JSON.stringify(nodes);
})()`

func CaptureCDP(ctx context.Context, rendererWS, pageURL string) (*Capture, error) {
	wsURL, err := resolveDebuggerWS(ctx, rendererWS)
	if err != nil {
		return nil, err
	}
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(ctx, wsURL)
	defer cancelAlloc()
	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 25*time.Second)
	defer cancelTimeout()

	var png []byte
	var html string
	var rawNodes string
	err = chromedp.Run(taskCtx,
		chromedp.EmulateViewport(ViewportW, ViewportH, chromedp.EmulateScale(1)),
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
		chromedp.Evaluate(extractJS, &rawNodes),
		chromedp.CaptureScreenshot(&png),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp: %w", err)
	}
	tree, err := domtree.Parse([]byte(html))
	if err != nil {
		return nil, err
	}
	var parsed []cdpNode
	if err := json.Unmarshal([]byte(rawNodes), &parsed); err != nil {
		// Evaluate may already return a JSON string or a quoted string.
		var s string
		if json.Unmarshal([]byte(rawNodes), &s) == nil {
			_ = json.Unmarshal([]byte(s), &parsed)
		}
	}
	nodes := make([]model.SnapshotNode, 0, len(parsed))
	for _, n := range parsed {
		box := model.Box{X: n.Box.X, Y: n.Box.Y, W: n.Box.W, H: n.Box.H}
		if tn := tree.ByID[n.NodeID]; tn != nil {
			tn.Box = box
		}
		nodes = append(nodes, model.SnapshotNode{NodeID: n.NodeID, Tag: n.Tag, Text: n.Text, Box: box})
	}
	if len(nodes) == 0 {
		AssignBoxes(tree)
		nodes = VisibleNodes(tree)
	}
	return &Capture{
		URL:    pageURL,
		Width:  ViewportW,
		Height: ViewportH,
		PNG:    png,
		HTML:   html,
		Tree:   tree,
		Nodes:  nodes,
		Source: "cdp",
	}, nil
}

func resolveDebuggerWS(ctx context.Context, rendererWS string) (string, error) {
	rendererWS = strings.TrimSpace(rendererWS)
	if rendererWS == "" {
		return "", fmt.Errorf("RENDERER_WS empty")
	}
	httpBase := rendererWS
	httpBase = strings.Replace(httpBase, "ws://", "http://", 1)
	httpBase = strings.Replace(httpBase, "wss://", "https://", 1)
	httpBase = strings.TrimRight(httpBase, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpBase+"/json/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var payload struct {
		WS string `json:"webSocketDebuggerUrl"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("parse debugger version: %w", err)
	}
	if payload.WS == "" {
		return rendererWS, nil
	}
	// Rewrite host if the container advertised localhost.
	if strings.Contains(payload.WS, "127.0.0.1") || strings.Contains(payload.WS, "localhost") {
		host := hostOf(rendererWS)
		payload.WS = replaceHost(payload.WS, host)
	}
	logger.Named("renderer").Debug("resolved debugger ws", zap.String("ws", payload.WS))
	return payload.WS, nil
}

func hostOf(ws string) string {
	ws = strings.TrimPrefix(ws, "ws://")
	ws = strings.TrimPrefix(ws, "wss://")
	ws = strings.TrimPrefix(ws, "http://")
	ws = strings.TrimPrefix(ws, "https://")
	if i := strings.Index(ws, "/"); i >= 0 {
		ws = ws[:i]
	}
	return ws
}

func replaceHost(ws, host string) string {
	// ws://127.0.0.1:9222/devtools/browser/xxx -> ws://renderer:9222/...
	rest := ws
	scheme := "ws://"
	if strings.HasPrefix(rest, "wss://") {
		scheme = "wss://"
		rest = rest[len("wss://"):]
	} else if strings.HasPrefix(rest, "ws://") {
		rest = rest[len("ws://"):]
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		return scheme + host + rest[i:]
	}
	return scheme + host
}
