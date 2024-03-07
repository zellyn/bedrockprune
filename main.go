package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/zellyn/bedrockprune/parse"
	"github.com/zellyn/bedrockprune/resources"
	"golang.org/x/exp/maps"
)

var asciiRe = regexp.MustCompile("^[a-zA-Z0-9_-]+$")
var asciiPrefix = regexp.MustCompile("^[a-zA-Z_-]{3,}")
var hexSuffix = regexp.MustCompile("_[a-f]{1,3}$")

var knownPrefixes = []string{
	"AutonomousEntities",
	"BiomeData",
	"LevelChunkMetaDataDictionary",
	"Nether",
	"Overworld",
	"TheEnd",
	"mobevents",
	"portals",
	"schedulerWT",
	"scoreboard",

	"VILLAGE_",
	"VILLAGE_Overworld_",
	"map_-",
	"player_server_",
	"player_",

	"actorprefix",
	"digp",

	// Not in my data, but from https://minecraft.wiki/w/Bedrock_Edition_level_format
	"~local_player",
	"game_flatworldlayers",
	"structuretemplate",
	"tickingarea",
}

func run() error {
	db, err := mcdb.Open("./worlds/survivalone")
	if err != nil {
		return err
	}

	ch, err := db.LoadColumn(world.ChunkPos{0, 0}, world.Overworld)
	if err != nil {
		return err
	}

	for x := uint8(0); x < 16; x++ {
		for z := uint8(0); z < 16; z++ {
			y := ch.HighestBlock(x, z)
			rid := ch.Block(x, y, z, 0)
			b, _ := world.BlockByRuntimeID(rid)

			fmt.Printf(" %d:%d:%T", y, rid, b)
		}
		fmt.Printf("\n")
	}

	return nil
}

func run2() error {
	db, err := leveldb.OpenFile("./worlds/survivalone/db", nil)
	if err != nil {
		return fmt.Errorf("error opening leveldb: %w", err)
	}

	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	count := 0
	counts := make([]int, 64)
	prefixCounts := make(map[string]int)

	for iter.Next() {
		key := iter.Key()
		sKey := string(key)
		lk := len(key)
		count++
		counts[lk]++

		if match := asciiPrefix.Find(key); match != nil {
			prefix := string(match)
			if suffix := hexSuffix.FindString(prefix); suffix != "" {
				prefix = prefix[:len(prefix)-len(suffix)+1]
			}
			if strings.HasPrefix(prefix, "digp") && prefix != "digp" {
				prefix = prefix[:len(prefix)-1]
			}
			if !slices.Contains(knownPrefixes, prefix) {
				fmt.Printf("Key with unhandled prefix: %q (prefix=%q)\n", key, prefix)
				os.Exit(1)
			}
			prefixCounts[prefix]++
		}

		switch lk {
		case 9, 10, 13, 14, 19:
			if !asciiRe.Match(key) {
				if asciiPrefix.Match(key) && !strings.HasPrefix(sKey, "actorprefix") {
					fmt.Printf("%v %q\n", key, key)
					os.Exit(1)
				}
				// chunk-related
				continue
			}
		}

		switch sKey {
		case "AutonomousEntities":
			continue
		case "BiomeData":
			continue
		case "LevelChunkMetaDataDictionary":
			continue
		case "Nether":
			continue
		case "Overworld":
			continue
		case "TheEnd":
			continue
		case "mobevents":
			continue
		case "portals":
			continue
		case "schedulerWT":
			continue
		case "scoreboard":
			continue
		}

		prefix, _, hasPrefix := strings.Cut(sKey, "_")

		if strings.HasPrefix(sKey, "digp") {
			continue
		}

		if !hasPrefix {
			fmt.Printf("Unhandled key: %q (length %d)\n", key, lk)
			os.Exit(1)
		}

		switch prefix {
		case "VILLAGE", "map", "player":
			fmt.Printf("%q\n", key)
			continue
		default:
			fmt.Printf("Unhandled prefix: %q\n", prefix)
			os.Exit(1)
		}
	}

	fmt.Printf("Total count: %d\n", count)
	for i, c := range counts {
		if c == 0 {
			continue
		}
		fmt.Printf(" Length %2d: %d\n", i, c)
	}

	fmt.Printf("Prefix counts:\n")
	for prefix, count := range prefixCounts {
		fmt.Printf("%s: %d\n", prefix, count)
	}

	return nil
}

