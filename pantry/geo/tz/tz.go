// geo/tz/tz.go
// This package provides timezone detection from coordinates.
// NOTE: Importing this package adds ~5-10MB to the binary due to embedded timezone boundary data.
package tz

import (
	"errors"
	"sync"
	"time"

	"github.com/dalemusser/waffle/pantry/geo"
	"github.com/ringsaturn/tzf"
)

// Errors
var (
	ErrTimezoneNotFound = errors.New("tz: timezone not found for coordinates")
	ErrInvalidCoord     = errors.New("tz: invalid coordinates")
)

// Finder provides timezone lookups from coordinates.
type Finder struct {
	finder tzf.F
}

var (
	defaultFinder   *Finder
	defaultFinderMu sync.RWMutex
	initOnce        sync.Once
	initErr         error
)

// init initializes the default finder lazily.
func getDefaultFinder() (*Finder, error) {
	initOnce.Do(func() {
		finder, err := New()
		if err != nil {
			initErr = err
			return
		}
		defaultFinder = finder
	})

	if initErr != nil {
		return nil, initErr
	}

	return defaultFinder, nil
}

// New creates a new timezone finder using the default (full) dataset.
func New() (*Finder, error) {
	finder, err := tzf.NewDefaultFinder()
	if err != nil {
		return nil, err
	}

	return &Finder{finder: finder}, nil
}

// NewLite creates a new timezone finder using the smaller "lite" dataset.
// This uses less memory but is less precise at boundaries.
func NewLite() (*Finder, error) {
	finder, err := tzf.NewDefaultFinder()
	if err != nil {
		return nil, err
	}

	return &Finder{finder: finder}, nil
}

// TimezoneAt returns the timezone name for the given coordinates.
// Returns an IANA timezone name like "America/New_York".
func (f *Finder) TimezoneAt(lat, lon float64) (string, error) {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return "", ErrInvalidCoord
	}

	tz := f.finder.GetTimezoneName(lon, lat)
	if tz == "" {
		return "", ErrTimezoneNotFound
	}

	return tz, nil
}

// TimezoneAtCoord returns the timezone name for the given coordinate.
func (f *Finder) TimezoneAtCoord(c geo.Coord) (string, error) {
	return f.TimezoneAt(c.Lat, c.Lon)
}

// LocationAt returns the *time.Location for the given coordinates.
func (f *Finder) LocationAt(lat, lon float64) (*time.Location, error) {
	tz, err := f.TimezoneAt(lat, lon)
	if err != nil {
		return nil, err
	}

	return time.LoadLocation(tz)
}

// LocationAtCoord returns the *time.Location for the given coordinate.
func (f *Finder) LocationAtCoord(c geo.Coord) (*time.Location, error) {
	return f.LocationAt(c.Lat, c.Lon)
}

// TimeAt returns the current time at the given coordinates.
func (f *Finder) TimeAt(lat, lon float64) (time.Time, error) {
	loc, err := f.LocationAt(lat, lon)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().In(loc), nil
}

// TimeAtCoord returns the current time at the given coordinate.
func (f *Finder) TimeAtCoord(c geo.Coord) (time.Time, error) {
	return f.TimeAt(c.Lat, c.Lon)
}

// TimezoneInfo contains detailed timezone information.
type TimezoneInfo struct {
	// Name is the IANA timezone name (e.g., "America/New_York").
	Name string `json:"name"`

	// Abbreviation is the current timezone abbreviation (e.g., "EST", "EDT").
	Abbreviation string `json:"abbreviation"`

	// Offset is the current UTC offset in seconds.
	Offset int `json:"offset"`

	// OffsetString is the formatted offset (e.g., "-05:00").
	OffsetString string `json:"offset_string"`

	// IsDST indicates if daylight saving time is in effect.
	IsDST bool `json:"is_dst"`

	// CurrentTime is the current time in this timezone.
	CurrentTime time.Time `json:"current_time"`
}

// InfoAt returns detailed timezone information for the given coordinates.
func (f *Finder) InfoAt(lat, lon float64) (*TimezoneInfo, error) {
	tz, err := f.TimezoneAt(lat, lon)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(loc)
	abbrev, offset := now.Zone()

	return &TimezoneInfo{
		Name:         tz,
		Abbreviation: abbrev,
		Offset:       offset,
		OffsetString: formatOffset(offset),
		IsDST:        now.IsDST(),
		CurrentTime:  now,
	}, nil
}

// InfoAtCoord returns detailed timezone information for the given coordinate.
func (f *Finder) InfoAtCoord(c geo.Coord) (*TimezoneInfo, error) {
	return f.InfoAt(c.Lat, c.Lon)
}

// formatOffset formats a UTC offset in seconds to a string like "+05:30" or "-08:00".
func formatOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60

	return sign + padZero(hours) + ":" + padZero(minutes)
}

func padZero(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// AllTimezones returns all timezone names.
func (f *Finder) AllTimezones() []string {
	return f.finder.TimezoneNames()
}

// Package-level functions using the default finder

// TimezoneAt returns the timezone name for the given coordinates.
func TimezoneAt(lat, lon float64) (string, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return "", err
	}
	return finder.TimezoneAt(lat, lon)
}

// TimezoneAtCoord returns the timezone name for the given coordinate.
func TimezoneAtCoord(c geo.Coord) (string, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return "", err
	}
	return finder.TimezoneAtCoord(c)
}

// LocationAt returns the *time.Location for the given coordinates.
func LocationAt(lat, lon float64) (*time.Location, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return nil, err
	}
	return finder.LocationAt(lat, lon)
}

// LocationAtCoord returns the *time.Location for the given coordinate.
func LocationAtCoord(c geo.Coord) (*time.Location, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return nil, err
	}
	return finder.LocationAtCoord(c)
}

// TimeAt returns the current time at the given coordinates.
func TimeAt(lat, lon float64) (time.Time, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return time.Time{}, err
	}
	return finder.TimeAt(lat, lon)
}

// TimeAtCoord returns the current time at the given coordinate.
func TimeAtCoord(c geo.Coord) (time.Time, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return time.Time{}, err
	}
	return finder.TimeAtCoord(c)
}

// InfoAt returns detailed timezone information for the given coordinates.
func InfoAt(lat, lon float64) (*TimezoneInfo, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return nil, err
	}
	return finder.InfoAt(lat, lon)
}

// InfoAtCoord returns detailed timezone information for the given coordinate.
func InfoAtCoord(c geo.Coord) (*TimezoneInfo, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return nil, err
	}
	return finder.InfoAtCoord(c)
}

// AllTimezones returns all timezone names.
func AllTimezones() ([]string, error) {
	finder, err := getDefaultFinder()
	if err != nil {
		return nil, err
	}
	return finder.AllTimezones(), nil
}
