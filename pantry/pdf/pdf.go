// pdf/pdf.go
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

// PageSize represents standard page sizes.
type PageSize struct {
	Width  float64 // Points (1/72 inch)
	Height float64
}

// Standard page sizes in points (1 point = 1/72 inch).
var (
	Letter    = PageSize{612, 792}      // 8.5 x 11 inches
	Legal     = PageSize{612, 1008}     // 8.5 x 14 inches
	Tabloid   = PageSize{792, 1224}     // 11 x 17 inches
	A3        = PageSize{841.89, 1190.55}
	A4        = PageSize{595.28, 841.89}
	A5        = PageSize{419.53, 595.28}
	B4        = PageSize{708.66, 1000.63}
	B5        = PageSize{498.90, 708.66}
)

// Orientation represents page orientation.
type Orientation int

const (
	Portrait Orientation = iota
	Landscape
)

// Unit conversion constants.
const (
	PtPerInch = 72.0
	PtPerMm   = 72.0 / 25.4
	PtPerCm   = 72.0 / 2.54
)

// Inches converts inches to points.
func Inches(n float64) float64 { return n * PtPerInch }

// Mm converts millimeters to points.
func Mm(n float64) float64 { return n * PtPerMm }

// Cm converts centimeters to points.
func Cm(n float64) float64 { return n * PtPerCm }

// Document represents a PDF document.
type Document struct {
	pages       []*Page
	currentPage *Page
	fonts       map[string]*font
	images      map[string]*pdfImage
	metadata    Metadata
	pageSize    PageSize
	orientation Orientation
	margins     Margins

	// Object tracking
	objects    []*pdfObject
	nextObjNum int
	catalog    *pdfObject
	pageTree   *pdfObject
	fontObjs   map[string]*pdfObject
	imageObjs  map[string]*pdfObject

	// Current state
	x, y       float64
	fontName   string
	fontSize   float64
	lineHeight float64
}

// Metadata contains document metadata.
type Metadata struct {
	Title        string
	Author       string
	Subject      string
	Keywords     string
	Creator      string
	Producer     string
	CreationDate time.Time
	ModDate      time.Time
}

// Margins represents page margins.
type Margins struct {
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

// Page represents a single page in the document.
type Page struct {
	doc         *Document
	size        PageSize
	orientation Orientation
	content     bytes.Buffer
	resources   map[string]bool
}

// pdfObject represents a PDF object.
type pdfObject struct {
	num      int
	gen      int
	data     []byte
	stream   []byte
	isStream bool
}

// font represents a font in the document.
type font struct {
	name     string
	baseFont string
	subtype  string
}

// pdfImage represents an image in the document.
type pdfImage struct {
	name   string
	width  int
	height int
	data   []byte
}

// New creates a new PDF document.
func New() *Document {
	doc := &Document{
		fonts:       make(map[string]*font),
		images:      make(map[string]*pdfImage),
		fontObjs:    make(map[string]*pdfObject),
		imageObjs:   make(map[string]*pdfObject),
		pageSize:    Letter,
		orientation: Portrait,
		margins:     Margins{Top: 72, Right: 72, Bottom: 72, Left: 72}, // 1 inch margins
		nextObjNum:  1,
		fontSize:    12,
		lineHeight:  1.2,
		metadata: Metadata{
			Producer:     "waffle/pdf",
			CreationDate: time.Now(),
			ModDate:      time.Now(),
		},
	}

	// Add standard fonts
	doc.addStandardFonts()

	// Set default font
	doc.fontName = "Helvetica"

	return doc
}

// addStandardFonts adds the 14 standard PDF fonts.
func (d *Document) addStandardFonts() {
	standardFonts := []string{
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Symbol", "ZapfDingbats",
	}

	for _, name := range standardFonts {
		d.fonts[name] = &font{
			name:     name,
			baseFont: name,
			subtype:  "Type1",
		}
	}
}

// SetPageSize sets the default page size.
func (d *Document) SetPageSize(size PageSize) *Document {
	d.pageSize = size
	return d
}

// SetOrientation sets the default page orientation.
func (d *Document) SetOrientation(o Orientation) *Document {
	d.orientation = o
	return d
}

// SetMargins sets the page margins.
func (d *Document) SetMargins(top, right, bottom, left float64) *Document {
	d.margins = Margins{Top: top, Right: right, Bottom: bottom, Left: left}
	return d
}

// SetMetadata sets document metadata.
func (d *Document) SetMetadata(m Metadata) *Document {
	if m.Producer == "" {
		m.Producer = "waffle/pdf"
	}
	if m.CreationDate.IsZero() {
		m.CreationDate = time.Now()
	}
	if m.ModDate.IsZero() {
		m.ModDate = time.Now()
	}
	d.metadata = m
	return d
}

// SetTitle sets the document title.
func (d *Document) SetTitle(title string) *Document {
	d.metadata.Title = title
	return d
}

// SetAuthor sets the document author.
func (d *Document) SetAuthor(author string) *Document {
	d.metadata.Author = author
	return d
}

// AddPage adds a new page to the document.
func (d *Document) AddPage() *Document {
	size := d.pageSize
	if d.orientation == Landscape {
		size = PageSize{Width: size.Height, Height: size.Width}
	}

	page := &Page{
		doc:         d,
		size:        size,
		orientation: d.orientation,
		resources:   make(map[string]bool),
	}

	d.pages = append(d.pages, page)
	d.currentPage = page

	// Reset position to top-left within margins
	d.x = d.margins.Left
	d.y = size.Height - d.margins.Top

	return d
}

// Page returns the current page number (1-indexed).
func (d *Document) Page() int {
	return len(d.pages)
}

// PageCount returns the total number of pages.
func (d *Document) PageCount() int {
	return len(d.pages)
}

// ensurePage ensures there's a current page.
func (d *Document) ensurePage() {
	if d.currentPage == nil {
		d.AddPage()
	}
}

// contentWidth returns the usable content width.
func (d *Document) contentWidth() float64 {
	if d.currentPage == nil {
		return d.pageSize.Width - d.margins.Left - d.margins.Right
	}
	return d.currentPage.size.Width - d.margins.Left - d.margins.Right
}

// contentHeight returns the usable content height.
func (d *Document) contentHeight() float64 {
	if d.currentPage == nil {
		return d.pageSize.Height - d.margins.Top - d.margins.Bottom
	}
	return d.currentPage.size.Height - d.margins.Top - d.margins.Bottom
}

// SetPos sets the current position.
func (d *Document) SetPos(x, y float64) *Document {
	d.x = x
	d.y = y
	return d
}

// GetPos returns the current position.
func (d *Document) GetPos() (x, y float64) {
	return d.x, d.y
}

// MoveTo moves to a position relative to the top-left of the content area.
func (d *Document) MoveTo(x, y float64) *Document {
	d.ensurePage()
	d.x = d.margins.Left + x
	d.y = d.currentPage.size.Height - d.margins.Top - y
	return d
}

// Bytes renders the document and returns the PDF as bytes.
func (d *Document) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write renders the document to the given writer.
func (d *Document) Write(w io.Writer) error {
	if len(d.pages) == 0 {
		d.AddPage()
	}

	return d.render(w)
}

// Save saves the document to a file.
func (d *Document) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return d.Write(f)
}

