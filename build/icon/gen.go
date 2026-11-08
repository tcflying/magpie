//go:build ignore

// Icon generator. Everything magpie shows is drawn from one shape: a filled
// disc on a 44-unit grid, the app's yellow. magpie.svg is the full drawing;
// the app tile uses it from 64px up. magpie-small.svg is the same disc (the
// bird's eye was the only thing it dropped, and a disc has no eye); the tray
// and the smaller tiles use it.
//
//	go run build/icon/gen.go tray internal/gui/tray.png         # 44px black template icon (macOS menu bar)
//	go run build/icon/gen.go tray-flap internal/gui/trayflap    # its frames when clicked (a size after the folder for a bigger look)
//	go run build/icon/gen.go app 1024 internal/gui/icon-1024.png # app icon at a given size
//	go run build/icon/gen.go win 256 build/windows/icon-256.png # the Windows icon: the mark, filling its square
//	go run build/icon/gen.go win-ico internal/gui/icon-windows.ico # it at the tray's sizes, one .ico
//	make icons                                                  # regenerates all of them plus magpie.icns
package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	//go:embed magpie.svg
	fullSVG string
	//go:embed magpie-small.svg
	smallSVG string
)

func main() {
	var img image.Image
	switch {
	case len(os.Args) == 3 && os.Args[1] == "tray":
		img = tray()
	case len(os.Args) == 3 && os.Args[1] == "tray-flap":
		if err := trayFlap(os.Args[2], 44); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case len(os.Args) == 4 && os.Args[1] == "tray-flap":
		n, _ := strconv.Atoi(os.Args[3])
		if err := trayFlap(os.Args[2], n); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case len(os.Args) == 4 && os.Args[1] == "glyph":
		n, _ := strconv.Atoi(os.Args[2])
		img = glyph(n)
	case len(os.Args) == 4 && (os.Args[1] == "app" || os.Args[1] == "app-dark"):
		n, _ := strconv.Atoi(os.Args[2])
		img = app(n, os.Args[1] == "app-dark")
	case len(os.Args) == 4 && os.Args[1] == "win":
		n, _ := strconv.Atoi(os.Args[2])
		img = win(n)
	case len(os.Args) == 3 && os.Args[1] == "win-ico":
		if err := winICO(os.Args[2], trayICOSizes); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	default:
		fmt.Fprintln(os.Stderr, "usage: gen.go tray <out.png> | gen.go app|app-dark|win <size> <out.png> | gen.go win-ico <out.ico>")
		os.Exit(2)
	}
	f, err := os.Create(os.Args[len(os.Args)-1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// trayK is the tray mark's size in its 22pt square: drawn to the edges it
// would be a full 22pt wide, bigger than the menu bar's own glyphs
// (@SherlockYoYo: 感觉这只鸟有些大); at .8 it is about 18pt across, centred.
const trayK = 0.8

// tray draws the silhouette as a macOS template image (black + alpha).
func tray() image.Image {
	const S = 44
	off := S / 2 * (1 - trayK)
	cov := coverage(parse(smallSVG), S, 8, trayK, off, off)
	img := image.NewNRGBA(image.Rect(0, 0, S, S))
	for i, a := range cov {
		img.SetNRGBA(i%S, i/S, color.NRGBA{0, 0, 0, uint8(a*255 + 0.5)})
	}
	return img
}

// The tray disc's flap: the disc draws in about its middle and settles back,
// over the .9s the header logo's own breath took (#117). Only the time is
// shared: the header mark swells, the tray draws in.
const (
	flapSecs  = 0.9
	flapEvery = 0.03 // a frame every 30ms: internal/gui's flap plays them
)

// trayFlap writes the frames the tray icon plays when it is clicked,
// flap-01.png on, one every flapEvery of the animation, the still disc at
// both ends left out. The disc draws in about its middle and settles back.
// It draws in rather than out because the menu bar's own glyphs are 18pt: a
// mark that grew past the still would stand out of the row
// (TestTrayIconSize).
func trayFlap(dir string, S int) error {
	shape := parse(smallSVG)
	n := int(math.Round(flapSecs / flapEvery))
	const ss = 8
	for f := 1; f < n; f++ {
		t := float64(f) / float64(n)
		// the same mapping the still disc is drawn with, scaled about the
		// middle: the viewBox's 22 stays at the canvas's middle. It draws
		// in and settles back, never past the still's own 18pt.
		s := 1 - 0.15*math.Sin(math.Pi*t)
		k := float64(S) / 44 * trayK * s
		off := float64(S)/2 - 22*k
		cov := coverage(shape, S, ss, k, off, off)
		img := image.NewNRGBA(image.Rect(0, 0, S, S))
		for i, a := range cov {
			img.SetNRGBA(i%S, i/S, color.NRGBA{0, 0, 0, uint8(a*255 + 0.5)})
		}
		out, err := os.Create(fmt.Sprintf("%s/flap-%02d.png", dir, f))
		if err != nil {
			return err
		}
		err = png.Encode(out, img)
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// glyph is the disc alone, in the app's yellow, as big on an n-pixel canvas
// as app draws it on its tile: the layer of build/darwin/AppIcon.icon, which
// macOS 26 lays on a tile of its own, light or dark (#117).
func glyph(n int) image.Image {
	S := float64(n)
	k := S * 0.8 / 44
	o := S/2 - 22*k
	cov := coverage(parse(fullSVG), n, 4, k, o, o)
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for i, a := range cov {
		if a <= 0 {
			continue
		}
		img.SetNRGBA(i%n, i/n, color.NRGBA{0xfa, 0xcc, 0x15, uint8(a*255 + 0.5)})
	}
	return img
}

// app draws the app's icon: the shape in the app's yellow, on transparency.
// The shape is a filled disc, so the white tile the bird sat on is gone:
// what shows is a yellow disc, the taskbar's, the tray's and the Dock's alike.
func app(n int, dark bool) image.Image {
	S := float64(n)
	// the disc fills the canvas, a hair's margin off its edge
	inset := S * 0.02
	side := S - 2*inset
	k := side / 44
	o := inset + side/2 - 22*k
	ss := 4
	if n <= 64 {
		ss = 8
	}
	// under 64px the eye is a grey speck; leave it out
	shape := fullSVG
	if n < 64 {
		shape = smallSVG
	}
	cov := coverage(parse(shape), n, ss, k, o, o)
	gr, gg, gb := uint8(0xfa), uint8(0xcc), uint8(0x15)
	if dark {
		// the Dock's dark appearance (macOS 26): the disc a shade lighter, so
		// it keeps its own colour on a dark tile (#117)
		gr, gg, gb = 0xfd, 0xe0, 0x47
	}
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for i, a := range cov {
		if a <= 0 {
			continue
		}
		img.SetNRGBA(i%n, i/n, color.NRGBA{gr, gg, gb, uint8(a*255 + 0.5)})
	}
	return img
}

// win is the app's mark for Windows: the exe's icon group
// (build/windows/winres.json) and the tray's .ico (win-ico) both take it at
// every size, the mark filling the canvas as the icons beside it do.
// Windows' notification area is the one place the mark is shown as big as
// the icons beside it, and the system scales the icon it is given down to
// the size it wants: the app's own icon, which keeps a margin for the
// larger icons Windows draws, came out a size smaller than its neighbours
// (#2404).
func win(n int) image.Image {
	S := float64(n)
	// the shape's disc is 43.20 units across in its 44-unit box (its path
	// runs from 0.40 to 43.60), so it is scaled to the canvas' own width
	k := S / 43.20
	o := S/2 - 22*k
	cov := coverage(parse(smallSVG), n, 8, k, o, o)
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for i, a := range cov {
		if a <= 0 {
			continue
		}
		img.SetNRGBA(i%n, i/n, color.NRGBA{0xfa, 0xcc, 0x15, uint8(a*255 + 0.5)})
	}
	return img
}

// trayICOSizes are the sizes Windows asks a tray icon for (SM_CXSMICON):
// 16px at 100% scaling, 20, 24, 32 and 40 at 125% to 250%, and 48 and 64
// beyond. Wails picks the one it needs from the .ico, where a single PNG
// was scaled down by the system and blurred.
var trayICOSizes = []int{16, 20, 24, 32, 40, 48, 64}

// winICO writes win's mark at each size into one .ico, each image a PNG
// (Windows Vista on reads them).
func winICO(out string, sizes []int) error {
	var pngs [][]byte
	for _, n := range sizes {
		var b bytes.Buffer
		if err := png.Encode(&b, win(n)); err != nil {
			return err
		}
		pngs = append(pngs, b.Bytes())
	}
	var ico bytes.Buffer
	le := binary.LittleEndian
	ico.Write(le.AppendUint16(le.AppendUint16(le.AppendUint16(nil, 0), 1), uint16(len(sizes))))
	at := 6 + 16*len(sizes)
	for i, n := range sizes {
		d := byte(n)
		if n >= 256 {
			d = 0
		}
		e := []byte{d, d, 0, 0}
		e = le.AppendUint16(e, 1)  // planes
		e = le.AppendUint16(e, 32) // bits per pixel
		e = le.AppendUint32(e, uint32(len(pngs[i])))
		e = le.AppendUint32(e, uint32(at))
		ico.Write(e)
		at += len(pngs[i])
	}
	for _, p := range pngs {
		ico.Write(p)
	}
	return os.WriteFile(out, ico.Bytes(), 0o644)
}

// coverage rasterises a shape, scaled by k and moved to (ox, oy), onto an n×n
// grid of pixels with ss×ss samples each, and gives each pixel's coverage in
// 0..1. It fills even-odd, a scanline at a time.
func coverage(rings [][][2]float64, n, ss int, k, ox, oy float64) []float64 {
	cov := make([]float64, n*n)
	per := 1 / float64(ss*ss)
	var xs []float64
	for y := 0; y < n*ss; y++ {
		v := ((float64(y)+0.5)/float64(ss) - oy) / k
		xs = xs[:0]
		for _, r := range rings {
			for i, j := 0, len(r)-1; i < len(r); j, i = i, i+1 {
				a, b := r[j], r[i]
				if (a[1] > v) != (b[1] > v) {
					u := a[0] + (v-a[1])*(b[0]-a[0])/(b[1]-a[1])
					xs = append(xs, (u*k+ox)*float64(ss))
				}
			}
		}
		sort.Float64s(xs)
		row := cov[(y/ss)*n:][:n]
		for i := 0; i+1 < len(xs); i += 2 {
			// sample centres x+0.5 that fall between the two crossings
			lo := int(math.Max(0, math.Ceil(xs[i]-0.5)))
			hi := int(math.Min(float64(n*ss), math.Ceil(xs[i+1]-0.5)))
			for x := lo; x < hi; x++ {
				row[x/ss] += per
			}
		}
	}
	return cov
}

// parse reads the path out of one of the SVGs: absolute M, L, C and Z
// commands, each curve cut into short segments, one polygon per subpath.
func parse(svg string) [][][2]float64 {
	d := svg[strings.Index(svg, ` d="`)+4:]
	d = d[:strings.IndexByte(d, '"')]
	f := strings.Fields(strings.NewReplacer("M", " M ", "L", " L ", "C", " C ", "Z", " Z ").Replace(d))
	num := func(i int) float64 { x, _ := strconv.ParseFloat(f[i], 64); return x }
	var rings [][][2]float64
	var pts [][2]float64
	cmd := ""
	for i := 0; i < len(f); {
		if c := f[i]; c == "M" || c == "L" || c == "C" || c == "Z" {
			cmd = c
			i++
		}
		switch cmd {
		case "M", "L":
			if cmd == "M" && len(pts) > 0 {
				rings, pts = append(rings, pts), nil
			}
			pts = append(pts, [2]float64{num(i), num(i + 1)})
			i += 2
			cmd = "L" // further pairs after M are lines
		case "C":
			p := pts[len(pts)-1]
			c := [8]float64{p[0], p[1], num(i), num(i + 1), num(i + 2), num(i + 3), num(i + 4), num(i + 5)}
			for s := 1; s <= 8; s++ {
				t := float64(s) / 8
				m := 1 - t
				pts = append(pts, [2]float64{
					m*m*m*c[0] + 3*m*m*t*c[2] + 3*m*t*t*c[4] + t*t*t*c[6],
					m*m*m*c[1] + 3*m*m*t*c[3] + 3*m*t*t*c[5] + t*t*t*c[7],
				})
			}
			i += 6
		case "Z":
			if len(pts) > 0 {
				rings, pts = append(rings, pts), nil
			}
		}
	}
	if len(pts) > 0 {
		rings = append(rings, pts)
	}
	return rings
}
