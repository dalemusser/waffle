// server/acmedns.go
package server

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme"
)

// DNS01Manager manages ACME certificates using DNS-01 challenges via Route 53.
type DNS01Manager struct {
	Domains          []string // One or more domains for the certificate (e.g., ["example.com", "*.example.com"])
	Email            string
	CacheDir         string
	HostedZoneID     string
	ACMEDirectoryURL string
	Logger           *zap.Logger

	client   *acme.Client
	clientMu sync.Mutex // protects client initialization
	r53      *route53.Client
	certMu   sync.RWMutex
	cert     *tls.Certificate
	certExpiry time.Time

	// renewMu ensures only one goroutine performs certificate renewal at a time.
	// Other goroutines will wait and receive the same result.
	//
	// Synchronization protocol:
	// 1. Acquire renewMu.Lock()
	// 2. While renewing == true, call renewCond.Wait() (releases lock, blocks until Broadcast)
	// 3. After waking, re-check certificate validity (another goroutine may have succeeded)
	// 4. If renewal needed, set renewing = true, release lock, and perform renewal
	// 5. After renewal (success or failure), acquire lock, set renewing = false, call Broadcast()
	//
	// IMPORTANT: renewCond.Wait() MUST be called while holding renewMu. The Wait() method
	// atomically releases the lock and suspends the goroutine. When Broadcast() is called,
	// Wait() reacquires the lock before returning. Failing to hold the lock during Wait()
	// will cause a runtime panic.
	renewMu   sync.Mutex
	renewing  bool
	renewCond *sync.Cond

	// dnsMu serializes DNS record operations to prevent rate limit issues
	// with Route 53 API and ensure consistent record state.
	dnsMu sync.Mutex

	// Background renewal
	bgCtx       context.Context
	bgCancel    context.CancelFunc
	bgWg        sync.WaitGroup
}

// acmeAccount represents a cached ACME account.
type acmeAccount struct {
	URI string `json:"uri"`
}

// NewDNS01Manager creates a new DNS-01 certificate manager.
// domains is a list of domains for the certificate (e.g., ["example.com", "*.example.com"]).
// acmeDirectoryURL specifies the ACME directory URL (e.g., Let's Encrypt production or staging).
func NewDNS01Manager(domains []string, email, cacheDir, hostedZoneID, acmeDirectoryURL string, logger *zap.Logger) (*DNS01Manager, error) {
	if len(domains) == 0 {
		return nil, errors.New("dns01: at least one domain is required")
	}
	// Validate all domains and check for duplicates
	seen := make(map[string]bool)
	for _, domain := range domains {
		if err := validateDomainFormat(domain); err != nil {
			return nil, fmt.Errorf("dns01: invalid domain %q: %w", domain, err)
		}
		if seen[domain] {
			return nil, fmt.Errorf("dns01: duplicate domain %q", domain)
		}
		seen[domain] = true
	}
	if email == "" {
		return nil, errors.New("dns01: email is required")
	}
	if cacheDir == "" {
		return nil, errors.New("dns01: cache directory is required")
	}
	if hostedZoneID == "" {
		return nil, errors.New("dns01: Route 53 hosted zone ID is required")
	}
	if acmeDirectoryURL == "" {
		return nil, errors.New("dns01: ACME directory URL is required")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("dns01: create cache dir: %w", err)
	}

	// Load AWS config from environment/credentials with a timeout
	// to prevent indefinite hangs if AWS credential services are unreachable.
	awsCtx, awsCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer awsCancel()

	awsCfg, err := awsconfig.LoadDefaultConfig(awsCtx)
	if err != nil {
		return nil, fmt.Errorf("dns01: load AWS config (check credentials): %w", err)
	}

	m := &DNS01Manager{
		Domains:          domains,
		Email:            email,
		CacheDir:         cacheDir,
		HostedZoneID:     hostedZoneID,
		ACMEDirectoryURL: acmeDirectoryURL,
		Logger:           logger,
		r53:              route53.NewFromConfig(awsCfg),
	}
	m.renewCond = sync.NewCond(&m.renewMu)
	return m, nil
}

// renewalBuffer is how far before expiry we start renewing certificates.
const renewalBuffer = 30 * 24 * time.Hour

// DNS propagation settings
const (
	// dnsPropagationTimeout is how long to wait for DNS TXT records to propagate.
	dnsPropagationTimeout = 5 * time.Minute

	// dnsPropagationInterval is how often to check if DNS has propagated.
	dnsPropagationInterval = 5 * time.Second
)

