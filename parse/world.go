package parse

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/goleveldb/leveldb"
)

func GetOccupiedChunkCoordinates(db *leveldb.DB) map[world.Dimension]map[world.ChunkPos]bool {
	res := make(map[world.Dimension]map[world.ChunkPos]bool)
	for _, dim := range []world.Dimension{world.Overworld, world.Nether, world.End} {
		res[dim] = make(map[world.ChunkPos]bool)
	}

	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		kv := NewKeyVal(iter.Key(), iter.Value())
		keyInfo := kv.KeyTypeAndChunkLocation()
		if keyInfo.HasLocation {
			res[keyInfo.Dimension][keyInfo.ChunkPos] = true
		}
	}
	return res
}
