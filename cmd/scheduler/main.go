package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joshestein/whatsapp-scheduler/internal/config"
	"github.com/joshestein/whatsapp-scheduler/internal/scheduler"
	"github.com/joshestein/whatsapp-scheduler/internal/session"
	"github.com/joshestein/whatsapp-scheduler/internal/store"
	"github.com/joshestein/whatsapp-scheduler/internal/web"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// run holds all wiring so that deferred cleanup (db.Close) runs on every
// exit path. main only reports the error.
func run(log *slog.Logger) error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return err
	}

	db, err := store.Open(filepath.Join(cfg.DataDir, "scheduler.db"))
	if err != nil {
		return err
	}
	defer db.Close()

	st, err := store.New(ctx, db)
	if err != nil {
		return err
	}

	sess, err := session.New(ctx, db, log)
	if err != nil {
		return err
	}
	sess.Start(ctx)
	defer sess.Stop()

	sched := scheduler.New(st, sess, log, cfg.Tick, cfg.GraceWindow)
	go sched.Run(ctx)

	ui, err := web.New(st, sess, log)
	if err != nil {
		return err
	}

	srv := &http.Server{Addr: cfg.ListenAddr, Handler: ui.Handler()}

	go func() {
		log.Info("listening", "addr", cfg.ListenAddr, "data_dir", cfg.DataDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
