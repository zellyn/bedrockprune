package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand/v2"
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

func main() {
	m, err := newMaze(121, 121)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	go func() {
		w := app.NewWindow(app.Title("Bedrock Pruner"))
		err := run(w, m)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

const zoomZeroDpPerBlock = 20

var one = f32.Pt(1, 1)

type view struct {
	maze       *maze
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

func (v *view) setContextInfo(metric unit.Metric, size image.Point) {
	v.metric = metric
	v.size = size
	v.haveContextInfo = true
	v.calc()
}

func (v *view) setZoom(zoom float32) {
	v.zoom = zoom
	if v.haveContextInfo {
		v.calc()
	}
}

func (v *view) left(fast bool) {
	v.cell.X--
	if fast {
		v.cell.X -= 9
	}
	if float32(v.cell.X)+1 <= v.topleft.X {
		v.topleft.X = float32(v.cell.X)
		v.calcpos()
	}
}

func (v *view) right(fast bool) {
	v.cell.X += 1
	if fast {
		v.cell.X += 9
	}
	if float32(v.cell.X) >= v.topleft.X+v.blockSize.X {
		v.topleft.X = float32(v.cell.X) + 1 - v.blockSize.X
		v.calcpos()
	}
}

func (v *view) up(fast bool) {
	v.cell.Y -= 1
	if fast {
		v.cell.Y -= 9
	}
	if float32(v.cell.Y)+1 <= v.topleft.Y {
		v.topleft.Y = float32(v.cell.Y)
		v.calcpos()
	}
}

func (v *view) down(fast bool) {
	v.cell.Y += 1
	if fast {
		v.cell.Y += 9
	}
	if float32(v.cell.Y) >= v.topleft.Y+v.blockSize.Y {
		v.topleft.Y = float32(v.cell.Y) + 1 - v.blockSize.Y
		v.calcpos()
	}
}

func (v *view) doZoom() {
	if v.lerping {
		now := time.Now()
		v.zoom = v.zoomLerp.At(now)
		factor := v.oldPow / math32.Pow(2, v.zoom)
		v.topleft.X = v.tlXLerp.At(now)
		v.topleft.Y = v.tlYLerp.At(now)
		v.topleft = v.oldCursorPos.Sub(v.zoomDiff.Mul(factor))
		if v.zoomLerp.Done(now) {
			v.lerping = false
		}
		v.calc()
	}
}

const zoomDuration = time.Second / 5

func (v *view) setZoomDetails(zoom float32, tl f32.Point, curve lerp.Curve) {
	if true {
		v.zoom = zoom
		v.topleft = tl
		v.calc()
	} else {

		now := time.Now()
		if !v.lerping {
			v.oldPow = math32.Pow(2, v.zoom)
			v.oldCursorPos = f32.Point{X: float32(v.cell.X) + 0.5, Y: float32(v.cell.Y) + 0.5}
			v.zoomDiff = v.oldCursorPos.Sub(v.topleft)
		}
		v.lerping = true
		v.zoomLerp = lerp.NewTimeLerp(v.zoom, zoom, now, zoomDuration, nil)
		v.tlXLerp = lerp.NewTimeLerp(v.topleft.X, tl.X, now, zoomDuration, curve)
		v.tlYLerp = lerp.NewTimeLerp(v.topleft.Y, tl.Y, now, zoomDuration, curve)
	}
}

func (v *view) targetZoom() float32 {
	if v.lerping {
		return v.zoomLerp.Target()
	}
	return v.zoom
}

func (v *view) zoomin() {
	cursorPos := f32.Point{X: float32(v.cell.X) + 0.5, Y: float32(v.cell.Y) + 0.5}
	diff := cursorPos.Sub(v.topleft)
	tl := v.topleft.Add(diff.Mul(0.5))

	v.setZoomDetails(v.targetZoom()+1, tl, lerp.Log2)
}

func (v *view) zoomout() {
	cursorPos := f32.Point{X: float32(v.cell.X) + 0.5, Y: float32(v.cell.Y) + 0.5}
	diff := cursorPos.Sub(v.topleft)
	tl := v.topleft.Sub(diff)

	v.setZoomDetails(v.targetZoom()-1, tl, lerp.Pow2)
}

func (v *view) calc() {
	v.dpPerBlock = unit.Dp(zoomZeroDpPerBlock * math32.Pow(2, v.zoom))
	v.pxPerBlock = v.metric.PxPerDp * float32(v.dpPerBlock)
	v.blockCount = image.Pt(int(float32(v.size.X)/v.pxPerBlock)+1, int(float32(v.size.Y)/v.pxPerBlock)+1)
	v.blockSize = f32.Pt(float32(v.size.X), float32(v.size.Y)).Div(float32(v.pxPerBlock))

	blockScale := float32(1) / 16
	v.unitsPerTile = 16
	dp := v.dpPerBlock
	for dp < minDpPerBlock {
		dp *= 2
		v.unitsPerTile *= 2
		blockScale *= 2
	}

	v.scale = f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(float32(v.pxPerBlock*blockScale), float32(v.pxPerBlock*blockScale)))

	v.calcpos()
}

func (v *view) calcpos() {
	v.min = f32.Pt(math32.Floor(v.topleft.X), math32.Floor(v.topleft.Y))
	v.max = f32.Pt(v.min.X+float32(v.blockCount.X), v.min.Y+float32(v.blockCount.Y))
}

func (v *view) screenPosition(x, y int) image.Rectangle {
	left := (float32(x) - v.topleft.X) * float32(v.dpPerBlock)
	top := (float32(y) - v.topleft.Y) * float32(v.dpPerBlock)
	return image.Rectangle{
		Min: image.Pt(int(v.metric.PxPerDp*left), int(v.metric.PxPerDp*top)),
		Max: image.Pt(int(v.metric.PxPerDp*(left+float32(v.dpPerBlock))), int(v.metric.PxPerDp*(top+float32(v.dpPerBlock)))),
	}
}

const minDpPerBlock = unit.Dp(5)

func run(w *app.Window, m *maze) error {
	if err := bringToFront(); err != nil {
		return err
	}
	v := &view{
		maze:       m,
		tileserver: tiles.NewServer(m),
		zoom:       1,
		topleft:    f32.Pt(1.5, 2.75),
		cell:       image.Pt(3, 3),
	}
	var ops op.Ops
	for {
		switch e := w.NextEvent().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			if v.lerping {
				v.doZoom()
				gtx.Execute(op.InvalidateCmd{})
			}

			event.Op(&ops, v)
			if !v.haveContextInfo || v.metric != gtx.Metric || v.size != gtx.Constraints.Max {
				v.setContextInfo(gtx.Metric, gtx.Constraints.Max)
			}

			area := image.Rectangle{Min: f32.PointFloor(v.min), Max: f32.PointCeil(v.max)}
			var empties []tiles.Tile

			// Process keypresses
			if quit := handleKeyPresses(gtx, v, v.unitsPerTile); quit {
				return nil
			}

			tiles, err := v.tileserver.Get(area, v.unitsPerTile)
			if err != nil {
				return err
			}
			for _, areaImage := range tiles {
				if areaImage.Empty {
					empties = append(empties, areaImage)
					continue
				}
				pos := v.screenPosition(areaImage.Area.Min.X, areaImage.Area.Min.Y)
				imageOp := paint.NewImageOp(areaImage.Image)
				imageOp.Add(&ops)
				offset := op.Affine(v.scale.Offset(f32.FromImagePt(pos.Min))).Push(&ops)
				paint.PaintOp{}.Add(&ops)

				width := 2.0 / v.scale.Transform(f32.Point{X: 1, Y: 1}).X

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
					pos := v.screenPosition(areaImage.Area.Min.X, areaImage.Area.Min.Y)
					offset := op.Affine(v.scale.Offset(f32.FromImagePt(pos.Min))).Push(&ops)
					paint.PaintOp{}.Add(&ops)
					offset.Pop()
				}
			}

			cursorPos := v.screenPosition(v.cell.X, v.cell.Y)
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

func handleKeyPresses(gtx layout.Context, v *view, units int) (quit bool) {
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
				v.up(ev.Modifiers&key.ModAlt > 0)
			case key.NameDownArrow:
				v.down(ev.Modifiers&key.ModAlt > 0)
			case key.NameLeftArrow:
				v.left(ev.Modifiers&key.ModAlt > 0)
			case key.NameRightArrow:
				v.right(ev.Modifiers&key.ModAlt > 0)
			case key.NameEscape, "Q":
				return true
			case "D":
				cursorPos := v.screenPosition(v.cell.X, v.cell.Y)
				fmt.Printf("Cursor: Cell: %v  Screen:%v-%v) Screensize:%v\n",
					v.cell, cursorPos.Min, cursorPos.Max, cursorPos.Size())
			case "=", "+":
				v.zoomin()
			case "-":
				v.zoomout()
			}
		}
	}

	return false
}

