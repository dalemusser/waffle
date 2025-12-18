// export/csv.go
package export

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Common errors.
var (
	ErrNoData       = errors.New("export: no data to export")
	ErrInvalidData  = errors.New("export: data must be a slice")
	ErrEmptyHeaders = errors.New("export: headers cannot be empty")
)

// CSV represents a CSV exporter.
type CSV struct {
	headers   []string
	rows      [][]string
	delimiter rune
	useCRLF   bool
}

// NewCSV creates a new CSV exporter.
func NewCSV() *CSV {
	return &CSV{
		delimiter: ',',
		useCRLF:   true,
	}
}

// Delimiter sets the field delimiter (default: comma).
func (c *CSV) Delimiter(d rune) *CSV {
	c.delimiter = d
	return c
}

// TabDelimited sets tab as the delimiter.
func (c *CSV) TabDelimited() *CSV {
	c.delimiter = '\t'
	return c
}

// UseLF uses LF line endings instead of CRLF.
func (c *CSV) UseLF() *CSV {
	c.useCRLF = false
	return c
}

// Headers sets the column headers.
func (c *CSV) Headers(headers ...string) *CSV {
	c.headers = headers
	return c
}

// Row adds a single row.
func (c *CSV) Row(values ...string) *CSV {
	c.rows = append(c.rows, values)
	return c
}

// Rows adds multiple rows.
func (c *CSV) Rows(rows [][]string) *CSV {
	c.rows = append(c.rows, rows...)
	return c
}

// From populates the CSV from a slice of structs or maps.
func (c *CSV) From(data any) *CSV {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return c
	}

	if v.Len() == 0 {
		return c
	}

	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	switch elem.Kind() {
	case reflect.Struct:
		c.fromStructs(v)
	case reflect.Map:
		c.fromMaps(v)
	}

	return c
}

// fromStructs converts a slice of structs to CSV.
func (c *CSV) fromStructs(v reflect.Value) {
	if v.Len() == 0 {
		return
	}

	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	t := elem.Type()

	// Build headers from struct fields if not set
	if len(c.headers) == 0 {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			name := field.Name
			if tag := field.Tag.Get("csv"); tag != "" {
				if tag == "-" {
					continue
				}
				name = strings.Split(tag, ",")[0]
			} else if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				name = strings.Split(tag, ",")[0]
			}
			c.headers = append(c.headers, name)
		}
	}

	// Build rows
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		var row []string
		for j := 0; j < t.NumField(); j++ {
			field := t.Field(j)
			if !field.IsExported() {
				continue
			}

			if tag := field.Tag.Get("csv"); tag == "-" {
				continue
			}

			fv := elem.Field(j)
			row = append(row, formatValue(fv))
		}
		c.rows = append(c.rows, row)
	}
}

// fromMaps converts a slice of maps to CSV.
func (c *CSV) fromMaps(v reflect.Value) {
	if v.Len() == 0 {
		return
	}

	// Collect all keys for headers if not set
	if len(c.headers) == 0 {
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
			c.headers = append(c.headers, key)
		}
	}

	// Build rows
	for i := 0; i < v.Len(); i++ {
		m := v.Index(i)
		if m.Kind() == reflect.Ptr {
			m = m.Elem()
		}

		row := make([]string, len(c.headers))
		for j, header := range c.headers {
			val := m.MapIndex(reflect.ValueOf(header))
			if val.IsValid() {
				row[j] = formatValue(val)
			}
		}
		c.rows = append(c.rows, row)
	}
}

