package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jibitesh/request-response-manager/configs"
	"github.com/jibitesh/request-response-manager/internal/server"
	"github.com/jibitesh/request-response-manager/pkg/instance"
)

func main() {
	err := configs.LoadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		os.Exit(1)
	}
	ins, err := instance.GetInstance()
	if err != nil {
		log.Printf("Error creating ins: %v", err)
		os.Exit(1)
	}
	ins.Port = 10000

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv, err := server.NewServer(configs.AppConfig, ins)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Error starting server: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	log.Println("Server exited")
}
