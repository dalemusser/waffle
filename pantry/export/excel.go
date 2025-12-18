// export/excel.go
package export

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Excel represents an Excel workbook exporter.
type Excel struct {
	file       *excelize.File
	sheets     []*ExcelSheet
	activeSheet string
}

// ExcelSheet represents a sheet in an Excel workbook.
type ExcelSheet struct {
	excel     *Excel
	name      string
	headers   []string
	rows      [][]any
	colWidths map[int]float64
	headerStyle int
	currentRow int
}

// CellStyle represents cell styling options.
type CellStyle struct {
	Bold       bool
	Italic     bool
	FontSize   float64
	FontColor  string
	FillColor  string
	Alignment  string // left, center, right
	NumFormat  string
	Border     bool
}

// NewExcel creates a new Excel workbook.
func NewExcel() *Excel {
	return &Excel{
		file: excelize.NewFile(),
	}
}

// Sheet creates or gets a sheet by name.
func (e *Excel) Sheet(name string) *ExcelSheet {
	// Check if sheet already exists
	for _, s := range e.sheets {
		if s.name == name {
			return s
		}
	}

	// Create new sheet
	idx, err := e.file.NewSheet(name)
	if err != nil {
		// Sheet might already exist
		idx, _ = e.file.GetSheetIndex(name)
	}

	if e.activeSheet == "" {
		e.file.SetActiveSheet(idx)
		e.activeSheet = name
		// Delete default Sheet1 if we created a new one
		if name != "Sheet1" {
			e.file.DeleteSheet("Sheet1")
		}
	}

	sheet := &ExcelSheet{
		excel:      e,
		name:       name,
		colWidths:  make(map[int]float64),
		currentRow: 1,
	}
	e.sheets = append(e.sheets, sheet)

	return sheet
}

// Headers sets the column headers.
func (s *ExcelSheet) Headers(headers ...string) *ExcelSheet {
	s.headers = headers
	return s
}

// Row adds a single row.
func (s *ExcelSheet) Row(values ...any) *ExcelSheet {
	s.rows = append(s.rows, values)
	return s
}

// Rows adds multiple rows.
func (s *ExcelSheet) Rows(rows [][]any) *ExcelSheet {
	s.rows = append(s.rows, rows...)
	return s
}

// From populates the sheet from a slice of structs or maps.
func (s *ExcelSheet) From(data any) *ExcelSheet {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return s
	}

	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	switch elem.Kind() {
	case reflect.Struct:
		s.fromStructs(v)
	case reflect.Map:
		s.fromMaps(v)
	}

	return s
}

// fromStructs converts a slice of structs to rows.
func (s *ExcelSheet) fromStructs(v reflect.Value) {
	if v.Len() == 0 {
		return
	}

	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	t := elem.Type()

	// Build headers from struct fields if not set
	if len(s.headers) == 0 {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			name := field.Name
			if tag := field.Tag.Get("excel"); tag != "" {
				if tag == "-" {
					continue
				}
				name = strings.Split(tag, ",")[0]
			} else if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				name = strings.Split(tag, ",")[0]
			}
			s.headers = append(s.headers, name)
		}
	}

	// Build rows
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		var row []any
		for j := 0; j < t.NumField(); j++ {
			field := t.Field(j)
			if !field.IsExported() {
				continue
			}

			if tag := field.Tag.Get("excel"); tag == "-" {
				continue
			}

			fv := elem.Field(j)
			row = append(row, extractValue(fv))
		}
		s.rows = append(s.rows, row)
	}
}

// fromMaps converts a slice of maps to rows.
func (s *ExcelSheet) fromMaps(v reflect.Value) {
	if v.Len() == 0 {
		return
	}

	// Collect all keys for headers if not set
	if len(s.headers) == 0 {
		keySet := make(map[string]bool)
		for i := 0; i < v.Len(); i++ {
			m := v.Index(i)
			if m.Kind() == reflect.Ptr {
				m = m.Elem()
			}
			for _, key := range m.MapKeys() {
				keySet[fmt.Sprintf("%v", key.Interface())] = true
			}
		}
		for key := range keySet {
			s.headers = append(s.headers, key)
		}
	}

	// Build rows
	for i := 0; i < v.Len(); i++ {
		m := v.Index(i)
		if m.Kind() == reflect.Ptr {
			m = m.Elem()
		}

		row := make([]any, len(s.headers))
		for j, header := range s.headers {
			val := m.MapIndex(reflect.ValueOf(header))
			if val.IsValid() {
				row[j] = extractValue(val)
			}
		}
		s.rows = append(s.rows, row)
	}
}

