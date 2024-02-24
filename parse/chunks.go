package parse

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/util"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// MakeChunkPrefix makes the eight- or twelve-byte leveldb key prefix
// for a chunk position in the given Dimension.
func MakeChunkPrefix(chunkPos world.ChunkPos, dimension world.Dimension) []byte {
	var res []byte
	if dimension == world.Overworld {
		res = make([]byte, 8)
	} else {
		res = make([]byte, 12)
		dimID, _ := world.DimensionID(dimension)
		putInt32(res[8:12], int32(dimID))
	}

	putInt32(res[0:4], chunkPos.X())
	putInt32(res[4:8], chunkPos.Z())

	return res
}

// ParseChunkPrefix parses the beginning 8 or 12 bytes of a key into a
// world.ChunkPos, and Dimension.
func ParseChunkPrefix(key []byte) (chunkPos world.ChunkPos, dimension world.Dimension, err error) {
	if len(key) < 8 {
		return chunkPos, dimension, fmt.Errorf("need at least 8-bytes key to parse chunk prefix; got %q (%d)", key, len(key))
	}

	chunkPos = getChunkPos(key)
	dimension = world.Overworld
	if len(key) >= 12 {
		var ok bool
		dimension, ok = world.DimensionByID(int(getInt32(key[8:12])))
		if !ok {
			return chunkPos, dimension, fmt.Errorf("unknown dimension: %d", getInt32(key[8:12]))
		}
	}
	return chunkPos, dimension, nil
}

// SaneChunkLimit is an upper limit on x and z chunk coordinates we
// expect to see. It's taken from the "Major Effects" section of
// https://minecraft.wiki/w/Bedrock_Edition_distance_effects
const SaneChunkLimit = 1 << 16

func ParseSaneChunkPrefix(key []byte) (chunkPos world.ChunkPos, dimension world.Dimension, err error) {
	chunkPos, dimension, err = ParseChunkPrefix(key)
	if err != nil {
		return chunkPos, dimension, err
	}

	if chunkPos.X() < -SaneChunkLimit || chunkPos.X() > SaneChunkLimit {
		return chunkPos, dimension, fmt.Errorf("x (%d) is beyond the distance where major floating point problems occur", chunkPos.X())
	}
	if chunkPos.Z() < -SaneChunkLimit || chunkPos.Z() > SaneChunkLimit {
		return chunkPos, dimension, fmt.Errorf("z (%d) is beyond the distance where major floating point problems occur", chunkPos.Z())
	}
	return chunkPos, dimension, nil
}

// AllEntriesWithChunkCoordinatePrefix returns a list of KeyVal items
// containing all entries in the world leveldb that have the chunk
// addressing prefix corresponding to the chunk coordinates and
// Dimension.
func AllEntriesWithChunkCoordinatePrefix(db *leveldb.DB, chunkPos world.ChunkPos, dimension world.Dimension) ([]*KeyVal, error) {
	var res []*KeyVal

	prefix := MakeChunkPrefix(chunkPos, dimension)

	iter := db.NewIterator(util.BytesPrefix(prefix), nil)

	for iter.Next() {
		res = append(res, NewKeyVal(iter.Key(), iter.Value()))
	}

	iter.Release()
	err := iter.Error()
	return res, err
}

// These are from
// https://learn.microsoft.com/en-us/minecraft/creator/documents/actorstorage?view=minecraft-bedrock-stable#non-actor-data-chunk-key-ids
//
// The minecraft wiki disagrees (in ways that look suspect): https://minecraft.wiki/w/Bedrock_Edition_level_format
type LevelChunkTag int

