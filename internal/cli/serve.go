package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"inventar/internal/config"
	"inventar/internal/mail"
	"inventar/internal/server"
	"inventar/internal/store"
)

func serve(ctx context.Context, cfg config.Config, stdout io.Writer) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	log := slog.New(slog.NewTextHandler(stdout, nil))
	st, err := store.Open(ctx, cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: server.New(server.Deps{
			Config: cfg,
			Store:  st,
			Mailer: mail.New(cfg, log),
			Logger: log,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	if cfg.TLSEnabled() {
		tlsCfg, err := tlsConfig(ctx, cfg, log)
		if err != nil {
			return err
		}
		srv.TLSConfig = tlsCfg
		log.Info("Server startet mit HTTPS", "adresse", cfg.Addr, "domain", cfg.DuckDNS.Domain)
		go func() { errc <- srv.ListenAndServeTLS("", "") }()
	} else {
		log.Info("Server startet ohne TLS", "url", fmt.Sprintf("http://%s", cfg.Addr))
		go func() { errc <- srv.ListenAndServe() }()
	}

	select {
	case err := <-errc:
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
