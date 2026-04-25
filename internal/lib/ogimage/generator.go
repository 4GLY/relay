// Package ogimage renders the 1200x630 PNG that ships with every shared
// packet snapshot. The artwork is intentionally minimal so it survives at
// thumbnail sizes (Slack/Twitter unfurls): a Fraunces project name, a Nunito
// subtitle, and a small swan-stamp glyph in the lower-right corner. The swan
// stamp here is a placeholder shape (V2.5 will replace it with the actual
// DESIGN.md duckling-to-swan contour).
package ogimage

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

//go:embed fonts/*.ttf
var fontFS embed.FS

const (
	width        = 1200
	height       = 630
	maxTitleLen  = 60
	maxSubLen    = 80
	swanCenterX  = float64(width) - 110
	swanCenterY  = float64(height) - 110
	swanRadius   = 36
	swanWingDX   = -12
	swanWingDY   = -8
	swanWingR    = 6
	titleFontPx  = 88
	subtitlePx   = 32
	titleYOffset = -40
	subYOffset   = 60
)

// Options is the public input to Generate.
type Options struct {
	ProjectName string
	Subtitle    string // e.g. "Snapshot from Relay"
}

// Generate renders an OG image as PNG bytes. It is deterministic for a
// given Options input (fonts are embedded so output is reproducible across
// machines with the same Go toolchain).
func Generate(opts Options) ([]byte, error) {
	dc := gg.NewContext(width, height)

	// Background — DESIGN.md `--canvas` (light theme).
	dc.SetColor(color.RGBA{0xFA, 0xF8, 0xF3, 0xFF})
	dc.Clear()

	// Subtle warm halo behind the title — `--halo` rgba(167,196,255,0.35).
	dc.DrawRectangle(0, float64(height)/2-180, float64(width), 360)
	dc.SetColor(color.RGBA{0xA7, 0xC4, 0xFF, 0x22})
	dc.Fill()

	titleFace, err := loadFont("fonts/Fraunces-Regular.ttf", titleFontPx)
	if err != nil {
		return nil, fmt.Errorf("load fraunces: %w", err)
	}
	subtitleFace, err := loadFont("fonts/Nunito-Regular.ttf", subtitlePx)
	if err != nil {
		return nil, fmt.Errorf("load nunito: %w", err)
	}

	title := truncate(strings.TrimSpace(opts.ProjectName), maxTitleLen)
	if title == "" {
		title = "Untitled snapshot"
	}
	subtitle := truncate(strings.TrimSpace(opts.Subtitle), maxSubLen)
	if subtitle == "" {
		subtitle = "Snapshot from Relay"
	}

	// Project name — DESIGN.md `--ink` (light theme).
	dc.SetColor(color.RGBA{0x0E, 0x1A, 0x35, 0xFF})
	dc.SetFontFace(titleFace)
	dc.DrawStringAnchored(title, float64(width)/2, float64(height)/2+titleYOffset, 0.5, 0.5)

	// Subtitle — DESIGN.md `--ink-muted` (light theme).
	dc.SetColor(color.RGBA{0x4A, 0x56, 0x69, 0xFF})
	dc.SetFontFace(subtitleFace)
	dc.DrawStringAnchored(subtitle, float64(width)/2, float64(height)/2+subYOffset, 0.5, 0.5)

	drawSwanStamp(dc, swanCenterX, swanCenterY)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func loadFont(path string, size float64) (font.Face, error) {
	raw, err := fontFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	parsed, err := truetype.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return truetype.NewFace(parsed, &truetype.Options{
		Size: size,
		DPI:  72,
	}), nil
}

// drawSwanStamp paints a placeholder swan glyph: a filled disc with a small
// inset wing. V2.5 will swap this for the real DESIGN.md duckling-to-swan
// contour so unfurls carry the brand object even when motion is forbidden.
func drawSwanStamp(dc *gg.Context, cx, cy float64) {
	// Body — DESIGN.md `--magic-primary-strong` (light theme).
	dc.SetColor(color.RGBA{0x6F, 0x96, 0xDB, 0xFF})
	dc.DrawCircle(cx, cy, swanRadius)
	dc.Fill()

	// Wing — DESIGN.md `--canvas` so the stamp reads as a contour.
	dc.SetColor(color.RGBA{0xFA, 0xF8, 0xF3, 0xFF})
	dc.DrawCircle(cx+swanWingDX, cy+swanWingDY, swanWingR)
	dc.Fill()
}

func truncate(input string, limit int) string {
	if len(input) <= limit {
		return input
	}
	if limit <= 1 {
		return input[:limit]
	}
	return input[:limit-1] + "…"
}
