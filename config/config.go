// config/config.go
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HTTPConfig groups HTTP/HTTPS port and protocol settings.
type HTTPConfig struct {
	HTTPPort  int  `mapstructure:"http_port"`
	HTTPSPort int  `mapstructure:"https_port"`
	UseHTTPS  bool `mapstructure:"use_https"`

	// Server timeouts
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}

// TLSConfig groups all TLS / ACME-related settings.
type TLSConfig struct {
	CertFile            string `mapstructure:"cert_file"`
	KeyFile             string `mapstructure:"key_file"`
	UseLetsEncrypt      bool   `mapstructure:"use_lets_encrypt"`
	LetsEncryptEmail    string `mapstructure:"lets_encrypt_email"`
	LetsEncryptCacheDir string `mapstructure:"lets_encrypt_cache_dir"`

	// Domain is the single domain for TLS/ACME (backward compatible).
	// Use Domains for multiple domains on a single certificate.
	// Cannot specify both Domain and Domains.
	Domain string `mapstructure:"domain"`

	// Domains is a list of domains for a single certificate (e.g., ["example.com", "*.example.com"]).
	// Use this for wildcard + apex domain certificates.
	// Cannot specify both Domain and Domains.
	Domains []string `mapstructure:"domains"`

	// LetsEncryptChallenge selects which ACME challenge type to use when
	// UseLetsEncrypt is true. Supported values:
	//   - "http-01" (default; uses an HTTP challenge endpoint)
	//   - "dns-01"  (for use with Route 53 DNS TXT records; required for wildcards)
	LetsEncryptChallenge string `mapstructure:"lets_encrypt_challenge"`

	// Route53HostedZoneID is required when using DNS-01 with Route 53 so the
	// ACME client knows which hosted zone to update.
	Route53HostedZoneID string `mapstructure:"route53_hosted_zone_id"`

	// ACMEDirectoryURL is the ACME directory URL to use. Defaults to Let's Encrypt
	// production for prod env, staging for other environments. Common values:
	//   - Production: https://acme-v02.api.letsencrypt.org/directory
	//   - Staging:    https://acme-staging-v02.api.letsencrypt.org/directory
	ACMEDirectoryURL string `mapstructure:"acme_directory_url"`
}

// EffectiveDomains returns the list of domains for certificate generation.
// It returns Domains if set, otherwise a single-element slice containing Domain.
// Returns nil if neither is configured.
func (t TLSConfig) EffectiveDomains() []string {
	if len(t.Domains) > 0 {
		return t.Domains
	}
	if strings.TrimSpace(t.Domain) != "" {
		return []string{t.Domain}
	}
	return nil
}

// CORSConfig groups all CORS behavior and lists.
type CORSConfig struct {
	EnableCORS           bool     `mapstructure:"enable_cors"`
	CORSAllowedOrigins   []string `mapstructure:"cors_allowed_origins"`
	CORSAllowedMethods   []string `mapstructure:"cors_allowed_methods"`
	CORSAllowedHeaders   []string `mapstructure:"cors_allowed_headers"`
	CORSExposedHeaders   []string `mapstructure:"cors_exposed_headers"`
	CORSAllowCredentials bool     `mapstructure:"cors_allow_credentials"`
	CORSMaxAge           int      `mapstructure:"cors_max_age"`
}

// CoreConfig holds the core configuration shared by all WAFFLE-based services.
type CoreConfig struct {
	// runtime
	Env      string `mapstructure:"env"`       // "dev" | "prod"
	LogLevel string `mapstructure:"log_level"` // debug, info, warn, error …

	// grouped config
	HTTP HTTPConfig `mapstructure:",squash"`
	TLS  TLSConfig  `mapstructure:",squash"`
	CORS CORSConfig `mapstructure:",squash"`

	// DB-related timeouts (no URIs/DB names here)
	DBConnectTimeout time.Duration `mapstructure:"db_connect_timeout"`
	IndexBootTimeout time.Duration `mapstructure:"index_boot_timeout"`

	// HTTP behavior
	MaxRequestBodyBytes int64 `mapstructure:"max_request_body_bytes"`

	// misc
	EnableCompression bool `mapstructure:"enable_compression"`
	CompressionLevel  int  `mapstructure:"compression_level"` // 1-9, default 5
}