// formatValue converts a reflect.Value to a string.
func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	// Handle interface
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	// Handle common types
	switch val := v.Interface().(type) {
	case time.Time:
		if val.IsZero() {
			return ""
		}
		return val.Format(time.RFC3339)
	case []byte:
		return string(val)
	case fmt.Stringer:
		return val.String()
	}

	// Handle basic types
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// Bytes returns the CSV as bytes.
func (c *CSV) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := c.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// String returns the CSV as a string.
func (c *CSV) String() (string, error) {
	b, err := c.Bytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Write writes the CSV to a writer.
func (c *CSV) Write(w io.Writer) error {
	writer := csv.NewWriter(w)
	writer.Comma = c.delimiter
	writer.UseCRLF = c.useCRLF

	if len(c.headers) > 0 {
		if err := writer.Write(c.headers); err != nil {
			return err
		}
	}

	for _, row := range c.rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

// Save saves the CSV to a file.
func (c *CSV) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return c.Write(f)
}

// ServeHTTP writes the CSV as an HTTP response.
func (c *CSV) ServeHTTP(w http.ResponseWriter, filename string) error {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Write(w)
}

// CSVReader represents a CSV reader with struct mapping.
type CSVReader struct {
	reader    *csv.Reader
	headers   []string
	delimiter rune
}

// ReadCSV creates a new CSV reader.
func ReadCSV(r io.Reader) *CSVReader {
	return &CSVReader{
		reader:    csv.NewReader(r),
		delimiter: ',',
	}
}

// ReadCSVFile opens and reads a CSV file.
func ReadCSVFile(filename string) (*CSVReader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// Note: caller should handle file closing if needed
	return ReadCSV(f), nil
}

// Delimiter sets the field delimiter.
func (r *CSVReader) Delimiter(d rune) *CSVReader {
	r.reader.Comma = d
	return r
}

// TabDelimited sets tab as the delimiter.
func (r *CSVReader) TabDelimited() *CSVReader {
	r.reader.Comma = '\t'
	return r
}

// ReadAll reads all records.
func (r *CSVReader) ReadAll() ([][]string, error) {
	return r.reader.ReadAll()
}

// ReadAllWithHeaders reads all records, treating the first row as headers.
func (r *CSVReader) ReadAllWithHeaders() ([]map[string]string, error) {
	records, err := r.reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	headers := records[0]
	var result []map[string]string

	for i := 1; i < len(records); i++ {
		row := make(map[string]string)
		for j, header := range headers {
			if j < len(records[i]) {
				row[header] = records[i][j]
			}
		}
		result = append(result, row)
	}

	return result, nil
}

// Into reads CSV into a slice of structs.
func (r *CSVReader) Into(dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return errors.New("export: dest must be a pointer to slice")
	}

	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return errors.New("export: dest must be a pointer to slice")
	}

	elemType := v.Type().Elem()
	isPtr := false
	if elemType.Kind() == reflect.Ptr {
		isPtr = true
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		return errors.New("export: dest must be a slice of structs")
	}

	records, err := r.reader.ReadAll()
	if err != nil {
		return err
	}

	if len(records) < 2 {
		return nil
	}

	headers := records[0]

	// Map headers to field indices
	fieldMap := make(map[string]int)
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		if tag := field.Tag.Get("csv"); tag != "" && tag != "-" {
			name = strings.Split(tag, ",")[0]
		} else if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			name = strings.Split(tag, ",")[0]
		}
		fieldMap[strings.ToLower(name)] = i
	}

	// Parse records
	for i := 1; i < len(records); i++ {
		elem := reflect.New(elemType).Elem()

		for j, header := range headers {
			if j >= len(records[i]) {
				continue
			}

			fieldIdx, ok := fieldMap[strings.ToLower(header)]
			if !ok {
				continue
			}

			field := elem.Field(fieldIdx)
			if err := setFieldValue(field, records[i][j]); err != nil {
				continue // Skip fields that can't be set
			}
		}

		if isPtr {
			v.Set(reflect.Append(v, elem.Addr()))
		} else {
			v.Set(reflect.Append(v, elem))
		}
	}

	return nil
}

// setFieldValue sets a struct field from a string value.
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return errors.New("cannot set field")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			return nil
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			return nil
		}
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		if value == "" {
			return nil
		}
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(n)
	case reflect.Bool:
		if value == "" {
			return nil
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) {
			if value == "" {
				return nil
			}
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				// Try other common formats
				for _, format := range []string{
					"2006-01-02",
					"2006-01-02 15:04:05",
					"01/02/2006",
					"02/01/2006",
				} {
					t, err = time.Parse(format, value)
					if err == nil {
						break
					}
				}
			}
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(t))
		}
	case reflect.Ptr:
		if value == "" {
			return nil
		}
		// Create new value and set it
		elem := reflect.New(field.Type().Elem())
		if err := setFieldValue(elem.Elem(), value); err != nil {
			return err
		}
		field.Set(elem)
	}

	return nil
}

// FromCSV is a convenience function to create a CSV from data.
func FromCSV(data any) *CSV {
	return NewCSV().From(data)
}