// GetCertificate returns a TLS certificate for the configured domain.
// It implements the tls.Config.GetCertificate callback.
//
// This method is safe for concurrent use. If multiple goroutines call it
// simultaneously when a renewal is needed, only one will perform the renewal
// while others wait and receive the same result.
//
// Panics during certificate obtainment are recovered and converted to errors
// to prevent crashing the server during TLS handshakes.
//
// Note: hello may be nil in certain edge cases (e.g., when building a self-signed
// cert without client info). This is handled by falling back to a background context.
func (m *DNS01Manager) GetCertificate(hello *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	// Fast path: check if we have a valid cached certificate.
	// The validity check must happen while holding the lock to prevent a TOCTOU race
	// where another goroutine updates certExpiry between our read and comparison.
	m.certMu.RLock()
	cert = m.cert
	expiry := m.certExpiry
	isValid := cert != nil && time.Now().Add(renewalBuffer).Before(expiry)
	m.certMu.RUnlock()

	if isValid {
		return cert, nil
	}

	// Slow path: need to obtain/renew certificate
	// Use condition variable to ensure only one goroutine renews at a time
	m.renewMu.Lock()
	for m.renewing {
		// Another goroutine is already renewing; wait for it to finish
		m.renewCond.Wait()
	}

	// Re-check after waking up - another goroutine may have renewed successfully.
	// Keep certMu held until we decide whether to return, eliminating TOCTOU gap.
	m.certMu.RLock()
	cert = m.cert
	expiry = m.certExpiry
	if cert != nil && time.Now().Add(renewalBuffer).Before(expiry) {
		m.certMu.RUnlock()
		m.renewMu.Unlock()
		return cert, nil
	}
	m.certMu.RUnlock()

	// We're the goroutine that will perform renewal.
	// CRITICAL: Set renewing = true while still holding renewMu to prevent race
	// where another goroutine could slip in between releasing the lock and
	// setting this flag.
	m.renewing = true
	m.renewMu.Unlock()

	// Ensure we always signal waiting goroutines, even on panic.
	// The order matters: Broadcast() must happen before recover() so that
	// waiting goroutines are woken. If a panic occurred, they'll see
	// m.renewing=false and re-check the certificate (which won't exist).
	// They'll then compete to become the next renewing goroutine and retry.
	// This provides automatic retry behavior for transient panics.
	defer func() {
		m.renewMu.Lock()
		m.renewing = false
		m.renewCond.Broadcast()
		m.renewMu.Unlock()

		// Recover from panics in certificate obtainment to prevent server crash.
		// The Broadcast above already woke waiting goroutines, which will retry.
		if r := recover(); r != nil {
			m.Logger.Error("panic during certificate obtainment",
				zap.Any("panic", r),
				zap.Strings("domains", m.Domains))
			cert = nil
			err = fmt.Errorf("internal error obtaining certificate: %v", r)
		}
	}()

	// Create a context for the certificate obtainment.
	// Use the connection context if available (Go 1.17+), with a 10-minute timeout
	// to prevent indefinite hangs during ACME operations.
	var ctx context.Context
	var cancel context.CancelFunc
	if hello != nil && hello.Context() != nil {
		parentCtx := hello.Context()
		// Check if parent context is already cancelled (e.g., during shutdown)
		select {
		case <-parentCtx.Done():
			return nil, fmt.Errorf("connection context already cancelled: %w", parentCtx.Err())
		default:
		}
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	}
	defer cancel()

	// Perform the actual certificate obtainment
	cert, err = m.doObtainCertificate(ctx)

	// Log timeout errors explicitly for easier debugging
	if errors.Is(err, context.DeadlineExceeded) {
		m.Logger.Error("certificate obtainment timed out",
			zap.Strings("domains", m.Domains),
			zap.Duration("timeout", 10*time.Minute),
			zap.Error(err))
	}

	return cert, err
}

