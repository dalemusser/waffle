# PDF - PDF Generation

The `pdf` package provides pure Go PDF generation with no external dependencies. It supports text, shapes, images, and tables.

## Features

- **Pure Go**: No external binaries or dependencies
- **Standard Fonts**: 14 built-in PDF fonts (Helvetica, Times, Courier, etc.)
- **Text**: Multiple fonts, sizes, styles, alignment, word wrapping
- **Drawing**: Lines, rectangles, circles, ellipses, polygons
- **Images**: JPEG and PNG support
- **Tables**: Headers, borders, cell padding, alternating rows
- **Multiple Pages**: Automatic page breaks

## Installation

```go
import "waffle/pdf"
```

## Quick Start

```go
doc := pdf.New()

doc.AddPage().
    Title("Hello World").
    Paragraph("This is a simple PDF document generated with pure Go.").
    Save("hello.pdf")
```

## Document Setup

### Page Size and Margins

```go
doc := pdf.New()

// Set page size
doc.SetPageSize(pdf.A4)        // A4
doc.SetPageSize(pdf.Letter)    // US Letter (default)
doc.SetPageSize(pdf.Legal)     // US Legal

// Custom page size (points)
doc.SetPageSize(pdf.PageSize{Width: 500, Height: 700})

// Orientation
doc.SetOrientation(pdf.Portrait)   // Default
doc.SetOrientation(pdf.Landscape)

// Margins (points)
doc.SetMargins(72, 72, 72, 72)     // 1 inch all around

// Unit helpers
doc.SetMargins(pdf.Inches(1), pdf.Inches(0.75), pdf.Inches(1), pdf.Inches(0.75))
doc.SetMargins(pdf.Cm(2.54), pdf.Cm(1.9), pdf.Cm(2.54), pdf.Cm(1.9))
doc.SetMargins(pdf.Mm(25.4), pdf.Mm(19), pdf.Mm(25.4), pdf.Mm(19))
```

### Metadata

```go
doc.SetTitle("My Document")
doc.SetAuthor("John Doe")

// Or set all metadata at once
doc.SetMetadata(pdf.Metadata{
    Title:    "My Document",
    Author:   "John Doe",
    Subject:  "Example PDF",
    Keywords: "pdf, example, waffle",
    Creator:  "My Application",
})
```

## Text

### Basic Text

```go
doc.AddPage()

// Write text at current position
doc.Text("Hello, World!")

// Move to next line
doc.Ln()

// Write and move to next line
doc.WriteLine("This is a line of text.")

// Formatted text
doc.WriteLinef("Page %d of %d", doc.Page(), doc.PageCount())
```

### Font and Size

```go
// Set font and size
doc.SetFont("Helvetica", 12)
doc.SetFont("Times-Roman", 14)
doc.SetFont("Courier", 10)

// Just change size
doc.SetFontSize(16)

// Fluent style
doc.Font("Helvetica").Size(12).Text("Hello")

// Bold, Italic, Regular
doc.Bold().Text("Bold text")
doc.Italic().Text("Italic text")
doc.Regular().Text("Regular text")
```

### Available Fonts

- Helvetica, Helvetica-Bold, Helvetica-Oblique, Helvetica-BoldOblique
- Times-Roman, Times-Bold, Times-Italic, Times-BoldItalic
- Courier, Courier-Bold, Courier-Oblique, Courier-BoldOblique
- Symbol, ZapfDingbats

### Text Alignment

```go
doc.Text("Left aligned (default)")
doc.CenterText("Centered text")
doc.RightText("Right aligned text")
```

### Headings

```go
doc.Title("Document Title")         // Large, bold, centered
doc.Heading("Section Heading")      // Bold, larger
doc.Subheading("Subsection")        // Bold, medium
```

### Paragraphs

```go
// Automatic word wrapping within margins
doc.Paragraph("This is a long paragraph of text that will automatically " +
    "wrap to fit within the page margins. It handles word breaking " +
    "and maintains proper line spacing.")
```

### Line Height

```go
doc.SetLineHeight(1.5)  // 1.5x font size spacing
```

## Positioning

```go
// Set absolute position
doc.SetPos(100, 200)

// Move relative to content area (top-left of margins)
doc.MoveTo(0, 0)   // Top-left of content area
doc.MoveTo(100, 50) // 100 points right, 50 points down

// Get current position
x, y := doc.GetPos()

// Text at specific position
doc.TextAt(100, 200, "Text at position")
```

## Drawing

### Colors

```go
// Predefined colors
pdf.Black, pdf.White, pdf.Red, pdf.Green, pdf.Blue
pdf.Gray, pdf.Yellow, pdf.Cyan, pdf.Magenta

// RGB (0-255)
color := pdf.RGB(255, 128, 0)

// Hex
color := pdf.Hex("#FF8000")
color := pdf.Hex("FF8000")

// Set colors
doc.SetStrokeColor(pdf.Red)    // Line color
doc.SetFillColor(pdf.Blue)     // Fill color
```

### Lines

```go
// Line between two points
doc.Line(x1, y1, x2, y2)

// Horizontal line across content area
doc.HLine()
doc.HLineAt(200)  // At specific y position

// Line width
doc.SetLineWidth(2)

// Dashed lines
doc.SetDash([]float64{5, 3}, 0)  // 5pt dash, 3pt gap
doc.ClearDash()                   // Solid lines
```

### Rectangles

```go
// Stroke only
doc.Rect(x, y, width, height)

// Filled
doc.RectFilled(x, y, width, height)

// Filled with stroke
doc.RectFilledStroke(x, y, width, height)
```

### Circles and Ellipses

