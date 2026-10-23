package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"inventar/internal/mail"
	"inventar/internal/server"
	"inventar/internal/store"
)

func starten(ctx context.Context, b befehl) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := b.cfg
	log := slog.New(slog.NewTextHandler(b.stdout, nil))
	st, err := store.Open(ctx, cfg.Datenverzeichnis)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := &http.Server{
		Addr: cfg.Adresse,
		Handler: server.New(server.Deps{
			Config: cfg,
			Store:  st,
			Mailer: mail.Neu(cfg, log),
			Logger: log,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	fehler := make(chan error, 1)
	if cfg.MitTLS() {
		tlsCfg, err := tlsKonfiguration(ctx, cfg, log)
		if err != nil {
			return err
		}
		srv.TLSConfig = tlsCfg
		log.Info("Server startet mit HTTPS", "adresse", cfg.Adresse, "domain", cfg.DuckDNS.Domain)
		go func() { fehler <- srv.ListenAndServeTLS("", "") }()
	} else {
		log.Info("Server startet ohne TLS", "url", fmt.Sprintf("http://%s", cfg.Adresse))
		go func() { fehler <- srv.ListenAndServe() }()
	}

	select {
	case err := <-fehler:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