// doObtainCertificate performs the actual certificate obtainment via DNS-01 challenge.
//
// Synchronization: Callers must ensure only one goroutine calls this at a time.
// This is enforced by GetCertificate via renewMu. The single-threaded guarantee
// means m.client is safe to access after ensureClient() returns successfully,
// as no concurrent goroutine can modify it.
func (m *DNS01Manager) doObtainCertificate(ctx context.Context) (*tls.Certificate, error) {
	// Try to load from cache first
	if cert, expiry, err := m.loadCachedCert(); err == nil {
		if time.Now().Add(renewalBuffer).Before(expiry) {
			m.certMu.Lock()
			m.cert = cert
			m.certExpiry = expiry
			m.certMu.Unlock()
			m.Logger.Info("loaded certificate from cache",
				zap.Strings("domains", m.Domains),
				zap.Time("expiry", expiry))
			return cert, nil
		}
		m.Logger.Info("cached certificate needs renewal",
			zap.Strings("domains", m.Domains),
			zap.Time("expiry", expiry))
	}

	m.Logger.Info("obtaining new certificate via DNS-01",
		zap.Strings("domains", m.Domains))

	// Initialize ACME client if needed
	if err := m.ensureClient(ctx); err != nil {
		return nil, fmt.Errorf("dns01: init client: %w", err)
	}

	// Create new order for all domains
	order, err := m.client.AuthorizeOrder(ctx, acme.DomainIDs(m.Domains...))
	if err != nil {
		return nil, fmt.Errorf("dns01: authorize order: %w", err)
	}
	if order == nil {
		return nil, errors.New("dns01: ACME server returned nil order")
	}

	// Process authorizations
	for _, authzURL := range order.AuthzURLs {
		// Check for context cancellation between authorizations
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("dns01: %w", ctx.Err())
		default:
		}

		authz, err := m.client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return nil, fmt.Errorf("dns01: get authorization: %w", err)
		}
		if authz == nil {
			return nil, errors.New("dns01: ACME server returned nil authorization")
		}

		if authz.Status == acme.StatusValid {
			continue
		}

		// Find DNS-01 challenge
		var chal *acme.Challenge
		for _, c := range authz.Challenges {
			if c.Type == "dns-01" {
				chal = c
				break
			}
		}
		if chal == nil {
			return nil, errors.New("dns01: no DNS-01 challenge found")
		}

		// Get the DNS record value
		txtValue, err := m.client.DNS01ChallengeRecord(chal.Token)
		if err != nil {
			return nil, fmt.Errorf("dns01: compute challenge record: %w", err)
		}

		// Create DNS TXT record
		// For wildcard domains (*.example.com), the challenge record goes on
		// _acme-challenge.example.com (without the wildcard prefix).
		// Get the domain from the authorization identifier, not the config,
		// since we may have multiple domains in the order.
		challengeDomain := authz.Identifier.Value
		if strings.HasPrefix(challengeDomain, "*.") {
			challengeDomain = challengeDomain[2:]
		}
		recordName := "_acme-challenge." + challengeDomain
		if err := m.createDNSRecord(ctx, recordName, txtValue); err != nil {
			return nil, fmt.Errorf("dns01: create DNS record: %w", err)
		}

		// Wait for DNS propagation by polling actual DNS lookups
		m.Logger.Info("waiting for DNS propagation",
			zap.String("record", recordName),
			zap.String("value", txtValue))
		if err := m.waitForDNSPropagation(ctx, recordName, txtValue); err != nil {
			// Best-effort cleanup; primary error takes precedence.
			// Log cleanup failures at debug level for diagnostic purposes without
			// cluttering logs during expected failures (e.g., context cancellation).
			if cleanupErr := m.deleteDNSRecord(ctx, recordName, txtValue); cleanupErr != nil {
				m.Logger.Debug("failed to cleanup DNS record after propagation error",
					zap.String("record", recordName),
					zap.Error(cleanupErr))
			}
			return nil, fmt.Errorf("dns01: DNS propagation: %w", err)
		}

		// Accept the challenge
		if _, err := m.client.Accept(ctx, chal); err != nil {
			if cleanupErr := m.deleteDNSRecord(ctx, recordName, txtValue); cleanupErr != nil {
				m.Logger.Debug("failed to cleanup DNS record after accept error",
					zap.String("record", recordName),
					zap.Error(cleanupErr))
			}
			return nil, fmt.Errorf("dns01: accept challenge: %w", err)
		}

		// Wait for authorization to be valid
		if _, err := m.client.WaitAuthorization(ctx, authzURL); err != nil {
			if cleanupErr := m.deleteDNSRecord(ctx, recordName, txtValue); cleanupErr != nil {
				m.Logger.Debug("failed to cleanup DNS record after authorization error",
					zap.String("record", recordName),
					zap.Error(cleanupErr))
			}
			return nil, fmt.Errorf("dns01: wait authorization: %w", err)
		}

		// Clean up DNS record after successful authorization.
		// Unlike error-path cleanup, we log failures here since there's no
		// primary error to report, and orphaned records may cause issues.
		if err := m.deleteDNSRecord(ctx, recordName, txtValue); err != nil {
			m.Logger.Warn("failed to delete DNS challenge record",
				zap.String("record", recordName),
				zap.Error(err))
		}
	}

	// Generate certificate key
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("dns01: generate cert key: %w", err)
	}

	// Create CSR with all domains
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		DNSNames: m.Domains,
	}, certKey)
	if err != nil {
		return nil, fmt.Errorf("dns01: create CSR: %w", err)
	}

	// Finalize order
	der, _, err := m.client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return nil, fmt.Errorf("dns01: finalize order: %w", err)
	}

	// Validate that ACME server returned a certificate chain
	if len(der) == 0 {
		return nil, errors.New("dns01: ACME server returned empty certificate chain")
	}
	// Validate leaf certificate is not empty (defense-in-depth)
	if len(der[0]) == 0 {
		return nil, errors.New("dns01: ACME server returned empty leaf certificate")
	}

	// Build tls.Certificate directly from der slice (no need to copy)
	cert := &tls.Certificate{
		Certificate: der,
		PrivateKey:  certKey,
	}

	// Parse leaf cert for expiry
	leaf, err := x509.ParseCertificate(der[0])
	if err != nil {
		return nil, fmt.Errorf("dns01: parse certificate: %w", err)
	}
	cert.Leaf = leaf

	// Cache the certificate to disk
	if err := m.cacheCert(cert); err != nil {
		m.Logger.Warn("failed to cache certificate", zap.Error(err))
	}

	// Store in memory (protected by certMu)
	m.certMu.Lock()
	m.cert = cert
	m.certExpiry = leaf.NotAfter
	m.certMu.Unlock()

	m.Logger.Info("obtained new certificate",
		zap.Strings("domains", m.Domains),
		zap.Time("expiry", leaf.NotAfter))

	return cert, nil
}

