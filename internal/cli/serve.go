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
	"inventar/internal/suggest"
)

func serve(ctx context.Context, c command) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := c.cfg
	log := slog.New(slog.NewTextHandler(c.stdout, nil))
	st, err := store.Open(ctx, cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	deps := server.Deps{
		Config: cfg,
		Store:  st,
		Mailer: mail.New(cfg, log),
		Logger: log,
	}
	if cfg.WithSuggestions() {
		claude := suggest.NewClaude(cfg.Anthropic.APIKey, cfg.Anthropic.Model)
		deps.Suggester = claude
		log.Info("suggestions from photos enabled, photos are sent to Anthropic (ADR-0006)", "model", claude.Model())
	}
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.New(deps),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errs := make(chan error, 1)
	if cfg.WithTLS() {
		tlsCfg, err := tlsConfig(ctx, cfg, log)
		if err != nil {
			return err
		}
		srv.TLSConfig = tlsCfg
		log.Info("server starting with HTTPS", "addr", cfg.Addr, "domain", cfg.DuckDNS.Domain)
		go func() { errs <- srv.ListenAndServeTLS("", "") }()
	} else if cfg.BehindProxy {
		log.Info("server starting behind a reverse proxy that terminates TLS", "addr", cfg.Addr)
		go func() { errs <- srv.ListenAndServe() }()
	} else {
		log.Info("server starting without TLS", "url", fmt.Sprintf("http://%s", cfg.Addr))
		go func() { errs <- srv.ListenAndServe() }()
	}

	select {
	case err := <-errs:
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
