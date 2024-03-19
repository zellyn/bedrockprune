package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/goleveldb/leveldb"
	_ "github.com/zellyn/bedrockprune/lerp"
	"github.com/zellyn/bedrockprune/occupation"
	"github.com/zellyn/bedrockprune/parse"
	"github.com/zellyn/bedrockprune/resources"
	"github.com/zellyn/bedrockprune/types"
	"github.com/zellyn/bedrockprune/zoomview"

	"gioui.org/app"
)

type worldTileSource16 struct {
	db             *leveldb.DB
	occupiedChunks map[world.ChunkPos]bool
	occupation     occupation.Map
	textureSource  *resources.TextureSource
	dimension      world.Dimension
	chunkCache     map[world.ChunkPos]*parse.Chunk
}

func run() error {
	fmt.Printf("Getting texture source from downloaded assets...")
	ts, err := resources.NewTextureSource(context.Background(), resources.UseOnlyCached)
	if err != nil {
		fmt.Println()
		return err
	}
	fmt.Printf(" done\n")

	fmt.Printf("Getting occupied chunks...")
	db, err := leveldb.OpenFile("./worlds/survivalone/db", nil)
	if err != nil {
		fmt.Println()
		return err
	}
	defer db.Close()

	occupiedChunks := parse.GetOccupiedChunkCoordinates(db)[world.Overworld]
	occ := occupation.New(occupiedChunks)
	fmt.Printf(" done\n")

	wts16 := &worldTileSource16{
		db:             db,
		occupiedChunks: occupiedChunks,
		occupation:     occ,
		textureSource:  ts,
		dimension:      world.Overworld,
		chunkCache:     make(map[world.ChunkPos]*parse.Chunk),
	}

	go func() {
		w := new(app.Window)
		w.Option(app.Title("Bedrock Pruner"))
		err := zoomview.Run(w, wts16)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func (wts *worldTileSource16) AllEmpty(area image.Rectangle) (bool, error) {
	minX := int32(area.Min.X >> 4)
	minY := int32(area.Min.Y >> 4)
	maxX := int32((area.Max.X + 15) >> 4)
	maxY := int32((area.Max.Y + 15) >> 4)

	return wts.occupation.AllEmpty(minX, minY, maxX, maxY), nil

	for x := minX; x < maxX; x++ {
		for y := minY; y < maxY; y++ {
			if !wts.occupiedChunks[world.ChunkPos{x, y}] {
				continue
			}
			chunk, err := wts.getChunk(world.ChunkPos{x, y})
			if err != nil {
				continue
			}
			if !chunk.Empty() {
				return false, nil
			}

		}
	}
	return true, nil
}

func (wts *worldTileSource16) getChunk(chunkPos world.ChunkPos) (*parse.Chunk, error) {
	chunk, ok := wts.chunkCache[chunkPos]
	if !ok {
		var err error
		chunk, err = parse.GetChunk(wts.db, chunkPos, wts.dimension)
		if err != nil {
			return nil, err
		}
		wts.chunkCache[chunkPos] = chunk
	}
	return chunk, nil
}

func (wts *worldTileSource16) Get(x, z int) (*image.RGBA, error) {
	chunkPos := world.ChunkPos{int32(x >> 4), int32(z >> 4)}
	if !wts.occupiedChunks[chunkPos] {
		return nil, fmt.Errorf("%w: no world data at position (%d,%d), dimension %s", types.ErrNotFound, x, z, wts.dimension)
	}

	chunk, err := wts.getChunk(chunkPos)
	if err != nil {
		return nil, err
	}

	layer := 0
	hm := chunk.GetHeightMap(layer)
	y := hm.Get(x&0xF, z&0xF)
	block, err := chunk.GetBlock(x&0xF, z&0xF, y, layer)
	if err != nil {
		return nil, err
	}

	img, err := wts.textureSource.Get(block)
	if err != nil {
		if img == nil {
			return nil, err
		}
		fmt.Printf("Error getting image at (%d,%d): %v\n", x, z, err)
	}

	return img, nil
}

func (wts *worldTileSource16) Info(x, z int) (string, error) {
	chunkPos := world.ChunkPos{int32(x >> 4), int32(z >> 4)}
	if !wts.occupiedChunks[chunkPos] {
		return "empty", nil
	}

	chunk, err := wts.getChunk(chunkPos)
	if err != nil {
		return "", err
	}

	layer := 0
	hm := chunk.GetHeightMap(layer)
	y := hm.Get(x&0xF, z&0xF)
	block, err := chunk.GetBlock(x&0xF, z&0xF, y, layer)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", block), nil
}