// ensureClient initializes the ACME client and registers/loads account.
// It uses clientMu to ensure only one goroutine initializes the client.
func (m *DNS01Manager) ensureClient(ctx context.Context) error {
	// Check context before acquiring lock to avoid wasted work
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before client initialization: %w", ctx.Err())
	default:
	}

	// Fast path: check if already initialized
	m.clientMu.Lock()
	defer m.clientMu.Unlock()

	if m.client != nil {
		return nil
	}

	// Re-check context after acquiring lock (may have waited)
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during client initialization: %w", ctx.Err())
	default:
	}

	// Generate or load account key
	accountKey, err := m.loadOrCreateAccountKey()
	if err != nil {
		return fmt.Errorf("account key: %w", err)
	}
	if accountKey == nil {
		return errors.New("account key is nil after load/create")
	}

	// Create ACME client (use Let's Encrypt production)
	m.client = &acme.Client{
		Key:          accountKey,
		DirectoryURL: m.ACMEDirectoryURL,
	}

	// Try to load cached account
	if acc, err := m.loadAccount(); err == nil && acc.URI != "" {
		// Reuse the cached account URI to avoid re-registration
		m.client.Key = accountKey
		// The acme.Client will use this URI for account operations when set
		// via GetReg, but we need to verify the account is still valid
		if verifiedAcc, err := m.client.GetReg(ctx, acc.URI); err == nil && verifiedAcc != nil {
			// Update cache if the server returned a different URI (edge case)
			if verifiedAcc.URI != "" && verifiedAcc.URI != acc.URI {
				if saveErr := m.saveAccount(&acmeAccount{URI: verifiedAcc.URI}); saveErr != nil {
					m.Logger.Warn("failed to update cached ACME account URI", zap.Error(saveErr))
				}
				m.Logger.Debug("updated ACME account URI in cache",
					zap.String("old_uri", acc.URI),
					zap.String("new_uri", verifiedAcc.URI))
			} else {
				m.Logger.Debug("loaded ACME account from cache", zap.String("uri", acc.URI))
			}
			return nil
		}
		m.Logger.Debug("cached ACME account invalid, will re-register", zap.String("uri", acc.URI))
	}

	// Register new account
	account := &acme.Account{
		Contact: []string{"mailto:" + m.Email},
	}
	registeredAccount, err := m.client.Register(ctx, account, acme.AcceptTOS)
	if err != nil {
		if isAccountExists(err) {
			// Account exists - fetch the existing account to get URI
			existingAccount, getErr := m.client.GetReg(ctx, "" /* empty URI triggers account lookup */)
			if getErr != nil {
				return fmt.Errorf("get existing account: %w", getErr)
			}
			if existingAccount == nil {
				return errors.New("ACME server returned nil account without error")
			}
			if existingAccount.URI == "" {
				return errors.New("ACME server returned existing account without URI")
			}
			// Cache the retrieved account URI
			if saveErr := m.saveAccount(&acmeAccount{URI: existingAccount.URI}); saveErr != nil {
				m.Logger.Warn("failed to cache existing ACME account", zap.Error(saveErr))
			}
			m.Logger.Info("using existing ACME account", zap.String("email", m.Email), zap.String("uri", existingAccount.URI))
			return nil
		}
		return fmt.Errorf("register account: %w", err)
	}

	// Validate that registration returned a valid URI
	if registeredAccount == nil || registeredAccount.URI == "" {
		return errors.New("ACME registration succeeded but returned no account URI")
	}

	// Cache account with the URI from the registration response
	if err := m.saveAccount(&acmeAccount{URI: registeredAccount.URI}); err != nil {
		m.Logger.Warn("failed to cache ACME account", zap.Error(err))
	}

	m.Logger.Info("registered ACME account", zap.String("email", m.Email), zap.String("uri", registeredAccount.URI))
	return nil
}

// loadOrCreateAccountKey loads or creates an ECDSA account key.
// This function is called with clientMu held, ensuring no concurrent access.
func (m *DNS01Manager) loadOrCreateAccountKey() (crypto.Signer, error) {
	keyPath := filepath.Join(m.CacheDir, "account.key")

	// Try to load existing key
	data, err := os.ReadFile(keyPath)
	if err == nil {
		block, rest := pem.Decode(data)
		if block == nil {
			m.Logger.Warn("cached account key file exists but contains no valid PEM data; generating new key",
				zap.String("path", keyPath))
		} else if block.Type != "EC PRIVATE KEY" {
			m.Logger.Warn("cached account key has unexpected PEM type; generating new key",
				zap.String("path", keyPath),
				zap.String("expected", "EC PRIVATE KEY"),
				zap.String("actual", block.Type))
		} else if len(rest) > 0 {
			m.Logger.Debug("cached account key file has trailing data after PEM block",
				zap.String("path", keyPath),
				zap.Int("trailing_bytes", len(rest)))
			// Still try to parse the key - trailing data might be harmless
			key, parseErr := x509.ParseECPrivateKey(block.Bytes)
			if parseErr == nil {
				return key, nil
			}
			m.Logger.Warn("cached account key PEM valid but parsing failed; generating new key",
				zap.String("path", keyPath),
				zap.Error(parseErr))
		} else {
			key, parseErr := x509.ParseECPrivateKey(block.Bytes)
			if parseErr == nil {
				return key, nil
			}
			m.Logger.Warn("cached account key PEM valid but parsing failed; generating new key",
				zap.String("path", keyPath),
				zap.Error(parseErr))
		}
	} else if !os.IsNotExist(err) {
		// Log unexpected read errors (permissions, etc.) but continue to generate new key
		m.Logger.Warn("failed to read cached account key; generating new key",
			zap.String("path", keyPath),
			zap.Error(err))
	}

	// Generate new key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// Save key atomically: write to temp file then rename
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: der}
	keyData := pem.EncodeToMemory(block)

	// Write to temporary file first
	tmpPath := keyPath + ".tmp"
	if err := os.WriteFile(tmpPath, keyData, 0600); err != nil {
		return nil, fmt.Errorf("write temp key file: %w", err)
	}

	// Atomic rename to final path
	if err := os.Rename(tmpPath, keyPath); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file on failure
		return nil, fmt.Errorf("rename key file: %w", err)
	}

	return key, nil
}

