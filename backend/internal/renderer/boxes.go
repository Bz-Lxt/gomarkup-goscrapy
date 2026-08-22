package renderer

import "goscrapy/internal/model"

func HitTest(nodes []model.SnapshotNode, x, y float64) *model.SnapshotNode {
	var best *model.SnapshotNode
	bestArea := 1e18
	for i := range nodes {
		n := &nodes[i]
		if x < n.Box.X || y < n.Box.Y || x > n.Box.X+n.Box.W || y > n.Box.Y+n.Box.H {
			continue
		}
		area := n.Box.W * n.Box.H
		if area < bestArea && area > 0 {
			bestArea = area
			best = n
		}
	}
	return best
}

func ClipBox(b model.Box, w, h float64) model.Box {
	if b.X < 0 {
		b.W += b.X
		b.X = 0
	}
	if b.Y < 0 {
		b.H += b.Y
		b.Y = 0
	}
	if b.X+b.W > w {
		b.W = w - b.X
	}
	if b.Y+b.H > h {
		b.H = h - b.Y
	}
	if b.W < 0 {
		b.W = 0
	}
	if b.H < 0 {
		b.H = 0
	}
	return b
}
