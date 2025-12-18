// server/server.go
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dalemusser/waffle/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/acme/autocert"
)

// WithShutdownSignals returns a context that is canceled when the process
// receives SIGINT or SIGTERM. It's a helper to tie OS signals into context
// cancellation, and should be used as the parent context for the HTTP server.
// The returned cancel function also cleans up the signal handler.
func WithShutdownSignals(parent context.Context, logger *zap.Logger) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	sigCh := make(chan os.Signal, 1)
	// Always listen for Ctrl+C (os.Interrupt). Add SIGTERM on non-Windows.
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigCh:
			if logger != nil {
				logger.Info("shutdown signal received", zap.Any("signal", sig))
			}
			cancel()
		case <-ctx.Done():
			// Context was cancelled externally (e.g., programmatic shutdown)
		}
		// Clean up signal handling: stop delivery to this channel.
		// After Stop(), no new signals will be sent to sigCh.
		signal.Stop(sigCh)
		// Note: We intentionally don't close sigCh. Closing is unnecessary since
		// no other goroutine reads from it after this point, and attempting to
		// drain+close introduces a race condition where a signal delivered just
		// before Stop() could cause issues. The channel will be garbage collected
		// when this goroutine exits and no references remain.
	}()

	return ctx, cancel
}

// ListenAndServeWithContext starts an HTTP or HTTPS server (with optional
// Let's Encrypt via http-01 or dns-01 challenge) and blocks until the context
// is canceled or the server encounters a terminal error.
//
// It does NOT wire any routes itself; callers must provide a fully
// configured http.Handler (e.g., chi.Router).
func ListenAndServeWithContext(
	ctx context.Context,
	cfg *config.CoreConfig,
	handler http.Handler,
	logger *zap.Logger,
) error {
	if cfg == nil {
		return fmt.Errorf("ListenAndServeWithContext: cfg is nil")
	}
	if handler == nil {
		return fmt.Errorf("ListenAndServeWithContext: handler is nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// Build base http.Server with configured timeouts.
	srv := &http.Server{
		Handler:           handler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	// Route stdlib error logs into zap at Warn level.
	if stdlog, err := zap.NewStdLogAt(logger, zapcore.WarnLevel); err == nil {
		srv.ErrorLog = stdlog
	} else {
		logger.Warn("failed to attach stdlib error logger", zap.Error(err))
	}

	httpAddr := ":" + strconv.Itoa(cfg.HTTP.HTTPPort)
	httpsAddr := ":" + strconv.Itoa(cfg.HTTP.HTTPSPort)

	var (
		auxSrv   *http.Server // :80 ACME or redirect server (when HTTPS/http-01)
		ln       net.Listener // primary listener we Serve() on
		baseLn   net.Listener // underlying TCP listener (for TLS cleanup)
		serveErr = make(chan error, 1)
		auxErr   chan error // lazily created if auxSrv is started; nil channels block forever in select
		err      error
	)

	// cleanupListener closes the base listener if set. Used in error paths
	// to prevent resource leaks when setup fails after listener creation.
	cleanupListener := func() {
		if baseLn != nil {
			_ = baseLn.Close()
		}
	}

	// Select serving mode based on config.
	switch {
	// ----------------------------- HTTP only -------------------------------
	case !cfg.HTTP.UseHTTPS:
		baseLn, err = net.Listen("tcp", httpAddr)
		if err != nil {
			return fmt.Errorf("listen http %s: %w", httpAddr, err)
		}
		ln = baseLn // No TLS wrapping in HTTP-only mode
		logger.Info("HTTP server listening", zap.String("addr", ln.Addr().String()))
		go servePrimary(srv, ln, serveErr)

	// ----------------------- HTTPS via Let's Encrypt -----------------------
	case cfg.TLS.UseLetsEncrypt:
		// LetsEncryptChallenge is normalized to lowercase by config.Load()
		challenge := cfg.TLS.LetsEncryptChallenge

		var tlsCfg *tls.Config

		if challenge == "dns-01" {
			// DNS-01 challenge via Route 53
			dns01, err := NewDNS01Manager(
				cfg.TLS.Domain,
				cfg.TLS.LetsEncryptEmail,
				cfg.TLS.LetsEncryptCacheDir,
				cfg.TLS.Route53HostedZoneID,
				cfg.TLS.ACMEDirectoryURL,
				logger,
			)
			if err != nil {
				return fmt.Errorf("dns-01 manager: %w", err)
			}

			// Pre-warm certificate before accepting connections
			logger.Info("obtaining certificate via DNS-01 challenge",
				zap.String("domain", cfg.TLS.Domain))
			if err := dns01.PreWarm(ctx); err != nil {
				return fmt.Errorf("dns-01 pre-warm: %w", err)
			}

			tlsCfg = &tls.Config{
				MinVersion:     tls.VersionTLS12,
				GetCertificate: dns01.GetCertificate,
			}

			// Port 80: redirect to HTTPS (no ACME challenge needed for dns-01)
			auxSrv = &http.Server{
				Addr:              ":80",
				Handler:           httpRedirectHandler(),
				ReadTimeout:       cfg.HTTP.ReadTimeout,
				ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
				WriteTimeout:      cfg.HTTP.WriteTimeout,
				IdleTimeout:       cfg.HTTP.IdleTimeout,
			}
			if stdlog, err := zap.NewStdLogAt(logger, zapcore.WarnLevel); err == nil {
				auxSrv.ErrorLog = stdlog
			}
			auxErr = make(chan error, 1)
			go serveAuxiliary(auxSrv, auxErr)
			logger.Info("HTTP → HTTPS redirect server listening", zap.String("addr", auxSrv.Addr))

		} else {
			// HTTP-01 challenge (default)
			m := &autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(cfg.TLS.Domain),
				Cache:      autocert.DirCache(cfg.TLS.LetsEncryptCacheDir),
				Email:      cfg.TLS.LetsEncryptEmail,
			}

			// Port 80: ACME challenge + HTTPS redirect for everything else.
			auxSrv = &http.Server{
				Addr:              ":80",
				Handler:           m.HTTPHandler(httpRedirectHandler()),
				ReadTimeout:       cfg.HTTP.ReadTimeout,
				ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
				WriteTimeout:      cfg.HTTP.WriteTimeout,
				IdleTimeout:       cfg.HTTP.IdleTimeout,
			}
			if stdlog, err := zap.NewStdLogAt(logger, zapcore.WarnLevel); err == nil {
				auxSrv.ErrorLog = stdlog
			}
			auxErr = make(chan error, 1)
			go serveAuxiliary(auxSrv, auxErr)
			logger.Info("ACME + redirect server listening", zap.String("addr", auxSrv.Addr))

			// Pre-warm before binding :443
			if err := waitForCert(ctx, m, cfg.TLS.Domain, 60*time.Second); err != nil {
				logger.Warn("autocert pre-warm failed; first HTTPS hits may see TLS errors", zap.Error(err))
			}

			tlsCfg = &tls.Config{
				MinVersion:     tls.VersionTLS12,
				GetCertificate: m.GetCertificate,
			}
		}

		srv.TLSConfig = tlsCfg

		var listenErr error
		baseLn, listenErr = net.Listen("tcp", httpsAddr)
		if listenErr != nil {
			// Cleanup auxiliary server that was already started
			_ = shutdownAux(auxSrv, context.Background())
			return fmt.Errorf("listen https %s: %w", httpsAddr, listenErr)
		}
		ln = tls.NewListener(baseLn, tlsCfg)
		logger.Info("HTTPS server (Let's Encrypt "+challenge+") listening",
			zap.String("addr", httpsAddr),
			zap.String("domain", cfg.TLS.Domain))
		go servePrimary(srv, ln, serveErr)

	// ----------------------- HTTPS via manual certs ------------------------
	default:
		if cfg.TLS.CertFile == "" || cfg.TLS.KeyFile == "" {
			return fmt.Errorf("manual TLS selected but cert_file / key_file not provided")
		}

		// Validate TLS files exist and are accessible before proceeding
		if err := validateTLSFiles(cfg.TLS.CertFile, cfg.TLS.KeyFile); err != nil {
			// Check if it's just a permissions warning vs a hard error
			if strings.Contains(err.Error(), "overly permissive permissions") {
				if cfg.Env == "prod" {
					// In production, insecure key permissions are a hard error
					return fmt.Errorf("production security: %w", err)
				}
				logger.Warn("TLS key file security warning (would block in prod)", zap.Error(err))
			} else {
				return err
			}
		}

		// Port 80: redirect everything to HTTPS.
		auxSrv = &http.Server{
			Addr:              ":80",
			Handler:           httpRedirectHandler(),
			ReadTimeout:       cfg.HTTP.ReadTimeout,
			ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
			WriteTimeout:      cfg.HTTP.WriteTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
		}
		if stdlog, err := zap.NewStdLogAt(logger, zapcore.WarnLevel); err == nil {
			auxSrv.ErrorLog = stdlog
		}
		auxErr = make(chan error, 1)
		go serveAuxiliary(auxSrv, auxErr)
		logger.Info("HTTP → HTTPS redirect server listening", zap.String("addr", auxSrv.Addr))

		// Port 443: primary HTTPS with provided certs.
		cert, loadErr := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if loadErr != nil {
			// Cleanup auxiliary server that was already started
			_ = shutdownAux(auxSrv, context.Background())
			return fmt.Errorf("load TLS cert/key: %w", loadErr)
		}
		tlsCfg := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
		srv.TLSConfig = tlsCfg

		var listenErr error
		baseLn, listenErr = net.Listen("tcp", httpsAddr)
		if listenErr != nil {
			// Cleanup auxiliary server that was already started
			_ = shutdownAux(auxSrv, context.Background())
			return fmt.Errorf("listen https %s: %w", httpsAddr, listenErr)
		}
		ln = tls.NewListener(baseLn, tlsCfg)
		logger.Info("HTTPS server (manual TLS) listening",
			zap.String("addr", httpsAddr),
			zap.String("cert_file", cfg.TLS.CertFile))
		go servePrimary(srv, ln, serveErr)
	}

	// ---------- wait for shutdown / errors ----------
	// Note: auxErr is nil in HTTP-only mode. In Go, receiving from a nil channel
	// blocks forever, which effectively disables that select case. This is
	// intentional - we only care about auxErr when an auxiliary server exists.
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown path requested by caller.
			logger.Info("shutting down server…")
			// Create shutdown context with configured timeout.
			// Use context.Background() as parent since ctx is already cancelled.
			// The shutdown timeout is intentionally independent of ctx's deadline
			// to ensure we have a consistent window for graceful shutdown regardless
			// of when cancellation occurred. Callers control total operation time
			// by when they cancel ctx, and ShutdownTimeout controls cleanup time.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
			defer cancel()
			_ = shutdownAux(auxSrv, shutdownCtx)
			if err := srv.Shutdown(shutdownCtx); err != nil {
				cleanupListener()
				return fmt.Errorf("server shutdown: %w", err)
			}
			cleanupListener()
			logger.Info("server stopped gracefully")
			return nil

		case err := <-serveErr:
			// Primary server crashed or closed unexpectedly.
			if err != nil && err != http.ErrServerClosed {
				_ = shutdownAux(auxSrv, context.Background())
				cleanupListener()
				return fmt.Errorf("primary server error: %w", err)
			}
			// nil or ErrServerClosed: ensure aux is stopped too.
			_ = shutdownAux(auxSrv, context.Background())
			cleanupListener()
			return nil

		case err := <-auxErr:
			// Auxiliary server (ACME / redirect) crashed.
			// Note: This case is only reachable when auxErr != nil (HTTPS modes).
			if err != nil && err != http.ErrServerClosed {
				// Close primary server and underlying listener
				if closeErr := srv.Close(); closeErr != nil {
					logger.Error("failed to close primary server after auxiliary crash", zap.Error(closeErr))
				}
				// srv.Close() doesn't close listeners passed to Serve(), so close explicitly
				cleanupListener()
				return fmt.Errorf("auxiliary server error: %w", err)
			}
			// nil (ErrServerClosed) → continue waiting for ctx or primary.
			// Setting auxErr to nil disables this case for subsequent iterations.
			// This is safe because serveAuxiliary sends at most once then exits,
			// so no other goroutine will attempt to send after we've received.
			auxSrv = nil
			auxErr = nil
		}
	}
}