// Dump returns a pretty, redacted JSON string of the config for debugging.
// Never logs secrets; use at debug level only.
func (c CoreConfig) Dump() string {
	s := c.redactedCopy()
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}

func (c CoreConfig) redactedCopy() CoreConfig {
	cp := c
	// Nothing sensitive yet in CoreConfig (no API keys, no URIs).
	// If later we add secrets to core, redact them here.
	return cp
}

// Load merges defaults → config.* file(s) → env vars → explicit flags into one CoreConfig.
// Final precedence (highest wins): flags(explicit) > env > config > defaults.
//
// For apps that need their own config keys, use LoadWithAppConfig instead.
func Load(logger *zap.Logger) (*CoreConfig, error) {
	core, _, err := LoadWithAppConfig(logger, "", nil)
	return core, err
}

// LoadWithAppConfig loads both WAFFLE core config and app-specific config.
// It merges defaults → config.* file(s) → env vars → explicit flags.
// Final precedence (highest wins): flags(explicit) > env > config > defaults.
//
// The appEnvPrefix is used for app config environment variables. For example,
// if appEnvPrefix is "STRATAHUB" and an AppKey has Name "session_name", the
// environment variable would be "STRATAHUB_SESSION_NAME".
//
// Config file keys and CLI flags use the key name directly (e.g., "session_name").
//
// Example:
//
//	appKeys := []config.AppKey{
//	    {Name: "mongo_uri", Default: "mongodb://localhost:27017", Desc: "MongoDB connection URI"},
//	    {Name: "session_name", Default: "myapp-session", Desc: "Session cookie name"},
//	}
//	coreCfg, appCfg, err := config.LoadWithAppConfig(logger, "MYAPP", appKeys)
//	mongoURI := appCfg.String("mongo_uri")
func LoadWithAppConfig(logger *zap.Logger, appEnvPrefix string, appKeys []AppKey) (*CoreConfig, AppConfigValues, error) {
	// 0) Optionally load .env (safe: real env still wins over .env)
	if err := godotenv.Load(); err == nil && logger != nil {
		logger.Info("Loaded .env file")
	}

	// 1) Define WAFFLE core flags (only *explicitly set* flags will override)
	pflag.String("env", "dev", `Runtime environment "dev"|"prod"`)
	pflag.String("log_level", "debug", "Log level")

	pflag.Int("http_port", 8080, "HTTP port")
	pflag.Int("https_port", 443, "HTTPS port")
	pflag.Bool("use_https", false, "Serve HTTPS")

	// TLS / Let’s Encrypt
	pflag.Bool("use_lets_encrypt", false, "Use Let's Encrypt")
	pflag.String("lets_encrypt_email", "", "ACME account e-mail")
	pflag.String("lets_encrypt_cache_dir", "letsencrypt-cache", "ACME cache dir")
	pflag.String("cert_file", "", "TLS cert file (manual TLS)")
	pflag.String("key_file", "", "TLS key file  (manual TLS)")
	pflag.String("domain", "", "Domain for TLS or ACME (single domain, backward compatible)")
	pflag.String("domains", "", `JSON array of domains for multi-domain cert, e.g. '["example.com", "*.example.com"]'`)
	pflag.String("lets_encrypt_challenge", "http-01", "ACME challenge type: http-01 or dns-01")
	pflag.String("route53_hosted_zone_id", "", "Route53 hosted zone ID (for dns-01)")
	pflag.String("acme_directory_url", "", "ACME directory URL (defaults to Let's Encrypt staging/prod based on env)")

	// DB Timeouts
	pflag.String("index_boot_timeout", "120s", "Startup timeout for building DB indexes (e.g., \"90s\", \"2m\")")
	pflag.String("db_connect_timeout", "10s", "Startup timeout for DB connection (e.g., \"10s\", \"30s\")")

	// HTTP Server Timeouts
	pflag.String("read_timeout", "15s", "HTTP server read timeout (e.g., \"15s\", \"30s\")")
	pflag.String("read_header_timeout", "10s", "HTTP server read header timeout (e.g., \"10s\")")
	pflag.String("write_timeout", "60s", "HTTP server write timeout (e.g., \"60s\", \"2m\")")
	pflag.String("idle_timeout", "120s", "HTTP server idle timeout (e.g., \"120s\", \"2m\")")
	pflag.String("shutdown_timeout", "15s", "Graceful shutdown timeout (e.g., \"15s\", \"30s\")")

	// misc / CORS
	pflag.Bool("enable_compression", true, "Enable HTTP compression")
	pflag.Int("compression_level", 5, "Compression level (1=fastest, 9=best compression)")
	pflag.Bool("enable_cors", false, "Enable CORS")

	// CORS lists as JSON strings or arrays
	pflag.String("cors_allowed_origins", "", `JSON array of origins, e.g. '["https://a.example","https://b.example"]'`)
	pflag.String("cors_allowed_methods", "", `JSON array of methods, e.g. '["GET","POST"]'`)
	pflag.String("cors_allowed_headers", "", `JSON array of headers, e.g. '["Accept","Authorization"]'`)
	pflag.String("cors_exposed_headers", "", `JSON array of headers, e.g. '["Link"]'`)
	pflag.Bool("cors_allow_credentials", false, "CORS: allow credentials")
	pflag.Int("cors_max_age", 0, "CORS: max age seconds (0 disables cache)")

	pflag.Int64("max_request_body_bytes", 2<<20, "Max HTTP request body size in bytes (0 = no limit, -1 = reject all)")

	// 1b) Register app-specific flags
	if err := registerAppFlags(appKeys); err != nil {
		return nil, nil, fmt.Errorf("failed to register app flags: %w", err)
	}

	pflag.Parse()

	// 2) Viper + env
	v := viper.New()
	v.SetEnvPrefix("WAFFLE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Bind env for all keys so Unmarshal sees them.
	for _, k := range allKeys() {
		_ = v.BindEnv(k)
	}

	// 3) Optional config.* files (yaml|yml|json|toml)
	for _, ext := range [...]string{"yaml", "yml", "json", "toml"} {
		file := "config." + ext
		if _, err := os.Stat(file); err != nil {
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			if logger != nil {
				logger.Warn("cannot read config file", zap.String("file", file), zap.Error(err))
			}
			continue
		}
		v.SetConfigType(ext)
		if err := v.MergeConfig(bytes.NewReader(b)); err != nil {
			if logger != nil {
				logger.Warn("cannot decode config file", zap.String("file", file), zap.Error(err))
			}
			continue
		}
		if logger != nil {
			logger.Info("Loaded config file", zap.String("file", file))
		}
	}

	// 4) Defaults (lowest precedence)
	setDefaults(v)

	// 5) Apply *explicit* flags (highest precedence)
	pflag.CommandLine.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			_ = v.BindPFlag(f.Name, f)
		}
	})

	// 6) Normalize list keys (accept JSON strings → []string)
	if err := normalizeListKeys(logger, v,
		"cors_allowed_origins",
		"cors_allowed_methods",
		"cors_allowed_headers",
		"cors_exposed_headers",
		"domains",
	); err != nil {
		return nil, nil, err
	}

	// 7) Build struct
	var cfg CoreConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, fmt.Errorf("unable to decode core config: %w", err)
	}

	// Parse durations
	dur, err := parseDurationFlexible(v.Get("index_boot_timeout"), 120*time.Second)
	if err != nil && logger != nil {
		logger.Warn("invalid index_boot_timeout; using default 120s",
			zap.Any("value", v.Get("index_boot_timeout")), zap.Error(err))
	}
	cfg.IndexBootTimeout = dur

	dbDur, err := parseDurationFlexible(v.Get("db_connect_timeout"), 10*time.Second)
	if err != nil && logger != nil {
		logger.Warn("invalid db_connect_timeout; using default 10s",
			zap.Any("value", v.Get("db_connect_timeout")), zap.Error(err))
	}
	cfg.DBConnectTimeout = dbDur

	// Parse HTTP server timeouts
	cfg.HTTP.ReadTimeout = parseDurationWithDefault(logger, v, "read_timeout", 15*time.Second)
	cfg.HTTP.ReadHeaderTimeout = parseDurationWithDefault(logger, v, "read_header_timeout", 10*time.Second)
	cfg.HTTP.WriteTimeout = parseDurationWithDefault(logger, v, "write_timeout", 60*time.Second)
	cfg.HTTP.IdleTimeout = parseDurationWithDefault(logger, v, "idle_timeout", 120*time.Second)
	cfg.HTTP.ShutdownTimeout = parseDurationWithDefault(logger, v, "shutdown_timeout", 15*time.Second)

	// Set ACME directory URL default based on environment if not explicitly configured
	if cfg.TLS.UseLetsEncrypt && cfg.TLS.ACMEDirectoryURL == "" {
		if cfg.Env == "prod" {
			cfg.TLS.ACMEDirectoryURL = "https://acme-v02.api.letsencrypt.org/directory"
		} else {
			cfg.TLS.ACMEDirectoryURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
			if logger != nil {
				logger.Info("using Let's Encrypt staging (non-prod env); set acme_directory_url for production")
			}
		}
	}

	// 8) Normalize values before validation so validators see canonical forms.
	// LetsEncryptChallenge is case-insensitive but should be lowercase
	// for consistent comparisons in validation and server code.
	cfg.TLS.LetsEncryptChallenge = strings.ToLower(strings.TrimSpace(cfg.TLS.LetsEncryptChallenge))

	// 9) Validate core config
	if err := validateCoreConfig(cfg); err != nil {
		return nil, nil, err
	}

	// 10) Load app config
	appCfg := loadAppConfig(logger, v, appEnvPrefix, appKeys)

	return &cfg, appCfg, nil
}

