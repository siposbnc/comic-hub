// Package image is the server's image pipeline boundary: reading dimensions and
// resizing/transcoding page images. The Processor interface lets the pure-Go
// implementation here be swapped for a libvips (govips) one later without touching call
// sites (see docs/04-server.md §5).
package image

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"

	// Decoders, registered for image.Decode / image.DecodeConfig.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Format is an output encoding for derived images.
type Format string

const (
	FormatJPEG Format = "jpeg"
	FormatPNG  Format = "png"
)

// DefaultQuality is the JPEG quality used when none is specified.
const DefaultQuality = 82

// Options controls a resize/transcode.
type Options struct {
	// Width is the target width in px. Images are never upscaled past their original
	// width; 0 means "original width".
	Width int
	// Format is the output encoding (default JPEG). WebP/AVIF arrive with the govips swap.
	Format Format
	// Quality is the JPEG quality (1-100); 0 uses DefaultQuality.
	Quality int
}

// Result is a derived image.
type Result struct {
	Data        []byte
	ContentType string
	Width       int
	Height      int
}

// Processor reads dimensions and produces resized/transcoded images.
type Processor interface {
	// Dimensions returns the pixel dimensions of an encoded image, reading only its
	// header where the format allows.
	Dimensions(r io.Reader) (width, height int, err error)
	// Resize decodes src and returns it scaled to Options.Width (aspect preserved) and
	// encoded in Options.Format.
	Resize(src []byte, opts Options) (Result, error)
}

// GoProcessor is the dependency-free implementation (std image + golang.org/x/image).
type GoProcessor struct{}

// New returns the default (pure-Go) processor.
func New() *GoProcessor { return &GoProcessor{} }

func (GoProcessor) Dimensions(r io.Reader) (int, int, error) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func (GoProcessor) Resize(src []byte, opts Options) (Result, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return Result{}, fmt.Errorf("decode: %w", err)
	}

	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	targetW := srcW
	if opts.Width > 0 && opts.Width < srcW {
		targetW = opts.Width
	}
	targetH := srcH
	if targetW != srcW && srcW > 0 {
		targetH = int(float64(srcH) * float64(targetW) / float64(srcW))
		if targetH < 1 {
			targetH = 1
		}
	}

	out := img
	if targetW != srcW {
		dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		out = dst
	}

	var buf bytes.Buffer
	contentType := "image/jpeg"
	switch opts.Format {
	case FormatPNG:
		if err := png.Encode(&buf, out); err != nil {
			return Result{}, fmt.Errorf("encode png: %w", err)
		}
		contentType = "image/png"
	default: // JPEG
		q := opts.Quality
		if q <= 0 {
			q = DefaultQuality
		}
		if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: q}); err != nil {
			return Result{}, fmt.Errorf("encode jpeg: %w", err)
		}
	}

	return Result{Data: buf.Bytes(), ContentType: contentType, Width: targetW, Height: targetH}, nil
}

var _ Processor = GoProcessor{}
