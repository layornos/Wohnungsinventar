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

	errs := make(chan error, 1)
	if cfg.WithTLS() {
		tlsCfg, err := tlsConfig(ctx, cfg, log)
		if err != nil {
			return err
		}
		srv.TLSConfig = tlsCfg
		log.Info("server starting with HTTPS", "addr", cfg.Addr, "domain", cfg.DuckDNS.Domain)
		go func() { errs <- srv.ListenAndServeTLS("", "") }()
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
