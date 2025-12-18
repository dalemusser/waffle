// geo/ip/ip.go
package ip

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/dalemusser/waffle/pantry/geo"
	"github.com/oschwald/maxminddb-golang"
)

// Common errors.
var (
	ErrDatabaseNotLoaded = errors.New("geoip: database not loaded")
	ErrInvalidIP         = errors.New("geoip: invalid IP address")
	ErrNotFound          = errors.New("geoip: location not found for IP")
	ErrDatabasePath      = errors.New("geoip: database path required")
)

// Location represents geolocation data for an IP address.
type Location struct {
	// IP is the IP address that was looked up.
	IP string `json:"ip"`

	// Coordinates
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  int     `json:"accuracy_radius,omitempty"` // km

	// Location details
	Continent     string `json:"continent,omitempty"`
	ContinentCode string `json:"continent_code,omitempty"`
	Country       string `json:"country,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
	Region        string `json:"region,omitempty"`
	RegionCode    string `json:"region_code,omitempty"`
	City          string `json:"city,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Timezone      string `json:"timezone,omitempty"`

	// Network info (if available)
	ISP          string `json:"isp,omitempty"`
	Organization string `json:"organization,omitempty"`
	ASN          int    `json:"asn,omitempty"`
	ASOrg        string `json:"as_org,omitempty"`

	// EU status
	IsEU bool `json:"is_eu,omitempty"`
}

// Coord returns the location as a geo.Coord.
func (l *Location) Coord() geo.Coord {
	return geo.Coord{Lat: l.Latitude, Lon: l.Longitude}
}

// HasCoordinates returns true if the location has valid coordinates.
func (l *Location) HasCoordinates() bool {
	return l.Latitude != 0 || l.Longitude != 0
}

// DB is a MaxMind GeoIP database reader.
type DB struct {
	city *maxminddb.Reader
	asn  *maxminddb.Reader
	mu   sync.RWMutex
}

// Config configures the GeoIP database.
type Config struct {
	// CityDBPath is the path to the GeoLite2-City.mmdb file.
	CityDBPath string

	// ASNDBPath is the optional path to the GeoLite2-ASN.mmdb file.
	ASNDBPath string
}

// Open opens the GeoIP databases.
func Open(cfg Config) (*DB, error) {
	if cfg.CityDBPath == "" {
		return nil, ErrDatabasePath
	}

	db := &DB{}

	cityReader, err := maxminddb.Open(cfg.CityDBPath)
	if err != nil {
		return nil, fmt.Errorf("geoip: failed to open city database: %w", err)
	}
	db.city = cityReader

	if cfg.ASNDBPath != "" {
		asnReader, err := maxminddb.Open(cfg.ASNDBPath)
		if err != nil {
			cityReader.Close()
			return nil, fmt.Errorf("geoip: failed to open ASN database: %w", err)
		}
		db.asn = asnReader
	}

	return db, nil
}

