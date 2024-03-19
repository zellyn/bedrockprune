package tiles

import (
	"errors"
	"fmt"
	"image"

	"github.com/zellyn/bedrockprune/types"
	"golang.org/x/image/draw"
)

// TileSource16 is the interface for a source of unit area images, where images are 16x16 pixels.
type TileSource16 interface {
	Get(x, y int) (*image.RGBA, error)
	AllEmpty(image.Rectangle) (bool, error)
	Info(x, y int) (string, error)
}

// TileServer is a source/cache of tile images generated from an
// underlying TileSource16.
type TileServer struct {
	source     TileSource16
	cache      map[int]map[image.Point]cacheEntry
	empty      map[int]*image.RGBA
	maxUnits   int
	empty16x16 *image.RGBA
}

type option func(ts *TileServer)

func OptionEmpty16x16(img *image.RGBA) option {
	return func(ts *TileServer) {
		ts.empty16x16 = img
	}
}

// NewServer creates a new TileServer, with the given TileSource16
func NewServer(source TileSource16, opts ...option) *TileServer {
	ts := &TileServer{
		source: source,
		cache:  make(map[int]map[image.Point]cacheEntry),
		empty:  make(map[int]*image.RGBA),
	}

	for _, opt := range opts {
		opt(ts)
	}

	return ts
}

type cacheEntry struct {
	Image *image.RGBA
	Empty bool
}

// Tile is the return type for a tile. It contains the covered area,
// the image itself, and whether the entire area was empty.
type Tile struct {
	Area  image.Rectangle // The area this tile covers
	Image *image.RGBA     // The image. If units was > 16, this will be scaled down
	Empty bool            // True if the whole area was "empty" according to the TileSource16
}

// Invalidate any cached tiles or images for the given area.
func (ts *TileServer) Invalidate(area image.Rectangle) {
	var pos image.Point
	for pos.Y = area.Min.Y; pos.Y <= area.Max.Y; pos.Y++ {
		for pos.X = area.Min.X; pos.X <= area.Max.X; pos.X++ {
			delete(ts.cache[1], pos)
		}
	}

	for units := 16; units <= ts.maxUnits; units *= 2 {
		area.Min.X &^= (units - 1)
		area.Min.Y &^= (units - 1)

		var pos image.Point
		for pos.Y = area.Min.Y; pos.Y <= area.Max.Y; pos.Y += units {
			for pos.X = area.Min.X; pos.X <= area.Max.X; pos.X += units {
				delete(ts.cache[units], pos)
			}
		}
	}
}

func (ts *TileServer) Get(area image.Rectangle, units int) ([]Tile, error) {

	if units < 16 || units&(units-1) > 1 {
		return nil, fmt.Errorf("units should be a power of 2 greater than 8; got %d", units)
	}

	if units > ts.maxUnits {
		ts.maxUnits = units
	}

	var res []Tile

	area.Min.X &^= (units - 1)
	area.Min.Y &^= (units - 1)

	for y := area.Min.Y; y <= area.Max.Y; y += units {
		for x := area.Min.X; x <= area.Max.X; x += units {
			im, empty, err := ts.get(image.Point{X: x, Y: y}, units)
			if err != nil {
				return nil, err
			}
			res = append(res, Tile{
				Area:  image.Rect(x, y, x+units, y+units),
				Image: im,
				Empty: empty,
			})
		}
	}

	// fmt.Printf("units: %d  imageSize: %d\n", units, res[0].Image.Bounds().Max.X)

	return res, nil
}

func (ts *TileServer) get(pos image.Point, units int) (*image.RGBA, bool, error) {
	if entry, ok := ts.cache[units][pos]; ok {
		return entry.Image, entry.Empty, nil
	}
	cache := ts.cache[units]

	if cache == nil {
		cache = make(map[image.Point]cacheEntry)
		ts.cache[units] = cache
	}

	emptyCheckRect := image.Rectangle{Min: pos, Max: image.Point{X: pos.X + units, Y: pos.Y + units}}

	if units == 16 {
		empty, err := ts.source.AllEmpty(emptyCheckRect)
		if err != nil {
			return nil, false, err
		}
		if empty {
			if ts.empty[units] == nil && ts.empty16x16 != nil {
				ts.empty[units] = ts.empty16x16
			}
			if ts.empty[units] != nil {
				cache[pos] = cacheEntry{
					Image: ts.empty[units],
					Empty: true,
				}
				return ts.empty[units], true, nil
			}
		}

		im := image.NewRGBA(image.Rect(0, 0, 256, 256))
		area := image.Rectangle{}
		for y := 0; y < 16; y++ {
			area.Min.Y = y * 16
			area.Max.Y = y*16 + 16
			for x := 0; x < 16; x++ {
				area.Min.X = x * 16
				area.Max.X = x*16 + 16
				littleIm, err := ts.source.Get(x+pos.X, y+pos.Y)
				if err != nil {
					if errors.Is(err, types.ErrNotFound) {
						continue
					}
					return nil, false, err
				}
				draw.Draw(im, area, littleIm, image.Point{}, draw.Src)
			}
		}

		if empty {
			ts.empty[units] = im
		}

		cache[pos] = cacheEntry{
			Image: im,
			Empty: empty,
		}
		return im, empty, nil
	}

	allEmpty := true
	half := units / 2

	// If we're missing the first underlying square, try to just ask about emptiness for the whole area.
	if _, haveBelow := ts.cache[half][pos]; !haveBelow {
		var err error
		allEmpty, err = ts.source.AllEmpty(emptyCheckRect)
		if err != nil {
			return nil, false, err
		}
		if allEmpty {
			if ts.empty[units] == nil && ts.empty16x16 != nil {
				ts.empty[units] = ts.empty16x16
			}
			if ts.empty[units] != nil {
				cache[pos] = cacheEntry{
					Image: ts.empty[units],
					Empty: true,
				}
				return ts.empty[units], true, nil
			}
		}
	}

	im := image.NewRGBA(image.Rect(0, 0, 256, 256))
	area := image.Rectangle{}
	for y := 0; y < 2; y++ {
		area.Min.Y = y * 128
		area.Max.Y = y*128 + 128
		for x := 0; x < 2; x++ {
			area.Min.X = x * 128
			area.Max.X = x*128 + 128
			littleIm, empty, err := ts.get(image.Point{X: pos.X + x*half, Y: pos.Y + y*half}, half)
			if err != nil {
				return nil, false, err
			}
			if !empty {
				allEmpty = false
			}

			if empty && ts.empty16x16 != nil {
				draw.Draw(im, area, ts.empty16x16, area.Min, draw.Src)
			} else {
				draw.BiLinear.Scale(im, area, littleIm, littleIm.Bounds(), draw.Src, nil)
			}
		}
	}

	if allEmpty {
		if ts.empty[units] != nil {
			im = ts.empty[units]
		} else {
			ts.empty[units] = im
		}
	}

	cache[pos] = cacheEntry{
		Image: im,
		Empty: allEmpty,
	}
	return im, allEmpty, nil
}
