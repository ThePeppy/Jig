package product

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// MarkKind selects the app icon or the menu-bar template.
type MarkKind int

const (
	// MarkApp is a full-bleed square. macOS applies the squircle mask.
	MarkApp MarkKind = iota
	// MarkTemplate is black on transparent, for a menu-bar status item.
	MarkTemplate
)

// The prism is the same geometry as the in-app brand mark: a point-up
// hexagon with the three upper edges and the vertical spine.
var (
	markT  = pt{10, 2}
	markUR = pt{17, 6}
	markLR = pt{17, 14}
	markB  = pt{10, 18}
	markLL = pt{3, 14}
	markUL = pt{3, 6}
	markC  = pt{10, 10}
)

// DrawMark renders the Jig mark at size×size pixels.
func DrawMark(size int, kind MarkKind) *image.NRGBA {
	if size < 1 {
		size = 1
	}
	ss := 4
	if size <= 64 {
		ss = 8
	}
	return downsample(renderMark(size*ss, kind), size, ss)
}

// PNG encodes DrawMark.
func PNG(size int, kind MarkKind) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, DrawMark(size, kind))
	return b.Bytes()
}

type pt struct{ x, y float64 }

type face struct {
	pts []pt
	col color.NRGBA
}

func renderMark(size int, kind MarkKind) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	s := float64(size)
	// The viewBox already insets the prism. Mapping it to most of the
	// canvas keeps the mark inside the squircle and the menu-bar slot.
	pad := 0.045
	if kind == MarkTemplate {
		pad = 0.11
	}
	inner := s * (1 - 2*pad)
	scale := inner / 20
	ox := (s - inner) / 2
	mapPt := func(p pt) pt { return pt{ox + p.x*scale, ox + p.y*scale} }

	var faces []face
	var strokes [][2]pt
	stroke := 0.0
	if kind == MarkApp {
		faces = mapFaces(markFaces(), mapPt)
	} else {
		strokes = markStrokes(mapPt)
		stroke = s * 0.052
	}

	for y := range size {
		for x := range size {
			px, py := float64(x)+0.5, float64(y)+0.5
			c := color.NRGBA{}
			if kind == MarkApp {
				c = appField(px, py, s)
				for i := range faces {
					if inside(faces[i].pts, px, py) {
						c = faces[i].col
					}
				}
			} else if onSegments(px, py, strokes, stroke) {
				c = color.NRGBA{A: 0xFF}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func mapFaces(faces []face, mapPt func(pt) pt) []face {
	out := make([]face, len(faces))
	for i, f := range faces {
		pts := make([]pt, len(f.pts))
		for j, p := range f.pts {
			pts[j] = mapPt(p)
		}
		out[i] = face{pts, f.col}
	}
	return out
}
func markFaces() []face {
	// Light from the upper right. Faces meet; no bevel.
	return []face{
		{[]pt{markT, markUL, markC}, rgb(0xD5, 0xD2, 0xE6)},
		{[]pt{markT, markC, markUR}, rgb(0xF4, 0xF1, 0xEA)},
		{[]pt{markUL, markLL, markB, markC}, rgb(0xB0, 0xAD, 0xC0)},
		{[]pt{markC, markUR, markLR, markB}, rgb(0x8B, 0x93, 0xE0)},
	}
}

func markStrokes(mapPt func(pt) pt) [][2]pt {
	hex := []pt{markT, markUR, markLR, markB, markLL, markUL}
	out := make([][2]pt, 0, 10)
	for i := range hex {
		out = append(out, [2]pt{mapPt(hex[i]), mapPt(hex[(i+1)%len(hex)])})
	}
	c := mapPt(markC)
	for _, p := range []pt{markT, markUR, markUL, markB} {
		out = append(out, [2]pt{c, mapPt(p)})
	}
	return out
}

func appField(x, y, size float64) color.NRGBA {
	cx, cy := size*0.5, size*0.40
	d := math.Hypot(x-cx, y-cy) / (size * 0.78)
	if d > 1 {
		d = 1
	}
	t := (y/size)*0.42 + d*0.58
	return lerp(rgb(0x2C, 0x2E, 0x34), rgb(0x12, 0x12, 0x10), t)
}

func inside(pts []pt, x, y float64) bool {
	in := false
	j := len(pts) - 1
	for i := range pts {
		xi, yi := pts[i].x, pts[i].y
		xj, yj := pts[j].x, pts[j].y
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
		j = i
	}
	return in
}

func onSegments(x, y float64, segs [][2]pt, half float64) bool {
	for _, s := range segs {
		if distToSeg(x, y, s[0], s[1]) <= half {
			return true
		}
	}
	return false
}

func distToSeg(px, py float64, a, b pt) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = ((px-a.x)*dx + (py-a.y)*dy) / l2
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
	}
	return math.Hypot(px-(a.x+t*dx), py-(a.y+t*dy))
}

func rgb(r, g, b byte) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 0xFF} }

func lerp(a, b color.NRGBA, t float64) color.NRGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	u := 1 - t
	return color.NRGBA{
		R: byte(u*float64(a.R) + t*float64(b.R)),
		G: byte(u*float64(a.G) + t*float64(b.G)),
		B: byte(u*float64(a.B) + t*float64(b.B)),
		A: 0xFF,
	}
}

func downsample(src *image.NRGBA, size, ss int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	n := float64(ss * ss)
	for y := range size {
		for x := range size {
			var sr, sg, sb, sa float64
			for sy := range ss {
				for sx := range ss {
					c := src.NRGBAAt(x*ss+sx, y*ss+sy)
					a := float64(c.A)
					sr += float64(c.R) * a
					sg += float64(c.G) * a
					sb += float64(c.B) * a
					sa += a
				}
			}
			if sa == 0 {
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(sr / sa),
				G: uint8(sg / sa),
				B: uint8(sb / sa),
				A: uint8(sa / n),
			})
		}
	}
	return dst
}
