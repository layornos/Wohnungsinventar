package cli

import (
	"context"
	"crypto/tls"
	"log/slog"
	"path/filepath"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/duckdns"

	"inventar/internal/config"
)

// certDir is the folder in the data directory for certificates. The name stays German so
// existing certificates are still found.
const certDir = "zertifikate"

// tlsConfig obtains and renews the Let's Encrypt certificate via DNS-01 through DuckDNS (ADR-0004).
// The Pi does not need to be reachable from outside; the domain may point to a private IP.
func tlsConfig(ctx context.Context, cfg config.Config, log *slog.Logger) (*tls.Config, error) {
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	certmagic.DefaultACME.DisableHTTPChallenge = true
	certmagic.DefaultACME.DisableTLSALPNChallenge = true
	certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
		DNSManager: certmagic.DNSManager{
			DNSProvider: &duckdns.Provider{APIToken: cfg.DuckDNS.Token},
		},
	}
	// Set the storage on the global template, not only on the configuration below:
	// for renewals in the background certmagic rebuilds its configuration from certmagic.Default.
	certmagic.Default.Storage = &certmagic.FileStorage{Path: filepath.Join(cfg.DataDir, certDir)}

	magic := certmagic.NewDefault()
	log.Info("checking certificate or obtaining it via DuckDNS", "domain", cfg.DuckDNS.Domain)
	if err := magic.ManageSync(ctx, []string{cfg.DuckDNS.Domain}); err != nil {
		return nil, err
	}
	tlsCfg := magic.TLSConfig()
	tlsCfg.NextProtos = append([]string{"h2", "http/1.1"}, tlsCfg.NextProtos...)
	return tlsCfg, nil
}
