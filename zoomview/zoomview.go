package zoomview

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/zellyn/bedrockprune/f32"
	"github.com/zellyn/bedrockprune/lerp"
	_ "github.com/zellyn/bedrockprune/lerp"
	"github.com/zellyn/bedrockprune/tiles"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/chewxy/math32"
)

var yellow = color.NRGBA{R: 0xFF, G: 0xFF, A: 0xFF}
var black = color.NRGBA{A: 0xFF}

const zoomZeroDpPerBlock = 20

var one = f32.Pt(1, 1)

type zoomView struct {
	tileserver *tiles.TileServer

	zoom        float32
	topleft     f32.Point
	bottomright f32.Point
	cell        image.Point

	dpPerBlock   unit.Dp
	pxPerBlock   float32
	min          f32.Point
	max          f32.Point
	blockSize    f32.Point
	blockCount   image.Point
	scale        f32.Affine2D
	unitsPerTile int

	haveContextInfo bool
	metric          unit.Metric
	size            image.Point

	lerping      bool
	oldPow       float32
	oldCursorPos f32.Point
	zoomDiff     f32.Point
	zoomLerp     lerp.TimeLerp[float32]
	tlXLerp      lerp.TimeLerp[float32]
	tlYLerp      lerp.TimeLerp[float32]
}

func (zv *zoomView) setContextInfo(metric unit.Metric, size image.Point) {
	zv.metric = metric
	zv.size = size
	zv.haveContextInfo = true
	zv.calc()
}

func (zv *zoomView) setZoom(zoom float32) {
	zv.zoom = zoom
	if zv.haveContextInfo {
		zv.calc()
	}
}

func (zv *zoomView) left(fast bool) {
	zv.cell.X--
	if fast {
		zv.cell.X -= 9
	}
	if float32(zv.cell.X)+1 <= zv.topleft.X {
		zv.topleft.X = float32(zv.cell.X)
		zv.calcpos()
	}
}

func (zv *zoomView) right(fast bool) {
	zv.cell.X += 1
	if fast {
		zv.cell.X += 9
	}
	if float32(zv.cell.X) >= zv.topleft.X+zv.blockSize.X {
		zv.topleft.X = float32(zv.cell.X) + 1 - zv.blockSize.X
		zv.calcpos()
	}
}

func (zv *zoomView) up(fast bool) {
	zv.cell.Y -= 1
	if fast {
		zv.cell.Y -= 9
	}
	if float32(zv.cell.Y)+1 <= zv.topleft.Y {
		zv.topleft.Y = float32(zv.cell.Y)
		zv.calcpos()
	}
}

func (zv *zoomView) down(fast bool) {
	zv.cell.Y += 1
	if fast {
		zv.cell.Y += 9
	}
	if float32(zv.cell.Y) >= zv.topleft.Y+zv.blockSize.Y {
		zv.topleft.Y = float32(zv.cell.Y) + 1 - zv.blockSize.Y
		zv.calcpos()
	}
}

func (zv *zoomView) doZoom() {
	if zv.lerping {
		now := time.Now()
		zv.zoom = zv.zoomLerp.At(now)
		factor := zv.oldPow / math32.Pow(2, zv.zoom)
		zv.topleft.X = zv.tlXLerp.At(now)
		zv.topleft.Y = zv.tlYLerp.At(now)
		zv.topleft = zv.oldCursorPos.Sub(zv.zoomDiff.Mul(factor))
		if zv.zoomLerp.Done(now) {
			zv.lerping = false
		}
		zv.calc()
	}
}

const zoomDuration = time.Second / 5

func (zv *zoomView) setZoomDetails(zoom float32, tl f32.Point, curve lerp.Curve) {
	if true {
		zv.zoom = zoom
		zv.topleft = tl
		zv.calc()
	} else {

		now := time.Now()
		if !zv.lerping {
			zv.oldPow = math32.Pow(2, zv.zoom)
			zv.oldCursorPos = f32.Point{X: float32(zv.cell.X) + 0.5, Y: float32(zv.cell.Y) + 0.5}
			zv.zoomDiff = zv.oldCursorPos.Sub(zv.topleft)
		}
		zv.lerping = true
		zv.zoomLerp = lerp.NewTimeLerp(zv.zoom, zoom, now, zoomDuration, nil)
		zv.tlXLerp = lerp.NewTimeLerp(zv.topleft.X, tl.X, now, zoomDuration, curve)
		zv.tlYLerp = lerp.NewTimeLerp(zv.topleft.Y, tl.Y, now, zoomDuration, curve)
	}
}

