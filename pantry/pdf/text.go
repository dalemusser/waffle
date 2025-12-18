// pdf/text.go
package pdf

import (
	"fmt"
	"strings"
)

// TextAlign represents text alignment.
type TextAlign int

const (
	AlignLeft TextAlign = iota
	AlignCenter
	AlignRight
	AlignJustify
)

// Color represents an RGB color.
type Color struct {
	R, G, B float64 // 0.0 to 1.0
}

// Common colors.
var (
	Black   = Color{0, 0, 0}
	White   = Color{1, 1, 1}
	Red     = Color{1, 0, 0}
	Green   = Color{0, 1, 0}
	Blue    = Color{0, 0, 1}
	Gray    = Color{0.5, 0.5, 0.5}
	Yellow  = Color{1, 1, 0}
	Cyan    = Color{0, 1, 1}
	Magenta = Color{1, 0, 1}
)

// RGB creates a color from 0-255 values.
func RGB(r, g, b int) Color {
	return Color{
		R: float64(r) / 255.0,
		G: float64(g) / 255.0,
		B: float64(b) / 255.0,
	}
}

// Hex creates a color from a hex string (e.g., "#FF5500" or "FF5500").
func Hex(hex string) Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return Black
	}
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return RGB(r, g, b)
}

// SetFont sets the current font.
func (d *Document) SetFont(name string, size float64) *Document {
	if _, ok := d.fonts[name]; ok {
		d.fontName = name
		d.fontSize = size
	}
	return d
}

// SetFontSize sets the current font size.
func (d *Document) SetFontSize(size float64) *Document {
	d.fontSize = size
	return d
}

// SetLineHeight sets the line height multiplier.
func (d *Document) SetLineHeight(h float64) *Document {
	d.lineHeight = h
	return d
}

// Font sets font and returns document for chaining.
func (d *Document) Font(name string) *Document {
	if _, ok := d.fonts[name]; ok {
		d.fontName = name
	}
	return d
}

// Size sets font size and returns document for chaining.
func (d *Document) Size(size float64) *Document {
	d.fontSize = size
	return d
}

// Bold switches to bold variant of current font.
func (d *Document) Bold() *Document {
	name := d.fontName
	switch {
	case strings.HasPrefix(name, "Helvetica"):
		if strings.Contains(name, "Oblique") {
			d.fontName = "Helvetica-BoldOblique"
		} else {
			d.fontName = "Helvetica-Bold"
		}
	case strings.HasPrefix(name, "Times"):
		if strings.Contains(name, "Italic") {
			d.fontName = "Times-BoldItalic"
		} else {
			d.fontName = "Times-Bold"
		}
	case strings.HasPrefix(name, "Courier"):
		if strings.Contains(name, "Oblique") {
			d.fontName = "Courier-BoldOblique"
		} else {
			d.fontName = "Courier-Bold"
		}
	}
	return d
}

// Italic switches to italic variant of current font.
func (d *Document) Italic() *Document {
	name := d.fontName
	switch {
	case strings.HasPrefix(name, "Helvetica"):
		if strings.Contains(name, "Bold") {
			d.fontName = "Helvetica-BoldOblique"
		} else {
			d.fontName = "Helvetica-Oblique"
		}
	case strings.HasPrefix(name, "Times"):
		if strings.Contains(name, "Bold") {
			d.fontName = "Times-BoldItalic"
		} else {
			d.fontName = "Times-Italic"
		}
	case strings.HasPrefix(name, "Courier"):
		if strings.Contains(name, "Bold") {
			d.fontName = "Courier-BoldOblique"
		} else {
			d.fontName = "Courier-Oblique"
		}
	}
	return d
}

// Regular switches to regular variant of current font.
func (d *Document) Regular() *Document {
	name := d.fontName
	switch {
	case strings.HasPrefix(name, "Helvetica"):
		d.fontName = "Helvetica"
	case strings.HasPrefix(name, "Times"):
		d.fontName = "Times-Roman"
	case strings.HasPrefix(name, "Courier"):
		d.fontName = "Courier"
	}
	return d
}

// Text writes text at the current position.
func (d *Document) Text(text string) *Document {
	d.ensurePage()
	d.writeText(text, d.x, d.y)
	return d
}

// TextAt writes text at the specified position.
func (d *Document) TextAt(x, y float64, text string) *Document {
	d.ensurePage()
	d.writeText(text, x, d.currentPage.size.Height-y)
	return d
}

// writeText writes text to the current page.
func (d *Document) writeText(text string, x, y float64) {
	d.currentPage.resources[d.fontName] = true

	escaped := escapeString(text)
	d.currentPage.content.WriteString(fmt.Sprintf(
		"BT /%s %.2f Tf %.2f %.2f Td (%s) Tj ET\n",
		d.fontName, d.fontSize, x, y, escaped))
}

