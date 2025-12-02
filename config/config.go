// config/config.go
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// HTTPConfig groups HTTP/HTTPS port and protocol settings.
type HTTPConfig struct {
	HTTPPort  int  `mapstructure:"http_port"`
	HTTPSPort int  `mapstructure:"https_port"`
	UseHTTPS  bool `mapstructure:"use_https"`
}

// TLSConfig groups all TLS / ACME-related settings.
type TLSConfig struct {
	CertFile            string `mapstructure:"cert_file"`
	KeyFile             string `mapstructure:"key_file"`
	UseLetsEncrypt      bool   `mapstructure:"use_lets_encrypt"`
	LetsEncryptEmail    string `mapstructure:"lets_encrypt_email"`
	LetsEncryptCacheDir string `mapstructure:"lets_encrypt_cache_dir"`
	Domain              string `mapstructure:"domain"`

	// LetsEncryptChallenge selects which ACME challenge type to use when
	// UseLetsEncrypt is true. Supported values:
	//   - "http-01" (default; uses an HTTP challenge endpoint)
	//   - "dns-01"  (for use with Route 53 DNS TXT records)
	LetsEncryptChallenge string `mapstructure:"lets_encrypt_challenge"`

	// Route53HostedZoneID is required when using DNS-01 with Route 53 so the
	// ACME client knows which hosted zone to update.
	Route53HostedZoneID string `mapstructure:"route53_hosted_zone_id"`
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
func Load(logger *zap.Logger) (*CoreConfig, error) {
	// 0) Optionally load .env (safe: real env still wins over .env)
	if err := godotenv.Load(); err == nil && logger != nil {
		logger.Info("Loaded .env file")
	}

	// 1) Define flags (only *explicitly set* flags will override)
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
	pflag.String("domain", "", "Domain for TLS or ACME")
	pflag.String("lets_encrypt_challenge", "http-01", "ACME challenge type: http-01 or dns-01")
	pflag.String("route53_hosted_zone_id", "", "Route53 hosted zone ID (for dns-01)")

	// Timeouts
	pflag.String("index_boot_timeout", "120s", "Startup timeout for building DB indexes (e.g., \"90s\", \"2m\")")
	pflag.String("db_connect_timeout", "10s", "Startup timeout for DB connection (e.g., \"10s\", \"30s\")")

	// misc / CORS
	pflag.Bool("enable_compression", true, "Enable HTTP compression")
	pflag.Bool("enable_cors", false, "Enable CORS")

	// CORS lists as JSON strings or arrays
	pflag.String("cors_allowed_origins", "", `JSON array of origins, e.g. '["https://a.example","https://b.example"]'`)
	pflag.String("cors_allowed_methods", "", `JSON array of methods, e.g. '["GET","POST"]'`)
	pflag.String("cors_allowed_headers", "", `JSON array of headers, e.g. '["Accept","Authorization"]'`)
	pflag.String("cors_exposed_headers", "", `JSON array of headers, e.g. '["Link"]'`)
	pflag.Bool("cors_allow_credentials", false, "CORS: allow credentials")
	pflag.Int("cors_max_age", 0, "CORS: max age seconds (0 disables cache)")

	pflag.Int64("max_request_body_bytes", 2<<20, "Max HTTP request body size in bytes (0 = unlimited)")
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
	); err != nil {
		return nil, err
	}

	// 7) Build struct
	var cfg CoreConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode core config: %w", err)
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

	// 8) Validate
	if err := validateCoreConfig(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func allKeys() []string {
	return []string{
		"env", "log_level",
		"http_port", "https_port", "use_https",
		"use_lets_encrypt", "lets_encrypt_email", "lets_encrypt_cache_dir",
		"cert_file", "key_file", "domain",
		"lets_encrypt_challenge", "route53_hosted_zone_id",
		"db_connect_timeout", "index_boot_timeout",
		"enable_compression",
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

	v.SetDefault("use_lets_encrypt", false)
	v.SetDefault("lets_encrypt_email", "")
	v.SetDefault("lets_encrypt_cache_dir", "letsencrypt-cache")
	v.SetDefault("cert_file", "")
	v.SetDefault("key_file", "")
	v.SetDefault("domain", "")
	v.SetDefault("lets_encrypt_challenge", "http-01")
	v.SetDefault("route53_hosted_zone_id", "")

	v.SetDefault("db_connect_timeout", "10s")
	v.SetDefault("index_boot_timeout", "120s")

	v.SetDefault("enable_compression", true)

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
				continue
			}
			var arr []string
			if err := json.Unmarshal([]byte(s), &arr); err != nil {
				return fmt.Errorf("config key %q expects a JSON array string, got %q: %w", key, s, err)
			}
			v.Set(key, arr)
		case []interface{}:
			arr := make([]string, 0, len(t))
			for _, e := range t {
				arr = append(arr, fmt.Sprint(e))
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

	// TLS / ACME consistency
	if cfg.TLS.UseLetsEncrypt && !cfg.HTTP.UseHTTPS {
		invalid = append(invalid, "use_lets_encrypt=true requires use_https=true")
	}
	if cfg.TLS.UseLetsEncrypt && (strings.TrimSpace(cfg.TLS.CertFile) != "" || strings.TrimSpace(cfg.TLS.KeyFile) != "") {
		invalid = append(invalid, "use_lets_encrypt=true cannot be combined with cert_file/key_file")
	}

	if cfg.TLS.UseLetsEncrypt {
		if strings.TrimSpace(cfg.TLS.Domain) == "" {
			missing = append(missing, "WAFFLE_DOMAIN (or --domain) for Let's Encrypt")
		}
		if s := strings.TrimSpace(cfg.TLS.LetsEncryptEmail); s == "" {
			missing = append(missing, "WAFFLE_LETS_ENCRYPT_EMAIL (or --lets_encrypt_email)")
		} else if !strings.Contains(cfg.TLS.LetsEncryptEmail, "@") {
			invalid = append(invalid, "lets_encrypt_email must look like an email address")
		}

		chal := strings.ToLower(strings.TrimSpace(cfg.TLS.LetsEncryptChallenge))
		if chal != "http-01" && chal != "dns-01" {
			invalid = append(invalid, "lets_encrypt_challenge must be \"http-01\" or \"dns-01\"")
		}
		if chal == "dns-01" && strings.TrimSpace(cfg.TLS.Route53HostedZoneID) == "" {
			missing = append(missing, "WAFFLE_ROUTE53_HOSTED_ZONE_ID (or --route53_hosted_zone_id) for dns-01")
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
		for _, o := range cfg.CORS.CORSAllowedOrigins {
			if o == "*" && cfg.CORS.CORSAllowCredentials {
				invalid = append(invalid, `CORS: cannot use "*" in cors_allowed_origins when cors_allow_credentials=true`)
				break
			}
		}
		if cfg.CORS.CORSMaxAge < 0 {
			invalid = append(invalid, "CORS: cors_max_age must be >= 0")
		}
	}

	// DB timeouts sanity
	if cfg.DBConnectTimeout <= 0 {
		invalid = append(invalid, "db_connect_timeout must be > 0")
	}
	if cfg.IndexBootTimeout <= 0 {
		invalid = append(invalid, "index_boot_timeout must be > 0")
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
