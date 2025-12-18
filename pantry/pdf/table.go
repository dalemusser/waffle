// pdf/table.go
package pdf

// Table represents a table in the document.
type Table struct {
	doc         *Document
	x, y        float64
	width       float64
	columns     []TableColumn
	rows        [][]string
	headerRow   []string
	cellPadding float64
	borderWidth float64
	borderColor Color
	headerBg    Color
	headerFg    Color
	altRowBg    *Color
	fontSize    float64
	fontName    string
}

// TableColumn represents a table column.
type TableColumn struct {
	Width float64
	Align TextAlign
}

// NewTable creates a new table at the current position.
func (d *Document) NewTable(columns []float64) *Table {
	d.ensurePage()

	cols := make([]TableColumn, len(columns))
	for i, w := range columns {
		cols[i] = TableColumn{Width: w, Align: AlignLeft}
	}

	totalWidth := 0.0
	for _, w := range columns {
		totalWidth += w
	}

	return &Table{
		doc:         d,
		x:           d.x,
		y:           d.y,
		width:       totalWidth,
		columns:     cols,
		cellPadding: 4,
		borderWidth: 0.5,
		borderColor: Black,
		headerBg:    RGB(220, 220, 220),
		headerFg:    Black,
		fontSize:    d.fontSize,
		fontName:    d.fontName,
	}
}

// NewTableAuto creates a table that auto-sizes columns to fill content width.
func (d *Document) NewTableAuto(numColumns int) *Table {
	d.ensurePage()

	colWidth := d.contentWidth() / float64(numColumns)
	columns := make([]float64, numColumns)
	for i := range columns {
		columns[i] = colWidth
	}

	return d.NewTable(columns)
}

// SetCellPadding sets the cell padding.
func (t *Table) SetCellPadding(padding float64) *Table {
	t.cellPadding = padding
	return t
}

// SetBorder sets the border style.
func (t *Table) SetBorder(width float64, color Color) *Table {
	t.borderWidth = width
	t.borderColor = color
	return t
}

// SetHeaderStyle sets the header row style.
func (t *Table) SetHeaderStyle(bg, fg Color) *Table {
	t.headerBg = bg
	t.headerFg = fg
	return t
}

// SetAlternateRowColor sets alternating row background color.
func (t *Table) SetAlternateRowColor(color Color) *Table {
	t.altRowBg = &color
	return t
}

// SetFont sets the table font.
func (t *Table) SetFont(name string, size float64) *Table {
	t.fontName = name
	t.fontSize = size
	return t
}

// SetColumnAlign sets the alignment for a column.
func (t *Table) SetColumnAlign(col int, align TextAlign) *Table {
	if col >= 0 && col < len(t.columns) {
		t.columns[col].Align = align
	}
	return t
}

// Header sets the header row.
func (t *Table) Header(cells ...string) *Table {
	t.headerRow = cells
	return t
}

// Row adds a data row.
func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// Rows adds multiple data rows.
func (t *Table) Rows(rows [][]string) *Table {
	t.rows = append(t.rows, rows...)
	return t
}

// Draw renders the table and returns the document.
func (t *Table) Draw() *Document {
	d := t.doc
	d.ensurePage()

	rowHeight := t.fontSize + t.cellPadding*2
	y := t.y

	// Save current font
	oldFont := d.fontName
	oldSize := d.fontSize

	d.fontName = t.fontName
	d.fontSize = t.fontSize

	// Draw header row
	if len(t.headerRow) > 0 {
		y = t.drawRow(y, t.headerRow, true, false)
	}

	// Draw data rows
	for i, row := range t.rows {
		isAlt := i%2 == 1 && t.altRowBg != nil

		// Check for page break
		if y-rowHeight < d.margins.Bottom {
			d.AddPage()
			y = d.y

			// Redraw header on new page
			if len(t.headerRow) > 0 {
				y = t.drawRow(y, t.headerRow, true, false)
			}
		}

		y = t.drawRow(y, row, false, isAlt)
	}

	// Update document position
	d.y = y
	d.x = t.x

	// Restore font
	d.fontName = oldFont
	d.fontSize = oldSize

	return d
}

