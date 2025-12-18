// geo/geo.go
package geo

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	// Earth's radius in kilometers
	EarthRadiusKm = 6371.0

	// Earth's radius in miles
	EarthRadiusMiles = 3958.8

	// Earth's radius in meters
	EarthRadiusMeters = 6371000.0

	// Earth's radius in nautical miles
	EarthRadiusNauticalMiles = 3440.065
)

// Unit represents a distance unit.
type Unit int

const (
	Kilometers Unit = iota
	Miles
	Meters
	Feet
	NauticalMiles
)

// String returns the unit name.
func (u Unit) String() string {
	switch u {
	case Kilometers:
		return "km"
	case Miles:
		return "mi"
	case Meters:
		return "m"
	case Feet:
		return "ft"
	case NauticalMiles:
		return "nm"
	default:
		return "unknown"
	}
}

// Coord represents a geographic coordinate.
type Coord struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// NewCoord creates a new coordinate.
func NewCoord(lat, lon float64) Coord {
	return Coord{Lat: lat, Lon: lon}
}

// Valid returns true if the coordinate is within valid ranges.
func (c Coord) Valid() bool {
	return c.Lat >= -90 && c.Lat <= 90 && c.Lon >= -180 && c.Lon <= 180
}

// IsZero returns true if the coordinate is at 0,0.
func (c Coord) IsZero() bool {
	return c.Lat == 0 && c.Lon == 0
}

// String returns the coordinate as a string.
func (c Coord) String() string {
	return fmt.Sprintf("%.6f,%.6f", c.Lat, c.Lon)
}

// DMS returns the coordinate in degrees, minutes, seconds format.
func (c Coord) DMS() string {
	latDir := "N"
	if c.Lat < 0 {
		latDir = "S"
	}
	lonDir := "E"
	if c.Lon < 0 {
		lonDir = "W"
	}

	latDMS := toDMS(math.Abs(c.Lat))
	lonDMS := toDMS(math.Abs(c.Lon))

	return fmt.Sprintf("%s%s %s%s", latDMS, latDir, lonDMS, lonDir)
}

// toDMS converts decimal degrees to DMS string.
func toDMS(dd float64) string {
	d := int(dd)
	m := int((dd - float64(d)) * 60)
	s := (dd - float64(d) - float64(m)/60) * 3600
	return fmt.Sprintf("%d°%d'%.2f\"", d, m, s)
}

// DistanceTo calculates the distance to another coordinate using the Haversine formula.
// Returns distance in kilometers.
func (c Coord) DistanceTo(other Coord) float64 {
	return Haversine(c, other)
}

// DistanceToUnit calculates the distance to another coordinate in the specified unit.
func (c Coord) DistanceToUnit(other Coord, unit Unit) float64 {
	km := Haversine(c, other)
	return ConvertDistance(km, Kilometers, unit)
}

// BearingTo calculates the initial bearing to another coordinate.
// Returns bearing in degrees (0-360, where 0 is north).
func (c Coord) BearingTo(other Coord) float64 {
	return Bearing(c, other)
}

// DestinationPoint calculates the destination point given distance and bearing.
// Distance is in kilometers, bearing is in degrees.
func (c Coord) DestinationPoint(distance, bearing float64) Coord {
	return Destination(c, distance, bearing)
}

// MidpointTo calculates the midpoint between two coordinates.
func (c Coord) MidpointTo(other Coord) Coord {
	return Midpoint(c, other)
}

// Haversine calculates the great-circle distance between two coordinates.
// Returns distance in kilometers.
func Haversine(c1, c2 Coord) float64 {
	lat1 := toRadians(c1.Lat)
	lat2 := toRadians(c2.Lat)
	deltaLat := toRadians(c2.Lat - c1.Lat)
	deltaLon := toRadians(c2.Lon - c1.Lon)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusKm * c
}

