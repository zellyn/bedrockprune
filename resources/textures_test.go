package resources

import (
	"context"
	"testing"
)

func TestZeroZeroBlocks(t *testing.T) {
	testdata := []struct {
		nbt     map[string]any
		partial bool
	}{
		{
			nbt: map[string]any{
				"name":    "minecraft:grass",
				"states":  map[string]any{},
				"version": 18100737,
			},
			partial: false,
		},
		{
			nbt: map[string]any{
				"name":    "minecraft:grass_path",
				"states":  map[string]any{},
				"version": 18100737,
			},
			partial: false,
		},
		{
			nbt: map[string]any{
				"name": "minecraft:oak_stairs",
				"states": map[string]any{
					"upside_down_bit":  uint8(0),
					"weirdo_direction": int32(2),
				},
				"version": 17959425,
			},
			partial: false,
		},
		{
			nbt: map[string]any{
				"name": "minecraft:oak_stairs",
				"states": map[string]any{
					"upside_down_bit":  uint8(0),
					"weirdo_direction": int32(3),
				},
				"version": 17959425,
			},
			partial: false,
		},
		{
			nbt: map[string]any{
				"name": "minecraft:snow_layer",
				"states": map[string]any{
					"covered_bit": uint8(0),
					"height":      int32(1),
				},
				"version": 18100737,
			},
			partial: false,
		},
		{
			nbt: map[string]any{
				"name": "minecraft:wooden_slab",
				"states": map[string]any{
					"top_slot_bit": uint8(0),
					"wood_type":    "oak",
				},
				"version": 17959425,
			},
			partial: false,
		},
		{
			nbt: map[string]any{
				"name": "minecraft:lantern",
				"states": map[string]any{
					"hanging": uint8(0),
				},
				"version": 18100737,
			},
			partial: true,
		},
		{
			nbt: map[string]any{
				"name": "minecraft:standing_sign",
				"states": map[string]any{
					"ground_sign_direction": int32(4),
				},
				"version": 18100737,
			},
			partial: true,
		},
	}

	ts, err := NewTextureSource(context.Background(), UseOnlyCached)
	if err != nil {
		t.Fatal(err)
	}

	for _, td := range testdata {
		if td.partial {
			continue
		}
		t.Run(td.nbt["name"].(string), func(t *testing.T) {
			img, err := ts.Get(td.nbt)
			if err != nil {
				t.Fatal(err)
			}
			if img == nil {
				t.Fatal("nil image")
			}
		})
	}
}