// servePrimary runs srv.Serve on the provided listener and reports terminal errors.
func servePrimary(srv *http.Server, ln net.Listener, ch chan<- error) {
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		ch <- err
		return
	}
	ch <- nil
}

// serveAuxiliary runs auxSrv.ListenAndServe and reports terminal errors.
func serveAuxiliary(auxSrv *http.Server, ch chan<- error) {
	if err := auxSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		ch <- err
		return
	}
	ch <- nil
}

// shutdownAux gracefully shuts down the auxiliary server (if any).
func shutdownAux(auxSrv *http.Server, ctx context.Context) error {
	if auxSrv == nil {
		return nil
	}
	return auxSrv.Shutdown(ctx)
}

// httpRedirectHandler redirects any HTTP request to HTTPS preserving host + path.
// It validates the Host header to prevent header injection and open redirect attacks.
func httpRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if !isValidHost(host) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// RequestURI returns the unmodified request-target. Validate it doesn't
		// contain control characters that could cause header injection.
		reqURI := r.URL.RequestURI()
		if !isValidRequestURI(reqURI) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		target := "https://" + host + reqURI
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

// isValidRequestURI checks that the request URI is safe to use in a redirect.
// It rejects URIs containing control characters that could enable header injection.
func isValidRequestURI(uri string) bool {
	for _, c := range uri {
		// Reject control characters (except tab which is technically allowed in some contexts)
		if c < 0x20 && c != '\t' {
			return false
		}
		if c == 0x7f { // DEL
			return false
		}
	}
	return true
}

