// pdf/draw.go
package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
)

// LineStyle represents line drawing styles.
type LineStyle struct {
	Width float64
	Color Color
	Dash  []float64 // Dash pattern
}

// DefaultLineStyle returns the default line style.
func DefaultLineStyle() LineStyle {
	return LineStyle{
		Width: 1,
		Color: Black,
	}
}

// SetLineWidth sets the line width for drawing operations.
func (d *Document) SetLineWidth(width float64) *Document {
	d.ensurePage()
	d.currentPage.content.WriteString(fmt.Sprintf("%.2f w\n", width))
	return d
}

// SetStrokeColor sets the stroke (line) color.
func (d *Document) SetStrokeColor(c Color) *Document {
	d.ensurePage()
	d.currentPage.content.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG\n", c.R, c.G, c.B))
	return d
}

// SetFillColor sets the fill color.
func (d *Document) SetFillColor(c Color) *Document {
	d.ensurePage()
	d.currentPage.content.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", c.R, c.G, c.B))
	return d
}

// SetDash sets the line dash pattern.
func (d *Document) SetDash(pattern []float64, phase float64) *Document {
	d.ensurePage()
	var buf bytes.Buffer
	buf.WriteString("[")
	for i, p := range pattern {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprintf("%.2f", p))
	}
	buf.WriteString(fmt.Sprintf("] %.2f d\n", phase))
	d.currentPage.content.Write(buf.Bytes())
	return d
}

// ClearDash clears the dash pattern (solid lines).
func (d *Document) ClearDash() *Document {
	d.ensurePage()
	d.currentPage.content.WriteString("[] 0 d\n")
	return d
}

// Line draws a line between two points.
func (d *Document) Line(x1, y1, x2, y2 float64) *Document {
	d.ensurePage()
	h := d.currentPage.size.Height
	d.currentPage.content.WriteString(fmt.Sprintf(
		"%.2f %.2f m %.2f %.2f l S\n",
		x1, h-y1, x2, h-y2))
	return d
}

// HLine draws a horizontal line at the current y position.
func (d *Document) HLine() *Document {
	d.ensurePage()
	return d.Line(d.margins.Left, d.currentPage.size.Height-d.y,
		d.currentPage.size.Width-d.margins.Right, d.currentPage.size.Height-d.y)
}

// HLineAt draws a horizontal line at a specific y position.
func (d *Document) HLineAt(y float64) *Document {
	d.ensurePage()
	return d.Line(d.margins.Left, y,
		d.currentPage.size.Width-d.margins.Right, y)
}

// Rect draws a rectangle.
func (d *Document) Rect(x, y, width, height float64) *Document {
	d.ensurePage()
	h := d.currentPage.size.Height
	d.currentPage.content.WriteString(fmt.Sprintf(
		"%.2f %.2f %.2f %.2f re S\n",
		x, h-y-height, width, height))
	return d
}

// RectFilled draws a filled rectangle.
func (d *Document) RectFilled(x, y, width, height float64) *Document {
	d.ensurePage()
	h := d.currentPage.size.Height
	d.currentPage.content.WriteString(fmt.Sprintf(
		"%.2f %.2f %.2f %.2f re f\n",
		x, h-y-height, width, height))
	return d
}

// RectFilledStroke draws a filled rectangle with a stroke.
func (d *Document) RectFilledStroke(x, y, width, height float64) *Document {
	d.ensurePage()
	h := d.currentPage.size.Height
	d.currentPage.content.WriteString(fmt.Sprintf(
		"%.2f %.2f %.2f %.2f re B\n",
		x, h-y-height, width, height))
	return d
}

// Circle draws a circle.
func (d *Document) Circle(x, y, radius float64) *Document {
	return d.Ellipse(x, y, radius, radius)
}

// CircleFilled draws a filled circle.
func (d *Document) CircleFilled(x, y, radius float64) *Document {
	return d.EllipseFilled(x, y, radius, radius)
}

// Ellipse draws an ellipse.
func (d *Document) Ellipse(x, y, rx, ry float64) *Document {
	d.ensurePage()
	d.drawEllipse(x, y, rx, ry, "S")
	return d
}

// EllipseFilled draws a filled ellipse.
func (d *Document) EllipseFilled(x, y, rx, ry float64) *Document {
	d.ensurePage()
	d.drawEllipse(x, y, rx, ry, "f")
	return d
}

// drawEllipse draws an ellipse using Bezier curves.
func (d *Document) drawEllipse(x, y, rx, ry float64, op string) {
	h := d.currentPage.size.Height
	y = h - y

	// Approximate ellipse with 4 Bezier curves
	k := 0.5522848 // Magic number for circle approximation

	d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f m\n", x+rx, y))
	d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f %.2f %.2f c\n",
		x+rx, y+ry*k, x+rx*k, y+ry, x, y+ry))
	d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f %.2f %.2f c\n",
		x-rx*k, y+ry, x-rx, y+ry*k, x-rx, y))
	d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f %.2f %.2f c\n",
		x-rx, y-ry*k, x-rx*k, y-ry, x, y-ry))
	d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f %.2f %.2f c\n",
		x+rx*k, y-ry, x+rx, y-ry*k, x+rx, y))
	d.currentPage.content.WriteString(op + "\n")
}