const (
	LevelChunkTagUnknown LevelChunkTag = 0

	LevelChunkTagData3D                             LevelChunkTag = 43  // '+'
	LevelChunkTagVersion                            LevelChunkTag = 44  // ',' This was moved to the front as needed for the extended heights feature. Old chunks will not have this data.
	LevelChunkTagData2D                             LevelChunkTag = 45  // '-'
	LevelChunkTagData2DLegacy                       LevelChunkTag = 46  // '.'
	LevelChunkTagSubChunkPrefix                     LevelChunkTag = 47  // '/'
	LevelChunkTagLegacyTerrain                      LevelChunkTag = 48  // '0'
	LevelChunkTagBlockEntity                        LevelChunkTag = 49  // '1'
	LevelChunkTagEntity                             LevelChunkTag = 50  // '2'
	LevelChunkTagPendingTicks                       LevelChunkTag = 51  // '3'
	LevelChunkTagLegacyBlockExtraData               LevelChunkTag = 52  // '4'
	LevelChunkTagBiomeState                         LevelChunkTag = 53  // '5'
	LevelChunkTagFinalizedState                     LevelChunkTag = 54  // '6'
	LevelChunkTagConversionData                     LevelChunkTag = 55  // '7' data that the converter provides, that are used at runtime for things like blending
	LevelChunkTagBorderBlocks                       LevelChunkTag = 56  // '8'
	LevelChunkTagHardcodedSpawners                  LevelChunkTag = 57  // '9'
	LevelChunkTagRandomTicks                        LevelChunkTag = 58  // ':'
	LevelChunkTagCheckSums                          LevelChunkTag = 59  // ';'
	LevelChunkTagGenerationSeed                     LevelChunkTag = 60  // '<'
	LevelChunkTagGeneratedPreCavesAndCliffsBlending LevelChunkTag = 61  // '=' not used, DON'T REMOVE
	LevelChunkTagBlendingBiomeHeight                LevelChunkTag = 62  // '>' not used, DON'T REMOVE
	LevelChunkTagMetaDataHash                       LevelChunkTag = 63  // '?'
	LevelChunkTagBlendingData                       LevelChunkTag = 64  // '@'
	LevelChunkTagActorDigestVersion                 LevelChunkTag = 65  // 'A'
	LevelChunkTagLegacyVersion                      LevelChunkTag = 118 // 'v'
)

// LevelChunkTagToString maps LevelChunkTag to a string with its name.
var LevelChunkTagToString = map[LevelChunkTag]string{
	LevelChunkTagUnknown:                            "LevelChunkTagUnknown",
	LevelChunkTagData3D:                             "LevelChunkTagData3D",
	LevelChunkTagVersion:                            "LevelChunkTagVersion",
	LevelChunkTagData2D:                             "LevelChunkTagData2D",
	LevelChunkTagData2DLegacy:                       "LevelChunkTagData2DLegacy",
	LevelChunkTagSubChunkPrefix:                     "LevelChunkTagSubChunkPrefix",
	LevelChunkTagLegacyTerrain:                      "LevelChunkTagLegacyTerrain",
	LevelChunkTagBlockEntity:                        "LevelChunkTagBlockEntity",
	LevelChunkTagEntity:                             "LevelChunkTagEntity",
	LevelChunkTagPendingTicks:                       "LevelChunkTagPendingTicks",
	LevelChunkTagLegacyBlockExtraData:               "LevelChunkTagLegacyBlockExtraData",
	LevelChunkTagBiomeState:                         "LevelChunkTagBiomeState",
	LevelChunkTagFinalizedState:                     "LevelChunkTagFinalizedState",
	LevelChunkTagConversionData:                     "LevelChunkTagConversionData",
	LevelChunkTagBorderBlocks:                       "LevelChunkTagBorderBlocks",
	LevelChunkTagHardcodedSpawners:                  "LevelChunkTagHardcodedSpawners",
	LevelChunkTagRandomTicks:                        "LevelChunkTagRandomTicks",
	LevelChunkTagCheckSums:                          "LevelChunkTagCheckSums",
	LevelChunkTagGenerationSeed:                     "LevelChunkTagGenerationSeed",
	LevelChunkTagGeneratedPreCavesAndCliffsBlending: "LevelChunkTagGeneratedPreCavesAndCliffsBlending",
	LevelChunkTagBlendingBiomeHeight:                "LevelChunkTagBlendingBiomeHeight",
	LevelChunkTagMetaDataHash:                       "LevelChunkTagMetaDataHash",
	LevelChunkTagBlendingData:                       "LevelChunkTagBlendingData",
	LevelChunkTagActorDigestVersion:                 "LevelChunkTagActorDigestVersion",
	LevelChunkTagLegacyVersion:                      "LevelChunkTagLegacyVersion",
}

// String returns the string representation of the given LevelChunkTag.
func (lct LevelChunkTag) String() string {
	if s := LevelChunkTagToString[lct]; s != "" {
		return s
	}
	return "LevelChunkTagINVALID:%d" + strconv.Itoa(int(lct))
}