// render generates the PDF output.
func (d *Document) render(w io.Writer) error {
	d.objects = nil
	d.nextObjNum = 1

	// Build object tree
	d.buildObjects()

	// Write PDF
	var buf bytes.Buffer

	// Header
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("%\xe2\xe3\xcf\xd3\n") // Binary marker

	// Objects
	offsets := make([]int, len(d.objects)+1)
	for _, obj := range d.objects {
		offsets[obj.num] = buf.Len()
		d.writeObject(&buf, obj)
	}

	// Cross-reference table
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", len(d.objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(d.objects); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}

	// Trailer
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root %d 0 R >>\n",
		len(d.objects)+1, d.catalog.num))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	_, err := w.Write(buf.Bytes())
	return err
}

// buildObjects builds the PDF object tree.
func (d *Document) buildObjects() {
	// Create font objects
	for name, f := range d.fonts {
		obj := d.newObject()
		obj.data = []byte(fmt.Sprintf("<< /Type /Font /Subtype /%s /BaseFont /%s >>",
			f.subtype, f.baseFont))
		d.fontObjs[name] = obj
	}

	// Create page objects
	pageObjs := make([]*pdfObject, len(d.pages))
	for i, page := range d.pages {
		// Content stream
		contentObj := d.newObject()
		contentObj.isStream = true
		contentObj.stream = page.content.Bytes()
		contentObj.data = []byte(fmt.Sprintf("<< /Length %d >>", len(contentObj.stream)))

		// Page object (will set parent later)
		pageObj := d.newObject()
		pageObjs[i] = pageObj

		// Build resources
		var fontRefs bytes.Buffer
		fontRefs.WriteString("<< ")
		for fontName := range page.resources {
			if fobj, ok := d.fontObjs[fontName]; ok {
				fontRefs.WriteString(fmt.Sprintf("/%s %d 0 R ", fontName, fobj.num))
			}
		}
		fontRefs.WriteString(">>")

		pageObj.data = []byte(fmt.Sprintf(
			"<< /Type /Page /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /Font %s >> >>",
			page.size.Width, page.size.Height, contentObj.num, fontRefs.String()))
	}

	// Pages object (parent of all pages)
	d.pageTree = d.newObject()
	var kids bytes.Buffer
	kids.WriteString("[")
	for i, po := range pageObjs {
		if i > 0 {
			kids.WriteString(" ")
		}
		kids.WriteString(fmt.Sprintf("%d 0 R", po.num))
	}
	kids.WriteString("]")
	d.pageTree.data = []byte(fmt.Sprintf("<< /Type /Pages /Kids %s /Count %d >>",
		kids.String(), len(pageObjs)))

	// Update page objects with parent reference
	for _, pageObj := range pageObjs {
		data := string(pageObj.data)
		data = data[:len(data)-2] + fmt.Sprintf(" /Parent %d 0 R >>", d.pageTree.num)
		pageObj.data = []byte(data)
	}

	// Catalog
	d.catalog = d.newObject()
	d.catalog.data = []byte(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", d.pageTree.num))
}

// newObject creates a new PDF object.
func (d *Document) newObject() *pdfObject {
	obj := &pdfObject{
		num: d.nextObjNum,
		gen: 0,
	}
	d.nextObjNum++
	d.objects = append(d.objects, obj)
	return obj
}

// writeObject writes a PDF object.
func (d *Document) writeObject(w *bytes.Buffer, obj *pdfObject) {
	w.WriteString(fmt.Sprintf("%d %d obj\n", obj.num, obj.gen))
	w.Write(obj.data)
	w.WriteString("\n")
	if obj.isStream {
		w.WriteString("stream\n")
		w.Write(obj.stream)
		w.WriteString("\nendstream\n")
	}
	w.WriteString("endobj\n")
}
