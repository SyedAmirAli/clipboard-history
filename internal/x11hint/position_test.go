package x11hint

import "testing"

func TestPlaceInMonitor(t *testing.T) {
	// A 1920×1080 primary monitor with a second 1920×1080 to its right.
	primary := rect{x: 0, y: 0, w: 1920, h: 1080}
	second := rect{x: 1920, y: 0, w: 1920, h: 1080}
	const w, h = 520, 620

	cases := []struct {
		name   string
		px, py int
		mon    rect
	}{
		{"plenty of room below-right", 100, 100, primary},
		{"near bottom → flips above", 300, 1000, primary},
		{"near right → slides left", 1800, 100, primary},
		{"bottom-right corner", 1900, 1060, primary},
		{"top-left corner", 0, 0, primary},
		{"second monitor near its right edge", 3800, 900, second},
		{"second monitor left edge", 1925, 500, second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := placeInMonitor(tc.px, tc.py, w, h, tc.mon)
			if x < tc.mon.x || y < tc.mon.y || x+w > tc.mon.x+tc.mon.w || y+h > tc.mon.y+tc.mon.h {
				t.Errorf("popup (%d,%d %dx%d) escapes monitor %+v (pointer %d,%d)",
					x, y, w, h, tc.mon, tc.px, tc.py)
			}
		})
	}

	// With room below, the popup should open below the pointer.
	if _, y := placeInMonitor(100, 100, w, h, primary); y < 100 {
		t.Errorf("expected popup below pointer, got y=%d", y)
	}
	// Near the bottom edge it should flip above the pointer.
	if _, y := placeInMonitor(300, 1000, w, h, primary); y+h > 1000 {
		t.Errorf("expected popup above pointer near bottom edge, got y=%d (h=%d)", y, h)
	}
}

func TestMonitorGeomParse(t *testing.T) {
	m := monitorGeomRe.FindStringSubmatch(" 0: +*eDP-1 1920/309x1080/174+0+0  eDP-1")
	if m == nil {
		t.Fatal("no geometry match for primary monitor line")
	}
	if m[1] != "1920" || m[2] != "1080" || m[3] != "0" || m[4] != "0" {
		t.Errorf("parsed %v, want [1920 1080 0 0]", m[1:])
	}
	m = monitorGeomRe.FindStringSubmatch(" 1: +HDMI-1 2560/597x1440/336+1920+0  HDMI-1")
	if m == nil {
		t.Fatal("no geometry match for secondary monitor line")
	}
	if m[1] != "2560" || m[2] != "1440" || m[3] != "1920" || m[4] != "0" {
		t.Errorf("parsed %v, want [2560 1440 1920 0]", m[1:])
	}
}