// https://minecraft.wiki/w/Bedrock_Edition_level_format
var levelChunkTagNames = map[byte]string{
	'+': "Data3D",
	',': "Version",
	'-': "Data2D",
	'.': "Data2DLegacy",
	'/': "SubChunkPrefix",
	'1': "BlockEntity",
	'2': "Entity",
	'3': "PendingTicks",
	'4': "LegacyBlockExtraData",
	'5': "BiomeState",
	'6': "FinalizedState",
	'7': "ConversionData",
	'8': "BorderBlocks",
	'9': "HardcodedSpawners",
	':': "RandomTicks",
	';': "Checksums",
	'=': "MetaDataHash",
	'>': "GeneratedPreCavesAndCliffsBlending",
	'?': "BlendingBiomeHeight",
	'@': "BlendingData",
	'A': "ActorDigestVersion",
	'v': "LegacyVersion",
}

type Chunk struct {
	Dimension world.Dimension
	ChunkPos  world.ChunkPos
	SubChunks []subChunk
	MaxLayer  int
	KeyVals   []*KeyVal
	HeightMap *HeightMap
}

type subChunk struct {
	subChunkIndex   int32
	subChunkVersion int
	layerCount      int
	yIndex          int32
	layers          []subChunkLayer
}

func (s subChunk) Print(w io.Writer) {
	fmt.Fprintf(w, "Subchunk {\n")
	fmt.Fprintf(w, "  index: %d\n", s.subChunkIndex)
	fmt.Fprintf(w, "  version: %d\n", s.subChunkVersion)
	fmt.Fprintf(w, "  layerCount: %d\n", s.layerCount)
	fmt.Fprintf(w, "  yIndex: %d\n", s.yIndex)
	fmt.Fprintf(w, "}\n")
}

type subChunkIndices [4096]int

func (s subChunkIndices) get(x, z, y int) int {
	return s[x<<8+z<<4+y]
}

type subChunkLayer struct {
	blockEntries *subChunkIndices
	palettes     []map[string]any
	allAir       *bool
	airIndex     *int
}

var seenBitsPerBlock = map[int]bool{}

var blocksPerWordForBitsPerBlock = map[int]int{
	1:  32,
	2:  16,
	3:  10,
	4:  8,
	5:  6,
	6:  5,
	8:  4,
	16: 2,
}

var wordCountForBitsPerBlock = map[int]int{
	0:  0,
	1:  128,
	2:  256,
	3:  410,
	4:  512,
	5:  683,
	6:  820,
	8:  1024,
	16: 2048,
}

