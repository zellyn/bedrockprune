package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand/v2"
	"os"

	_ "github.com/zellyn/bedrockprune/lerp"
	"github.com/zellyn/bedrockprune/zoomview"

	"gioui.org/app"
)

func main() {
	m, err := newMaze(121, 121)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Bedrock Pruner"))
		err := zoomview.Run(w, m)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

type maze struct {
	width, height int
	cells         map[image.Point]rune
	grass         *image.RGBA
	stone         *image.RGBA
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

func (m *maze) Get(x, y int) (*image.RGBA, error) {
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

func getImage(base uint32, c color.Color) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 16, 16))
	r, g, b, _ := c.RGBA()
	for i := 0; i < 256; i++ {
		raw := uint32(rand.IntN(80)) + base
		col := color.RGBA{R: uint8(r * raw >> 16), G: uint8(g * raw >> 16), B: uint8(b * raw >> 16), A: 0xFF}
		im.Set(i%16, i/16, col)
	}
	return im
}
