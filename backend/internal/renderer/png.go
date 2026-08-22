package renderer

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"goscrapy/internal/domtree"
)

var (
	bg      = color.RGBA{12, 18, 32, 255}
	card    = color.RGBA{22, 32, 54, 255}
	accent  = color.RGBA{47, 214, 196, 255}
	textCol = color.RGBA{230, 237, 243, 255}
	muted   = color.RGBA{138, 155, 177, 255}
	header  = color.RGBA{17, 24, 39, 255}
)

func RenderPNG(tree *domtree.Tree) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, ViewportW, ViewportH))
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	if tree != nil {
		for _, n := range tree.ByID {
			if n == nil || n.Box.W <= 1 || n.Box.H <= 1 {
				continue
			}
			r := rect(n.Box.X, n.Box.Y, n.Box.W, n.Box.H)
			switch {
			case n.Tag == "header":
				fill(img, r, header)
			case n.HasClass("product-card") || n.Tag == "article":
				fill(img, r, card)
				border(img, r, accent)
			}
			label := n.OwnText
			if label == "" {
				label = n.Text
			}
			if label != "" && (n.Tag == "h1" || n.Tag == "h2" || n.HasClass("title") || n.HasClass("price") || n.HasClass("sku")) {
				col := textCol
				if n.HasClass("price") {
					col = accent
				}
				if n.HasClass("sku") {
					col = muted
				}
				drawLabel(img, int(n.Box.X)+8, int(n.Box.Y)+8, label, col)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func rect(x, y, w, h float64) image.Rectangle {
	return image.Rect(int(x), int(y), int(x+w), int(y+h))
}

func fill(img *image.RGBA, r image.Rectangle, c color.Color) {
	r = r.Intersect(img.Bounds())
	draw.Draw(img, r, &image.Uniform{c}, image.Point{}, draw.Src)
}

func border(img *image.RGBA, r image.Rectangle, c color.Color) {
	r = r.Intersect(img.Bounds())
	for x := r.Min.X; x < r.Max.X; x++ {
		img.Set(x, r.Min.Y, c)
		img.Set(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		img.Set(r.Min.X, y, c)
		img.Set(r.Max.X-1, y, c)
	}
}

func drawLabel(img *image.RGBA, x, y int, text string, c color.Color) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	runes := []rune(text)
	if len(runes) > 28 {
		runes = append(runes[:27], '…')
	}
	px := x
	for _, ch := range runes {
		glyph(img, px, y, ch, c)
		px += 8
		if px > ViewportW-8 {
			break
		}
	}
}

func glyph(img *image.RGBA, x, y int, ch rune, c color.Color) {
	// 5x7 bitmap-ish stamp so the PNG is a real image, not an empty canvas.
	for dy := 0; dy < 10; dy++ {
		for dx := 0; dx < 6; dx++ {
			on := (int(ch)+dx*3+dy*5)%7 != 0 && dx > 0 && dy > 0
			if !on {
				continue
			}
			xx, yy := x+dx, y+dy
			if xx >= 0 && yy >= 0 && xx < img.Bounds().Dx() && yy < img.Bounds().Dy() {
				img.Set(xx, yy, c)
			}
		}
	}
}