// Close closes the databases.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	var errs []error

	if db.city != nil {
		if err := db.city.Close(); err != nil {
			errs = append(errs, err)
		}
		db.city = nil
	}

	if db.asn != nil {
		if err := db.asn.Close(); err != nil {
			errs = append(errs, err)
		}
		db.asn = nil
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// MaxMind database record structures
type cityRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Continent struct {
		Code  string            `maxminddb:"code"`
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
		IsInEU  bool              `maxminddb:"is_in_european_union"`
	} `maxminddb:"country"`
	Location struct {
		Latitude       float64 `maxminddb:"latitude"`
		Longitude      float64 `maxminddb:"longitude"`
		AccuracyRadius int     `maxminddb:"accuracy_radius"`
		TimeZone       string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`
	Subdivisions []struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	Traits struct {
		ISP          string `maxminddb:"isp"`
		Organization string `maxminddb:"organization"`
	} `maxminddb:"traits"`
}

type asnRecord struct {
	AutonomousSystemNumber       int    `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// Lookup looks up the location for an IP address.
func (db *DB) Lookup(ipStr string) (*Location, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.city == nil {
		return nil, ErrDatabaseNotLoaded
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, ErrInvalidIP
	}

	var record cityRecord
	err := db.city.Lookup(ip, &record)
	if err != nil {
		return nil, fmt.Errorf("geoip: lookup failed: %w", err)
	}

	loc := &Location{
		IP:            ipStr,
		Latitude:      record.Location.Latitude,
		Longitude:     record.Location.Longitude,
		Accuracy:      record.Location.AccuracyRadius,
		Continent:     record.Continent.Names["en"],
		ContinentCode: record.Continent.Code,
		Country:       record.Country.Names["en"],
		CountryCode:   record.Country.ISOCode,
		City:          record.City.Names["en"],
		PostalCode:    record.Postal.Code,
		Timezone:      record.Location.TimeZone,
		ISP:           record.Traits.ISP,
		Organization:  record.Traits.Organization,
		IsEU:          record.Country.IsInEU,
	}

	if len(record.Subdivisions) > 0 {
		loc.Region = record.Subdivisions[0].Names["en"]
		loc.RegionCode = record.Subdivisions[0].ISOCode
	}

	// Look up ASN if available
	if db.asn != nil {
		var asn asnRecord
		if err := db.asn.Lookup(ip, &asn); err == nil {
			loc.ASN = asn.AutonomousSystemNumber
			loc.ASOrg = asn.AutonomousSystemOrganization
		}
	}

	return loc, nil
}

// LookupIP looks up the location for a net.IP.
func (db *DB) LookupIP(ip net.IP) (*Location, error) {
	if ip == nil {
		return nil, ErrInvalidIP
	}
	return db.Lookup(ip.String())
}

// Context key for storing location
type contextKey struct{}

var locationKey = contextKey{}

// FromContext retrieves the location from the context.
func FromContext(ctx context.Context) *Location {
	loc, _ := ctx.Value(locationKey).(*Location)
	return loc
}

// WithLocation adds a location to the context.
func WithLocation(ctx context.Context, loc *Location) context.Context {
	return context.WithValue(ctx, locationKey, loc)
}

// Middleware creates HTTP middleware that adds geolocation to the request context.
func (db *DB) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)
		if ip != "" {
			if loc, err := db.Lookup(ip); err == nil {
				r = r.WithContext(WithLocation(r.Context(), loc))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc creates HTTP middleware as a function.
func (db *DB) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)
		if ip != "" {
			if loc, err := db.Lookup(ip); err == nil {
				r = r.WithContext(WithLocation(r.Context(), loc))
			}
		}
		next(w, r)
	}
}

// GetClientIP extracts the client IP from a request.
// Checks common proxy headers first.
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if isValidIP(ip) {
			return ip
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" && isValidIP(xri) {
		return xri
	}

	// Check CF-Connecting-IP (Cloudflare)
	cfip := r.Header.Get("CF-Connecting-IP")
	if cfip != "" && isValidIP(cfip) {
		return cfip
	}

	// Check True-Client-IP (Akamai, Cloudflare)
	tcip := r.Header.Get("True-Client-IP")
	if tcip != "" && isValidIP(tcip) {
		return tcip
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// isValidIP checks if a string is a valid IP address.
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsPrivateIP checks if an IP is in a private range.
func IsPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsLinkLocalUnicast()
}

// Global database instance
var (
	defaultDB   *DB
	defaultDBMu sync.RWMutex
)

// SetDefault sets the default database.
func SetDefault(db *DB) {
	defaultDBMu.Lock()
	defer defaultDBMu.Unlock()
	defaultDB = db
}

// Default returns the default database.
func Default() *DB {
	defaultDBMu.RLock()
	defer defaultDBMu.RUnlock()
	return defaultDB
}

// Lookup uses the default database to look up an IP.
func Lookup(ip string) (*Location, error) {
	db := Default()
	if db == nil {
		return nil, ErrDatabaseNotLoaded
	}
	return db.Lookup(ip)
}

// Middleware creates middleware using the default database.
func Middleware(next http.Handler) http.Handler {
	db := Default()
	if db == nil {
		return next
	}
	return db.Middleware(next)
}
