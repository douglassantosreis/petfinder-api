package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/yourname/go-backend/docs"
	"github.com/yourname/go-backend/internal/platform/config"
	"github.com/yourname/go-backend/internal/server"
)

// @title Lost Pet Finder API
// @version 1.0
// @description API para cadastro de animais perdidos e comunicação interna.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	ctx := context.Background()
	app, err := server.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	addr := ":" + cfg.Port
	httpServer := &http.Server{
		Addr:    addr,
		Handler: app.Routes(),
	}

	slog.Info("server starting", "addr", addr)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("shutdown warning", "error", err)
	}
}
