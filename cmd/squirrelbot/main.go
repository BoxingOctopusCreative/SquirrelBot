package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/api"
	"github.com/BoxingOctopus/squirrelbot/internal/config"
	"github.com/BoxingOctopus/squirrelbot/internal/db"
	"github.com/BoxingOctopus/squirrelbot/internal/discord"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("configuration", "err", err.Error())
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err.Error())
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "err", err.Error())
		os.Exit(1)
	}
	st := store.New(pool)
	if err := st.RecoverStuckWork(ctx); err != nil {
		log.Error("recover", "err", err.Error())
		os.Exit(1)
	}

	bot, err := discord.New(cfg.DiscordToken, st, log)
	if err != nil {
		log.Error("discord", "err", err.Error())
		os.Exit(1)
	}
	go bot.RunScheduler(ctx)

	server := api.New(cfg, st, bot, log)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Listen()
	}()
	if err := bot.Open(); err != nil {
		log.Error("discord gateway", "err", err.Error())
		_ = server.Shutdown()
		os.Exit(1)
	}
	defer bot.Close()
	log.Info("squirrelbot listening", "addr", cfg.HTTPAddr)

	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case err := <-errCh:
		if err != nil {
			log.Error("http", "err", err.Error())
		}
	}
	if err := server.Shutdown(); err != nil {
		log.Error("shutdown", "err", err.Error())
	}
	time.Sleep(200 * time.Millisecond)
}