// loadAccount loads a cached ACME account.
// Returns an error if the cached account has an invalid or empty URI.
func (m *DNS01Manager) loadAccount() (*acmeAccount, error) {
	data, err := os.ReadFile(filepath.Join(m.CacheDir, "account.json"))
	if err != nil {
		return nil, err
	}
	var acc acmeAccount
	if err := json.Unmarshal(data, &acc); err != nil {
		return nil, err
	}
	// Validate the URI is a valid HTTPS URL (ACME account URIs are always HTTPS)
	if acc.URI != "" {
		parsed, err := url.Parse(acc.URI)
		if err != nil {
			return nil, fmt.Errorf("cached account has invalid URI: %w", err)
		}
		if parsed.Scheme != "https" || parsed.Host == "" {
			return nil, fmt.Errorf("cached account URI must be HTTPS with a host: %s", acc.URI)
		}
	}
	return &acc, nil
}

// saveAccount caches an ACME account.
func (m *DNS01Manager) saveAccount(acc *acmeAccount) error {
	data, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.CacheDir, "account.json"), data, 0600)
}

// safeCachePath constructs a file path within the cache directory, ensuring
// the result cannot escape via path traversal. This is defense in depth -
// the domain is already validated by validateDomainFormat(), but we ensure
// the constructed path stays within CacheDir regardless.
func (m *DNS01Manager) safeCachePath(filename string) (string, error) {
	// Use filepath.Base to strip any directory components
	safeFilename := filepath.Base(filename)
	if safeFilename == "." || safeFilename == ".." || safeFilename == "" {
		return "", fmt.Errorf("invalid cache filename: %q", filename)
	}
	fullPath := filepath.Join(m.CacheDir, safeFilename)

	// Verify the path is actually within CacheDir (handles symlinks, etc.)
	absCache, err := filepath.Abs(m.CacheDir)
	if err != nil {
		return "", fmt.Errorf("resolve cache dir: %w", err)
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve cache path: %w", err)
	}
	// Path traversal check: absPath must be either:
	//   1. Exactly absCache (the cache directory itself), OR
	//   2. A path that starts with absCache + separator (a subdirectory/file)
	// The separator suffix prevents "/tmp" from matching "/tmp-evil/file"
	isSubpath := strings.HasPrefix(absPath, absCache+string(filepath.Separator))
	isExact := absPath == absCache
	if !isSubpath && !isExact {
		return "", fmt.Errorf("path %q escapes cache directory", filename)
	}
	return fullPath, nil
}

// cachePrefix returns a filesystem-safe prefix for cache files.
// Uses the primary (first) domain with wildcard asterisk replaced.
func (m *DNS01Manager) cachePrefix() string {
	if len(m.Domains) == 0 {
		return "cert"
	}
	// Use first domain as cache key, replacing wildcard for filesystem safety
	name := m.Domains[0]
	name = strings.Replace(name, "*.", "wildcard.", 1)
	return name
}

// loadCachedCert loads a cached certificate.
func (m *DNS01Manager) loadCachedCert() (*tls.Certificate, time.Time, error) {
	prefix := m.cachePrefix()
	certPath, err := m.safeCachePath(prefix + ".crt")
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("cert path: %w", err)
	}
	keyPath, err := m.safeCachePath(prefix + ".key")
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("key path: %w", err)
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, time.Time{}, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, time.Time{}, err
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, time.Time{}, err
	}

	// Parse leaf for expiry and domain validation
	if cert.Leaf == nil && len(cert.Certificate) > 0 {
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("parse leaf certificate: %w", err)
		}
		cert.Leaf = leaf
	}
	if cert.Leaf == nil {
		return nil, time.Time{}, errors.New("no leaf certificate in chain")
	}

	// Validate that the cached certificate covers all expected domains.
	// This is defense-in-depth against cache corruption or tampering.
	if !m.certMatchesDomains(cert.Leaf) {
		return nil, time.Time{}, fmt.Errorf("cached certificate doesn't cover all domains (expected %v, got %v)",
			m.Domains, cert.Leaf.DNSNames)
	}

	return &cert, cert.Leaf.NotAfter, nil
}

// certMatchesDomains checks if a certificate covers all configured domains.
func (m *DNS01Manager) certMatchesDomains(leaf *x509.Certificate) bool {
	for _, expectedDomain := range m.Domains {
		if !m.domainCoveredByCert(leaf, expectedDomain) {
			return false
		}
	}
	return true
}