// isValidHost checks that the host header is safe to use in a redirect.
// It rejects hosts containing characters that could enable header injection
// or other attacks (newlines, carriage returns, null bytes, etc.).
func isValidHost(host string) bool {
	if host == "" {
		return false
	}

	// Use net.SplitHostPort for robust parsing of host:port and IPv6 addresses
	// like "[::1]:8080". If it fails, the host might not have a port.
	hostPart, portStr, err := net.SplitHostPort(host)
	if err != nil {
		// No port present, or malformed - treat entire string as hostname.
		// Note: A bare IPv6 address like "::1" (without brackets or port) is
		// technically ambiguous but valid for the Host header. We accept it
		// and validate the characters below.
		hostPart = host
	} else {
		// Valid host:port - validate port range
		if portStr != "" {
			port, parseErr := strconv.Atoi(portStr)
			if parseErr != nil || port <= 0 || port > 65535 {
				return false
			}
		}
	}

	// Reject empty host part
	if hostPart == "" {
		return false
	}

	// Strip brackets from IPv6 addresses for validation
	if strings.HasPrefix(hostPart, "[") && strings.HasSuffix(hostPart, "]") {
		if len(hostPart) < 3 {
			// Just "[]" with nothing inside
			return false
		}
		ipv6Part := hostPart[1 : len(hostPart)-1]
		// IPv6 addresses may include a zone ID (e.g., "fe80::1%eth0")
		// which is valid for link-local addresses. Strip the zone for IP parsing.
		if zoneIdx := strings.Index(ipv6Part, "%"); zoneIdx != -1 {
			ipv6Part = ipv6Part[:zoneIdx]
		}
		// Validate it's actually a valid IP address
		if net.ParseIP(ipv6Part) == nil {
			return false
		}
		// Use the original hostPart (with zone) for subsequent character checks
	}

	// Reject any control characters or whitespace that could enable injection
	for _, c := range hostPart {
		if c < 0x20 || c == 0x7f { // ASCII control characters
			return false
		}
		if c == '\r' || c == '\n' || c == '\t' {
			return false
		}
	}

	// Reject hosts that look like they're trying URL schemes or paths
	if strings.Contains(host, "://") || strings.HasPrefix(host, "/") {
		return false
	}

	return true
}