// drawRow draws a single row and returns the new y position.
func (t *Table) drawRow(y float64, cells []string, isHeader, isAlt bool) float64 {
	d := t.doc
	rowHeight := t.fontSize + t.cellPadding*2

	x := t.x

	// Draw background
	if isHeader {
		d.SetFillColor(t.headerBg)
		d.RectFilled(x, d.currentPage.size.Height-y, t.width, rowHeight)
	} else if isAlt && t.altRowBg != nil {
		d.SetFillColor(*t.altRowBg)
		d.RectFilled(x, d.currentPage.size.Height-y, t.width, rowHeight)
	}

	// Draw cell borders and text
	d.SetStrokeColor(t.borderColor)
	d.SetLineWidth(t.borderWidth)

	for i, col := range t.columns {
		// Draw cell border
		d.Rect(x, d.currentPage.size.Height-y, col.Width, rowHeight)

		// Draw cell text
		text := ""
		if i < len(cells) {
			text = cells[i]
		}

		textY := y - t.cellPadding - t.fontSize*0.8
		textX := x + t.cellPadding

		// Handle alignment
		textWidth := d.measureText(text)
		switch col.Align {
		case AlignCenter:
			textX = x + (col.Width-textWidth)/2
		case AlignRight:
			textX = x + col.Width - t.cellPadding - textWidth
		}

		// Set text color
		if isHeader {
			d.SetFillColor(t.headerFg)
		} else {
			d.SetFillColor(Black)
		}

		// Draw text
		if isHeader {
			oldFont := d.fontName
			d.Bold()
			d.writeText(text, textX, textY)
			d.fontName = oldFont
		} else {
			d.writeText(text, textX, textY)
		}

		x += col.Width
	}

	return y - rowHeight
}

// SimpleTable is a convenience function to draw a simple table.
func (d *Document) SimpleTable(headers []string, rows [][]string) *Document {
	t := d.NewTableAuto(len(headers))
	t.Header(headers...)
	t.Rows(rows)
	return t.Draw()
}

// DataTable creates a table from a map of column names to values.
func (d *Document) DataTable(data map[string]string) *Document {
	// Calculate column widths
	colWidth := d.contentWidth() / 2

	t := d.NewTable([]float64{colWidth, colWidth})
	t.SetColumnAlign(0, AlignLeft)
	t.SetColumnAlign(1, AlignLeft)
	t.Header("Field", "Value")

	for key, value := range data {
		t.Row(key, value)
	}

	return t.Draw()
}

// KeyValueTable creates a simple two-column key-value table without header.
func (d *Document) KeyValueTable(pairs []struct{ Key, Value string }) *Document {
	colWidth := d.contentWidth() / 2

	t := d.NewTable([]float64{colWidth, colWidth})
	t.SetColumnAlign(0, AlignRight)
	t.SetColumnAlign(1, AlignLeft)

	for _, pair := range pairs {
		t.Row(pair.Key+":", pair.Value)
	}

	return t.Draw()
}

// List writes a bulleted list.
func (d *Document) List(items []string) *Document {
	d.ensurePage()

	indent := d.fontSize * 1.5

	for _, item := range items {
		// Draw bullet
		d.Text("•")

		// Draw text with indent
		oldX := d.x
		d.x += indent
		d.Text(item)
		d.x = oldX
		d.Ln()
	}

	return d
}

// NumberedList writes a numbered list.
func (d *Document) NumberedList(items []string) *Document {
	d.ensurePage()

	indent := d.fontSize * 2

	for i, item := range items {
		// Draw number
		d.Writef("%d.", i+1)

		// Draw text with indent
		oldX := d.x
		d.x = d.margins.Left + indent
		d.Text(item)
		d.x = oldX
		d.Ln()
	}

	return d
}
