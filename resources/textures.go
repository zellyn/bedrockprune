package resources

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"path"
	"strings"

	"github.com/tailscale/hujson"
)

// type TileSource16 interface {
// 	Get(x, y int) (*image.RGBA, error)
// 	AllEmpty(image.Rectangle) (bool, error)
// }

const (
	bedrockBlockTextureDir = "assets/assets/resource_packs/vanilla/textures/blocks"
	javaBlockTextureDir    = "assets/minecraft/textures/block"

	bedrockBlocksJSONPath           = "assets/assets/resource_packs/vanilla/blocks.json"
	bedrockBlocksTerrainTexturePath = "assets/assets/resource_packs/vanilla/textures/terrain_texture.json"
	bedrockMissingTexturePath       = "assets/assets/resource_packs/vanilla/textures/misc/missing_texture.png"
	bedrockVanillaBase              = "assets/assets/resource_packs/vanilla"
)

type TextureSource struct {
	cache           map[string]*image.RGBA
	bedrockZip      *zip.ReadCloser
	javaZip         *zip.ReadCloser
	bedrockBlocks   bedrockBlocks
	terrainTextures terrainTextures
	missing         *image.RGBA
}

func (ts *TextureSource) Get(nbt map[string]any) (*image.RGBA, error) {
	name, ok := nbt["name"].(string)
	if !ok {
		return ts.missing, fmt.Errorf(`cannot get "name" from block nbt %v`, nbt)
	}
	// version := nbt["version"]
	// states := nbt["states"]

	if !strings.HasPrefix(name, "minecraft:") {
		return ts.missing, fmt.Errorf(`name %q doesn't start with "minecraft:"`, name)
	}

	name = name[10:]

	if name == "grass" {
		return ts.getGrass()
	}

	textureName, err := ts.bedrockBlocks.getTextureName(name)
	if err != nil {
		return ts.missing, err
	}
	texturePath, err := ts.terrainTextures.getTerrainTextureName(textureName)
	if err != nil {
		return ts.missing, err
	}

	fullPath := path.Join(bedrockVanillaBase, texturePath) + ".png"
	png, err := getPng(fullPath, ts.bedrockZip)
	if err == nil {
		if png.Bounds() == sixteenBounds {
			return rgba(png), nil
		}
		return ts.missing, fmt.Errorf("%s found in bedrock apk, but has dimensions %v", fullPath, png.Bounds().Size())
	}
	return ts.missing, fmt.Errorf("not implemented yet")
}

var sixteenBounds = image.Rect(0, 0, 16, 16)

func (ts *TextureSource) getBlockTexture(name string) (*image.RGBA, error) {
	png, err := getPng(path.Join(bedrockBlockTextureDir, name)+".png", ts.bedrockZip)
	if err == nil && png.Bounds() == sixteenBounds {
		return rgba(png), nil
	}

	png, err = getPng(path.Join(javaBlockTextureDir, name)+".png", ts.javaZip)
	if err == nil && png.Bounds() == sixteenBounds {
		return rgba(png), nil
	}

	return nil, fmt.Errorf("cannot get texture for %q", name)
}

func getPng(fullpath string, zipReader *zip.ReadCloser) (image.Image, error) {
	file, err := zipReader.Open(fullpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func rgba(img image.Image) *image.RGBA {
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), img, image.Point{}, draw.Src)
	return m
}

func (ts *TextureSource) getGrass() (*image.RGBA, error) {
	m, err := ts.getBlockTexture("grass_top")
	if err != nil {
		return nil, err
	}
	for x := range 16 {
		for y := range 16 {
			r, _, _, _ := m.At(x, y).RGBA()

			// 0x92 0xBD 0x59
			c := color.NRGBA{R: uint8(r * 0x92 >> 16), G: uint8(r * 0xBD >> 16), B: uint8(r * 0x59 >> 16), A: 0xFF}
			m.Set(x, y, c)
		}
	}

	return m, nil
}

type downloadMode bool

const (
	UseOnlyCached        downloadMode = false
	CheckForNewDownloads downloadMode = true
)

