# Export - CSV and Excel Data Export

The `export` package provides data export capabilities for CSV and Excel formats, with support for struct and map data sources.

## Features

- **CSV Export**: Generate CSV from structs, maps, or manual rows
- **CSV Import**: Parse CSV into structs or maps
- **Excel Export**: Generate .xlsx files with multiple sheets, styling, and formulas
- **HTTP Helpers**: Serve files directly as downloads
- **Struct Tags**: Use `csv` or `excel` tags to customize column names

## Installation

```go
import "waffle/export"
```

## CSV Export

### From Structs

```go
type User struct {
    ID    int    `csv:"id"`
    Name  string `csv:"name"`
    Email string `csv:"email"`
}

users := []User{
    {ID: 1, Name: "Alice", Email: "alice@example.com"},
    {ID: 2, Name: "Bob", Email: "bob@example.com"},
}

// Save to file
export.FromCSV(users).Save("users.csv")

// Get as string
csv, _ := export.FromCSV(users).String()

// Get as bytes
data, _ := export.FromCSV(users).Bytes()
```

### From Maps

```go
data := []map[string]string{
    {"name": "Alice", "age": "30"},
    {"name": "Bob", "age": "25"},
}

export.FromCSV(data).Save("data.csv")
```

### Manual Rows

```go
csv := export.NewCSV().
    Headers("Product", "Price", "Quantity").
    Row("Widget", "9.99", "100").
    Row("Gadget", "19.99", "50").
    Row("Gizmo", "14.99", "75")

csv.Save("products.csv")
```

### Options

```go
csv := export.NewCSV().
    Delimiter(';').           // Use semicolon
    TabDelimited().           // Or use tab
    UseLF().                  // Use LF instead of CRLF
    Headers("A", "B", "C").
    Row("1", "2", "3")
```

### HTTP Download

```go
func handler(w http.ResponseWriter, r *http.Request) {
    users := getUsers()
    export.FromCSV(users).ServeHTTP(w, "users.csv")
}
```

### Struct Tags

```go
type Product struct {
    ID        int     `csv:"product_id"`
    Name      string  `csv:"product_name"`
    Price     float64 `csv:"price"`
    Internal  string  `csv:"-"`  // Excluded
}
```

If no `csv` tag is present, the `json` tag is used as a fallback.

## CSV Import

### Read All Records

```go
reader, _ := export.ReadCSVFile("data.csv")
records, _ := reader.ReadAll()

for _, row := range records {
    fmt.Println(row)
}
```

### Read with Headers

```go
reader, _ := export.ReadCSVFile("data.csv")
rows, _ := reader.ReadAllWithHeaders()

for _, row := range rows {
    fmt.Printf("Name: %s, Age: %s\n", row["name"], row["age"])
}
```

### Read into Structs

```go
type User struct {
    ID    int    `csv:"id"`
    Name  string `csv:"name"`
    Email string `csv:"email"`
}

var users []User
reader, _ := export.ReadCSVFile("users.csv")
reader.Into(&users)

for _, user := range users {
    fmt.Printf("%d: %s\n", user.ID, user.Name)
}
```

### Tab-Delimited

```go
reader, _ := export.ReadCSVFile("data.tsv")
reader.TabDelimited().ReadAll()
```

## Excel Export

### From Structs

```go
type Order struct {
    ID       int       `excel:"Order ID"`
    Customer string    `excel:"Customer"`
    Amount   float64   `excel:"Amount"`
    Date     time.Time `excel:"Order Date"`
}

orders := []Order{...}

export.FromExcel(orders).
    AutoWidth().
    FreezeHeader().
    Save("orders.xlsx")
```

### Multiple Sheets

```go
excel := export.NewExcel()

excel.Sheet("Users").
    From(users).
    AutoWidth()

excel.Sheet("Orders").
    From(orders).
    AutoWidth()

excel.Save("report.xlsx")
```

### Manual Rows

```go
sheet := export.NewExcel().Sheet("Report")

sheet.Headers("Month", "Revenue", "Expenses", "Profit").
    Row("January", 10000, 7000, 3000).
    Row("February", 12000, 8000, 4000).
    Row("March", 15000, 9000, 6000)

sheet.AutoWidth().Save("report.xlsx")
```

### Styling

```go
sheet := export.NewExcel().Sheet("Styled")
sheet.Headers("Name", "Value").
    Row("Total", 1234.56)

// Apply style to a range
sheet.SetCellStyle(1, 1, 2, 1, export.CellStyle{
    Bold:      true,
    FillColor: "#4472C4",
    FontColor: "#FFFFFF",
    Alignment: "center",
})

// Number format for column
sheet.SetNumberFormat(2, export.NumberFormatCurrency)

sheet.Build().Save("styled.xlsx")
```

