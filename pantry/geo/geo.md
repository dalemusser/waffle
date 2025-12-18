# Geo - Geolocation Utilities

The `geo` package provides core geolocation utilities including coordinate handling, distance calculations, and geohashing. This is a lightweight package with no external data dependencies.

## Related Packages

- **[geo/ip](ip/ip.md)** - IP geolocation using MaxMind databases
- **[geo/tz](tz/tz.md)** - Timezone detection from coordinates (adds ~5-10MB to binary)

## Installation

```go
import "waffle/geo"
```

## Coordinates

```go
// Create a coordinate
coord := geo.NewCoord(40.7128, -74.0060) // New York City

// Parse from string
coord, err := geo.ParseCoord("40.7128,-74.0060")
coord, err := geo.ParseCoord("40°42'46\"N 74°0'22\"W") // DMS format

// Validate
if coord.Valid() {
    fmt.Println("Valid coordinate")
}

// Format
fmt.Println(coord.String())  // "40.712800,-74.006000"
fmt.Println(coord.DMS())     // "40°42'46.08"N 74°0'21.60"W"
```

## Distance Calculations

### Haversine (Great-Circle Distance)

Fast calculation assuming spherical Earth. Accuracy ~0.3% max error.

```go
nyc := geo.NewCoord(40.7128, -74.0060)
london := geo.NewCoord(51.5074, -0.1278)

// Distance in kilometers
km := geo.Haversine(nyc, london)
// or
km := nyc.DistanceTo(london)
```

### Vincenty (Ellipsoid Distance)

More accurate calculation using WGS-84 ellipsoid. Better for long distances.

```go
km := geo.Vincenty(nyc, london)
```

### Distance Units

```go
// Get distance in specific unit
miles := nyc.DistanceToUnit(london, geo.Miles)
meters := nyc.DistanceToUnit(london, geo.Meters)
feet := nyc.DistanceToUnit(london, geo.Feet)
nm := nyc.DistanceToUnit(london, geo.NauticalMiles)

// Convert between units
miles := geo.ConvertDistance(100, geo.Kilometers, geo.Miles)
```

Available units: `Kilometers`, `Miles`, `Meters`, `Feet`, `NauticalMiles`

## Bearing and Direction

```go
nyc := geo.NewCoord(40.7128, -74.0060)
london := geo.NewCoord(51.5074, -0.1278)

// Initial bearing (degrees, 0 = north, clockwise)
bearing := geo.Bearing(nyc, london)
// or
bearing := nyc.BearingTo(london)

// Final bearing (bearing at destination)
finalBearing := geo.FinalBearing(nyc, london)

// Compass direction from bearing
direction := geo.CompassDirection(bearing) // "NE", "SSW", etc.
```

## Destination and Midpoint

```go
start := geo.NewCoord(40.7128, -74.0060)

// Find point 100km away at bearing 45° (northeast)
dest := geo.Destination(start, 100, 45)
// or
dest := start.DestinationPoint(100, 45)

// Midpoint between two coordinates
nyc := geo.NewCoord(40.7128, -74.0060)
london := geo.NewCoord(51.5074, -0.1278)
mid := geo.Midpoint(nyc, london)
// or
mid := nyc.MidpointTo(london)
```

## Bounding Box

```go
center := geo.NewCoord(40.7128, -74.0060)

// Create bounding box with 10km radius
bounds := geo.BoundsFromCenter(center, 10)

// Check if point is within bounds
point := geo.NewCoord(40.72, -74.01)
if bounds.Contains(point) {
    fmt.Println("Point is within bounds")
}

// Get center of bounding box
center := bounds.Center()

// Expand bounds by 0.1 degrees
expanded := bounds.Expand(0.1)
```

## Point in Polygon

```go
polygon := []geo.Coord{
    {Lat: 40.0, Lon: -75.0},
    {Lat: 41.0, Lon: -75.0},
    {Lat: 41.0, Lon: -74.0},
    {Lat: 40.0, Lon: -74.0},
}

point := geo.NewCoord(40.5, -74.5)
if geo.PointInPolygon(point, polygon) {
    fmt.Println("Point is inside polygon")
}

// Calculate polygon area (approximate, in km²)
area := geo.PolygonArea(polygon)
```

## Geohash

Geohashes encode coordinates into short strings useful for spatial indexing and proximity searches.

```go
coord := geo.NewCoord(40.7128, -74.0060)

// Encode to geohash (precision 1-12)
hash := geo.Geohash(coord, 9) // "dr5ru6j2c"

// Decode geohash to coordinate (center of cell)
center, err := geo.GeohashDecode("dr5ru6j2c")

// Get bounding box of geohash
bounds, err := geo.GeohashBounds("dr5ru6j2c")

// Get 8 neighboring geohashes
neighbors, err := geo.GeohashNeighbors("dr5ru6j")

// Find all geohashes in a bounding box
hashes := geo.GeohashesInBounds(bounds, 7)

// Check if one geohash contains another
if geo.GeohashContains("dr5ru", "dr5ru6j2c") {
    fmt.Println("dr5ru contains dr5ru6j2c")
}

// Validate geohash
if geo.GeohashValid("dr5ru6j2c") {
    fmt.Println("Valid geohash")
}

// Get approximate cell dimensions for precision level
latErr, lonErr := geo.GeohashPrecision(7)
```

### Geohash Precision Reference

| Precision | Cell Size | Use Case |
|-----------|-----------|----------|
| 1 | ~5000km | Continent |
| 2 | ~1250km | Large country |
| 3 | ~156km | State/region |
| 4 | ~39km | City |
| 5 | ~4.9km | Neighborhood |
| 6 | ~1.2km | Street |
| 7 | ~153m | Block |
| 8 | ~38m | Building |
| 9 | ~4.8m | Room |
| 10 | ~1.2m | Precise |
| 11 | ~15cm | Very precise |
| 12 | ~4cm | Extremely precise |

## Constants

```go
geo.EarthRadiusKm           // 6371.0
geo.EarthRadiusMiles        // 3958.8
geo.EarthRadiusMeters       // 6371000.0
geo.EarthRadiusNauticalMiles // 3440.065
```

## Accuracy Notes

- **Haversine**: Assumes spherical Earth, ~0.3% maximum error
- **Vincenty**: Uses WGS-84 ellipsoid, sub-millimeter accuracy
- **Geohash**: Precision depends on hash length (see table above)