type bedrockBlocks struct {
	json map[string]any
}

type terrainTextures struct {
	json map[string]any
}

func NewTextureSource(ctx context.Context, downloadMode downloadMode) (*TextureSource, error) {
	var bedrockZip, javaZip *zip.ReadCloser
	var err error

	if downloadMode == UseOnlyCached {
		bedrockZip, err = LatestCachedZip("bedrock-", "apk")
		if err != nil {
			return nil, err
		}
		javaZip, err = LatestCachedZip("java-", "jar")
		if err != nil {
			return nil, err
		}
	} else {
		bedrockZip, err = LatestBedrockReleaseClientZip(ctx)
		if err != nil {
			return nil, err
		}
		javaZip, err = LatestJavaReleaseClientZip(ctx)
		if err != nil {
			return nil, err
		}
	}

	bb, err := readHuJSON(bedrockZip, bedrockBlocksJSONPath)
	if err != nil {
		return nil, err
	}
	tt, err := readHuJSON(bedrockZip, bedrockBlocksTerrainTexturePath)
	if err != nil {
		return nil, err
	}

	missing, err := getPng(bedrockMissingTexturePath, bedrockZip)
	if err != nil {
		return nil, fmt.Errorf("cannot find missing texture in bedrock apk: %w", err)
	}

	return &TextureSource{
		cache:           make(map[string]*image.RGBA),
		bedrockZip:      bedrockZip,
		javaZip:         javaZip,
		bedrockBlocks:   bedrockBlocks{json: bb},
		terrainTextures: terrainTextures{json: tt},
		missing:         rgba(missing),
	}, nil
}

func readHuJSON(reader *zip.ReadCloser, path string) (map[string]any, error) {
	file, err := reader.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading contents of %q from zip: %w", path, err)
	}
	b, err = hujson.Standardize(b)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON contents of %q from zip: %w", path, err)
	}

	var v map[string]any
	if err = json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("error parsing JSON contents of %q from zip: %w", path, err)
	}
	return v, nil
}

func (bb *bedrockBlocks) getTextureName(name string) (string, error) {
	obj, ok := bb.json[name]
	if !ok {
		return "", fmt.Errorf("%s not found in blocks.json", name)
	}
	textures, ok := obj.(map[string]any)["textures"]
	if !ok {
		return "", fmt.Errorf("%s.textures not found in blocks.json", name)
	}

	switch v := textures.(type) {
	case string:
		return v, nil
	case map[string]any:
		up, ok := v["up"]
		if !ok {
			return "", fmt.Errorf("%s.textures.up not found in blocks.json. textures=%v", name, v)
		}
		return up.(string), nil
	}
	return "", fmt.Errorf("unknown shape for %s.textures in blocks.json. textures=%v", name, textures)
}

func (tt *terrainTextures) getTerrainTextureName(textureName string) (string, error) {
	td, ok := tt.json["texture_data"]
	if !ok {
		return "", fmt.Errorf(`cannot find "texture_data" field of terrain_texture.json`)
	}

	obj, ok := td.(map[string]any)[textureName]
	if !ok {
		return "", fmt.Errorf("texture_data.%s not found in terrain_texture.json", textureName)
	}

	textures, ok := obj.(map[string]any)["textures"]
	if !ok {
		return "", fmt.Errorf("texture_data.%s.textures not found in terrain_texture.json", textureName)
	}

	switch v := textures.(type) {
	case string:
		return v, nil
	case map[string]any:
		path, ok := v["path"]
		if !ok {
			return "", fmt.Errorf(`cannot get "path" field from %v`, v)
		}
		return path.(string), nil
	case []any:
		if s := v[0].(string); ok {
			return s, nil
		}
		if m := v[0].(map[string]any); ok {
			path, ok := m["path"]
			if !ok {
				return "", fmt.Errorf(`cannot get "path" field from index 0 of %v`, v)
			}
			return path.(string), nil
		}
	}

	return "", fmt.Errorf("unknown shape for texture_data.%s.textures: %v", textureName, textures)
}