// Vincenty calculates distance using Vincenty's formulae for higher accuracy.
// Returns distance in kilometers.
// This is more accurate than Haversine, especially for long distances.
func Vincenty(c1, c2 Coord) float64 {
	// WGS-84 ellipsoid parameters
	a := 6378137.0          // Semi-major axis (meters)
	f := 1 / 298.257223563  // Flattening
	b := a * (1 - f)        // Semi-minor axis

	lat1 := toRadians(c1.Lat)
	lat2 := toRadians(c2.Lat)
	lon1 := toRadians(c1.Lon)
	lon2 := toRadians(c2.Lon)

	L := lon2 - lon1

	U1 := math.Atan((1 - f) * math.Tan(lat1))
	U2 := math.Atan((1 - f) * math.Tan(lat2))

	sinU1, cosU1 := math.Sin(U1), math.Cos(U1)
	sinU2, cosU2 := math.Sin(U2), math.Cos(U2)

	lambda := L
	var lambdaP float64
	var sinSigma, cosSigma, sigma, sinAlpha, cosSqAlpha, cos2SigmaM float64

	for i := 0; i < 100; i++ {
		sinLambda := math.Sin(lambda)
		cosLambda := math.Cos(lambda)

		sinSigma = math.Sqrt(
			(cosU2*sinLambda)*(cosU2*sinLambda) +
				(cosU1*sinU2-sinU1*cosU2*cosLambda)*(cosU1*sinU2-sinU1*cosU2*cosLambda))

		if sinSigma == 0 {
			return 0 // Co-incident points
		}

		cosSigma = sinU1*sinU2 + cosU1*cosU2*cosLambda
		sigma = math.Atan2(sinSigma, cosSigma)

		sinAlpha = cosU1 * cosU2 * sinLambda / sinSigma
		cosSqAlpha = 1 - sinAlpha*sinAlpha

		if cosSqAlpha != 0 {
			cos2SigmaM = cosSigma - 2*sinU1*sinU2/cosSqAlpha
		} else {
			cos2SigmaM = 0
		}

		C := f / 16 * cosSqAlpha * (4 + f*(4-3*cosSqAlpha))

		lambdaP = lambda
		lambda = L + (1-C)*f*sinAlpha*
			(sigma+C*sinSigma*(cos2SigmaM+C*cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)))

		if math.Abs(lambda-lambdaP) < 1e-12 {
			break
		}
	}

	uSq := cosSqAlpha * (a*a - b*b) / (b * b)
	A := 1 + uSq/16384*(4096+uSq*(-768+uSq*(320-175*uSq)))
	B := uSq / 1024 * (256 + uSq*(-128+uSq*(74-47*uSq)))

	deltaSigma := B * sinSigma * (cos2SigmaM + B/4*(cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)-
		B/6*cos2SigmaM*(-3+4*sinSigma*sinSigma)*(-3+4*cos2SigmaM*cos2SigmaM)))

	s := b * A * (sigma - deltaSigma)

	return s / 1000 // Convert to kilometers
}

// Bearing calculates the initial bearing from c1 to c2.
// Returns bearing in degrees (0-360).
func Bearing(c1, c2 Coord) float64 {
	lat1 := toRadians(c1.Lat)
	lat2 := toRadians(c2.Lat)
	deltaLon := toRadians(c2.Lon - c1.Lon)

	y := math.Sin(deltaLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLon)

	bearing := toDegrees(math.Atan2(y, x))

	// Normalize to 0-360
	return math.Mod(bearing+360, 360)
}

// FinalBearing calculates the final bearing from c1 to c2.
// Returns bearing in degrees (0-360).
func FinalBearing(c1, c2 Coord) float64 {
	return math.Mod(Bearing(c2, c1)+180, 360)
}

// Destination calculates the destination point from a start point,
// given a distance (km) and bearing (degrees).
func Destination(start Coord, distance, bearing float64) Coord {
	lat1 := toRadians(start.Lat)
	lon1 := toRadians(start.Lon)
	brng := toRadians(bearing)

	angularDist := distance / EarthRadiusKm

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angularDist) +
		math.Cos(lat1)*math.Sin(angularDist)*math.Cos(brng))

	lon2 := lon1 + math.Atan2(
		math.Sin(brng)*math.Sin(angularDist)*math.Cos(lat1),
		math.Cos(angularDist)-math.Sin(lat1)*math.Sin(lat2))

	// Normalize longitude to -180 to 180
	lon2 = math.Mod(lon2+3*math.Pi, 2*math.Pi) - math.Pi

	return Coord{
		Lat: toDegrees(lat2),
		Lon: toDegrees(lon2),
	}
}