// validateTLSFiles checks that the certificate and key files exist, are readable,
// and have reasonably secure permissions. Returns nil if valid, or a descriptive error.
func validateTLSFiles(certFile, keyFile string) error {
	// Check certificate file
	certInfo, err := os.Stat(certFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("TLS certificate file does not exist: %s", certFile)
		}
		return fmt.Errorf("cannot access TLS certificate file %s: %w", certFile, err)
	}
	if certInfo.IsDir() {
		return fmt.Errorf("TLS certificate path is a directory, not a file: %s", certFile)
	}

	// Check key file
	keyInfo, err := os.Stat(keyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("TLS key file does not exist: %s", keyFile)
		}
		return fmt.Errorf("cannot access TLS key file %s: %w", keyFile, err)
	}
	if keyInfo.IsDir() {
		return fmt.Errorf("TLS key path is a directory, not a file: %s", keyFile)
	}

	// Check key file permissions (warn if world-readable on Unix)
	// Skip this check on Windows where Unix-style permissions don't apply.
	// On Windows, os.FileMode.Perm() returns values that don't represent
	// actual file permissions, making this check meaningless.
	if runtime.GOOS != "windows" {
		// Permission bits: 0o077 checks if group or others have any access
		if keyInfo.Mode().Perm()&0o077 != 0 {
			// This is a warning-level issue, not a hard error, but we return an error
			// to let the caller decide. In practice, we'll log a warning and continue.
			return fmt.Errorf("TLS key file %s has overly permissive permissions %o (recommended: 0600)", keyFile, keyInfo.Mode().Perm())
		}
	}

	return nil
}

// waitForCert blocks until autocert has a certificate for host (or times out).
// It respects both the provided timeout and any deadline set on the parent context,
// using whichever comes first.
func waitForCert(ctx context.Context, m *autocert.Manager, host string, timeout time.Duration) error {
	// Use the earlier of: provided timeout or parent context deadline
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	var lastErr error
	for {
		// Respect shutdown
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := m.GetCertificate(&tls.ClientHelloInfo{ServerName: host})
		if err == nil {
			return nil // cert is ready and cached
		}
		lastErr = err

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for cert for %q: %w", host, lastErr)
		}

		// Use context-aware sleep so shutdown signals are respected during wait
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}