// domainCoveredByCert checks if a single domain is covered by the certificate.
// It handles both exact matches and wildcard certificates.
func (m *DNS01Manager) domainCoveredByCert(leaf *x509.Certificate, expectedDomain string) bool {
	expectedBase := strings.TrimPrefix(expectedDomain, "*.")

	for _, dnsName := range leaf.DNSNames {
		// Exact match
		if dnsName == expectedDomain {
			return true
		}
		// Wildcard in cert matches our base domain
		if strings.HasPrefix(dnsName, "*.") {
			certBase := dnsName[2:]
			if certBase == expectedBase {
				return true
			}
		}
		// Our wildcard domain matches cert's base
		if strings.HasPrefix(expectedDomain, "*.") && dnsName == expectedBase {
			return true
		}
	}
	return false
}

// cacheCert saves a certificate to the cache directory using atomic writes.
// Files are written to temporary paths first, then renamed to prevent partial
// writes from corrupting the cache if the process is interrupted.
//
// Thread safety: This function is only called from doObtainCertificate, which
// is serialized by renewMu in GetCertificate. Therefore, concurrent cache
// writes are not possible within a single DNS01Manager instance.
func (m *DNS01Manager) cacheCert(cert *tls.Certificate) error {
	prefix := m.cachePrefix()
	certPath, err := m.safeCachePath(prefix + ".crt")
	if err != nil {
		return fmt.Errorf("cert path: %w", err)
	}
	keyPath, err := m.safeCachePath(prefix + ".key")
	if err != nil {
		return fmt.Errorf("key path: %w", err)
	}

	// Encode certificate chain
	var certPEM []byte
	for _, der := range cert.Certificate {
		block := &pem.Block{Type: "CERTIFICATE", Bytes: der}
		certPEM = append(certPEM, pem.EncodeToMemory(block)...)
	}

	// Encode private key
	if cert.PrivateKey == nil {
		return errors.New("certificate missing private key")
	}
	var keyPEM []byte
	switch k := cert.PrivateKey.(type) {
	case *ecdsa.PrivateKey:
		der, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return fmt.Errorf("marshal ECDSA key: %w", err)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	case *rsa.PrivateKey:
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		})
	default:
		return fmt.Errorf("unsupported key type %T (expected *ecdsa.PrivateKey or *rsa.PrivateKey)", cert.PrivateKey)
	}

	// Write key first (more sensitive), then cert - both atomically.
	// Write to temp file then rename to prevent partial writes.
	keyTmp := keyPath + ".tmp"
	if err := os.WriteFile(keyTmp, keyPEM, 0600); err != nil {
		return fmt.Errorf("write temp key: %w", err)
	}
	if err := os.Rename(keyTmp, keyPath); err != nil {
		_ = os.Remove(keyTmp)
		return fmt.Errorf("rename key file: %w", err)
	}

	certTmp := certPath + ".tmp"
	if err := os.WriteFile(certTmp, certPEM, 0600); err != nil {
		return fmt.Errorf("write temp cert: %w", err)
	}
	if err := os.Rename(certTmp, certPath); err != nil {
		_ = os.Remove(certTmp)
		return fmt.Errorf("rename cert file: %w", err)
	}

	return nil
}

// waitForDNSPropagation polls DNS until the expected TXT record is visible.
// This is more reliable than a fixed sleep because:
// - It proceeds as soon as DNS is ready (no unnecessary waiting)
// - It handles slow DNS providers that take longer than a fixed timeout
// - It respects context cancellation for graceful shutdown
//
// Note: We use net.DefaultResolver.LookupTXT with context support so that
// DNS lookups can be interrupted by context cancellation (e.g., during shutdown).
func (m *DNS01Manager) waitForDNSPropagation(ctx context.Context, recordName, expectedValue string) error {
	// Track start time for accurate elapsed duration logging
	startTime := time.Now()

	// Use the smaller of internal timeout or context deadline
	deadline := startTime.Add(dnsPropagationTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if deadline exceeded
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %v waiting for DNS propagation of %s", dnsPropagationTimeout, recordName)
		}

		// Perform DNS lookup with context support for cancellation
		records, err := net.DefaultResolver.LookupTXT(ctx, recordName)
		if err == nil {
			// Check if expected value is in the records
			for _, record := range records {
				if record == expectedValue {
					m.Logger.Info("DNS propagation confirmed",
						zap.String("record", recordName),
						zap.Duration("elapsed", time.Since(startTime)))
					return nil
				}
			}
			// Distinguish empty results from mismatched values for clearer debugging
			if len(records) == 0 {
				m.Logger.Debug("DNS lookup succeeded but returned no records",
					zap.String("record", recordName))
			} else {
				m.Logger.Debug("DNS record found but value doesn't match yet",
					zap.String("record", recordName),
					zap.Strings("found", records),
					zap.String("expected", expectedValue))
			}
		} else {
			// Distinguish between transient errors (retry) and potential permanent errors
			var dnsErr *net.DNSError
			if errors.As(err, &dnsErr) {
				if dnsErr.IsNotFound {
					// NXDOMAIN - record doesn't exist yet, this is expected during propagation
					m.Logger.Debug("DNS record not found yet (NXDOMAIN)",
						zap.String("record", recordName))
				} else if dnsErr.IsTemporary {
					// Temporary error (e.g., network issue) - worth retrying
					m.Logger.Debug("temporary DNS error, will retry",
						zap.String("record", recordName),
						zap.Error(err))
				} else {
					// Permanent DNS error - log at warn level
					m.Logger.Warn("DNS lookup error (may be permanent)",
						zap.String("record", recordName),
						zap.Error(err))
				}
			} else {
				// Non-DNS error (e.g., context cancellation, I/O error, resolver issues).
				// Log with error type for debugging unexpected failure modes.
				m.Logger.Debug("DNS lookup failed with non-DNS error",
					zap.String("record", recordName),
					zap.String("error_type", fmt.Sprintf("%T", err)),
					zap.Error(err))
			}
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(dnsPropagationInterval):
		}
	}
}

