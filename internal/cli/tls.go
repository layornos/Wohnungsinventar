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

// tlsKonfiguration holt und erneuert das Let's-Encrypt-Zertifikat per DNS-01 über DuckDNS (ADR-0004).
// Der Pi muss dafür von außen nicht erreichbar sein; die Domain darf auf eine private IP zeigen.
func tlsKonfiguration(ctx context.Context, cfg config.Config, log *slog.Logger) (*tls.Config, error) {
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	certmagic.DefaultACME.DisableHTTPChallenge = true
	certmagic.DefaultACME.DisableTLSALPNChallenge = true
	certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
		DNSManager: certmagic.DNSManager{
			DNSProvider: &duckdns.Provider{APIToken: cfg.DuckDNS.Token},
		},
	}
	// Speicher an der globalen Vorlage setzen, nicht nur an der Konfiguration unten:
	// Bei der Erneuerung im Hintergrund baut certmagic seine Konfiguration aus certmagic.Default neu.
	certmagic.Default.Storage = &certmagic.FileStorage{Path: filepath.Join(cfg.Datenverzeichnis, "zertifikate")}

	magic := certmagic.NewDefault()
	log.Info("Zertifikat wird geprüft bzw. per DuckDNS geholt", "domain", cfg.DuckDNS.Domain)
	if err := magic.ManageSync(ctx, []string{cfg.DuckDNS.Domain}); err != nil {
		return nil, err
	}
	tlsCfg := magic.TLSConfig()
	tlsCfg.NextProtos = append([]string{"h2", "http/1.1"}, tlsCfg.NextProtos...)
	return tlsCfg, nil
}
