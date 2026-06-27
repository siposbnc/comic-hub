package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// makePNG builds a w×h PNG with a simple gradient.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestDimensions(t *testing.T) {
	src := makePNG(t, 120, 80)
	w, h, err := New().Dimensions(bytes.NewReader(src))
	if err != nil {
		t.Fatalf("dimensions: %v", err)
	}
	if w != 120 || h != 80 {
		t.Fatalf("dims = %dx%d, want 120x80", w, h)
	}
}

func TestResizeDownscaleJPEG(t *testing.T) {
	src := makePNG(t, 200, 100)
	res, err := New().Resize(src, Options{Width: 50})
	if err != nil {
		t.Fatalf("resize: %v", err)
	}
	if res.ContentType != "image/jpeg" {
		t.Errorf("content type = %q, want image/jpeg", res.ContentType)
	}
	if res.Width != 50 || res.Height != 25 { // aspect preserved
		t.Fatalf("result dims = %dx%d, want 50x25", res.Width, res.Height)
	}
	// Output must actually decode at the new size.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(res.Data))
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if cfg.Width != 50 || cfg.Height != 25 {
		t.Fatalf("decoded dims = %dx%d, want 50x25", cfg.Width, cfg.Height)
	}
}

func TestResizeNoUpscale(t *testing.T) {
	src := makePNG(t, 80, 80)
	res, err := New().Resize(src, Options{Width: 300})
	if err != nil {
		t.Fatalf("resize: %v", err)
	}
	if res.Width != 80 || res.Height != 80 {
		t.Fatalf("dims = %dx%d, want 80x80 (no upscale)", res.Width, res.Height)
	}
}

func TestResizePNGFormat(t *testing.T) {
	src := makePNG(t, 100, 100)
	res, err := New().Resize(src, Options{Width: 40, Format: FormatPNG})
	if err != nil {
		t.Fatalf("resize: %v", err)
	}
	if res.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", res.ContentType)
	}
}
