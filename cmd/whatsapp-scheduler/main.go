package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
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

	// Bind the port before touching the database or the WhatsApp session. If
	// another instance is running (the launchd agent, say) this fails here, and
	// two processes never share one session.
	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}

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
	schedDone := make(chan struct{})
	go func() {
		defer close(schedDone)
		sched.Run(ctx)
	}()
	// Registered after sess.Stop and db.Close, so it runs before them: cancel,
	// then wait for the current tick to finish writing before the DB and session go away.
	defer func() {
		stop()
		<-schedDone
	}()

	ui, err := web.New(st, sess, log)
	if err != nil {
		return err
	}

	srv := &http.Server{Handler: ui.Handler()}

	go func() {
		log.Info("listening", "addr", cfg.ListenAddr, "data_dir", cfg.DataDir)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
