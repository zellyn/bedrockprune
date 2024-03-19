package occupation

import (
	"github.com/df-mc/dragonfly/server/world"
)

type Map interface {
	AllEmpty(minX, minZ, maxX, maxZ int32) bool
}

type occupation struct {
	levels   int
	presence map[int]map[world.ChunkPos]bool
}

func New(occupiedChunks map[world.ChunkPos]bool) Map {
	res := occupation{
		levels:   3,
		presence: make(map[int]map[world.ChunkPos]bool),
	}

	for level := 0; level < res.levels; level++ {
		res.presence[level] = make(map[world.ChunkPos]bool)
	}

	for pos := range occupiedChunks {
		x, z := pos.X(), pos.Z()

		for level := 0; level < res.levels; level++ {
			mask := int32(1)<<(level*4) - 1
			maskedPos := world.ChunkPos{x &^ mask, z &^ mask}
			res.presence[level][maskedPos] = true
		}
	}

	return res
}

func (occ occupation) allEmptyAtLevel(level int, minX, minZ, maxX, maxZ int32) bool {
	// indent := occ.levels - 1 - level
	// fmt.Printf("%slevel: %d, (%d,%d) - (%d,%d)\n", strings.Repeat(" ", indent*2), level, minX, minZ, maxX, maxZ)
	gap := int32(1) << (level * 4)
	mask := gap - 1

	for startX := minX &^ mask; startX < maxX; startX += gap {
		for startZ := minZ &^ mask; startZ < maxZ; startZ += gap {
			// No presence; continue for other pieces.
			if !occ.presence[level][world.ChunkPos{startX, startZ}] {
				continue
			}
			if level == 0 {
				return false
			}
			if !occ.allEmptyAtLevel(level-1, max(startX, minX), max(startZ, minZ), min(startX+gap, maxX), min(startZ+gap, maxZ)) {
				return false
			}
		}
	}

	return true
}

func (occ occupation) AllEmpty(minX, minZ, maxX, maxZ int32) bool {
	if maxX-minX+maxZ-minZ < 30 {
		return occ.allEmptyAtLevel(0, minX, minZ, maxX, maxZ)
	}
	return occ.allEmptyAtLevel(occ.levels-1, minX, minZ, maxX, maxZ)
}