// dns01ValueLength is the expected length of a DNS-01 challenge value.
// ACME DNS-01 values are base64url(SHA256(token || "." || thumbprint)).
// SHA-256 produces 32 bytes; base64url encoding without padding produces 43 characters.
const dns01ValueLength = 43

// validateDNS01Value checks that a DNS-01 challenge value is safe to use.
// DNS-01 values are base64url-encoded SHA-256 hashes, so they should only
// contain URL-safe base64 characters (A-Z, a-z, 0-9, -, _) and be exactly 43 chars.
func validateDNS01Value(value string) error {
	if len(value) == 0 {
		return errors.New("dns01: challenge value is empty")
	}
	// ACME DNS-01 values are base64url(SHA256(token || "." || thumbprint))
	// SHA-256 produces 32 bytes, base64url encoding produces exactly 43 characters (no padding)
	if len(value) != dns01ValueLength {
		return fmt.Errorf("dns01: challenge value has invalid length %d (expected %d)", len(value), dns01ValueLength)
	}
	for _, c := range value {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return fmt.Errorf("dns01: invalid character %q in challenge value", c)
		}
	}
	return nil
}

// validateDNSRecordName checks that a DNS record name is valid.
// This is defense-in-depth validation for Route 53 API calls.
func validateDNSRecordName(name string) error {
	if name == "" {
		return errors.New("dns01: record name cannot be empty")
	}
	// Strip trailing dot if present (Route 53 format)
	checkName := strings.TrimSuffix(name, ".")
	if len(checkName) > 253 {
		return fmt.Errorf("dns01: record name exceeds maximum length of 253 characters")
	}
	// Basic validation: no control characters
	for _, c := range checkName {
		if c < 0x20 || c == 0x7f {
			return fmt.Errorf("dns01: record name contains invalid control character")
		}
	}
	return nil
}

// createDNSRecord creates a TXT record in Route 53.
// Operations are serialized via dnsMu to prevent Route 53 rate limiting.
func (m *DNS01Manager) createDNSRecord(ctx context.Context, name, value string) error {
	// Normalize name first: ensure it ends with a dot for Route 53
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}

	// Validate after normalization to ensure validation sees the final form
	if err := validateDNSRecordName(name); err != nil {
		return err
	}
	// Validate the DNS-01 challenge value before using it
	if err := validateDNS01Value(value); err != nil {
		return err
	}

	// Serialize DNS operations to prevent rate limiting
	m.dnsMu.Lock()
	defer m.dnsMu.Unlock()

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(m.HostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(name),
						Type: types.RRTypeTxt,
						TTL:  aws.Int64(60),
						ResourceRecords: []types.ResourceRecord{
							{Value: aws.String(`"` + value + `"`)},
						},
					},
				},
			},
		},
	}

	result, err := m.r53.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("create DNS record: %w", err)
	}

	// Validate Route53 response structure before accessing nested fields.
	// While unlikely, a malformed response could cause a nil pointer panic.
	if result == nil || result.ChangeInfo == nil || result.ChangeInfo.Id == nil {
		return errors.New("Route53 returned invalid response: missing ChangeInfo or Id")
	}

	// Wait for change to propagate
	waiter := route53.NewResourceRecordSetsChangedWaiter(m.r53)
	if err := waiter.Wait(ctx, &route53.GetChangeInput{
		Id: result.ChangeInfo.Id,
	}, 5*time.Minute); err != nil {
		m.Logger.Error("Route53 DNS record propagation failed",
			zap.String("record", name),
			zap.String("changeId", aws.ToString(result.ChangeInfo.Id)),
			zap.Error(err))
		return fmt.Errorf("waiting for DNS record propagation: %w", err)
	}
	return nil
}

// deleteDNSRecord deletes a TXT record from Route 53.
// Operations are serialized via dnsMu to prevent Route 53 rate limiting.
func (m *DNS01Manager) deleteDNSRecord(ctx context.Context, name, value string) error {
	// Normalize name first: ensure it ends with a dot for Route 53
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}

	// Validate after normalization to ensure validation sees the final form
	if err := validateDNSRecordName(name); err != nil {
		return err
	}
	// Validate the DNS-01 challenge value for consistency with createDNSRecord.
	// This ensures we attempt to delete exactly what we created.
	if err := validateDNS01Value(value); err != nil {
		return err
	}

	// Serialize DNS operations to prevent rate limiting
	m.dnsMu.Lock()
	defer m.dnsMu.Unlock()

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(m.HostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeActionDelete,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(name),
						Type: types.RRTypeTxt,
						TTL:  aws.Int64(60),
						ResourceRecords: []types.ResourceRecord{
							{Value: aws.String(`"` + value + `"`)},
						},
					},
				},
			},
		},
	}

	_, err := m.r53.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("delete DNS record: %w", err)
	}
	return nil
}

