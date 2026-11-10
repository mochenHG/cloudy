package app

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// The zooms of the window the listener can choose, as a browser's: for
// screens of many pixels where text comes out small, or few where it is
// too large.
const (
	minUIZoom = 0.8
	maxUIZoom = 2
)

// uiZoom is how much larger than as designed the main window shows
// everything.
func (a *app) uiZoom() float32 {
	if z := a.settings.UIZoom; z > 0 {
		return float32(math.Max(minUIZoom, math.Min(z, maxUIZoom)))
	}
	return 1
}

// zoomBy zooms the window by a step of 10%, or back to as designed for 0.
func (a *app) zoomBy(by float64) {
	z := 1.0
	if by != 0 {
		z = math.Round((float64(a.uiZoom())+by)*10) / 10
		z = math.Max(minUIZoom, math.Min(z, maxUIZoom))
	}
	if z == 1 {
		z = 0
	}
	if z == a.settings.UIZoom {
		return
	}
	a.settings.UIZoom = z
	a.changed()
}

// zoom zooms the main window as chosen, and takes Ctrl+= and Ctrl+- (⌘ on
// macOS) to zoom it in and out, and Ctrl+0 back, as browsers do.
func (a *app) zoom(c *ui.Context) {
	c.OnShortcut(ui.Cmd, ui.KeyEqual, func() { a.zoomBy(0.1) })
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyEqual, func() { a.zoomBy(0.1) })
	c.OnShortcut(ui.Cmd, ui.KeyMinus, func() { a.zoomBy(-0.1) })
	c.OnShortcut(ui.Cmd, ui.Key0, func() { a.zoomBy(0) })
	c.SetZoom(a.uiZoom())
}
