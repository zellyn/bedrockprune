package tiles

import (
	"image"

	"gioui.org/unit"
	"github.com/chewxy/math32"
	"github.com/zellyn/bedrockprune/f32"
)

type TileSource16 interface {
	Get(x, y int) *image.NRGBA
}

type TileServer struct {
	source TileSource16
}

func NewServer(source TileSource16) *TileServer {
	return &TileServer{
		source: source,
	}
}

type ImageRect struct {
	Area  f32.Rectangle
	Image *image.NRGBA
	scale *f32.Point
}

func (ir ImageRect) Scale() f32.Point {
	if ir.scale == nil {
		ir.scale = &f32.Point{X: 1, Y: 1}
	}
	return *ir.scale
}

var sixteenth = f32.Pt(1.0/16, 1.0/16)

func (ts *TileServer) Get(area f32.Rectangle, dpPerUnit unit.Dp) []ImageRect {
	// first pass: let's do what the original code did.
	var res []ImageRect

	for y := math32.Floor(area.Min.Y); y < area.Max.Y; y++ {
		for x := math32.Floor(area.Min.X); x < area.Max.X; x++ {
			res = append(res, ImageRect{
				Area:  f32.Rect(x, y, x+1, y+1),
				Image: ts.source.Get(int(x), int(y)),
				scale: &sixteenth,
			})
		}
	}

	return res
}