// Midpoint calculates the midpoint between two coordinates.
func Midpoint(c1, c2 Coord) Coord {
	lat1 := toRadians(c1.Lat)
	lon1 := toRadians(c1.Lon)
	lat2 := toRadians(c2.Lat)
	deltaLon := toRadians(c2.Lon - c1.Lon)

	Bx := math.Cos(lat2) * math.Cos(deltaLon)
	By := math.Cos(lat2) * math.Sin(deltaLon)

	lat3 := math.Atan2(
		math.Sin(lat1)+math.Sin(lat2),
		math.Sqrt((math.Cos(lat1)+Bx)*(math.Cos(lat1)+Bx)+By*By))

	lon3 := lon1 + math.Atan2(By, math.Cos(lat1)+Bx)

	return Coord{
		Lat: toDegrees(lat3),
		Lon: toDegrees(lon3),
	}
}

// BoundingBox represents a geographic bounding box.
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLon float64 `json:"max_lon"`
}

// BoundsFromCenter calculates a bounding box around a center point.
// Radius is in kilometers.
func BoundsFromCenter(center Coord, radius float64) BoundingBox {
	// Angular distance in radians
	angular := radius / EarthRadiusKm

	lat := toRadians(center.Lat)
	lon := toRadians(center.Lon)

	minLat := lat - angular
	maxLat := lat + angular

	// Account for longitude variation with latitude
	deltaLon := math.Asin(math.Sin(angular) / math.Cos(lat))

	minLon := lon - deltaLon
	maxLon := lon + deltaLon

	return BoundingBox{
		MinLat: toDegrees(minLat),
		MaxLat: toDegrees(maxLat),
		MinLon: toDegrees(minLon),
		MaxLon: toDegrees(maxLon),
	}
}

// Contains returns true if the coordinate is within the bounding box.
func (bb BoundingBox) Contains(c Coord) bool {
	return c.Lat >= bb.MinLat && c.Lat <= bb.MaxLat &&
		c.Lon >= bb.MinLon && c.Lon <= bb.MaxLon
}

// Center returns the center of the bounding box.
func (bb BoundingBox) Center() Coord {
	return Coord{
		Lat: (bb.MinLat + bb.MaxLat) / 2,
		Lon: (bb.MinLon + bb.MaxLon) / 2,
	}
}

// Expand expands the bounding box by the given amount (in degrees).
func (bb BoundingBox) Expand(degrees float64) BoundingBox {
	return BoundingBox{
		MinLat: bb.MinLat - degrees,
		MaxLat: bb.MaxLat + degrees,
		MinLon: bb.MinLon - degrees,
		MaxLon: bb.MaxLon + degrees,
	}
}

// ConvertDistance converts a distance from one unit to another.
func ConvertDistance(value float64, from, to Unit) float64 {
	// Convert to meters first
	var meters float64
	switch from {
	case Kilometers:
		meters = value * 1000
	case Miles:
		meters = value * 1609.344
	case Meters:
		meters = value
	case Feet:
		meters = value * 0.3048
	case NauticalMiles:
		meters = value * 1852
	}

	// Convert from meters to target unit
	switch to {
	case Kilometers:
		return meters / 1000
	case Miles:
		return meters / 1609.344
	case Meters:
		return meters
	case Feet:
		return meters / 0.3048
	case NauticalMiles:
		return meters / 1852
	}

	return value
}