### Cell Styles

```go
style := export.CellStyle{
    Bold:      true,
    Italic:    true,
    FontSize:  14,
    FontColor: "#FF0000",
    FillColor: "#FFFF00",
    Alignment: "center",  // left, center, right
    NumFormat: "0.00%",
    Border:    true,
}

sheet.SetCellStyle(startCol, startRow, endCol, endRow, style)
```

### Number Formats

```go
// Predefined formats
export.NumberFormatGeneral    // "General"
export.NumberFormatInteger    // "0"
export.NumberFormatDecimal2   // "0.00"
export.NumberFormatPercent    // "0%"
export.NumberFormatPercent2   // "0.00%"
export.NumberFormatCurrency   // "$#,##0.00"
export.NumberFormatDate       // "yyyy-mm-dd"
export.NumberFormatDateTime   // "yyyy-mm-dd hh:mm:ss"
export.NumberFormatTime       // "hh:mm:ss"
export.NumberFormatAccounting // Accounting format

// Apply to column
sheet.SetNumberFormat(3, export.NumberFormatCurrency)
sheet.SetDateFormat(4, "mm/dd/yyyy")
```

### Column Widths

```go
sheet.ColWidth(1, 30).   // Column A: 30
    ColWidth(2, 15).     // Column B: 15
    AutoWidth()          // Or auto-calculate all
```

### Formulas

```go
sheet.Headers("A", "B", "Sum").
    Row(10, 20, nil).
    Row(30, 40, nil)

// Add SUM formula to column C
sheet.AddFormula(3, 2, "SUM(A2:B2)")
sheet.AddFormula(3, 3, "SUM(A3:B3)")
```

### Merge Cells

```go
sheet.MergeCells(1, 1, 3, 1)  // Merge A1:C1
```

### Freeze Header

```go
sheet.FreezeHeader()  // Freeze first row
```

### Excel Table

```go
// Format as Excel table with sorting/filtering
sheet.From(data).AddTable()
```

### HTTP Download

```go
func handler(w http.ResponseWriter, r *http.Request) {
    export.FromExcel(data).
        AutoWidth().
        ServeHTTP(w, "report.xlsx")
}
```

## Complete Examples

### CSV Report

```go
type SalesReport struct {
    Date     time.Time `csv:"Date"`
    Product  string    `csv:"Product"`
    Quantity int       `csv:"Qty"`
    Revenue  float64   `csv:"Revenue"`
}

func generateReport(w http.ResponseWriter) {
    report := getSalesData()

    export.NewCSV().
        From(report).
        ServeHTTP(w, "sales_report.csv")
}
```

### Excel Dashboard

```go
func generateDashboard() {
    excel := export.NewExcel()

    // Summary sheet
    summary := excel.Sheet("Summary")
    summary.Headers("Metric", "Value").
        Row("Total Revenue", 150000).
        Row("Total Orders", 1234).
        Row("Average Order", 121.52)
    summary.SetNumberFormat(2, export.NumberFormatCurrency)
    summary.AutoWidth()

    // Details sheet
    excel.Sheet("Orders").
        From(getAllOrders()).
        AutoWidth().
        FreezeHeader().
        AddTable()

    // Customers sheet
    excel.Sheet("Customers").
        From(getAllCustomers()).
        AutoWidth()

    excel.Save("dashboard.xlsx")
}
```

### Import and Transform

```go
// Read CSV
var products []Product
reader, _ := export.ReadCSVFile("products.csv")
reader.Into(&products)

// Transform
for i := range products {
    products[i].Price *= 1.1  // 10% increase
}

// Export to Excel
export.FromExcel(products).
    SetNumberFormat(3, export.NumberFormatCurrency).
    AutoWidth().
    Save("products_updated.xlsx")
```

## Struct Tag Reference

Both `csv` and `excel` tags are supported:

```go
type Example struct {
    // Custom column name
    Field1 string `csv:"custom_name"`
    Field1 string `excel:"Custom Name"`

    // Exclude from export
    Internal string `csv:"-"`
    Internal string `excel:"-"`

    // Falls back to json tag if csv/excel not present
    Field2 string `json:"field_two"`
}
```

## Dependencies

- CSV: Uses Go standard library `encoding/csv`
- Excel: Uses `github.com/xuri/excelize/v2`
