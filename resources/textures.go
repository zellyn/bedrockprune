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
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/blezek/tga"
	"github.com/tailscale/hujson"
	"github.com/zellyn/bedrockprune/types"
)

// type TileSource16 interface {
// 	Get(x, y int) (*image.RGBA, error)
// 	AllEmpty(image.Rectangle) (bool, error)
// }

const (
	bedrockGrassTexturePath   = "assets/assets/resource_packs/vanilla/textures/blocks/grass_top.png"
	bedrockWaterTexturePath   = "assets/assets/resource_packs/vanilla/textures/blocks/water_still_grey.png"
	bedrockMissingTexturePath = "assets/assets/resource_packs/vanilla/textures/misc/missing_texture.png"
	bedrockVanillaBase        = "assets/assets/resource_packs/vanilla"
	bedrockResourcePacksPath  = "assets/assets/resource_packs"
)

type TextureSource struct {
	cache         map[string]*image.RGBA
	bedrockZip    *zip.ReadCloser
	javaZip       *zip.ReadCloser
	resourcePacks resourcePacks
	missing       *image.RGBA
}

type resourcePacks []*resourcePack

type resourcePack struct {
	basepath        string
	bedrockZip      *zip.ReadCloser
	blocks          bedrockBlocks
	terrainTextures terrainTextures
}

var versionRegexp = regexp.MustCompile(" version:[0-9]+")

func (ts *TextureSource) Get(nbt map[string]any) (*image.RGBA, error) {
	cacheKey := versionRegexp.ReplaceAllString(fmt.Sprintf("%v", nbt), "")
	if img, ok := ts.cache[cacheKey]; ok {
		return img, nil
	}
	name, ok := nbt["name"].(string)
	if !ok {
		ts.cache[cacheKey] = ts.missing
		return ts.missing, fmt.Errorf(`cannot get "name" from block nbt %v`, nbt)
	}
	// version := nbt["version"]
	// states := nbt["states"]

	if !strings.HasPrefix(name, "minecraft:") {
		ts.cache[cacheKey] = ts.missing
		return ts.missing, fmt.Errorf(`name %q doesn't start with "minecraft:"`, name)
	}

	name = name[10:]

	if name == "grass" {
		img, err := ts.getGrass()
		if err != nil {
			img = ts.missing
		}
		ts.cache[cacheKey] = img
		return img, err
	}

	if name == "water" {
		img, err := ts.getWater()
		if err != nil {
			img = ts.missing
		}
		ts.cache[cacheKey] = img
		return img, err
	}

	if name == "glow_lichen" {
		fmt.Printf("DEBUG: cacheKey=%q\n", cacheKey)
	}

	img, err := ts.resourcePacks.Get(name)
	if err != nil {
		ts.cache[cacheKey] = ts.missing
		return ts.missing, fmt.Errorf("%w: %q", err, cacheKey)
	}
	ts.cache[cacheKey] = img
	return img, nil
}

var sixteenBounds = image.Rect(0, 0, 16, 16)

func getPng(fullpath string, zipReader *zip.ReadCloser) (image.Image, error) {
	file, err := zipReader.Open(fullpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func getTGA(fullpath string, zipReader *zip.ReadCloser) (image.Image, error) {
	file, err := zipReader.Open(fullpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return tga.Decode(file)
}

func rgba(img image.Image) *image.RGBA {
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), img, image.Point{}, draw.Src)
	return m
}

func rgbaForce16x16(img image.Image) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, 16, 16))
	draw.Draw(m, m.Bounds(), img, image.Point{}, draw.Src)
	return m
}

