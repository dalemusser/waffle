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
	"strconv"
	"syscall"
	"time"

	"github.com/dalemusser/waffle/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/acme/autocert"
)

// WithShutdownSignals returns a context that is canceled when the process
// receives SIGINT or SIGTERM. It’s a helper to tie OS signals into context
// cancellation, and should be used as the parent context for the HTTP server.
func WithShutdownSignals(parent context.Context, logger *zap.Logger) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	sigCh := make(chan os.Signal, 1)
	// Always listen for Ctrl+C (os.Interrupt). Add SIGTERM on non-Windows.
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		if logger != nil {
			logger.Info("shutdown signal received", zap.Any("signal", sig))
		}
		cancel()
	}()

	return ctx, cancel
}

// ListenAndServeWithContext starts an HTTP or HTTPS server (with optional
// Let's Encrypt http-01) and blocks until the context is canceled or the
// server encounters a terminal error.
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

	// Build base http.Server with sane timeouts.
	srv := &http.Server{
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
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
		serveErr = make(chan error, 1)
		auxErr   chan error // lazily created if auxSrv is started
		err      error
	)

	// Select serving mode based on config.
	switch {
	// ----------------------------- HTTP only -------------------------------
	case !cfg.HTTP.UseHTTPS:
		ln, err = net.Listen("tcp", httpAddr)
		if err != nil {
			return fmt.Errorf("listen http %s: %w", httpAddr, err)
		}
		logger.Info("HTTP server listening", zap.String("addr", ln.Addr().String()))
		go servePrimary(srv, ln, serveErr)

	// ----------------------- HTTPS via Let's Encrypt (http-01) ------------
	case cfg.TLS.UseLetsEncrypt:
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
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
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

		// Port 443: primary HTTPS.
		tlsCfg := &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: m.GetCertificate,
		}
		srv.TLSConfig = tlsCfg

		base, e := net.Listen("tcp", httpsAddr)
		if e != nil {
			return fmt.Errorf("listen https %s: %w", httpsAddr, e)
		}
		ln = tls.NewListener(base, tlsCfg)
		logger.Info("HTTPS server (Let's Encrypt http-01) listening",
			zap.String("addr", httpsAddr),
			zap.String("domain", cfg.TLS.Domain))
		go servePrimary(srv, ln, serveErr)

	// ----------------------- HTTPS via manual certs ------------------------
	default:
		if cfg.TLS.CertFile == "" || cfg.TLS.KeyFile == "" {
			return fmt.Errorf("manual TLS selected but cert_file / key_file not provided")
		}

		// Port 80: redirect everything to HTTPS.
		auxSrv = &http.Server{
			Addr:              ":80",
			Handler:           httpRedirectHandler(),
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		if stdlog, err := zap.NewStdLogAt(logger, zapcore.WarnLevel); err == nil {
			auxSrv.ErrorLog = stdlog
		}
		auxErr = make(chan error, 1)
		go serveAuxiliary(auxSrv, auxErr)
		logger.Info("HTTP → HTTPS redirect server listening", zap.String("addr", auxSrv.Addr))

		// Port 443: primary HTTPS with provided certs.
		cert, e := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if e != nil {
			return fmt.Errorf("load TLS cert/key: %w", e)
		}
		tlsCfg := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
		srv.TLSConfig = tlsCfg

		base, e := net.Listen("tcp", httpsAddr)
		if e != nil {
			return fmt.Errorf("listen https %s: %w", httpsAddr, e)
		}
		ln = tls.NewListener(base, tlsCfg)
		logger.Info("HTTPS server (manual TLS) listening",
			zap.String("addr", httpsAddr),
			zap.String("cert_file", cfg.TLS.CertFile))
		go servePrimary(srv, ln, serveErr)
	}

	// ---------- wait for shutdown / errors ----------
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown path requested by caller.
			logger.Info("shutting down server…")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = shutdownAux(auxSrv, shutdownCtx)
			if err := srv.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("server shutdown: %w", err)
			}
			logger.Info("server stopped gracefully")
			return nil

		case err := <-serveErr:
			// Primary server crashed or closed unexpectedly.
			if err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("primary server error: %w", err)
			}
			// nil or ErrServerClosed: ensure aux is stopped too.
			_ = shutdownAux(auxSrv, context.Background())
			return nil

		case err := <-auxErr:
			// Auxiliary server (ACME / redirect) crashed.
			if err != nil && err != http.ErrServerClosed {
				_ = srv.Close()
				return fmt.Errorf("auxiliary server error: %w", err)
			}
			// nil (ErrServerClosed) → continue waiting for ctx or primary.
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
func httpRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

// waitForCert blocks until autocert has a certificate for host (or times out).
func waitForCert(ctx context.Context, m *autocert.Manager, host string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
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
		time.Sleep(1 * time.Second)
	}
}
