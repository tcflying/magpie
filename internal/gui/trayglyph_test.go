//go:build !nogui

package gui

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"image/png"
	"testing"

	"github.com/godbus/dbus/v5"
)

// ylorn on Discord: a Linux bar's tray icon was the app's own icon, and a
// bar tinting its icons by their alpha tinted the whole of it. It is the
// mark's silhouette alone, in one colour on transparency, light unless the
// desktop prefers light; the Mac keeps its template and Windows the app's
// icon.
func TestTrayIconPerOS(t *testing.T) {
	for _, c := range []struct {
		light bool
		want  color.NRGBA
	}{{false, trayGlyphLight}, {true, trayGlyphDark}} {
		icon, template := trayIconFor("linux", c.light)
		if template {
			t.Errorf("linux: a template icon, which no Linux bar tints")
		}
		img, err := png.Decode(bytes.NewReader(icon))
		if err != nil {
			t.Fatal(err)
		}
		glyph, _ := png.Decode(bytes.NewReader(trayIcon))
		if img.Bounds() != glyph.Bounds() {
			t.Fatalf("linux: %v, want the menu bar mark's %v", img.Bounds(), glyph.Bounds())
		}
		b := img.Bounds()
		shown := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				p := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				g := color.NRGBAModel.Convert(glyph.At(x, y)).(color.NRGBA)
				if p.A != g.A {
					t.Fatalf("linux light=%v: alpha %d at %d,%d, want the mark's %d (a tile or another shape)", c.light, p.A, x, y, g.A)
				}
				if p.A > 0 {
					shown++
					if p.R != c.want.R || p.G != c.want.G || p.B != c.want.B {
						t.Fatalf("linux light=%v: %v at %d,%d, want every pixel %v", c.light, p, x, y, c.want)
					}
				}
			}
		}
		if shown == 0 || shown == b.Dx()*b.Dy() {
			t.Fatalf("linux: %d of %d pixels shown, want the silhouette on transparency", shown, b.Dx()*b.Dy())
		}
	}
	if icon, template := trayIconFor("darwin", false); !template || !bytes.Equal(icon, trayIcon) {
		t.Errorf("darwin: want the template mark")
	}
	if icon, template := trayIconFor("windows", false); template || !bytes.Equal(icon, trayWindowsIcon) {
		t.Errorf("windows: want the tray's own icon")
	}
	if img, _ := png.Decode(bytes.NewReader(trayGlyph(trayGlyphLight))); img == nil || img.ColorModel() != color.NRGBAModel {
		t.Errorf("trayGlyph: want an NRGBA png, got %T", img)
	}
}

// The Windows tray's icon is an icon file with a size for each DPI the
// system asks for (SM_CXSMICON is 16 at 100% and grows with the DPI: 20 at
// 125%, 36 at 225% and 40 at 250%), so the notification area takes one of
// them as it is instead of scaling a bigger one: the app's own icon, scaled
// down to it, came out a size smaller than the icons beside it (#2404). Each
// size's mark fills its own canvas, or the icon is smaller again.
func TestTrayWindowsIconSizes(t *testing.T) {
	entries := icoEntries(t, trayWindowsIcon)
	for _, size := range []int{16, 20, 24, 28, 32, 36, 40, 48, 64} {
		e, ok := icoSize(entries, size)
		if !ok {
			t.Errorf("windows: no %dx%d size in the tray icon, want one for every size the system asks for (16 at 100%% to 40 at 250%%): the system scales the nearest one, so a missing size is drawn soft", size, size)
			continue
		}
		img, err := png.Decode(bytes.NewReader(e.data))
		if err != nil {
			t.Fatalf("windows: %dx%d: %v", size, size, err)
		}
		b := img.Bounds()
		if b.Dx() != size || b.Dy() != size {
			t.Errorf("windows: %dx%d: the picture is %dx%d, want the size it is filed under", size, size, b.Dx(), b.Dy())
			continue
		}
		minX, minY, maxX, maxY := b.Dx(), b.Dy(), -1, -1
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA).A == 0 {
					continue
				}
				minX = min(minX, x-b.Min.X)
				minY = min(minY, y-b.Min.Y)
				maxX = max(maxX, x-b.Min.X)
				maxY = max(maxY, y-b.Min.Y)
			}
		}
		w, h := maxX-minX+1, maxY-minY+1
		// the mark has to reach within a pixel of the canvas' edge, in every
		// size: a canvas that keeps a margin is drawn smaller than the icons
		// beside it
		want := size - 1
		if w < want || h < want {
			t.Errorf("windows: %dx%d: the mark is %dx%d, want %dx%d or more: a canvas that keeps a margin is drawn smaller than the icons beside it",
				size, size, w, h, want, want)
		}
	}
}

// icoEntry is one size of an icon file: the directory entry and its PNG.
type icoEntry struct {
	width, height int
	data          []byte
}

// icoEntries reads the sizes of an icon file: the header, a directory entry
// per size and the PNG of each.
func icoEntries(t *testing.T, icon []byte) []icoEntry {
	t.Helper()
	if len(icon) < 6 || icon[0] != 0 || icon[1] != 0 || icon[2] != 1 || icon[3] != 0 {
		t.Fatalf("windows: not an icon file: % x", icon[:min(6, len(icon))])
	}
	n := int(binary.LittleEndian.Uint16(icon[4:6]))
	if len(icon) < 6+16*n {
		t.Fatalf("windows: %d sizes claimed, the file is %d bytes", n, len(icon))
	}
	entries := make([]icoEntry, 0, n)
	for i := 0; i < n; i++ {
		e := icon[6+16*i : 6+16*(i+1)]
		size := int(binary.LittleEndian.Uint32(e[8:12]))
		off := int(binary.LittleEndian.Uint32(e[12:16]))
		if off+size > len(icon) {
			t.Fatalf("windows: size %d runs past the end of the file", i)
		}
		// a size of 256 is filed as 0, as the byte has no room for it
		w, h := int(e[0]), int(e[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		entries = append(entries, icoEntry{width: w, height: h, data: icon[off : off+size]})
	}
	return entries
}

// icoSize is the size of an icon file's entries, if it is filed under one.
func icoSize(entries []icoEntry, size int) (icoEntry, bool) {
	for _, e := range entries {
		if e.width == size && e.height == size {
			return e, true
		}
	}
	return icoEntry{}, false
}

// The portal's color-scheme: 1 prefers dark, 2 light, 0 neither; Read
// wraps it in a second variant, ReadOne and SettingChanged don't.
func TestPortalLight(t *testing.T) {
	for _, c := range []struct {
		v    dbus.Variant
		want bool
	}{
		{dbus.MakeVariant(uint32(2)), true},
		{dbus.MakeVariant(dbus.MakeVariant(uint32(2))), true},
		{dbus.MakeVariant(uint32(1)), false},
		{dbus.MakeVariant(dbus.MakeVariant(uint32(0))), false},
		{dbus.MakeVariant("light"), false},
	} {
		if got := portalLight(c.v); got != c.want {
			t.Errorf("portalLight(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}
