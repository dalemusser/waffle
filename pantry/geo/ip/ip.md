# IP Geolocation

The `geo/ip` package provides IP geolocation using MaxMind GeoLite2 or GeoIP2 databases. It requires an external database file that you download from MaxMind.

## Installation

```go
import "waffle/geo/ip"
```

## Database Setup

1. Create a free MaxMind account at https://www.maxmind.com/en/geolite2/signup
2. Download GeoLite2-City.mmdb (required)
3. Optionally download GeoLite2-ASN.mmdb (for ISP/ASN data)
4. Store the files securely and update periodically (MaxMind updates weekly)

## Quick Start

```go
// Open database
db, err := ip.Open(ip.Config{
    CityDBPath: "/path/to/GeoLite2-City.mmdb",
    ASNDBPath:  "/path/to/GeoLite2-ASN.mmdb", // optional
})
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Lookup IP
loc, err := db.Lookup("8.8.8.8")
if err != nil {
    log.Printf("Lookup failed: %v", err)
    return
}

fmt.Printf("Country: %s (%s)\n", loc.Country, loc.CountryCode)
fmt.Printf("City: %s\n", loc.City)
fmt.Printf("Region: %s (%s)\n", loc.Region, loc.RegionCode)
fmt.Printf("Postal: %s\n", loc.PostalCode)
fmt.Printf("Coordinates: %.4f, %.4f (±%dkm)\n", loc.Latitude, loc.Longitude, loc.Accuracy)
fmt.Printf("Timezone: %s\n", loc.Timezone)
fmt.Printf("EU Member: %v\n", loc.IsEU)

// With ASN database
fmt.Printf("ISP: %s\n", loc.ISP)
fmt.Printf("Organization: %s\n", loc.Organization)
fmt.Printf("ASN: AS%d %s\n", loc.ASN, loc.ASOrg)
```

## Location Fields

```go
type Location struct {
    IP            string  // IP address looked up
    Latitude      float64 // Latitude
    Longitude     float64 // Longitude
    Accuracy      int     // Accuracy radius in km

    Continent     string  // "North America"
    ContinentCode string  // "NA"
    Country       string  // "United States"
    CountryCode   string  // "US"
    Region        string  // "California"
    RegionCode    string  // "CA"
    City          string  // "Mountain View"
    PostalCode    string  // "94035"
    Timezone      string  // "America/Los_Angeles"

    ISP           string  // "Google LLC"
    Organization  string  // "Google LLC"
    ASN           int     // 15169
    ASOrg         string  // "GOOGLE"

    IsEU          bool    // EU member state
}
```

## Integration with geo.Coord

```go
loc, err := db.Lookup("8.8.8.8")
if err != nil {
    return err
}

// Get as geo.Coord for distance calculations
coord := loc.Coord()

// Calculate distance to another location
destination := geo.NewCoord(40.7128, -74.0060)
distance := coord.DistanceTo(destination)
```

## HTTP Middleware

### Basic Middleware

```go
// Set as default for package-level functions
ip.SetDefault(db)

// Use middleware - adds location to request context
mux.Handle("/", ip.Middleware(myHandler))

// Access location in handler
func myHandler(w http.ResponseWriter, r *http.Request) {
    loc := ip.FromContext(r.Context())
    if loc != nil {
        fmt.Printf("Request from %s, %s\n", loc.City, loc.Country)
    }
}
```

### Function-style Middleware

```go
http.HandleFunc("/", db.MiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
    loc := ip.FromContext(r.Context())
    // ...
}))
```

## Client IP Detection

The `GetClientIP` function extracts the real client IP from requests, handling common proxy headers:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    clientIP := ip.GetClientIP(r)

    loc, err := db.Lookup(clientIP)
    // ...
}
```

Headers checked (in order):
1. `X-Forwarded-For` (first IP in chain)
2. `X-Real-IP`
3. `CF-Connecting-IP` (Cloudflare)
4. `True-Client-IP` (Akamai, Cloudflare)
5. `RemoteAddr` (fallback)

## Private IP Detection

```go
clientIP := ip.GetClientIP(r)

if ip.IsPrivateIP(clientIP) {
    // Local network, localhost, or link-local address
    fmt.Println("Private/local IP - no geolocation available")
    return
}

loc, err := db.Lookup(clientIP)
```

## Global Default

```go
// Set default at startup
db, _ := ip.Open(ip.Config{CityDBPath: "GeoLite2-City.mmdb"})
ip.SetDefault(db)

// Use package-level functions anywhere
loc, err := ip.Lookup("8.8.8.8")

// Get default database
db := ip.Default()
```

## Error Handling

```go
loc, err := db.Lookup(ipAddress)
if err != nil {
    switch err {
    case ip.ErrDatabaseNotLoaded:
        // Database not opened
    case ip.ErrInvalidIP:
        // Invalid IP address format
    case ip.ErrNotFound:
        // IP not in database (rare)
    default:
        // Other error
    }
}
```

## Database Options

### GeoLite2 (Free)

- **GeoLite2-City**: Country, region, city, postal, coordinates, timezone
- **GeoLite2-ASN**: ASN and organization name
- **GeoLite2-Country**: Country only (smaller file)

Download: https://dev.maxmind.com/geoip/geolite2-free-geolocation-data

### GeoIP2 (Commercial)

Higher accuracy, more frequent updates, additional data fields.

See: https://www.maxmind.com/en/geoip2-databases

## Accuracy Notes

- **Country**: ~99% accurate
- **Region/State**: ~80% accurate
- **City**: ~70-80% accurate
- **Coordinates**: Can be off by several kilometers (use `Accuracy` field)
- **VPNs/Proxies**: Will show VPN exit location, not user's actual location
- **Mobile IPs**: Often geolocate to carrier headquarters

## Best Practices

1. **Update regularly**: MaxMind updates databases weekly
2. **Handle missing data**: Not all IPs have complete data
3. **Check accuracy**: Use the `Accuracy` field for confidence
4. **Private IPs**: Check `IsPrivateIP` before lookup
5. **Cache results**: IP-to-location mappings are stable short-term
6. **Secure database**: The .mmdb files contain licensed data