// extractValue extracts the actual value from a reflect.Value.
func extractValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	return v.Interface()
}

// ColWidth sets the width of a column (1-indexed).
func (s *ExcelSheet) ColWidth(col int, width float64) *ExcelSheet {
	s.colWidths[col] = width
	return s
}

// AutoWidth enables auto-width for all columns based on content.
func (s *ExcelSheet) AutoWidth() *ExcelSheet {
	// Calculate max width for each column
	for i, header := range s.headers {
		width := float64(len(header)) * 1.2
		if width < 10 {
			width = 10
		}
		s.colWidths[i+1] = width
	}

	for _, row := range s.rows {
		for i, val := range row {
			str := fmt.Sprintf("%v", val)
			width := float64(len(str)) * 1.1
			if existing, ok := s.colWidths[i+1]; !ok || width > existing {
				if width > 50 {
					width = 50
				}
				s.colWidths[i+1] = width
			}
		}
	}

	return s
}

// Build writes the sheet data to the Excel file.
func (s *ExcelSheet) Build() *Excel {
	file := s.excel.file
	row := 1

	// Write headers with styling
	if len(s.headers) > 0 {
		for i, header := range s.headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			file.SetCellValue(s.name, cell, header)
		}

		// Apply header style
		style, _ := file.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{"#E0E0E0"},
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "bottom", Color: "#000000", Style: 1},
			},
		})

		startCell, _ := excelize.CoordinatesToCellName(1, row)
		endCell, _ := excelize.CoordinatesToCellName(len(s.headers), row)
		file.SetCellStyle(s.name, startCell, endCell, style)

		row++
	}

	// Write data rows
	for _, rowData := range s.rows {
		for i, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			file.SetCellValue(s.name, cell, val)
		}
		row++
	}

	// Set column widths
	for col, width := range s.colWidths {
		colName, _ := excelize.ColumnNumberToName(col)
		file.SetColWidth(s.name, colName, colName, width)
	}

	return s.excel
}

// Sheet returns to the Excel builder to add more sheets.
func (s *ExcelSheet) Sheet(name string) *ExcelSheet {
	s.Build()
	return s.excel.Sheet(name)
}

// Bytes returns the Excel file as bytes.
func (e *Excel) Bytes() ([]byte, error) {
	// Build any unbuilt sheets
	for _, s := range e.sheets {
		s.Build()
	}

	buf, err := e.file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write writes the Excel file to a writer.
func (e *Excel) Write(w io.Writer) error {
	// Build any unbuilt sheets
	for _, s := range e.sheets {
		s.Build()
	}

	return e.file.Write(w)
}

// Save saves the Excel file.
func (e *Excel) Save(filename string) error {
	// Build any unbuilt sheets
	for _, s := range e.sheets {
		s.Build()
	}

	return e.file.SaveAs(filename)
}

// ServeHTTP writes the Excel file as an HTTP response.
func (e *Excel) ServeHTTP(w http.ResponseWriter, filename string) error {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return e.Write(w)
}

// Close closes the Excel file and releases resources.
func (e *Excel) Close() error {
	return e.file.Close()
}

// AddFormula adds a formula to a cell.
func (s *ExcelSheet) AddFormula(col, row int, formula string) *ExcelSheet {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	s.excel.file.SetCellFormula(s.name, cell, formula)
	return s
}

// MergeCells merges a range of cells.
func (s *ExcelSheet) MergeCells(startCol, startRow, endCol, endRow int) *ExcelSheet {
	startCell, _ := excelize.CoordinatesToCellName(startCol, startRow)
	endCell, _ := excelize.CoordinatesToCellName(endCol, endRow)
	s.excel.file.MergeCell(s.name, startCell, endCell)
	return s
}

// SetCellStyle sets style for a range of cells.
func (s *ExcelSheet) SetCellStyle(startCol, startRow, endCol, endRow int, style CellStyle) *ExcelSheet {
	excelStyle := &excelize.Style{}

	if style.Bold || style.Italic || style.FontSize > 0 || style.FontColor != "" {
		excelStyle.Font = &excelize.Font{
			Bold:   style.Bold,
			Italic: style.Italic,
			Size:   style.FontSize,
			Color:  style.FontColor,
		}
	}

	if style.FillColor != "" {
		excelStyle.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{style.FillColor},
			Pattern: 1,
		}
	}

	if style.Alignment != "" {
		excelStyle.Alignment = &excelize.Alignment{
			Horizontal: style.Alignment,
		}
	}

	if style.NumFormat != "" {
		excelStyle.CustomNumFmt = &style.NumFormat
	}

	if style.Border {
		excelStyle.Border = []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		}
	}

	styleID, _ := s.excel.file.NewStyle(excelStyle)
	startCell, _ := excelize.CoordinatesToCellName(startCol, startRow)
	endCell, _ := excelize.CoordinatesToCellName(endCol, endRow)
	s.excel.file.SetCellStyle(s.name, startCell, endCell, styleID)

	return s
}

