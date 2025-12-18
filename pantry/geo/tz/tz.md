# Timezone Detection

The `geo/tz` package provides timezone detection from geographic coordinates. It uses embedded timezone boundary data to determine the IANA timezone for any latitude/longitude pair.

**Important**: Importing this package adds approximately 5-10MB to your binary due to the embedded timezone boundary data. Only import it if you need coordinate-to-timezone lookups.

## Installation

```go
import "waffle/geo/tz"
```

## Quick Start

```go
// Get timezone name from coordinates
timezone, err := tz.TimezoneAt(40.7128, -74.0060)
if err != nil {
    log.Printf("Timezone lookup failed: %v", err)
    return
}
fmt.Println(timezone) // "America/New_York"

// Get *time.Location for time operations
loc, err := tz.LocationAt(40.7128, -74.0060)
if err != nil {
    return err
}
localTime := time.Now().In(loc)

// Get current time at a location
tokyoTime, err := tz.TimeAt(35.6762, 139.6503)
fmt.Println(tokyoTime) // Current time in Tokyo
```

## Using with geo.Coord

```go
import (
    "waffle/geo"
    "waffle/geo/tz"
)

coord := geo.NewCoord(40.7128, -74.0060)

timezone, err := tz.TimezoneAtCoord(coord)
location, err := tz.LocationAtCoord(coord)
currentTime, err := tz.TimeAtCoord(coord)
info, err := tz.InfoAtCoord(coord)
```

## Detailed Timezone Info

```go
info, err := tz.InfoAt(40.7128, -74.0060)
if err != nil {
    return err
}

fmt.Printf("Timezone: %s\n", info.Name)           // "America/New_York"
fmt.Printf("Abbreviation: %s\n", info.Abbreviation) // "EST" or "EDT"
fmt.Printf("Offset: %d seconds\n", info.Offset)   // -18000 or -14400
fmt.Printf("Offset: %s\n", info.OffsetString)     // "-05:00" or "-04:00"
fmt.Printf("Is DST: %v\n", info.IsDST)            // true/false
fmt.Printf("Current time: %v\n", info.CurrentTime)
```

### TimezoneInfo Fields

```go
type TimezoneInfo struct {
    Name         string    // IANA name: "America/New_York"
    Abbreviation string    // Current abbreviation: "EST", "EDT"
    Offset       int       // UTC offset in seconds
    OffsetString string    // Formatted offset: "-05:00"
    IsDST        bool      // Daylight saving time active
    CurrentTime  time.Time // Current time in this timezone
}
```

## List All Timezones

```go
timezones, err := tz.AllTimezones()
if err != nil {
    return err
}

for _, name := range timezones {
    fmt.Println(name)
}
// Africa/Abidjan
// Africa/Accra
// America/New_York
// ...
```

## Custom Finder Instance

For advanced use cases or multiple instances:

```go
// Create a new finder
finder, err := tz.New()
if err != nil {
    return err
}

// Use the finder
timezone, err := finder.TimezoneAt(lat, lon)
info, err := finder.InfoAt(lat, lon)
```

## Error Handling

```go
timezone, err := tz.TimezoneAt(lat, lon)
if err != nil {
    switch err {
    case tz.ErrInvalidCoord:
        // Coordinates out of range (lat: -90 to 90, lon: -180 to 180)
    case tz.ErrTimezoneNotFound:
        // No timezone found (ocean areas, Antarctica)
    default:
        // Other error
    }
}
```

## Common Use Cases

### Convert Time Between Locations

```go
// Get timezone locations
nycLoc, _ := tz.LocationAt(40.7128, -74.0060)
tokyoLoc, _ := tz.LocationAt(35.6762, 139.6503)

// Convert a time from NYC to Tokyo
nycTime := time.Date(2024, 1, 15, 9, 0, 0, 0, nycLoc)
tokyoTime := nycTime.In(tokyoLoc)
fmt.Printf("9 AM in NYC is %s in Tokyo\n", tokyoTime.Format("3:04 PM"))
```

### Display Local Time for User Location

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Get user's location from IP (using geo/ip)
    ipLoc := ip.FromContext(r.Context())
    if ipLoc == nil {
        return
    }

    // Get timezone at that location
    loc, err := tz.LocationAt(ipLoc.Latitude, ipLoc.Longitude)
    if err != nil {
        // Fall back to timezone from IP database
        loc, _ = time.LoadLocation(ipLoc.Timezone)
    }

    localTime := time.Now().In(loc)
    fmt.Fprintf(w, "Your local time: %s", localTime.Format("3:04 PM"))
}
```

### Schedule at User's Local Time

```go
// User wants a reminder at 9 AM their time
userLoc, _ := tz.LocationAt(userLat, userLon)

// Create time in user's timezone
reminderTime := time.Date(2024, 1, 15, 9, 0, 0, 0, userLoc)

// Convert to UTC for storage
reminderUTC := reminderTime.UTC()
```

## Accuracy Notes

- Uses official timezone boundary data from timezone-boundary-builder
- Accurate to boundary precision (~100m)
- Ocean areas may return empty or nearest land timezone
- Some disputed territories may have multiple valid timezones
- Antarctica has limited coverage

## Binary Size

The embedded timezone data adds approximately:
- Full dataset: ~10MB to binary
- This is a one-time cost regardless of how many lookups you perform

If binary size is critical and you only need timezone from IP addresses, consider using the `Timezone` field from `geo/ip.Location` instead, which doesn't require this package.

## Dependencies

This package uses:
- `github.com/ringsaturn/tzf` - Timezone finder with embedded data