func allKeys() []string {
	return []string{
		"env", "log_level",
		"http_port", "https_port", "use_https",
		"read_timeout", "read_header_timeout", "write_timeout", "idle_timeout", "shutdown_timeout",
		"use_lets_encrypt", "lets_encrypt_email", "lets_encrypt_cache_dir",
		"cert_file", "key_file", "domain", "domains",
		"lets_encrypt_challenge", "route53_hosted_zone_id", "acme_directory_url",
		"db_connect_timeout", "index_boot_timeout",
		"enable_compression", "compression_level",
		"enable_cors",
		"cors_allowed_origins", "cors_allowed_methods", "cors_allowed_headers",
		"cors_exposed_headers", "cors_allow_credentials", "cors_max_age",
		"max_request_body_bytes",
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("env", "dev")
	v.SetDefault("log_level", "debug")

	v.SetDefault("http_port", 8080)
	v.SetDefault("https_port", 443)
	v.SetDefault("use_https", false)

	// HTTP server timeouts
	v.SetDefault("read_timeout", "15s")
	v.SetDefault("read_header_timeout", "10s")
	v.SetDefault("write_timeout", "60s")
	v.SetDefault("idle_timeout", "120s")
	v.SetDefault("shutdown_timeout", "15s")

	v.SetDefault("use_lets_encrypt", false)
	v.SetDefault("lets_encrypt_email", "")
	v.SetDefault("lets_encrypt_cache_dir", "letsencrypt-cache")
	v.SetDefault("cert_file", "")
	v.SetDefault("key_file", "")
	v.SetDefault("domain", "")
	v.SetDefault("domains", []string{})
	v.SetDefault("lets_encrypt_challenge", "http-01")
	v.SetDefault("route53_hosted_zone_id", "")
	v.SetDefault("acme_directory_url", "") // Empty means auto-detect based on env

	v.SetDefault("db_connect_timeout", "10s")
	v.SetDefault("index_boot_timeout", "120s")

	v.SetDefault("enable_compression", true)
	v.SetDefault("compression_level", 5)

	// Neutral CORS defaults
	v.SetDefault("enable_cors", false)
	v.SetDefault("cors_allowed_origins", []string{})
	v.SetDefault("cors_allowed_methods", []string{})
	v.SetDefault("cors_allowed_headers", []string{})
	v.SetDefault("cors_exposed_headers", []string{})
	v.SetDefault("cors_allow_credentials", false)
	v.SetDefault("cors_max_age", 0)

	v.SetDefault("max_request_body_bytes", int64(2<<20))
}

// normalizeListKeys coerces JSON-string values into []string for the given keys.
func normalizeListKeys(logger *zap.Logger, v *viper.Viper, keys ...string) error {
	for _, key := range keys {
		val := v.Get(key)
		switch t := val.(type) {
		case string:
			s := strings.TrimSpace(t)
			if s == "" {
				// Empty string should be normalized to empty slice for consistency
				v.Set(key, []string{})
				continue
			}
			var arr []string
			if err := json.Unmarshal([]byte(s), &arr); err != nil {
				return fmt.Errorf("config key %q expects a JSON array string, got %q: %w", key, s, err)
			}
			v.Set(key, arr)
		case []interface{}:
			arr := make([]string, 0, len(t))
			for i, e := range t {
				s, ok := e.(string)
				if !ok {
					// Non-string element in array - warn and coerce
					if logger != nil {
						logger.Warn("non-string element in config array, coercing to string",
							zap.String("key", key),
							zap.Int("index", i),
							zap.Any("value", e),
							zap.String("type", fmt.Sprintf("%T", e)))
					}
					s = fmt.Sprint(e)
				}
				arr = append(arr, s)
			}
			v.Set(key, arr)
		case []string, nil:
			// already correct or unset
		default:
			if logger != nil {
				logger.Warn("unexpected type for list key; expected JSON array/string",
					zap.String("key", key), zap.Any("value", t))
			}
		}
	}
	return nil
}

func validateCoreConfig(cfg CoreConfig) error {
	var missing []string
	var invalid []string

	// Log level validation (case-insensitive)
	if cfg.LogLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(strings.ToLower(cfg.LogLevel))); err != nil {
			invalid = append(invalid, fmt.Sprintf("log_level %q is invalid; valid levels: debug, info, warn, error, dpanic, panic, fatal", cfg.LogLevel))
		}
	}

	// TLS / ACME consistency
	if cfg.TLS.UseLetsEncrypt && !cfg.HTTP.UseHTTPS {
		invalid = append(invalid, "use_lets_encrypt=true requires use_https=true")
	}
	if cfg.TLS.UseLetsEncrypt && (strings.TrimSpace(cfg.TLS.CertFile) != "" || strings.TrimSpace(cfg.TLS.KeyFile) != "") {
		invalid = append(invalid, "use_lets_encrypt=true cannot be combined with cert_file/key_file")
	}

	if cfg.TLS.UseLetsEncrypt {
		// Check domain/domains configuration
		hasSingleDomain := strings.TrimSpace(cfg.TLS.Domain) != ""
		hasMultipleDomains := len(cfg.TLS.Domains) > 0

		if hasSingleDomain && hasMultipleDomains {
			invalid = append(invalid, "cannot specify both domain and domains; use one or the other")
		} else if !hasSingleDomain && !hasMultipleDomains {
			missing = append(missing, "WAFFLE_DOMAIN or WAFFLE_DOMAINS for Let's Encrypt")
		}

		if s := strings.TrimSpace(cfg.TLS.LetsEncryptEmail); s == "" {
			missing = append(missing, "WAFFLE_LETS_ENCRYPT_EMAIL (or --lets_encrypt_email)")
		} else if !isValidEmail(cfg.TLS.LetsEncryptEmail) {
			invalid = append(invalid, "lets_encrypt_email must be a valid email address (e.g., user@example.com)")
		}

		chal := strings.ToLower(strings.TrimSpace(cfg.TLS.LetsEncryptChallenge))
		if chal != "http-01" && chal != "dns-01" {
			invalid = append(invalid, "lets_encrypt_challenge must be \"http-01\" or \"dns-01\"")
		}
		if chal == "dns-01" && strings.TrimSpace(cfg.TLS.Route53HostedZoneID) == "" {
			missing = append(missing, "WAFFLE_ROUTE53_HOSTED_ZONE_ID (or --route53_hosted_zone_id) for dns-01")
		}

		// Check for wildcards - they require dns-01 challenge
		allDomains := cfg.TLS.Domains
		if hasSingleDomain {
			allDomains = []string{cfg.TLS.Domain}
		}
		for _, d := range allDomains {
			if strings.HasPrefix(d, "*.") && chal != "dns-01" {
				invalid = append(invalid, fmt.Sprintf("wildcard domain %q requires dns-01 challenge (set lets_encrypt_challenge=dns-01)", d))
				break
			}
		}

		// Validate ACME directory URL format if explicitly provided.
		// Note: We only validate URL format here, not reachability. Network
		// connectivity is checked at runtime when the ACME client initializes,
		// which provides better error messages in context.
		if cfg.TLS.ACMEDirectoryURL != "" {
			u, err := url.Parse(cfg.TLS.ACMEDirectoryURL)
			if err != nil {
				invalid = append(invalid, fmt.Sprintf("acme_directory_url is not a valid URL: %v", err))
			} else if u.Scheme != "https" {
				invalid = append(invalid, "acme_directory_url must use HTTPS scheme")
			} else if u.Host == "" {
				invalid = append(invalid, "acme_directory_url must include a host")
			}
		}
	}

	// Manual TLS requirements
	if cfg.HTTP.UseHTTPS && !cfg.TLS.UseLetsEncrypt {
		if strings.TrimSpace(cfg.TLS.CertFile) == "" || strings.TrimSpace(cfg.TLS.KeyFile) == "" {
			missing = append(missing, "WAFFLE_CERT_FILE and WAFFLE_KEY_FILE (or --cert_file/--key_file) for manual TLS")
		}
	}

	// Port sanity
	if cfg.HTTP.HTTPPort <= 0 || cfg.HTTP.HTTPPort > 65535 {
		invalid = append(invalid, "http_port must be in 1..65535")
	}
	if cfg.HTTP.HTTPSPort <= 0 || cfg.HTTP.HTTPSPort > 65535 {
		invalid = append(invalid, "https_port must be in 1..65535")
	}
	if cfg.HTTP.UseHTTPS {
		if cfg.HTTP.HTTPPort == cfg.HTTP.HTTPSPort {
			invalid = append(invalid, "http_port and https_port cannot be equal when use_https=true")
		}
		if cfg.HTTP.HTTPSPort == 80 {
			invalid = append(invalid, "https_port cannot be 80; port 80 is used by the ACME/redirect server")
		}
	}

	// CORS sanity
	if cfg.CORS.EnableCORS {
		if len(cfg.CORS.CORSAllowedOrigins) == 0 {
			missing = append(missing, "CORS: cors_allowed_origins (JSON array) required when enable_cors=true")
		}
		if len(cfg.CORS.CORSAllowedMethods) == 0 {
			missing = append(missing, "CORS: cors_allowed_methods (JSON array) required when enable_cors=true")
		}
		// Validate HTTP methods
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true,
			"PATCH": true, "HEAD": true, "OPTIONS": true, "TRACE": true, "CONNECT": true,
		}
		for _, method := range cfg.CORS.CORSAllowedMethods {
			if !validMethods[strings.ToUpper(method)] {
				invalid = append(invalid, fmt.Sprintf("CORS: invalid HTTP method %q", method))
			}
		}
		for _, o := range cfg.CORS.CORSAllowedOrigins {
			if o == "*" {
				if cfg.CORS.CORSAllowCredentials {
					invalid = append(invalid, `CORS: cannot use "*" in cors_allowed_origins when cors_allow_credentials=true`)
				}
				continue
			}
			// Validate non-wildcard origins are proper URLs with scheme
			parsed, err := url.Parse(o)
			if err != nil {
				invalid = append(invalid, fmt.Sprintf("CORS: invalid origin URL %q: %v", o, err))
			} else if parsed.Scheme == "" || parsed.Host == "" {
				invalid = append(invalid, fmt.Sprintf("CORS: origin %q must be a full URL with scheme (e.g., https://example.com)", o))
			} else if parsed.Scheme != "http" && parsed.Scheme != "https" {
				invalid = append(invalid, fmt.Sprintf("CORS: origin %q has invalid scheme %q (must be http or https)", o, parsed.Scheme))
			}
		}
		if cfg.CORS.CORSMaxAge < 0 {
			invalid = append(invalid, "CORS: cors_max_age must be >= 0")
		}
		// Most browsers cap max-age (Chrome: 2 hours, Firefox: 24 hours).
		// Values beyond a week are almost certainly configuration errors.
		const maxCORSMaxAge = 7 * 24 * 60 * 60 // 1 week in seconds
		if cfg.CORS.CORSMaxAge > maxCORSMaxAge {
			invalid = append(invalid, fmt.Sprintf("CORS: cors_max_age %d exceeds maximum of %d seconds (1 week)", cfg.CORS.CORSMaxAge, maxCORSMaxAge))
		}
	}

	// DB timeouts sanity
	if cfg.DBConnectTimeout <= 0 {
		invalid = append(invalid, "db_connect_timeout must be > 0")
	}
	if cfg.IndexBootTimeout <= 0 {
		invalid = append(invalid, "index_boot_timeout must be > 0")
	}

	// Compression level sanity - validate even if disabled to catch config errors early
	if cfg.CompressionLevel != 0 && (cfg.CompressionLevel < 1 || cfg.CompressionLevel > 9) {
		invalid = append(invalid, "compression_level must be between 1 and 9 (or 0 for default)")
	}

	// MaxRequestBodyBytes validation: negative values other than -1 are invalid.
	// -1 means reject all request bodies, 0 means no limit, positive means limit.
	if cfg.MaxRequestBodyBytes < -1 {
		invalid = append(invalid, "max_request_body_bytes must be >= -1 (-1 = reject all, 0 = no limit)")
	}

	// HTTP timeout sanity checks
	if cfg.HTTP.ReadTimeout <= 0 {
		invalid = append(invalid, "read_timeout must be > 0")
	}
	if cfg.HTTP.ReadHeaderTimeout <= 0 {
		invalid = append(invalid, "read_header_timeout must be > 0 (required for Slowloris protection)")
	}
	if cfg.HTTP.WriteTimeout <= 0 {
		invalid = append(invalid, "write_timeout must be > 0")
	}
	if cfg.HTTP.IdleTimeout <= 0 {
		invalid = append(invalid, "idle_timeout must be > 0")
	}
	if cfg.HTTP.ShutdownTimeout <= 0 {
		invalid = append(invalid, "shutdown_timeout must be > 0")
	}

	// Timeout consistency: read_header_timeout should not exceed read_timeout
	if cfg.HTTP.ReadHeaderTimeout > 0 && cfg.HTTP.ReadTimeout > 0 {
		if cfg.HTTP.ReadHeaderTimeout > cfg.HTTP.ReadTimeout {
			invalid = append(invalid, "read_header_timeout should not exceed read_timeout")
		}
	}

	if len(missing) == 0 && len(invalid) == 0 {
		return nil
	}

	var parts []string
	if len(missing) > 0 {
		parts = append(parts, "missing: "+strings.Join(missing, ", "))
	}
	if len(invalid) > 0 {
		parts = append(parts, "invalid: "+strings.Join(invalid, ", "))
	}
	return fmt.Errorf("core configuration errors: %s", strings.Join(parts, " | "))
}