// SetDateFormat sets date format for a column.
func (s *ExcelSheet) SetDateFormat(col int, format string) *ExcelSheet {
	if format == "" {
		format = "yyyy-mm-dd"
	}

	style, _ := s.excel.file.NewStyle(&excelize.Style{
		CustomNumFmt: &format,
	})

	colName, _ := excelize.ColumnNumberToName(col)

	// Apply to all data rows
	startRow := 2 // Skip header
	if len(s.headers) == 0 {
		startRow = 1
	}
	endRow := startRow + len(s.rows) - 1

	startCell, _ := excelize.CoordinatesToCellName(col, startRow)
	endCell, _ := excelize.CoordinatesToCellName(col, endRow)
	s.excel.file.SetCellStyle(s.name, startCell, endCell, style)

	_ = colName // Avoid unused warning

	return s
}

// SetNumberFormat sets number format for a column.
func (s *ExcelSheet) SetNumberFormat(col int, format string) *ExcelSheet {
	style, _ := s.excel.file.NewStyle(&excelize.Style{
		CustomNumFmt: &format,
	})

	startRow := 2
	if len(s.headers) == 0 {
		startRow = 1
	}
	endRow := startRow + len(s.rows) - 1

	startCell, _ := excelize.CoordinatesToCellName(col, startRow)
	endCell, _ := excelize.CoordinatesToCellName(col, endRow)
	s.excel.file.SetCellStyle(s.name, startCell, endCell, style)

	return s
}

// FreezeHeader freezes the header row.
func (s *ExcelSheet) FreezeHeader() *ExcelSheet {
	s.excel.file.SetPanes(s.name, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})
	return s
}

// AddTable formats the data as an Excel table.
func (s *ExcelSheet) AddTable() *ExcelSheet {
	if len(s.headers) == 0 || len(s.rows) == 0 {
		return s
	}

	startCell, _ := excelize.CoordinatesToCellName(1, 1)
	endCell, _ := excelize.CoordinatesToCellName(len(s.headers), len(s.rows)+1)

	s.excel.file.AddTable(s.name, &excelize.Table{
		Range:     startCell + ":" + endCell,
		Name:      strings.ReplaceAll(s.name, " ", "_") + "_Table",
		StyleName: "TableStyleMedium2",
	})

	return s
}

// FromExcel is a convenience function to create an Excel file from data.
func FromExcel(data any) *ExcelSheet {
	return NewExcel().Sheet("Sheet1").From(data)
}

// Common number formats
const (
	NumberFormatGeneral    = "General"
	NumberFormatInteger    = "0"
	NumberFormatDecimal2   = "0.00"
	NumberFormatPercent    = "0%"
	NumberFormatPercent2   = "0.00%"
	NumberFormatCurrency   = `"$"#,##0.00`
	NumberFormatDate       = "yyyy-mm-dd"
	NumberFormatDateTime   = "yyyy-mm-dd hh:mm:ss"
	NumberFormatTime       = "hh:mm:ss"
	NumberFormatAccounting = `_("$"* #,##0.00_)`
)

// TimeValue wraps a time.Time with a specific format for Excel.
type TimeValue struct {
	Time   time.Time
	Format string
}

// ExcelTime creates a TimeValue for proper Excel date handling.
func ExcelTime(t time.Time) TimeValue {
	return TimeValue{Time: t, Format: NumberFormatDate}
}

// ExcelDateTime creates a TimeValue with date and time format.
func ExcelDateTime(t time.Time) TimeValue {
	return TimeValue{Time: t, Format: NumberFormatDateTime}
}