// Polygon draws a polygon from a series of points.
func (d *Document) Polygon(points []struct{ X, Y float64 }) *Document {
	if len(points) < 3 {
		return d
	}
	d.ensurePage()
	h := d.currentPage.size.Height

	d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f m\n",
		points[0].X, h-points[0].Y))
	for i := 1; i < len(points); i++ {
		d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f l\n",
			points[i].X, h-points[i].Y))
	}
	d.currentPage.content.WriteString("h S\n")

	return d
}

// PolygonFilled draws a filled polygon.
func (d *Document) PolygonFilled(points []struct{ X, Y float64 }) *Document {
	if len(points) < 3 {
		return d
	}
	d.ensurePage()
	h := d.currentPage.size.Height

	d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f m\n",
		points[0].X, h-points[0].Y))
	for i := 1; i < len(points); i++ {
		d.currentPage.content.WriteString(fmt.Sprintf("%.2f %.2f l\n",
			points[i].X, h-points[i].Y))
	}
	d.currentPage.content.WriteString("h f\n")

	return d
}

// Image draws an image at the specified position.
func (d *Document) Image(x, y, width, height float64, img image.Image) *Document {
	d.ensurePage()

	// Encode image to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return d
	}

	imgData := buf.Bytes()
	imgName := fmt.Sprintf("Img%d", len(d.images)+1)

	d.images[imgName] = &pdfImage{
		name:   imgName,
		width:  img.Bounds().Dx(),
		height: img.Bounds().Dy(),
		data:   imgData,
	}

	// Add image to page content
	h := d.currentPage.size.Height
	d.currentPage.content.WriteString("q\n") // Save graphics state
	d.currentPage.content.WriteString(fmt.Sprintf("%.2f 0 0 %.2f %.2f %.2f cm\n",
		width, height, x, h-y-height))
	d.currentPage.content.WriteString(fmt.Sprintf("/%s Do\n", imgName))
	d.currentPage.content.WriteString("Q\n") // Restore graphics state

	return d
}

// ImageFromFile draws an image from a file.
func (d *Document) ImageFromFile(x, y, width, height float64, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	d.Image(x, y, width, height, img)
	return nil
}

// ImageFromReader draws an image from a reader.
func (d *Document) ImageFromReader(x, y, width, height float64, r io.Reader) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}

	d.Image(x, y, width, height, img)
	return nil
}

// ImageFromBytes draws an image from bytes.
func (d *Document) ImageFromBytes(x, y, width, height float64, data []byte) error {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	d.Image(x, y, width, height, img)
	return nil
}

// ImageFromBase64 draws an image from base64-encoded data.
func (d *Document) ImageFromBase64(x, y, width, height float64, b64 string) error {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return d.ImageFromBytes(x, y, width, height, data)
}

// SaveState saves the current graphics state.
func (d *Document) SaveState() *Document {
	d.ensurePage()
	d.currentPage.content.WriteString("q\n")
	return d
}

// RestoreState restores the previously saved graphics state.
func (d *Document) RestoreState() *Document {
	d.ensurePage()
	d.currentPage.content.WriteString("Q\n")
	return d
}

// Translate applies a translation transformation.
func (d *Document) Translate(tx, ty float64) *Document {
	d.ensurePage()
	d.currentPage.content.WriteString(fmt.Sprintf("1 0 0 1 %.2f %.2f cm\n", tx, -ty))
	return d
}

// Scale applies a scaling transformation.
func (d *Document) Scale(sx, sy float64) *Document {
	d.ensurePage()
	d.currentPage.content.WriteString(fmt.Sprintf("%.2f 0 0 %.2f 0 0 cm\n", sx, sy))
	return d
}

// Rotate applies a rotation transformation (degrees).
func (d *Document) Rotate(angle float64) *Document {
	d.ensurePage()
	rad := angle * 3.14159265358979323846 / 180
	cos := float64(int(1000*cosApprox(rad)+0.5)) / 1000
	sin := float64(int(1000*sinApprox(rad)+0.5)) / 1000
	d.currentPage.content.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f 0 0 cm\n",
		cos, sin, -sin, cos))
	return d
}

// Simple trig approximations to avoid math import in this file
func sinApprox(x float64) float64 {
	// Taylor series approximation
	x3 := x * x * x
	x5 := x3 * x * x
	return x - x3/6 + x5/120
}

func cosApprox(x float64) float64 {
	x2 := x * x
	x4 := x2 * x2
	return 1 - x2/2 + x4/24
}

// Register image decoders
func init() {
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG", png.Decode, png.DecodeConfig)
}