func (zv *zoomView) targetZoom() float32 {
	if zv.lerping {
		return zv.zoomLerp.Target()
	}
	return zv.zoom
}

func (zv *zoomView) zoomin() {
	cursorPos := f32.Point{X: float32(zv.cell.X) + 0.5, Y: float32(zv.cell.Y) + 0.5}
	diff := cursorPos.Sub(zv.topleft)
	tl := zv.topleft.Add(diff.Mul(0.5))

	zv.setZoomDetails(zv.targetZoom()+1, tl, lerp.Log2)
}

func (zv *zoomView) zoomout() {
	cursorPos := f32.Point{X: float32(zv.cell.X) + 0.5, Y: float32(zv.cell.Y) + 0.5}
	diff := cursorPos.Sub(zv.topleft)
	tl := zv.topleft.Sub(diff)

	zv.setZoomDetails(zv.targetZoom()-1, tl, lerp.Pow2)
}

func (zv *zoomView) calc() {
	zv.dpPerBlock = unit.Dp(zoomZeroDpPerBlock * math32.Pow(2, zv.zoom))
	zv.pxPerBlock = zv.metric.PxPerDp * float32(zv.dpPerBlock)
	zv.blockCount = image.Pt(int(float32(zv.size.X)/zv.pxPerBlock)+1, int(float32(zv.size.Y)/zv.pxPerBlock)+1)
	zv.blockSize = f32.Pt(float32(zv.size.X), float32(zv.size.Y)).Div(float32(zv.pxPerBlock))

	blockScale := float32(1) / 16
	zv.unitsPerTile = 16
	dp := zv.dpPerBlock
	for dp < minDpPerBlock {
		dp *= 2
		zv.unitsPerTile *= 2
		blockScale *= 2
	}

	zv.scale = f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(float32(zv.pxPerBlock*blockScale), float32(zv.pxPerBlock*blockScale)))

	zv.calcpos()
}

func (zv *zoomView) calcpos() {
	zv.min = f32.Pt(math32.Floor(zv.topleft.X), math32.Floor(zv.topleft.Y))
	zv.max = f32.Pt(zv.min.X+float32(zv.blockCount.X), zv.min.Y+float32(zv.blockCount.Y))
}

func (zv *zoomView) screenPosition(x, y int) image.Rectangle {
	left := (float32(x) - zv.topleft.X) * float32(zv.dpPerBlock)
	top := (float32(y) - zv.topleft.Y) * float32(zv.dpPerBlock)
	return image.Rectangle{
		Min: image.Pt(int(zv.metric.PxPerDp*left), int(zv.metric.PxPerDp*top)),
		Max: image.Pt(int(zv.metric.PxPerDp*(left+float32(zv.dpPerBlock))), int(zv.metric.PxPerDp*(top+float32(zv.dpPerBlock)))),
	}
}

const minDpPerBlock = unit.Dp(5)

func Run(w *app.Window, ts16 tiles.TileSource16) error {
	if err := bringToFront(); err != nil {
		return err
	}
	zv := &zoomView{
		tileserver: tiles.NewServer(ts16, tiles.OptionEmpty16x16(empty16x16())),
		zoom:       1,
		topleft:    f32.Pt(1.5, 2.75),
		cell:       image.Pt(3, 3),
	}
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			if zv.lerping {
				zv.doZoom()
				gtx.Execute(op.InvalidateCmd{})
			}

			event.Op(&ops, zv)
			if !zv.haveContextInfo || zv.metric != gtx.Metric || zv.size != gtx.Constraints.Max {
				zv.setContextInfo(gtx.Metric, gtx.Constraints.Max)
			}

			area := image.Rectangle{Min: f32.PointFloor(zv.min), Max: f32.PointCeil(zv.max)}
			var empties []tiles.Tile

			// Process keypresses
			if quit := zv.handleKeyPresses(gtx, zv.unitsPerTile); quit {
				return nil
			}

			tiles, err := zv.tileserver.Get(area, zv.unitsPerTile)
			if err != nil {
				return err
			}
			for _, areaImage := range tiles {
				if areaImage.Empty {
					empties = append(empties, areaImage)
					continue
				}
				pos := zv.screenPosition(areaImage.Area.Min.X, areaImage.Area.Min.Y)
				imageOp := paint.NewImageOp(areaImage.Image)
				imageOp.Add(&ops)
				offset := op.Affine(zv.scale.Offset(f32.FromImagePt(pos.Min))).Push(&ops)
				paint.PaintOp{}.Add(&ops)

				width := 2.0 / zv.scale.Transform(f32.Point{X: 1, Y: 1}).X

				// draw rectangles around tiles?
				if false {
					bounds := areaImage.Image.Bounds()
					rrect := clip.RRect{Rect: bounds}
					paint.FillShape(&ops, black,
						clip.Stroke{
							Path:  rrect.Path(&ops),
							Width: float32(gtx.Metric.PxPerDp) * width,
						}.Op(),
					)
				}

				offset.Pop()
			}

			if len(empties) > 0 {
				imageOp := paint.NewImageOp(empties[0].Image)
				imageOp.Add(&ops)
				for _, areaImage := range empties {
					pos := zv.screenPosition(areaImage.Area.Min.X, areaImage.Area.Min.Y)
					offset := op.Affine(zv.scale.Offset(f32.FromImagePt(pos.Min))).Push(&ops)
					paint.PaintOp{}.Add(&ops)
					offset.Pop()
				}
			}

			cursorPos := zv.screenPosition(zv.cell.X, zv.cell.Y)
			rrect := clip.RRect{Rect: cursorPos, SE: 3, SW: 3, NE: 3, NW: 3}
			paint.FillShape(&ops, black,
				clip.Stroke{
					Path:  rrect.Path(&ops),
					Width: float32(gtx.Metric.Dp(unit.Dp(4))),
				}.Op(),
			)
			paint.FillShape(&ops, yellow,
				clip.Stroke{
					Path:  rrect.Path(&ops),
					Width: float32(gtx.Metric.Dp(unit.Dp(2))),
				}.Op(),
			)

			// Pass the drawing operations to the GPU.
			e.Frame(gtx.Ops)

		}
	}
}