```go
// Circle
doc.Circle(centerX, centerY, radius)
doc.CircleFilled(centerX, centerY, radius)

// Ellipse
doc.Ellipse(centerX, centerY, radiusX, radiusY)
doc.EllipseFilled(centerX, centerY, radiusX, radiusY)
```

### Polygons

```go
points := []struct{ X, Y float64 }{
    {100, 100},
    {150, 50},
    {200, 100},
    {175, 150},
    {125, 150},
}
doc.Polygon(points)
doc.PolygonFilled(points)
```

### Transformations

```go
doc.SaveState()       // Save current graphics state
doc.Translate(50, 50) // Move origin
doc.Scale(2, 2)       // Scale
doc.Rotate(45)        // Rotate (degrees)
// ... draw something ...
doc.RestoreState()    // Restore state
```

## Images

```go
// From file
err := doc.ImageFromFile(x, y, width, height, "photo.jpg")

// From image.Image
doc.Image(x, y, width, height, img)

// From bytes
err := doc.ImageFromBytes(x, y, width, height, imageData)

// From reader
err := doc.ImageFromReader(x, y, width, height, reader)

// From base64
err := doc.ImageFromBase64(x, y, width, height, base64String)
```

Supported formats: JPEG, PNG

## Tables

### Basic Table

```go
// Auto-sized columns
doc.SimpleTable(
    []string{"Name", "Age", "City"},
    [][]string{
        {"Alice", "30", "New York"},
        {"Bob", "25", "Los Angeles"},
        {"Carol", "35", "Chicago"},
    },
)
```

### Custom Table

```go
table := doc.NewTable([]float64{100, 60, 150})  // Column widths

table.Header("Product", "Price", "Description")
table.Row("Widget", "$9.99", "A useful widget")
table.Row("Gadget", "$19.99", "An amazing gadget")
table.Row("Gizmo", "$14.99", "A wonderful gizmo")

table.Draw()
```

### Table Styling

```go
table := doc.NewTableAuto(3)  // 3 equal columns

table.SetCellPadding(8)
table.SetBorder(1, pdf.Black)
table.SetHeaderStyle(pdf.RGB(50, 50, 150), pdf.White)  // Blue header, white text
table.SetAlternateRowColor(pdf.RGB(240, 240, 240))     // Light gray alternating
table.SetFont("Helvetica", 10)
table.SetColumnAlign(1, pdf.AlignRight)  // Right-align column 1

table.Header("Item", "Amount", "Notes")
table.Row("Consulting", "$5,000", "10 hours")
table.Row("Development", "$15,000", "Sprint 1")

table.Draw()
```

### Key-Value Table

```go
doc.KeyValueTable([]struct{ Key, Value string }{
    {"Name", "John Doe"},
    {"Email", "john@example.com"},
    {"Phone", "(555) 123-4567"},
})
```

## Lists

```go
// Bulleted list
doc.List([]string{
    "First item",
    "Second item",
    "Third item",
})

// Numbered list
doc.NumberedList([]string{
    "Step one",
    "Step two",
    "Step three",
})
```

## Multiple Pages

```go
doc.AddPage()
doc.Title("First Page")
doc.Paragraph("Content on page 1...")

doc.AddPage()
doc.Title("Second Page")
doc.Paragraph("Content on page 2...")

// Page numbers
fmt.Printf("Current: %d, Total: %d\n", doc.Page(), doc.PageCount())
```

Automatic page breaks occur when:
- Text reaches the bottom margin
- Tables detect insufficient space for a row

## Saving

```go
// Save to file
err := doc.Save("output.pdf")

// Write to any io.Writer
err := doc.Write(w)

// Get as bytes
data, err := doc.Bytes()
```

## Complete Example

```go
package main

import "waffle/pdf"

func main() {
    doc := pdf.New()
    doc.SetTitle("Invoice #12345")
    doc.SetAuthor("ACME Corp")
    doc.SetMargins(pdf.Inches(1), pdf.Inches(1), pdf.Inches(1), pdf.Inches(1))

    doc.AddPage()

    // Header
    doc.Title("INVOICE")
    doc.Ln()

    // Company info
    doc.Bold().Text("ACME Corporation")
    doc.Regular().Ln()
    doc.WriteLine("123 Business Street")
    doc.WriteLine("New York, NY 10001")
    doc.Ln()

    // Invoice details
    doc.KeyValueTable([]struct{ Key, Value string }{
        {"Invoice #", "12345"},
        {"Date", "January 15, 2024"},
        {"Due Date", "February 15, 2024"},
    })
    doc.Ln()

    // Line items
    table := doc.NewTable([]float64{250, 60, 80, 80})
    table.SetColumnAlign(1, pdf.AlignRight)
    table.SetColumnAlign(2, pdf.AlignRight)
    table.SetColumnAlign(3, pdf.AlignRight)

    table.Header("Description", "Qty", "Price", "Total")
    table.Row("Consulting Services", "10", "$150.00", "$1,500.00")
    table.Row("Software License", "1", "$500.00", "$500.00")
    table.Row("Support (Annual)", "1", "$200.00", "$200.00")
    table.Draw()

    doc.Ln()

    // Total
    doc.RightText("Total: $2,200.00")
    doc.Ln()
    doc.Bold().RightText("Amount Due: $2,200.00")

    doc.Save("invoice.pdf")
}
```

## Limitations

This is a pure Go implementation with the following limitations compared to full PDF libraries:

- **Fonts**: Only the 14 standard PDF fonts (no TrueType/OpenType embedding)
- **Images**: JPEG and PNG only
- **No HTML**: Cannot convert HTML to PDF
- **No CSS**: Manual positioning and styling
- **Text Metrics**: Approximate character widths (may affect justification)

For complex layouts or HTML-to-PDF conversion, consider using Chromium/wkhtmltopdf-based solutions.
