package natsort

import (
	"reflect"
	"testing"
)

func TestSliceStable(t *testing.T) {
	in := []string{"page10.jpg", "page2.jpg", "page1.jpg", "page20.jpg", "cover.jpg"}
	want := []string{"cover.jpg", "page1.jpg", "page2.jpg", "page10.jpg", "page20.jpg"}
	got := append([]string(nil), in...)
	SliceStable(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SliceStable = %v, want %v", got, want)
	}
}

func TestLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"img2", "img10", true},
		{"img10", "img2", false},
		{"a", "b", true},
		{"page-001", "page-2", true}, // 1 < 2 numerically despite zero padding
		{"Page2", "page10", true},    // case-insensitive
		{"x", "x", false},
		{"chapter1/p1", "chapter1/p2", true},
	}
	for _, c := range cases {
		if got := Less(c.a, c.b); got != c.want {
			t.Errorf("Less(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
