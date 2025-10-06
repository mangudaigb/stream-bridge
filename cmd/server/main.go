package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jibitesh/request-response-manager/internal/config"
	"github.com/jibitesh/request-response-manager/internal/logger"
	"github.com/jibitesh/request-response-manager/internal/server"
	"github.com/jibitesh/request-response-manager/pkg/instance"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}

	if err := logger.Init(); err != nil {
		panic(err)
	}
	defer logger.Sync()

	ins, err := instance.GetInstance()
	if err != nil {
		logger.Error("Error creating ins: %v", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv, err := server.NewServer(config.AppConfig, ins)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("Error starting server: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down server: %v", err)
	}
	logger.Info("Server exited")
}
