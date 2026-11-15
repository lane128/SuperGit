// Command icongen turns a square app drawing into a macOS app icon: the
// artwork is scaled into the system's icon grid and masked with the rounded
// square macOS draws, with a soft shadow underneath.
//
// The supplied drawings fill their whole canvas edge to edge. Handed to the
// Dock as-is they show up as hard-cornered squares that sit outside every
// other icon's grid, so the mask is applied once, here, and the embedded PNGs
// are the finished icon.
//
// Usage:
//
//	go run ./tools/icongen -src drawing.png -out app-light.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	xdraw "golang.org/x/image/draw"
)

const (
	// canvasSize is the pixel size of the icon macOS expects (1024 for @2x
	// 512pt icons).
	canvasSize = 1024
	// tileSize is the rounded square inside that canvas. macOS's grid leaves
	// 100px of breathing room on every side, so a full-bleed drawing is scaled
	// down into it rather than cropped.
	tileSize = 824
	// cornerExponent shapes the squircle. A superellipse with n=5 matches the
	// continuous corners macOS uses; a plain rounded rectangle reads as too
	// circular at the corners.
	cornerExponent = 5.0
	// shadowOffset and shadowBlur place the soft shadow under the tile, as the
	// system's own icon template does.
	shadowOffset = 12.0
	shadowBlur   = 22.0
	shadowAlpha  = 0.30
	// supersample is the grid used to antialias the mask edge.
	supersample = 4
)

func main() {
	src := flag.String("src", "", "source square PNG")
	out := flag.String("out", "", "output icon PNG")
	flag.Parse()
	if *src == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: icongen -src drawing.png -out icon.png")
		os.Exit(2)
	}
	if err := run(*src, *out); err != nil {
		fmt.Fprintln(os.Stderr, "icongen:", err)
		os.Exit(1)
	}
}

func run(srcPath, outPath string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	src, err := png.Decode(file)
	file.Close()
	if err != nil {
		return err
	}
	bounds := src.Bounds()
	if bounds.Dx() != bounds.Dy() {
		return fmt.Errorf("%s is %dx%d; want a square drawing", srcPath, bounds.Dx(), bounds.Dy())
	}

	art := scale(src, tileSize, tileSize)
	origin := (canvasSize - tileSize) / 2
	mask := squircleMask(origin, tileSize)
	shadow := dropShadow(mask)

	icon := image.NewNRGBA(image.Rect(0, 0, canvasSize, canvasSize))
	for y := 0; y < canvasSize; y++ {
		for x := 0; x < canvasSize; x++ {
			i := y*canvasSize + x
			shade := shadow[i]
			coverage := mask[i]
			// The artwork sits over its own shadow; both are opaque black
			// underneath a colored top, so one alpha-over composite is enough.
			alpha := coverage + shade*(1-coverage)
			if alpha <= 0 {
				continue
			}
			var r, g, b float64
			if coverage > 0 {
				ax, ay := x-origin, y-origin
				if ax >= 0 && ax < tileSize && ay >= 0 && ay < tileSize {
					c := art.NRGBAAt(ax, ay)
					r, g, b = float64(c.R), float64(c.G), float64(c.B)
				}
			}
			// Colors stay straight (not premultiplied) because NRGBA stores
			// unassociated alpha; the shadow contributes black, so it only
			// dims the color where the tile does not cover it.
			icon.SetNRGBA(x, y, color.NRGBA{
				R: clamp8(r),
				G: clamp8(g),
				B: clamp8(b),
				A: clamp8(alpha * 255),
			})
		}
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	if err := png.Encode(out, icon); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// scale resamples src into a w×h image with a smooth kernel, so the 1024px
// drawing keeps its edges when it shrinks into the 824px tile.
func scale(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// squircleMask returns 0..1 coverage for every pixel of the canvas: 1 inside
// the rounded square, 0 outside, with partial coverage on the edge.
func squircleMask(origin, size int) []float64 {
	mask := make([]float64, canvasSize*canvasSize)
	half := float64(size) / 2
	cx := float64(origin) + half
	cy := float64(origin) + half
	for y := 0; y < canvasSize; y++ {
		for x := 0; x < canvasSize; x++ {
			samples := 0
			for sy := 0; sy < supersample; sy++ {
				for sx := 0; sx < supersample; sx++ {
					px := float64(x) + (float64(sx)+0.5)/supersample - 0.5
					py := float64(y) + (float64(sy)+0.5)/supersample - 0.5
					dx := math.Abs(px-cx) / half
					dy := math.Abs(py-cy) / half
					if math.Pow(dx, cornerExponent)+math.Pow(dy, cornerExponent) <= 1 {
						samples++
					}
				}
			}
			if samples > 0 {
				mask[y*canvasSize+x] = float64(samples) / (supersample * supersample)
			}
		}
	}
	return mask
}

// dropShadow offsets the mask down and blurs it into the soft shadow the
// system's icon grid draws under an app icon.
func dropShadow(mask []float64) []float64 {
	shadow := make([]float64, len(mask))
	for y := 0; y < canvasSize; y++ {
		from := float64(y) - shadowOffset
		sy := int(math.Round(from))
		for x := 0; x < canvasSize; x++ {
			if sy >= 0 && sy < canvasSize {
				shadow[y*canvasSize+x] = mask[sy*canvasSize+x]
			}
		}
	}
	boxBlur(shadow, int(math.Round(shadowBlur)))
	for i := range shadow {
		shadow[i] *= shadowAlpha
	}
	return shadow
}

// boxBlur applies a separable box blur three times, which is close enough to
// a Gaussian for a shadow this faint and costs one pass per axis per round.
func boxBlur(pixels []float64, radius int) {
	if radius <= 0 {
		return
	}
	tmp := make([]float64, len(pixels))
	for pass := 0; pass < 3; pass++ {
		blurAxis(pixels, tmp, radius, true)
		blurAxis(tmp, pixels, radius, false)
	}
}

func blurAxis(src, dst []float64, radius int, horizontal bool) {
	width, height := canvasSize, canvasSize
	window := 2*radius + 1
	step := 1
	lines := height
	lineLen := width
	if !horizontal {
		step = canvasSize
		lines = width
		lineLen = height
	}
	for line := 0; line < lines; line++ {
		base := line
		if horizontal {
			base = line * canvasSize
		}
		sum := 0.0
		for k := -radius; k <= radius; k++ {
			sum += sampleLine(src, base, k, lineLen, step)
		}
		for i := 0; i < lineLen; i++ {
			dst[base+i*step] = sum / float64(window)
			sum -= sampleLine(src, base, i-radius, lineLen, step)
			sum += sampleLine(src, base, i+radius+1, lineLen, step)
		}
	}
}

// sampleLine reads an index on a scanline, clamping at the edges so the blur
// does not darken the canvas border.
func sampleLine(src []float64, base int, index, length, step int) float64 {
	if index < 0 {
		index = 0
	}
	if index >= length {
		index = length - 1
	}
	return src[base+index*step]
}

func clamp8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(math.Round(v))
}