// ParseCoord parses a coordinate from various string formats.
// Supports:
//   - Decimal: "40.7128,-74.0060" or "40.7128, -74.0060"
//   - DMS: "40°42'46\"N 74°0'22\"W"
//   - Degrees and decimal minutes: "40°42.767'N 74°0.367'W"
func ParseCoord(s string) (Coord, error) {
	s = strings.TrimSpace(s)

	// Try decimal format first
	if coord, err := parseDecimal(s); err == nil {
		return coord, nil
	}

	// Try DMS format
	if coord, err := parseDMS(s); err == nil {
		return coord, nil
	}

	return Coord{}, fmt.Errorf("geo: unable to parse coordinate: %s", s)
}

// parseDecimal parses "lat,lon" format.
func parseDecimal(s string) (Coord, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return Coord{}, fmt.Errorf("invalid format")
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return Coord{}, err
	}

	lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return Coord{}, err
	}

	coord := Coord{Lat: lat, Lon: lon}
	if !coord.Valid() {
		return Coord{}, fmt.Errorf("coordinate out of range")
	}

	return coord, nil
}

// DMS regex pattern
var dmsPattern = regexp.MustCompile(`(\d+)[°]\s*(\d+)[′']\s*([\d.]+)[″"]?\s*([NSns])\s+(\d+)[°]\s*(\d+)[′']\s*([\d.]+)[″"]?\s*([EWew])`)

// parseDMS parses DMS format.
func parseDMS(s string) (Coord, error) {
	matches := dmsPattern.FindStringSubmatch(s)
	if matches == nil {
		return Coord{}, fmt.Errorf("invalid DMS format")
	}

	latD, _ := strconv.ParseFloat(matches[1], 64)
	latM, _ := strconv.ParseFloat(matches[2], 64)
	latS, _ := strconv.ParseFloat(matches[3], 64)
	latDir := strings.ToUpper(matches[4])

	lonD, _ := strconv.ParseFloat(matches[5], 64)
	lonM, _ := strconv.ParseFloat(matches[6], 64)
	lonS, _ := strconv.ParseFloat(matches[7], 64)
	lonDir := strings.ToUpper(matches[8])

	lat := latD + latM/60 + latS/3600
	if latDir == "S" {
		lat = -lat
	}

	lon := lonD + lonM/60 + lonS/3600
	if lonDir == "W" {
		lon = -lon
	}

	coord := Coord{Lat: lat, Lon: lon}
	if !coord.Valid() {
		return Coord{}, fmt.Errorf("coordinate out of range")
	}

	return coord, nil
}

// Helper functions

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func toDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

// PointInPolygon checks if a coordinate is inside a polygon.
// Uses the ray casting algorithm.
func PointInPolygon(point Coord, polygon []Coord) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		if (polygon[i].Lon > point.Lon) != (polygon[j].Lon > point.Lon) &&
			point.Lat < (polygon[j].Lat-polygon[i].Lat)*(point.Lon-polygon[i].Lon)/
				(polygon[j].Lon-polygon[i].Lon)+polygon[i].Lat {
			inside = !inside
		}
		j = i
	}

	return inside
}

// PolygonArea calculates the area of a polygon in square kilometers.
// Uses the Shoelace formula adapted for spherical coordinates.
func PolygonArea(polygon []Coord) float64 {
	if len(polygon) < 3 {
		return 0
	}

	// Simple approximation using Shoelace formula
	// For more accuracy, use proper spherical excess calculation
	var area float64
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		area += (polygon[j].Lon + polygon[i].Lon) * (polygon[j].Lat - polygon[i].Lat)
		j = i
	}

	area = math.Abs(area) / 2

	// Convert from degrees^2 to km^2 (approximate at equator)
	// 1 degree ≈ 111 km at equator
	return area * 111 * 111
}

// CompassDirection returns the compass direction for a bearing.
func CompassDirection(bearing float64) string {
	directions := []string{
		"N", "NNE", "NE", "ENE",
		"E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW",
		"W", "WNW", "NW", "NNW",
	}

	// Normalize bearing
	bearing = math.Mod(bearing+360, 360)

	// Each direction covers 22.5 degrees
	index := int((bearing + 11.25) / 22.5)
	if index >= 16 {
		index = 0
	}

	return directions[index]
}