// Ln moves to the next line.
func (d *Document) Ln() *Document {
	d.ensurePage()
	d.x = d.margins.Left
	d.y -= d.fontSize * d.lineHeight

	// Check for page break
	if d.y < d.margins.Bottom {
		d.AddPage()
	}

	return d
}

// Br adds a line break without resetting x position.
func (d *Document) Br() *Document {
	d.ensurePage()
	d.y -= d.fontSize * d.lineHeight

	if d.y < d.margins.Bottom {
		d.AddPage()
	}

	return d
}

// WriteText writes text and advances position (inline).
func (d *Document) WriteText(text string) *Document {
	d.ensurePage()
	d.writeText(text, d.x, d.y)
	d.x += d.measureText(text)
	return d
}

// Writef writes formatted text.
func (d *Document) Writef(format string, args ...any) *Document {
	return d.WriteText(fmt.Sprintf(format, args...))
}

// WriteLine writes text and moves to next line.
func (d *Document) WriteLine(text string) *Document {
	return d.Text(text).Ln()
}

// WriteLinef writes formatted text and moves to next line.
func (d *Document) WriteLinef(format string, args ...any) *Document {
	return d.WriteLine(fmt.Sprintf(format, args...))
}

// Paragraph writes a paragraph with word wrapping.
func (d *Document) Paragraph(text string) *Document {
	d.ensurePage()

	width := d.contentWidth()
	words := strings.Fields(text)

	var line strings.Builder
	lineWidth := 0.0
	spaceWidth := d.measureText(" ")

	for _, word := range words {
		wordWidth := d.measureText(word)

		if lineWidth+wordWidth > width && line.Len() > 0 {
			// Write current line
			d.Text(line.String()).Ln()
			line.Reset()
			lineWidth = 0
		}

		if line.Len() > 0 {
			line.WriteString(" ")
			lineWidth += spaceWidth
		}
		line.WriteString(word)
		lineWidth += wordWidth
	}

	// Write remaining text
	if line.Len() > 0 {
		d.Text(line.String()).Ln()
	}

	return d
}

// TextWidth returns the width of text in points.
func (d *Document) TextWidth(text string) float64 {
	return d.measureText(text)
}

// measureText calculates the width of text.
func (d *Document) measureText(text string) float64 {
	// Approximate widths for standard fonts
	// This is a simplification - real implementation would use font metrics
	var avgWidth float64

	switch {
	case strings.HasPrefix(d.fontName, "Courier"):
		avgWidth = 0.6 // Monospace
	case strings.HasPrefix(d.fontName, "Times"):
		avgWidth = 0.45
	default: // Helvetica and others
		avgWidth = 0.5
	}

	return float64(len(text)) * avgWidth * d.fontSize
}

// CenterText writes centered text at the current y position.
func (d *Document) CenterText(text string) *Document {
	d.ensurePage()

	width := d.measureText(text)
	x := d.margins.Left + (d.contentWidth()-width)/2
	d.writeText(text, x, d.y)

	return d
}

// RightText writes right-aligned text at the current y position.
func (d *Document) RightText(text string) *Document {
	d.ensurePage()

	width := d.measureText(text)
	x := d.currentPage.size.Width - d.margins.Right - width
	d.writeText(text, x, d.y)

	return d
}

// Title writes a title (large, bold, centered).
func (d *Document) Title(text string) *Document {
	oldFont := d.fontName
	oldSize := d.fontSize

	d.Bold().Size(24)
	d.CenterText(text).Ln().Ln()

	d.fontName = oldFont
	d.fontSize = oldSize

	return d
}

// Heading writes a heading (bold, larger).
func (d *Document) Heading(text string) *Document {
	oldFont := d.fontName
	oldSize := d.fontSize

	d.Bold().Size(16)
	d.Text(text).Ln()

	d.fontName = oldFont
	d.fontSize = oldSize

	return d
}

// Subheading writes a subheading.
func (d *Document) Subheading(text string) *Document {
	oldFont := d.fontName
	oldSize := d.fontSize

	d.Bold().Size(14)
	d.Text(text).Ln()

	d.fontName = oldFont
	d.fontSize = oldSize

	return d
}

// escapeString escapes special characters in PDF strings.
func escapeString(s string) string {
	var result strings.Builder
	for _, c := range s {
		switch c {
		case '\\':
			result.WriteString("\\\\")
		case '(':
			result.WriteString("\\(")
		case ')':
			result.WriteString("\\)")
		case '\n':
			result.WriteString("\\n")
		case '\r':
			result.WriteString("\\r")
		case '\t':
			result.WriteString("\\t")
		default:
			if c >= 32 && c < 127 {
				result.WriteRune(c)
			} else if c < 256 {
				result.WriteString(fmt.Sprintf("\\%03o", c))
			}
		}
	}
	return result.String()
}
