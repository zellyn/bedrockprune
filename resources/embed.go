package resources

import (
	"embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"path"
)

//go:embed textures
var embedded embed.FS

// GetBlockTexture will get the texture of the given block as an
// Image.NRGBA, if possible.
func GetBlockTexture(name string) (*image.NRGBA, error) {
	f, err := embedded.Open(path.Join("textures/blocks", name+".png"))
	if err != nil {
		return nil, err
	}

	src, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	b := src.Bounds()
	m := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), src, b.Min, draw.Src)

	if name == "grass_top" {
		for x := range 16 {
			for y := range 16 {
				r, _, _, _ := m.At(x, y).RGBA()

				// 0x92 0xBD 0x59
				c := color.NRGBA{R: uint8(r * 0x92 >> 16), G: uint8(r * 0xBD >> 16), B: uint8(r * 0x59 >> 16), A: 0xFF}
				m.Set(x, y, c)
			}
		}
	}

	return m, nil
}
