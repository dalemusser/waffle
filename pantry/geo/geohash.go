// geo/geohash.go
package geo

import (
	"errors"
	"strings"
)

// Geohash encoding alphabet
const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"

// Geohash errors
var (
	ErrInvalidGeohash = errors.New("geo: invalid geohash")
)

// Geohash encodes a coordinate to a geohash string.
// Precision determines the length (1-12). Higher precision = smaller area.
// Default precision is 9 (~4.77m x 4.77m).
func Geohash(c Coord, precision int) string {
	if precision < 1 {
		precision = 1
	}
	if precision > 12 {
		precision = 12
	}

	minLat, maxLat := -90.0, 90.0
	minLon, maxLon := -180.0, 180.0

	var hash strings.Builder
	hash.Grow(precision)

	bit := 0
	ch := 0
	isLon := true

	for hash.Len() < precision {
		if isLon {
			mid := (minLon + maxLon) / 2
			if c.Lon >= mid {
				ch |= 1 << (4 - bit)
				minLon = mid
			} else {
				maxLon = mid
			}
		} else {
			mid := (minLat + maxLat) / 2
			if c.Lat >= mid {
				ch |= 1 << (4 - bit)
				minLat = mid
			} else {
				maxLat = mid
			}
		}

		isLon = !isLon
		bit++

		if bit == 5 {
			hash.WriteByte(base32[ch])
			bit = 0
			ch = 0
		}
	}

	return hash.String()
}

// GeohashDecode decodes a geohash string to a coordinate.
// Returns the center point of the geohash cell.
func GeohashDecode(hash string) (Coord, error) {
	bounds, err := GeohashBounds(hash)
	if err != nil {
		return Coord{}, err
	}
	return bounds.Center(), nil
}

// GeohashBounds returns the bounding box for a geohash.
func GeohashBounds(hash string) (BoundingBox, error) {
	hash = strings.ToLower(hash)

	minLat, maxLat := -90.0, 90.0
	minLon, maxLon := -180.0, 180.0

	isLon := true

	for _, c := range hash {
		idx := strings.IndexRune(base32, c)
		if idx == -1 {
			return BoundingBox{}, ErrInvalidGeohash
		}

		for bit := 4; bit >= 0; bit-- {
			if isLon {
				mid := (minLon + maxLon) / 2
				if idx&(1<<bit) != 0 {
					minLon = mid
				} else {
					maxLon = mid
				}
			} else {
				mid := (minLat + maxLat) / 2
				if idx&(1<<bit) != 0 {
					minLat = mid
				} else {
					maxLat = mid
				}
			}
			isLon = !isLon
		}
	}

	return BoundingBox{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}, nil
}

// GeohashNeighbors returns the 8 neighboring geohashes.
func GeohashNeighbors(hash string) ([]string, error) {
	bounds, err := GeohashBounds(hash)
	if err != nil {
		return nil, err
	}

	center := bounds.Center()
	latDelta := (bounds.MaxLat - bounds.MinLat)
	lonDelta := (bounds.MaxLon - bounds.MinLon)
	precision := len(hash)

	neighbors := make([]string, 8)

	// N, NE, E, SE, S, SW, W, NW
	offsets := []struct{ lat, lon float64 }{
		{1, 0},   // N
		{1, 1},   // NE
		{0, 1},   // E
		{-1, 1},  // SE
		{-1, 0},  // S
		{-1, -1}, // SW
		{0, -1},  // W
		{1, -1},  // NW
	}

	for i, offset := range offsets {
		neighborCoord := Coord{
			Lat: center.Lat + offset.lat*latDelta,
			Lon: center.Lon + offset.lon*lonDelta,
		}
		neighbors[i] = Geohash(neighborCoord, precision)
	}

	return neighbors, nil
}

// GeohashPrecision returns the approximate cell dimensions for a precision level.
func GeohashPrecision(precision int) (latError, lonError float64) {
	// Approximate dimensions in degrees
	precisions := []struct{ lat, lon float64 }{
		{23, 23},          // 1: ±2500km
		{2.8, 5.6},        // 2: ±630km
		{0.70, 0.70},      // 3: ±78km
		{0.087, 0.18},     // 4: ±20km
		{0.022, 0.022},    // 5: ±2.4km
		{0.0027, 0.0055},  // 6: ±610m
		{0.00068, 0.00068},// 7: ±76m
		{0.000086, 0.00017},// 8: ±19m
		{0.000021, 0.000021},// 9: ±2.4m
		{0.0000027, 0.0000053},// 10: ±0.6m
		{0.00000067, 0.00000067},// 11: ±0.074m
		{0.000000083, 0.00000017},// 12: ±0.019m
	}

	if precision < 1 {
		precision = 1
	}
	if precision > 12 {
		precision = 12
	}

	p := precisions[precision-1]
	return p.lat, p.lon
}

// GeohashContains checks if a geohash contains another.
// A geohash contains another if the other starts with the first.
func GeohashContains(parent, child string) bool {
	return strings.HasPrefix(strings.ToLower(child), strings.ToLower(parent))
}

// GeohashesInBounds returns all geohashes of the given precision
// that intersect with the bounding box.
func GeohashesInBounds(bounds BoundingBox, precision int) []string {
	if precision < 1 {
		precision = 1
	}
	if precision > 12 {
		precision = 12
	}

	var hashes []string
	seen := make(map[string]bool)

	// Get approximate step size for this precision
	latErr, lonErr := GeohashPrecision(precision)
	latStep := latErr * 2
	lonStep := lonErr * 2

	for lat := bounds.MinLat; lat <= bounds.MaxLat; lat += latStep {
		for lon := bounds.MinLon; lon <= bounds.MaxLon; lon += lonStep {
			hash := Geohash(Coord{Lat: lat, Lon: lon}, precision)
			if !seen[hash] {
				seen[hash] = true
				hashes = append(hashes, hash)
			}
		}
	}

	// Also check corners and edges
	corners := []Coord{
		{bounds.MinLat, bounds.MinLon},
		{bounds.MinLat, bounds.MaxLon},
		{bounds.MaxLat, bounds.MinLon},
		{bounds.MaxLat, bounds.MaxLon},
	}

	for _, corner := range corners {
		hash := Geohash(corner, precision)
		if !seen[hash] {
			seen[hash] = true
			hashes = append(hashes, hash)
		}
	}

	return hashes
}

// GeohashValid returns true if the string is a valid geohash.
func GeohashValid(hash string) bool {
	if len(hash) == 0 || len(hash) > 12 {
		return false
	}

	hash = strings.ToLower(hash)
	for _, c := range hash {
		if strings.IndexRune(base32, c) == -1 {
			return false
		}
	}

	return true
}
