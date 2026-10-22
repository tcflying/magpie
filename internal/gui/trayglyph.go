//go:build !nogui

package gui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/godbus/dbus/v5"
)

// A Linux bar's tray icon is the menu bar's disc alone, in one colour on
// transparency, as the icons beside it are (ylorn on Discord): the app's
// own coloured icon stood out in a row of them, and a bar that tints its
// icons to the theme by their alpha tints the mark like them. A bar that
// doesn't tint shows the colour given: light, as most bars are dark,
// unless the desktop prefers light (traySchemeLight).
var (
	trayGlyphLight = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	trayGlyphDark  = color.NRGBA{0x1f, 0x1f, 0x1f, 0xff}
)

// trayGlyph is trayIcon, the disc's silhouette (a black glyph on
// transparency), in c: each pixel keeps the glyph's alpha. trayIcon itself
// when it can't be read, which is still a silhouette.
func trayGlyph(c color.NRGBA) []byte {
	src, err := png.Decode(bytes.NewReader(trayIcon))
	if err != nil {
		return trayIcon
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			a := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA).A
			out.SetNRGBA(x, y, color.NRGBA{c.R, c.G, c.B, uint8(uint(a) * uint(c.A) / 0xff)})
		}
	}
	var buf bytes.Buffer
	if png.Encode(&buf, out) != nil {
		return trayIcon
	}
	return buf.Bytes()
}

// trayIconFor is the tray's icon on goos, and whether it is a template
// image the system tints (the Mac's menu bar): the mark alone on the Mac
// and Linux (linuxTrayIcon, light saying the desktop prefers a light
// scheme), and in Windows' tray the app's mark at the size the system asks
// for (trayWindowsIcon), which is an icon with a size for every DPI, so the
// notification area takes one of it as it is instead of scaling a bigger
// one (#2404).
func trayIconFor(goos string, light bool) (icon []byte, template bool) {
	switch goos {
	case "darwin":
		return trayIcon, true
	case "linux":
		return linuxTrayIcon(light), false
	}
	return trayWindowsIcon, false
}

// linuxTrayIcon is the glyph for a desktop whose colour scheme preference
// (the portal's org.freedesktop.appearance color-scheme) is light or not:
// dark only where light is asked for; "no preference" or one that can't be
// read is light, the bars of GNOME, Omarchy and waybar's own style being
// dark.
func linuxTrayIcon(light bool) []byte {
	if light {
		return trayGlyph(trayGlyphDark)
	}
	return trayGlyph(trayGlyphLight)
}

// portalLight reads the portal's color-scheme value as linuxTrayIcon takes
// it: light only for 2 (prefer light). Settings.Read wraps the value in a
// second variant and ReadOne and SettingChanged don't, so either is
// unwrapped; anything else is no preference.
func portalLight(v dbus.Variant) bool {
	for range 2 {
		if in, ok := v.Value().(dbus.Variant); ok {
			v = in
		}
	}
	n, ok := v.Value().(uint32)
	return ok && n == 2
}
