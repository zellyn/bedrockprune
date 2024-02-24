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

	"github.com/zellyn/bedrockprune/f32"
	"github.com/zellyn/bedrockprune/tiles"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
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

	dpPerBlock unit.Dp
	pxPerBlock int
	min        f32.Point
	max        f32.Point
	blockSize  f32.Point
	blockCount image.Point
	scale      f32.Affine2D

	haveContextInfo bool
	metric          unit.Metric
	size            image.Point
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

func (v *view) left() {
	v.cell.X--
	if float32(v.cell.X)+1 <= v.topleft.X {
		v.topleft.X = float32(v.cell.X)
		v.calcpos()
	}
}

func (v *view) right() {
	v.cell.X += 1
	if float32(v.cell.X) >= v.topleft.X+v.blockSize.X {
		v.topleft.X = float32(v.cell.X) + 1 - v.blockSize.X
		v.calcpos()
	}
}

func (v *view) up() {
	v.cell.Y -= 1
	if float32(v.cell.Y)+1 <= v.topleft.Y {
		v.topleft.Y = float32(v.cell.Y)
		v.calcpos()
	}
}

func (v *view) down() {
	v.cell.Y += 1
	if float32(v.cell.Y) >= v.topleft.Y+v.blockSize.Y {
		v.topleft.Y = float32(v.cell.Y) + 1 - v.blockSize.Y
		v.calcpos()
	}
}

func (v *view) zoomin() {
	v.zoom++
	v.calc()
	fmt.Printf("zoom: %g  dpPerBlock: %g  blockSize: %v\n", v.zoom, v.dpPerBlock, v.blockSize)
}

func (v *view) zoomout() {
	v.zoom--
	v.calc()
	fmt.Printf("zoom: %g  dpPerBlock: %g  blockSize: %v\n", v.zoom, v.dpPerBlock, v.blockSize)
}

func (v *view) calc() {
	v.dpPerBlock = unit.Dp(zoomZeroDpPerBlock * math32.Pow(2, v.zoom))
	v.pxPerBlock = v.metric.Dp(v.dpPerBlock)
	v.blockCount = image.Pt(int(v.size.X/v.pxPerBlock)+1, int(v.size.Y/v.pxPerBlock)+1)
	v.blockSize = f32.Pt(float32(v.size.X), float32(v.size.Y)).Div(float32(v.pxPerBlock))
	v.scale = f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(float32(v.pxPerBlock), float32(v.pxPerBlock)))
	v.calcpos()
}

func (v *view) calcpos() {
	v.min = f32.Pt(math32.Floor(v.topleft.X), math32.Floor(v.topleft.Y))
	v.max = f32.Pt(v.min.X+float32(v.blockCount.X), v.min.Y+float32(v.blockCount.Y))
}

func (v *view) screenPosition(x, y float32) f32.Rectangle {
	left := unit.Dp(x-v.topleft.X) * v.dpPerBlock
	top := unit.Dp(y-v.topleft.Y) * v.dpPerBlock
	return f32.Rectangle{
		Min: f32.Pt(float32(v.metric.Dp(left)), float32(v.metric.Dp(top))),
		Max: f32.Pt(float32(v.metric.Dp(left+v.dpPerBlock)), float32(v.metric.Dp(top+v.dpPerBlock))),
	}
}

func run(w *app.Window, m *maze) error {
	if err := bringToFront(); err != nil {
		return err
	}
	v := view{
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
			event.Op(&ops, v)
			if !v.haveContextInfo || v.metric != gtx.Metric || v.size != gtx.Constraints.Max {
				v.setContextInfo(gtx.Metric, gtx.Constraints.Max)
			}

			// Process keypresses
			for {
				event, ok := gtx.Event(
					key.Filter{Name: key.NameUpArrow},
					key.Filter{Name: key.NameDownArrow},
					key.Filter{Name: key.NameLeftArrow},
					key.Filter{Name: key.NameRightArrow},
					key.Filter{Name: key.NameEscape}, key.Filter{Name: "Q", Optional: key.ModShift},
					key.Filter{Name: "="}, key.Filter{Name: "+", Optional: key.ModShift},
					key.Filter{Name: "-"},
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
						v.up()
					case key.NameDownArrow:
						v.down()
					case key.NameLeftArrow:
						v.left()
					case key.NameRightArrow:
						v.right()
					case key.NameEscape, "Q":
						return nil
					case "=", "+":
						v.zoomin()
					case "-":
						v.zoomout()
					}
				}
			}

			/*
				for y := v.min.Y; y <= v.max.Y; y++ {
					for x := v.min.X; x <= v.max.X; x++ {
						pos := v.screenPosition(x, y)
						imageOp := paint.NewImageOp(m.Get(int(x), int(y)))
						imageOp.Add(&ops)
						offset := op.Affine(v.scale.Offset(pos.Min)).Push(&ops)
						paint.PaintOp{}.Add(&ops)
						offset.Pop()
					}
				}
			*/

			area := f32.Rectangle{Min: v.min, Max: v.max}
			for _, imrect := range v.tileserver.Get(area, v.dpPerBlock) {
				pos := v.screenPosition(imrect.Area.Min.X, imrect.Area.Min.Y)
				imageOp := paint.NewImageOp(imrect.Image)
				imageOp.Add(&ops)
				offset := op.Affine(v.scale.Scale(f32.Point{}, imrect.Scale()).Offset(pos.Min)).Push(&ops)
				paint.PaintOp{}.Add(&ops)
				offset.Pop()
			}

			cursorPos := v.screenPosition(float32(v.cell.X), float32(v.cell.Y))
			bounds := cursorPos.ImRect()
			rrect := clip.RRect{Rect: bounds, SE: 3, SW: 3, NE: 3, NW: 3}
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

func (m *maze) Get(x, y int) *image.NRGBA {
	if m.cells[image.Point{X: x, Y: y}] == 0 {
		return m.grass
	}
	return m.stone
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
