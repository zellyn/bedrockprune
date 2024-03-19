package parse

import (
	"fmt"
	"math"
	"sort"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/goleveldb/leveldb"
)

const NoHeight = math.MinInt32

type HeightMap [16][16]int32

func (hm *HeightMap) Get(x, z int) int32 {
	if hm == nil {
		return 0
	}
	return hm[z][x]
}

func GetChunk(db *leveldb.DB, chunkPos world.ChunkPos, dimension world.Dimension) (*Chunk, error) {
	res := &Chunk{
		Dimension: dimension,
		ChunkPos:  chunkPos,
	}

	kvs, err := AllEntriesWithChunkCoordinatePrefix(db, chunkPos, dimension)
	if err != nil {
		return nil, err
	}

	typeMap := make(map[LevelChunkTag][]*KeyVal)

	for _, kv := range kvs {
		kt := kv.KeyType()
		if kt.IsChunkDataForDimension(dimension) {
			res.KeyVals = append(res.KeyVals, kv)
			lct := kt.LevelChunkTag()
			typeMap[lct] = append(typeMap[lct], kv)
		}
	}

	for _, kv := range typeMap[LevelChunkTagSubChunkPrefix] {
		subchunk, err := ParseSubChunk(kv)
		if err != nil {
			return nil, err
		}
		res.SubChunks = append(res.SubChunks, subchunk)
		if subchunk.layerCount > res.MaxLayer {
			res.MaxLayer = subchunk.layerCount
		}
	}

	sort.Slice(res.SubChunks, func(i, j int) bool {
		return res.SubChunks[i].yIndex < res.SubChunks[j].yIndex
	})

	return res, nil
}

func (ch Chunk) GetBlock(x, z int, y int32, layer int) (map[string]any, error) {
	if layer > ch.MaxLayer {
		return nil, nil
	}
	if x < 0 || x > 15 || z < 0 || z > 15 {
		return nil, fmt.Errorf("GetBlock expects 0<=x<=15 (got %d), 0<=z<=15 (got %d)", x, z)
	}
	yIndex := y >> 4
	y16 := int(y & 0xf)
	for _, sc := range ch.SubChunks {
		if sc.yIndex != yIndex {
			continue
		}
		if len(sc.layers) <= layer {
			return nil, nil
		}

		subChunkLayer := sc.layers[layer]
		if subChunkLayer.blockEntries == nil {
			return nil, fmt.Errorf("no blockEntries found for subChunkLayer with yIndex==%d", yIndex)
		}
		paletteIndex := subChunkLayer.blockEntries.get(x, z, y16)
		if len(subChunkLayer.palettes) <= paletteIndex {
			return nil, fmt.Errorf("block palette index (%d) points past end of palette slice (%d)", paletteIndex, len(subChunkLayer.palettes))
		}
		return subChunkLayer.palettes[paletteIndex], nil
	}
	return nil, nil
}

func (ch Chunk) MaxY() int32 {
	yIndex := ch.SubChunks[len(ch.SubChunks)-1].yIndex
	return yIndex<<4 + 15
}

func (ch Chunk) MinY() int32 {
	if ch.Empty() {
		return NoHeight
	}
	yIndex := ch.SubChunks[0].yIndex
	return yIndex << 4
}

func (ch Chunk) Empty() bool {
	return len(ch.SubChunks) == 0
}

func (ch *Chunk) GetHeightMap(layer int) *HeightMap {
	if len(ch.HeightMaps) > layer && ch.HeightMaps[layer] != nil {
		return ch.HeightMaps[layer]
	}

	var hm HeightMap
	for z := range 16 {
		for x := range 16 {
			hm[z][x] = NoHeight
			for y := ch.MaxY(); y >= ch.MinY(); y-- {
				b, err := ch.GetBlock(x, z, y, layer)
				if err == nil && b != nil && b["name"] != "minecraft:air" {
					hm[z][x] = y
					break
				}
			}
		}
	}

	for len(ch.HeightMaps) <= layer {
		ch.HeightMaps = append(ch.HeightMaps, nil)
	}

	ch.HeightMaps[layer] = &hm
	return &hm
}