// validateMongoURI is left out here on purpose; CoreConfig does not know about URIs.
// Apps can bring their own DB config validation as needed.

// isValidEmail performs basic email validation for ACME account registration.
// It checks for:
// - Non-empty local part and domain
// - Exactly one @ symbol
// - Domain has at least one dot
// - Local part length <= 64, domain length <= 255 (RFC 5321 limits)
// - Domain doesn't start/end with dot or hyphen
// - No spaces or control characters
//
// This is intentionally not RFC 5322 compliant but catches common mistakes.
//
// Note: This is more thorough than pantry/validate.SimpleEmailValid, which only
// checks for @ and a dot in the domain. This stricter validation is appropriate
// for ACME registration where a valid, routable email is required for account
// recovery. For general-purpose UI validation where leniency is preferred,
// consider using SimpleEmailValid instead.
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	// Must have exactly one @
	atIdx := strings.Index(email, "@")
	if atIdx == -1 || atIdx == 0 || atIdx == len(email)-1 {
		return false
	}
	// Check for multiple @
	if strings.Count(email, "@") > 1 {
		return false
	}

	local := email[:atIdx]
	domain := email[atIdx+1:]

	// Local part checks
	if len(local) == 0 || len(local) > 64 {
		return false
	}

	// Domain checks
	if len(domain) == 0 || len(domain) > 255 {
		return false
	}
	// Domain must have at least one dot (e.g., example.com)
	if !strings.Contains(domain, ".") {
		return false
	}
	// Domain can't start or end with a dot or hyphen
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") ||
		strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return false
	}

	// No spaces or control characters in email
	for _, c := range email {
		if c <= 0x20 || c == 0x7f {
			return false
		}
	}

	return true
}
