// server/certrenewer.go
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

// CertRenewer provides certificate renewal capabilities.
type CertRenewer interface {
	// ForceRenewal forces an immediate certificate renewal.
	// Returns the new certificate expiry time on success.
	ForceRenewal(ctx context.Context) (time.Time, error)

	// ChallengeType returns the ACME challenge type ("http-01" or "dns-01").
	ChallengeType() string
}

// activeCertRenewer holds the current certificate renewer (if any).
var (
	activeCertRenewer   CertRenewer
	activeCertRenewerMu sync.RWMutex
)

// SetCertRenewer sets the active certificate renewer.
// This is called by ListenAndServeWithContext when using Let's Encrypt.
func SetCertRenewer(r CertRenewer) {
	activeCertRenewerMu.Lock()
	activeCertRenewer = r
	activeCertRenewerMu.Unlock()
}

// GetCertRenewer returns the active certificate renewer, or nil if not using Let's Encrypt.
func GetCertRenewer() CertRenewer {
	activeCertRenewerMu.RLock()
	defer activeCertRenewerMu.RUnlock()
	return activeCertRenewer
}

// ChallengeType returns "dns-01" for DNS01Manager.
func (m *DNS01Manager) ChallengeType() string {
	return "dns-01"
}

// AutocertRenewer wraps an autocert.Manager to implement CertRenewer.
type AutocertRenewer struct {
	Manager  *autocert.Manager
	Domain   string
	CacheDir string
	Logger   *zap.Logger
}

// ChallengeType returns "http-01" for autocert.
func (r *AutocertRenewer) ChallengeType() string {
	return "http-01"
}

// ForceRenewal forces an immediate certificate renewal for autocert.
// It clears the disk cache and requests a fresh certificate from Let's Encrypt.
func (r *AutocertRenewer) ForceRenewal(ctx context.Context) (time.Time, error) {
	if r.Logger != nil {
		r.Logger.Info("forcing certificate renewal (http-01)", zap.String("domain", r.Domain))
	}

	// Delete the cached certificate file to force autocert to get a new one.
	// autocert.DirCache stores certs in files named after the domain.
	if r.CacheDir != "" {
		certFile := filepath.Join(r.CacheDir, r.Domain)
		if err := os.Remove(certFile); err != nil && !os.IsNotExist(err) {
			if r.Logger != nil {
				r.Logger.Warn("failed to remove cached cert file", zap.String("path", certFile), zap.Error(err))
			}
		} else if err == nil {
			if r.Logger != nil {
				r.Logger.Info("removed cached cert file", zap.String("path", certFile))
			}
		}
	}

	// Request a new certificate - with cache cleared, autocert will fetch from ACME
	hello := &tls.ClientHelloInfo{
		ServerName: r.Domain,
	}

	cert, err := r.Manager.GetCertificate(hello)
	if err != nil {
		return time.Time{}, fmt.Errorf("get certificate: %w", err)
	}

	// Parse the leaf certificate to get expiry
	var expiry time.Time
	if cert.Leaf != nil {
		expiry = cert.Leaf.NotAfter
	} else if len(cert.Certificate) > 0 {
		// Parse the leaf if not already parsed
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return time.Time{}, fmt.Errorf("parse certificate: %w", err)
		}
		expiry = leaf.NotAfter
	}

	if r.Logger != nil {
		r.Logger.Info("forced certificate renewal succeeded (http-01)",
			zap.String("domain", r.Domain),
			zap.Time("new_expiry", expiry))
	}

	return expiry, nil
}