func (ts *TextureSource) getGrass() (*image.RGBA, error) {
	png, err := getPng(bedrockGrassTexturePath, ts.bedrockZip)
	if err != nil {
		return nil, err
	}
	m := rgba(png)
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

func (ts *TextureSource) getWater() (*image.RGBA, error) {
	png, err := getPng(bedrockWaterTexturePath, ts.bedrockZip)
	if err != nil {
		return nil, err
	}
	m := rgbaForce16x16(png)
	for x := range 16 {
		for y := range 16 {
			r, _, _, a := m.At(x, y).RGBA()

			// #44aff5
			c := color.NRGBA{R: uint8(r * 0x44 >> 16), G: uint8(r * 0xAF >> 16), B: uint8(r * 0xF5 >> 16), A: uint8(a >> 8)}
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

	rps, err := newResourcePacks(bedrockZip)
	if err != nil {
		return nil, err
	}

	for _, rp := range rps {
		fmt.Printf("resource pack: %s\n", path.Base(rp.basepath))
	}

	missing, err := getPng(bedrockMissingTexturePath, bedrockZip)
	if err != nil {
		return nil, fmt.Errorf("cannot find missing texture in bedrock apk: %w", err)
	}

	return &TextureSource{
		cache:         make(map[string]*image.RGBA),
		bedrockZip:    bedrockZip,
		javaZip:       javaZip,
		resourcePacks: rps,
		missing:       rgba(missing),
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
		return "", fmt.Errorf("%w: %s in blocks.json", types.ErrNotFound, name)
	}
	textures, ok := obj.(map[string]any)["textures"]
	if !ok {
		return "", fmt.Errorf("%w: %s in blocks.json", types.ErrNotFound, name)
	}

	switch v := textures.(type) {
	case string:
		return v, nil
	case map[string]any:
		up, ok := v["up"]
		if !ok {
			return "", fmt.Errorf("%w: %s.textures.up in blocks.json. textures=%v", types.ErrNotFound, name, v)
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
		if s, ok := v[0].(string); ok {
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

func newResourcePacks(bedrockZip *zip.ReadCloser) (resourcePacks, error) {

	var rps []*resourcePack

	dirFile, err := bedrockZip.Open(bedrockResourcePacksPath)
	if err != nil {
		return nil, fmt.Errorf("error reading bedrock apk: %w", err)
	}
	defer dirFile.Close()
	dir, ok := dirFile.(fs.ReadDirFile)
	if !ok {
		return nil, fmt.Errorf("cannot read directory %q in bedrock apk", bedrockResourcePacksPath)
	}
	files, err := dir.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %q in bedrock apk: %e", bedrockResourcePacksPath, err)
	}

	for _, file := range files {
		name := file.Name()
		if !file.IsDir() || !(strings.HasPrefix(name, "vanilla_") || name == "vanilla") {
			continue
		}
		rp, err := newResourcePack(path.Join(bedrockResourcePacksPath, name), bedrockZip)
		if err != nil {
			continue
		}
		rps = append(rps, rp)
	}

	versions := make(map[string]version)
	for _, rp := range rps {
		v, err := versionFromDirectoryName(path.Base(rp.basepath))
		if err != nil {
			return nil, err
		}
		versions[rp.basepath] = v
	}

	slices.SortFunc(rps, func(a, b *resourcePack) int {
		return versions[a.basepath].cmp(versions[b.basepath])
	})

	return rps, nil
}

func newResourcePack(basepath string, bedrockZip *zip.ReadCloser) (*resourcePack, error) {
	blocksJSONPath := path.Join(basepath, "blocks.json")
	texturesJSONPath := path.Join(basepath, "textures/terrain_texture.json")

	bb, err := readHuJSON(bedrockZip, blocksJSONPath)
	if err != nil {
		return nil, err
	}
	tt, err := readHuJSON(bedrockZip, texturesJSONPath)
	if err != nil {
		return nil, err
	}

	return &resourcePack{
		basepath:        basepath,
		bedrockZip:      bedrockZip,
		blocks:          bedrockBlocks{json: bb},
		terrainTextures: terrainTextures{json: tt},
	}, nil
}

func (rps resourcePacks) Get(name string) (*image.RGBA, error) {
	debug := name == "glow_lichen"
	if debug {
		fmt.Printf("DEBUG: debugging\n")
	}
	// First resource pack is the main one.
	img, err0 := rps[0].Get(name)
	if err0 == nil {
		return img, nil
	}
	if debug {
		fmt.Printf("DEBUG: didn't get image from base vanilla resource pack (%v)\n", err0)
	}

	// If we can find it elsewhere, do so.
	for i := 1; i < len(rps); i++ {
		if debug {
			fmt.Printf("DEBUG: trying resource pack %s\n", rps[i].basepath)
		}

		img, err := rps[i].Get(name)
		if err == nil {
			return img, nil
		}
		if debug {
			fmt.Printf("DEBUG:  no luck (%v)\n", err)
		}
	}

	// Otherwise return the base error.
	return nil, err0
}

func (rp *resourcePack) Get(name string) (*image.RGBA, error) {
	textureName, err := rp.blocks.getTextureName(name)
	if err != nil {
		textureName = name
	}

	texturePath, err := rp.terrainTextures.getTerrainTextureName(textureName)
	if err != nil {
		return nil, err
	}

	pngPath := path.Join(rp.basepath, texturePath) + ".png"
	png, err := getPng(pngPath, rp.bedrockZip)
	if err == nil {
		if png.Bounds() == sixteenBounds {
			img := rgba(png)
			return img, nil
		}
		return nil, fmt.Errorf("%s found in bedrock apk, but has dimensions %v", pngPath, png.Bounds().Size())
	}
	tgaPath := path.Join(rp.basepath, texturePath) + ".tga"
	targa, err := getTGA(tgaPath, rp.bedrockZip)
	if err == nil {
		if targa.Bounds() == sixteenBounds {
			img := rgba(targa)
			return img, nil
		}
		return nil, fmt.Errorf("%s found in bedrock apk, but has dimensions %v", tgaPath, targa.Bounds().Size())
	}
	return nil, fmt.Errorf("not implemented yet")
}