// isAccountExists checks if the error indicates an existing account.
func isAccountExists(err error) bool {
	if err == nil {
		return false
	}
	// ACME returns a specific error for existing accounts
	return strings.Contains(err.Error(), "already exists") ||
		strings.Contains(err.Error(), "Account already exists")
}

// validateDomainFormat checks that a domain name is syntactically valid.
// It validates according to RFC 1035 / RFC 1123 rules:
// - Labels separated by dots
// - Each label 1-63 characters
// - Total length <= 253 characters
// - Labels contain only alphanumeric characters and hyphens
// - Labels cannot start or end with hyphens
// - At least one dot (TLD required)
func validateDomainFormat(domain string) error {
	// Preserve original domain for error messages
	originalDomain := domain

	// Check total length
	if len(domain) > 253 {
		return fmt.Errorf("domain %q exceeds maximum length of 253 characters", originalDomain)
	}

	// Check for wildcard prefix (allowed for certificates)
	if strings.HasPrefix(domain, "*.") {
		domain = domain[2:] // Validate the rest
	}

	// Split into labels
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return fmt.Errorf("domain %q must have at least two labels (e.g., example.com)", originalDomain)
	}

	for i, label := range labels {
		// Check label length
		if len(label) == 0 {
			return fmt.Errorf("domain %q has empty label at position %d", originalDomain, i)
		}
		if len(label) > 63 {
			return fmt.Errorf("domain label %q exceeds maximum length of 63 characters", label)
		}

		// Check label characters
		for j, c := range label {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-') {
				return fmt.Errorf("domain label %q contains invalid character %q", label, c)
			}
			// Check hyphen position
			if c == '-' && (j == 0 || j == len(label)-1) {
				return fmt.Errorf("domain label %q cannot start or end with hyphen", label)
			}
		}
	}

	return nil
}

// PreWarm obtains a certificate before the server starts accepting connections.
// If the provided context has no deadline, a default 15-minute timeout is applied
// to prevent indefinite hangs during ACME operations.
func (m *DNS01Manager) PreWarm(ctx context.Context) error {
	// Apply default timeout if context has no deadline
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
	}
	_, err := m.doObtainCertificate(ctx)
	return err
}

// Background renewal constants
const (
	// renewalCheckInterval is how often to check if renewal is needed.
	renewalCheckInterval = 12 * time.Hour
)

// StartBackgroundRenewal starts a goroutine that proactively renews certificates
// before they expire. This prevents renewal latency during user TLS handshakes.
// Call StopBackgroundRenewal to stop the background goroutine gracefully.
func (m *DNS01Manager) StartBackgroundRenewal() {
	m.bgCtx, m.bgCancel = context.WithCancel(context.Background())
	m.bgWg.Add(1)

	go func() {
		defer m.bgWg.Done()

		ticker := time.NewTicker(renewalCheckInterval)
		defer ticker.Stop()

		m.Logger.Info("started background certificate renewal",
			zap.Duration("check_interval", renewalCheckInterval),
			zap.Duration("renewal_buffer", renewalBuffer),
			zap.Strings("domains", m.Domains))

		for {
			select {
			case <-m.bgCtx.Done():
				m.Logger.Info("stopping background certificate renewal")
				return
			case <-ticker.C:
				m.checkAndRenewIfNeeded()
			}
		}
	}()
}

// StopBackgroundRenewal stops the background renewal goroutine and waits for it to exit.
func (m *DNS01Manager) StopBackgroundRenewal() {
	if m.bgCancel != nil {
		m.bgCancel()
		m.bgWg.Wait()
	}
}

// checkAndRenewIfNeeded checks if the certificate needs renewal and renews if necessary.
func (m *DNS01Manager) checkAndRenewIfNeeded() {
	m.certMu.RLock()
	expiry := m.certExpiry
	m.certMu.RUnlock()

	timeUntilExpiry := time.Until(expiry)
	needsRenewal := time.Now().Add(renewalBuffer).After(expiry)

	if !needsRenewal {
		m.Logger.Debug("certificate still valid, no renewal needed",
			zap.Time("expiry", expiry),
			zap.Duration("time_remaining", timeUntilExpiry),
			zap.Strings("domains", m.Domains))
		return
	}

	m.Logger.Info("proactively renewing certificate",
		zap.Time("expiry", expiry),
		zap.Duration("time_remaining", timeUntilExpiry),
		zap.Strings("domains", m.Domains))

	// Use a reasonable timeout for renewal (10 minutes)
	renewCtx, cancel := context.WithTimeout(m.bgCtx, 10*time.Minute)
	defer cancel()

	// Use GetCertificate which handles synchronization
	if _, err := m.GetCertificate(nil); err != nil {
		// Check if we were cancelled
		if m.bgCtx.Err() != nil {
			m.Logger.Info("background renewal cancelled during shutdown")
			return
		}
		m.Logger.Error("background certificate renewal failed",
			zap.Error(err),
			zap.Strings("domains", m.Domains))
	} else {
		m.certMu.RLock()
		newExpiry := m.certExpiry
		m.certMu.RUnlock()
		m.Logger.Info("background certificate renewal succeeded",
			zap.Strings("domains", m.Domains),
			zap.Time("new_expiry", newExpiry))
	}
	_ = renewCtx // suppress unused warning
}