/*
Notes:

	subChunkIndex may be the sum of a counter and the y-coordinate?
	https://github.com/df-mc/dragonfly/blob/f392edaffa84d73d48628fa2bd85c93908bc7166/server/world/mcdb/db.go#L224
	```
	y := uint8(i + (r[0] >> 4))
	sub[i], err = db.ldb.Get(k.Sum(keySubChunkData, y), nil)
	```
*/
func ParseSubChunk(kv *KeyVal) (subChunk, error) {
	var res subChunk

	if !kv.KeyType().IsSubChunkPrefix() {
		return res, fmt.Errorf("cannot parse subChunk for key/value of type %s", kv.KeyType())
	}
	res.subChunkIndex = int32(int8(kv.Key[len(kv.Key)-1]))

	res.subChunkVersion = int(kv.Val[0])
	res.layerCount = int(kv.Val[1])
	res.yIndex = int32(int8(kv.Val[2]))
	// if res.layerCount > 1 {
	// 	fmt.Printf("layercount=%d\n", res.layerCount)
	// }

	if res.subChunkIndex != res.yIndex {
		return res, fmt.Errorf("subChunkIndex (%d) != yIndex (%d)", res.subChunkIndex, res.yIndex)
	}

	if res.subChunkVersion != 9 {
		return res, fmt.Errorf("decoding for subChunk version %d not implemented (key=%v)", res.subChunkVersion, kv.Key)
	}

	buf := bytes.NewBuffer(kv.Val[3:])

	for layerIndex := 0; layerIndex < res.layerCount; layerIndex++ {
		if layerIndex > 0 {
			// fmt.Printf("skipping layer 2\n")
			return res, nil
		}
		var layer subChunkLayer
		paletteType, err := buf.ReadByte()
		if err != nil {
			return res, fmt.Errorf("cannot read palette type (key=%v, layer=%d", kv.Key, layerIndex)
		}

		// fmt.Printf("subChunkIndex:%d subChunkVersion:%d layerCount:%d yIndex:%d paletteType:%d\n", res.subChunkIndex, res.subChunkVersion, res.layerCount, res.yIndex, paletteType)
		if paletteType&1 == 1 {
			// Runtime format? We don't expect that.
			return res, fmt.Errorf("parsing of runtime subChunk representations not implemented (key=%v, layer=%d, paletteType=%d)\n%v", kv.Key, layerIndex, paletteType, kv.Val)
		}

		bitsPerBlock := int(paletteType) >> 1
		if bitsPerBlock == 0x7f {
			// https://github.com/df-mc/dragonfly/blob/f392edaffa84d73d48628fa2bd85c93908bc7166/server/world/chunk/decode.go#L159
			return res, fmt.Errorf("Unable to decode block with 0x7F (127) bits per block (key=%v, layer=%d)", kv.Key, layerIndex)
		}

		if !seenBitsPerBlock[bitsPerBlock] {
			seenBitsPerBlock[bitsPerBlock] = true
			fmt.Printf("First sight of %d bits per block for subchunk (key=%v, layer=%d)\n", bitsPerBlock, kv.Key, layerIndex)
		}

		paletteEntryCount := 1

		wordCount, ok := wordCountForBitsPerBlock[bitsPerBlock]
		if !ok {
			return res, fmt.Errorf("unimplemented bits-per-block: %d (key=%v, layer=%d)", bitsPerBlock, kv.Key, layerIndex)
		}

		// fmt.Printf("reading %d * 4 = %d bytes for block entries\n", wordCount, wordCount*4)
		wordBytes := buf.Next(wordCount * 4)
		if len(wordBytes) < wordCount*4 {
			return res, fmt.Errorf("ran out of bytes for block entries (key=%v, layer=%d)", kv.Key, layerIndex)
		}
		layer.blockEntries = readBlockEntries(wordBytes, bitsPerBlock)

		if bitsPerBlock > 0 {
			paletteEntryCount, err = readUint32AsInt(buf)
			if err != nil {
				return res, fmt.Errorf("unable to read palette entry count for subchunk (key=%v, layer=%d): %w", kv.Key, layerIndex, err)
			}
		}
		// fmt.Printf("%d palette entries\n", paletteEntryCount)

		for i, paletteIndex := range layer.blockEntries {
			if paletteIndex >= paletteEntryCount {
				return res, fmt.Errorf("block %d has palette index %d, which is >= %d (key=%v, layer=%d): %w", i, paletteIndex, paletteEntryCount, kv.Key, layerIndex, err)
			}
		}

		d := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)
		for i := 0; i < paletteEntryCount; i++ {
			// fmt.Printf("About to read palette %d at at: %v\n", i, []byte(buf.String()[:10]))
			var m map[string]any
			err := d.Decode(&m)
			if err != nil {
				// fmt.Printf("Key: %v\nValue:\n%v\n", kv.Key, kv.Val)
				return res, fmt.Errorf("unable to decode palette entry %d (key=%v, layer=%d): %w", i, kv.Key, layerIndex, err)
			}

			// fmt.Printf("%v\n", m)

			if m["name"] == "air" {
				layer.airIndex = &i
			}

			layer.palettes = append(layer.palettes, m)
		}

		if bitsPerBlock == 0 && layer.airIndex != nil && *layer.airIndex == 0 {
			t := true
			layer.allAir = &t
		}

		res.layers = append(res.layers, layer)
	}

	return res, nil
}

func readBlockEntries(bb []byte, bitsPerBlock int) *subChunkIndices {
	index := 0
	var indices subChunkIndices
	mask := (1 << bitsPerBlock) - 1
	bitsLeft := 0
	word := 0
	for i := range indices {
		if bitsLeft < bitsPerBlock {
			word = int(bb[index]) + int(bb[index+1])<<8 + int(bb[index+2])<<16 + int(bb[index+3])<<24
			index += 4
			bitsLeft = 32
		}
		bitsLeft -= bitsPerBlock
		indices[i] = int(word & mask)
		word >>= bitsPerBlock
	}

	if index != len(bb) {
		panic(fmt.Sprintf("Want zero bytes left after reading block entry bits; got %d", len(bb)-index))
	}

	return &indices
}
