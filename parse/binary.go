package parse

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/df-mc/dragonfly/server/world"
)

func putInt32(b []byte, i int32) {
	binary.LittleEndian.PutUint32(b, uint32(i))
}

func getInt32(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b))
}

func getChunkPos(b []byte) world.ChunkPos {
	return world.ChunkPos{
		getInt32(b[0:4]),
		getInt32(b[4:8]),
	}
}

func readInt32(b *bytes.Buffer) (int32, error) {
	bytes4 := b.Next(4)
	if len(bytes4) < 4 {
		return 0, fmt.Errorf("ran out of bytes while trying to read int32")
	}

	unsigned := binary.LittleEndian.Uint32(bytes4)
	return int32(unsigned), nil
}

func readUint32(b *bytes.Buffer) (uint32, error) {
	bytes4 := b.Next(4)
	if len(bytes4) < 4 {
		return 0, fmt.Errorf("ran out of bytes while trying to read uint32")
	}

	// Dragonfly code suggested this was faster than binary.LittleEndian
	// https://github.com/df-mc/dragonfly/blob/f392edaffa/server/world/chunk/decode.go#L17
	return uint32(bytes4[0]) | uint32(bytes4[1])<<8 | uint32(bytes4[2])<<16 | uint32(bytes4[3])<<24, nil
}

func readUint32AsInt(b *bytes.Buffer) (int, error) {
	bytes4 := b.Next(4)
	if len(bytes4) < 4 {
		return 0, fmt.Errorf("ran out of bytes while trying to read uint32")
	}

	// Dragonfly code suggested this was faster than binary.LittleEndian
	// https://github.com/df-mc/dragonfly/blob/f392edaffa/server/world/chunk/decode.go#L17
	return int(bytes4[0]) | int(bytes4[1])<<8 | int(bytes4[2])<<16 | int(bytes4[3])<<24, nil
}