type maze struct {
	width, height int
	cells         map[image.Point]rune
	grass         *image.NRGBA
	stone         *image.NRGBA
}

func newMaze(width, height int) (*maze, error) {
	if width < 3 || height < 3 || width%2 == 0 || height%2 == 0 {
		return nil, fmt.Errorf("width and height must be positive odd numbers greater than 1")
	}

	m := &maze{
		width:  width,
		height: height,
		cells:  make(map[image.Point]rune),
		grass:  getImage(120, color.RGBA{R: 0x92, G: 0xBD, B: 0x59}),
		stone:  getImage(40, color.Alpha{A: 0xFF}),
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			m.cells[image.Pt(x, y)] = '#'
		}
	}

	dirs4 := []image.Point{{X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: -1}, {X: 0, Y: 1}}
	todo := []image.Point{{X: 1, Y: 1}}

OUTER:
	for len(todo) > 0 {
		pos := todo[len(todo)-1]

		for _, dirNum := range rand.Perm(4) {
			dir := dirs4[dirNum]
			if m.cells[pos.Add(dir.Mul(2))] == 0 {
				continue
			}

			newPos := pos.Add(dir.Mul(2))
			todo = append(todo, newPos)
			delete(m.cells, newPos)
			delete(m.cells, pos.Add(dir))
			continue OUTER
		}

		todo = todo[:len(todo)-1]
	}

	return m, nil
}

func (m *maze) Get(x, y int) (*image.NRGBA, error) {
	if m.cells[image.Point{X: x, Y: y}] == 0 {
		return m.grass, nil
	}
	return m.stone, nil
}

func (m *maze) AllEmpty(area image.Rectangle) (bool, error) {
	if area.Min.X > m.width || area.Min.Y > m.height || area.Max.X < 0 || area.Max.Y < 0 {
		return true, nil
	}
	for x := area.Min.X; x < area.Max.X; x++ {
		for y := area.Min.Y; y < area.Max.Y; y++ {
			if m.cells[image.Point{X: x, Y: y}] != 0 {
				return false, nil
			}
		}
	}
	return true, nil
}

func getImage(base uint32, c color.Color) *image.NRGBA {
	im := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	r, g, b, _ := c.RGBA()
	for i := 0; i < 256; i++ {
		raw := uint32(rand.IntN(80)) + base
		col := color.NRGBA{R: uint8(r * raw >> 16), G: uint8(g * raw >> 16), B: uint8(b * raw >> 16), A: 0xFF}
		im.Set(i%16, i/16, col)
	}
	return im
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