func run4() error {
	keyTypeCounts := map[parse.KeyType]int{
		parse.KeyTypePlayerServer:                 0,
		parse.KeyTypePlayer:                       0,
		parse.KeyTypeMap:                          0,
		parse.KeyTypeVillageDwellers:              0,
		parse.KeyTypeVillageInfo:                  0,
		parse.KeyTypeVillagePlayers:               0,
		parse.KeyTypeVillagePOI:                   0,
		parse.KeyTypeVillageOverworldDwellers:     0,
		parse.KeyTypeVillageOverworldInfo:         0,
		parse.KeyTypeVillageOverworldPlayers:      0,
		parse.KeyTypeVillageOverworldPOI:          0,
		parse.KeyTypeAutonomousEntities:           0,
		parse.KeyTypeBiomeData:                    0,
		parse.KeyTypeLevelChunkMetaDataDictionary: 0,
		parse.KeyTypeNether:                       0,
		parse.KeyTypeOverworld:                    0,
		parse.KeyTypeTheEnd:                       0,
		parse.KeyTypeMobevents:                    0,
		parse.KeyTypePortals:                      0,
		parse.KeyTypeSchedulerWT:                  0,
		parse.KeyTypeScoreboard:                   0,
		parse.KeyTypeActorprefix:                  0,
		parse.KeyTypeDigp:                         0,

		// parse.KeyTypeLocalPlayer         // Not seen in my data
		// parse.KeyTypeGameFlatworldLayers // Not seen in my data
		// parse.KeyTypeStructureTemplate   // Not seen in my data
		// parse.KeyTypeTickingArea         // Not seen in my data

		parse.KeyTypeOverworldData3D:  0,
		parse.KeyTypeOverworldVersion: 0,
		parse.KeyTypeOverworldData2D:  0,
		// parse.KeyTypeOverworldData2DLegacy:                       0,
		parse.KeyTypeOverworldSubChunkPrefix: 0,
		parse.KeyTypeOverworldLegacyTerrain:  0,
		parse.KeyTypeOverworldBlockEntity:    0,
		parse.KeyTypeOverworldEntity:         0,
		parse.KeyTypeOverworldPendingTicks:   0,
		// parse.KeyTypeOverworldLegacyBlockExtraData:               0,
		parse.KeyTypeOverworldBiomeState:                         0,
		parse.KeyTypeOverworldFinalizedState:                     0,
		parse.KeyTypeOverworldConversionData:                     0,
		parse.KeyTypeOverworldBorderBlocks:                       0,
		parse.KeyTypeOverworldHardcodedSpawners:                  0,
		parse.KeyTypeOverworldRandomTicks:                        0,
		parse.KeyTypeOverworldCheckSums:                          0,
		parse.KeyTypeOverworldGenerationSeed:                     0,
		parse.KeyTypeOverworldGeneratedPreCavesAndCliffsBlending: 0,
		parse.KeyTypeOverworldBlendingBiomeHeight:                0,
		parse.KeyTypeOverworldMetaDataHash:                       0,
		parse.KeyTypeOverworldBlendingData:                       0,
		parse.KeyTypeOverworldActorDigestVersion:                 0,
		parse.KeyTypeOverworldLegacyVersion:                      0,

		parse.KeyTypeNetherData3D:  0,
		parse.KeyTypeNetherVersion: 0,
		parse.KeyTypeNetherData2D:  0,
		// parse.KeyTypeNetherData2DLegacy:                       0,
		parse.KeyTypeNetherSubChunkPrefix: 0,
		parse.KeyTypeNetherLegacyTerrain:  0,
		parse.KeyTypeNetherBlockEntity:    0,
		parse.KeyTypeNetherEntity:         0,
		parse.KeyTypeNetherPendingTicks:   0,
		// parse.KeyTypeNetherLegacyBlockExtraData:               0,
		parse.KeyTypeNetherBiomeState:                         0,
		parse.KeyTypeNetherFinalizedState:                     0,
		parse.KeyTypeNetherConversionData:                     0,
		parse.KeyTypeNetherBorderBlocks:                       0,
		parse.KeyTypeNetherHardcodedSpawners:                  0,
		parse.KeyTypeNetherRandomTicks:                        0,
		parse.KeyTypeNetherCheckSums:                          0,
		parse.KeyTypeNetherGenerationSeed:                     0,
		parse.KeyTypeNetherGeneratedPreCavesAndCliffsBlending: 0,
		parse.KeyTypeNetherBlendingBiomeHeight:                0,
		parse.KeyTypeNetherMetaDataHash:                       0,
		parse.KeyTypeNetherBlendingData:                       0,
		parse.KeyTypeNetherActorDigestVersion:                 0,
		parse.KeyTypeNetherLegacyVersion:                      0,

		parse.KeyTypeEndData3D:  0,
		parse.KeyTypeEndVersion: 0,
		parse.KeyTypeEndData2D:  0,
		// parse.KeyTypeEndData2DLegacy:                       0,
		parse.KeyTypeEndSubChunkPrefix: 0,
		parse.KeyTypeEndLegacyTerrain:  0,
		parse.KeyTypeEndBlockEntity:    0,
		parse.KeyTypeEndEntity:         0,
		parse.KeyTypeEndPendingTicks:   0,
		// parse.KeyTypeEndLegacyBlockExtraData:               0,
		parse.KeyTypeEndBiomeState:                         0,
		parse.KeyTypeEndFinalizedState:                     0,
		parse.KeyTypeEndConversionData:                     0,
		parse.KeyTypeEndBorderBlocks:                       0,
		parse.KeyTypeEndHardcodedSpawners:                  0,
		parse.KeyTypeEndRandomTicks:                        0,
		parse.KeyTypeEndCheckSums:                          0,
		parse.KeyTypeEndGenerationSeed:                     0,
		parse.KeyTypeEndGeneratedPreCavesAndCliffsBlending: 0,
		parse.KeyTypeEndBlendingBiomeHeight:                0,
		parse.KeyTypeEndMetaDataHash:                       0,
		parse.KeyTypeEndBlendingData:                       0,
		parse.KeyTypeEndActorDigestVersion:                 0,
		parse.KeyTypeEndLegacyVersion:                      0,
	}

	db, err := leveldb.OpenFile("./worlds/survivalone/db", nil)
	if err != nil {
		return err
	}
	defer db.Close()

	count := 0

	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
		kv := parse.NewKeyVal(iter.Key(), iter.Value())
		keyType := kv.KeyType()
		_, ok := keyTypeCounts[keyType]
		if !ok {
			fmt.Printf("Unexpected KeyType: %s for key %v (%d)\n", keyType, kv.Key, len(kv.Key))
		}
		keyTypeCounts[keyType]++

		if keyType.IsSubChunkPrefix() {
			_, err := parse.ParseSubChunk(kv)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	fmt.Printf("Saw %d keys and values\n", count)

	keys := maps.Keys(keyTypeCounts)
	slices.Sort(keys)
	longest := 0
	for _, key := range keys {
		if l := len(key.String()); l > longest {
			longest = l
		}
	}

	for _, k := range keys {
		fmt.Printf("%s:%s%d\n", k, strings.Repeat(" ", longest+3-len(k.String())), keyTypeCounts[k])
	}

	return nil
}

func runShowZeroZero() error {
	db, err := leveldb.OpenFile("./worlds/survivalone/db", nil)
	if err != nil {
		return err
	}

	chunk, err := parse.GetChunk(db, world.ChunkPos{}, world.Overworld)
	if err != nil {
		return err
	}

	for layer := 0; layer < chunk.MaxLayer; layer++ {
		fmt.Printf("layer %d\n", layer)
		hm := chunk.GetHeightMap(layer)
		for z := 0; z < 16; z++ {
			for x := 0; x < 16; x++ {
				y := hm.Get(x, z)
				block, err := chunk.GetBlock(x, z, y, layer)
				if err != nil {
					return err
				}
				fmt.Printf("%d: %v\n", y, block)
				if true {
					continue
				}
				// h := hm[z][x]
				// if h == parse.NoHeight {
				// 	fmt.Printf(" none")
				// } else {
				// 	fmt.Printf(" %4d", hm[z][x])
				// }
				name, ok := block["name"].(string)
				if !ok {
					name = "unknown"
				} else {
					if strings.HasPrefix(name, "minecraft:") {
						name = name[10:]
					}
				}
				fmt.Printf(" %d:%-11s", y, name)
			}
			fmt.Printf("\n")
		}
	}
	return nil
}

func run6() error {
	bb := []byte{
		10, 0, 0, 8, 4, 0, 110, 97, 109, 101, 17, 0, 109, 105, 110, 101,
		99, 114, 97, 102, 116, 58, 98, 101, 100, 114, 111, 99, 107, 10, 6, 0, 115, 116, 97, 116,
		101, 115, 1, 14, 0, 105, 110, 102, 105, 110, 105, 98, 117, 114, 110, 95, 98, 105, 116, 0,
		0, 3, 7, 0, 118, 101, 114, 115, 105, 111, 110, 1, 10, 18, 1, 0, 10, 0, 0, 8,
		4, 0, 110, 97, 109, 101, 20, 0, 109, 105, 110, 101, 99, 114, 97, 102, 116, 58, 110, 101,
		116, 104, 101, 114, 114, 97, 99, 107, 10, 6, 0, 115, 116, 97, 116, 101, 115, 0, 3, 7,
		0, 118, 101, 114, 115, 105, 111, 110, 1, 10, 18, 1, 0, 10, 0, 0, 8, 4, 0, 110,
		97, 109, 101, 20, 0, 109, 105, 110, 101, 99, 114, 97, 102, 116, 58, 113, 117, 97, 114, 116,
		122, 95, 111, 114, 101, 10, 6, 0, 115, 116, 97, 116, 101, 115, 0, 3, 7, 0, 118, 101,
		114, 115, 105, 111, 110, 1, 10, 18, 1, 0, 10, 0, 0, 8, 4, 0, 110, 97, 109, 101,
		24, 0, 109, 105, 110, 101, 99, 114, 97, 102, 116, 58, 97, 110, 99, 105, 101, 110, 116, 95,
		100, 101, 98, 114, 105, 115, 10, 6, 0, 115, 116, 97, 116, 101, 115, 0, 3, 7, 0, 118,
		101, 114, 115, 105, 111, 110, 1, 10, 18, 1, 0, 10, 0, 0, 8, 4, 0, 110, 97, 109,
		101, 25, 0, 109, 105, 110, 101, 99, 114, 97, 102, 116, 58, 110, 101, 116, 104, 101, 114, 95,
		103, 111, 108, 100, 95, 111, 114, 101, 10, 6, 0, 115, 116, 97, 116, 101, 115, 0, 3, 7,
		0, 118, 101, 114, 115, 105, 111, 110, 1, 10, 18, 1, 0, 10, 0, 0, 8, 4, 0, 110,
		97, 109, 101, 16, 0, 109, 105, 110, 101, 99, 114, 97, 102, 116, 58, 103, 114, 97, 118, 101,
		108, 10, 6, 0, 115, 116, 97, 116, 101, 115, 0, 3, 7, 0, 118, 101, 114, 115, 105, 111,
		110, 1, 10, 18, 1, 0, 10, 0, 0, 8, 4, 0, 110, 97, 109, 101, 20, 0, 109, 105,
		110, 101, 99, 114, 97, 102, 116, 58, 98, 108, 97, 99, 107, 115, 116, 111, 110, 101, 10, 6,
		0, 115, 116, 97, 116, 101, 115, 0, 3, 7, 0, 118, 101, 114, 115, 105, 111, 110, 1, 10,
		18, 1, 0,
	}

	buf := bytes.NewBuffer(bb)
	d := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)
	for i := 0; i < 7; i++ {
		fmt.Printf("decoding nbt %d starting at %v\n", i, []byte(buf.String()[:10]))
		var m map[string]any
		err := d.Decode(&m)
		if err != nil {
			return err
		}
	}
	return nil
}

func runScanEntireWorld() error {
	db, err := leveldb.OpenFile("./worlds/survivalone/db", nil)
	if err != nil {
		return err
	}
	defer db.Close()

	occupiedChunks := parse.GetOccupiedChunkCoordinates(db)

	for dim, chunkMap := range occupiedChunks {
		fmt.Printf("%v: %d occupied chunks\n", dim, len(chunkMap))
		i := 0
		for pos := range chunkMap {
			i++
			if i%10000 == 0 {
				fmt.Printf(" %d\n", i)
			}
			chunk, err := parse.GetChunk(db, pos, dim)
			if err != nil {
				return err
			}
			_ = chunk
		}
	}

	return nil
}

func runLoadImage() error {
	img, err := resources.GetBlockTexture("grass_top")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", img.Bounds())
	return nil
}

func main() {
	if err := runShowZeroZero(); err != nil {
		// if err := runLoadImage(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
