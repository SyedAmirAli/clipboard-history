package shortcut

import (
	"reflect"
	"testing"
)

func TestParsePathList(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"@as []", nil},
		{"[]", nil},
		{"['/org/a/custom0/']", []string{"/org/a/custom0/"}},
		{
			"['/org/a/custom0/', '/org/a/clipd/']",
			[]string{"/org/a/custom0/", "/org/a/clipd/"},
		},
		{"['/org/a/custom0/']\n", []string{"/org/a/custom0/"}},
	}
	for _, c := range cases {
		got := parsePathList(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parsePathList(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFormatPathList(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, "@as []"},
		{[]string{"/a/"}, "['/a/']"},
		{[]string{"/a/", "/b/"}, "['/a/', '/b/']"},
	}
	for _, c := range cases {
		if got := formatPathList(c.in); got != c.want {
			t.Errorf("formatPathList(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Round-trip: a formatted list must parse back to the same paths.
func TestPathListRoundTrip(t *testing.T) {
	in := []string{"/org/gnome/x/custom0/", "/org/gnome/x/clipd/"}
	got := parsePathList(formatPathList(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip = %v, want %v", got, in)
	}
}

func TestToGtkAccelerator(t *testing.T) {
	ok := []struct {
		spec string
		want string
	}{
		{"Super+V", "<Super>v"},
		{"super+v", "<Super>v"},
		{"Ctrl+Alt+H", "<Control><Alt>h"},
		{"Ctrl+Shift+,", "<Control><Shift>,"},
		{"Meta+Space", "<Super>space"},
	}
	for _, c := range ok {
		got, err := toGtkAccelerator(c.spec)
		if err != nil {
			t.Errorf("toGtkAccelerator(%q) unexpected error: %v", c.spec, err)
			continue
		}
		if got != c.want {
			t.Errorf("toGtkAccelerator(%q) = %q, want %q", c.spec, got, c.want)
		}
	}

	bad := []string{"", "V", "Foo+V", "+"}
	for _, spec := range bad {
		if _, err := toGtkAccelerator(spec); err == nil {
			t.Errorf("toGtkAccelerator(%q) = nil error, want error", spec)
		}
	}
}
