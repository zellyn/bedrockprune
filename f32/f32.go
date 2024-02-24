package f32

import (
	"image"

	gf32 "gioui.org/f32"
	"github.com/chewxy/math32"
)

type Point = gf32.Point
type Affine2D = gf32.Affine2D

// Constructors
var Pt = gf32.Pt
var NewAffine2D = gf32.NewAffine2D

// Rectangle is like image.Rectangle, but with f32.Point for Min and Max.
type Rectangle struct {
	Min Point
	Max Point
}

// Rect is shorthand for [Rectangle]{Pt(x0, y0), [Pt](x1, y1)}. The returned
// rectangle has minimum and maximum coordinates swapped if necessary so that
// it is well-formed.
func Rect(x0, y0, x1, y1 float32) Rectangle {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return Rectangle{Min: Point{X: x0, Y: y0}, Max: Point{X: x1, Y: y1}}
}

// ImRect converts a Rectangle (with float32 coordinates) to an
// image.Rectangle (with integer coordinates) by simple casting.
func (r Rectangle) ImRect() image.Rectangle {
	return image.Rect(int(r.Min.X), int(r.Min.Y), int(r.Max.X), int(r.Max.Y))
}

// InclusiveInt converts to an integer rectangle guaranteed to include
// the original. It does a Floor on the minimum coordinates, Ceil on
// the maximum ones.
func (r Rectangle) InclusiveInt() image.Rectangle {
	return image.Rect(int(math32.Floor(r.Min.X)), int(math32.Floor(r.Min.Y)), int(math32.Ceil(r.Max.X)), int(math32.Ceil(r.Max.Y)))
}