func (zv *zoomView) handleKeyPresses(gtx layout.Context, units int) (quit bool) {
	for {
		event, ok := gtx.Event(
			key.Filter{Name: key.NameUpArrow, Optional: key.ModShift | key.ModAlt},
			key.Filter{Name: key.NameDownArrow, Optional: key.ModShift | key.ModAlt},
			key.Filter{Name: key.NameLeftArrow, Optional: key.ModShift | key.ModAlt},
			key.Filter{Name: key.NameRightArrow, Optional: key.ModShift | key.ModAlt},
			key.Filter{Name: key.NameEscape}, key.Filter{Name: "Q", Optional: key.ModShift},
			key.Filter{Name: "="}, key.Filter{Name: "+", Optional: key.ModShift},
			key.Filter{Name: "-"},
			key.Filter{Name: "D", Optional: key.ModShift},
		)
		if !ok {
			break
		}
		ev, ok := event.(key.Event)
		if !ok {
			continue
		}

		if ev.State == key.Press {
			switch ev.Name {
			case key.NameUpArrow:
				zv.up(ev.Modifiers&key.ModAlt > 0)
			case key.NameDownArrow:
				zv.down(ev.Modifiers&key.ModAlt > 0)
			case key.NameLeftArrow:
				zv.left(ev.Modifiers&key.ModAlt > 0)
			case key.NameRightArrow:
				zv.right(ev.Modifiers&key.ModAlt > 0)
			case key.NameEscape, "Q":
				return true
			case "D":
				cursorPos := zv.screenPosition(zv.cell.X, zv.cell.Y)
				fmt.Printf("Cursor: Cell: %v  Screen:%v-%v) Screensize:%v\n",
					zv.cell, cursorPos.Min, cursorPos.Max, cursorPos.Size())
			case "=", "+":
				zv.zoomin()
			case "-":
				zv.zoomout()
			}
		}
	}

	return false
}

func bringToFront() error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	cmd := exec.Command("osascript")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		fmt.Fprintf(stdin, `tell application "System Events" to set frontmost of the first application process whose unix id is %d to true`, os.Getpid())
	}()

	return cmd.Run()
}

func empty16x16() *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 256, 256))

	lightGray := image.NewUniform(color.Gray{Y: 0x00})
	darkGray := image.NewUniform(color.Gray{Y: 0x33})

	draw.Draw(im, im.Bounds(), lightGray, image.Point{}, draw.Src)
	count := 8
	inc := 256 / count
	for x := 0; x < count; x++ {
		for y := 0; y < count; y++ {
			if x&1 == y&1 {
				draw.Draw(im, image.Rect(x*inc, y*inc, x*inc+inc, y*inc+inc), darkGray, image.Point{}, draw.Src)
			}
		}
	}
	// draw.Draw(im, image.Rect(0, 0, 128, 128), darkGray, image.Point{}, draw.Src)
	// draw.Draw(im, image.Rect(128, 128, 256, 256), darkGray, image.Point{}, draw.Src)

	return im
}
