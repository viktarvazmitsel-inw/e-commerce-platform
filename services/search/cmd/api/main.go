package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shop/services/search/internal/broker"
	"github.com/shop/services/search/internal/config"
	"github.com/shop/services/search/internal/handler"
	"github.com/shop/services/search/internal/search"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting Search Service on port %s...", cfg.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	esClient, err := search.NewClient(cfg.ESHost)
	if err != nil {
		log.Printf("Warning: Elasticsearch client init error: %v", err)
	} else {
		go esClient.ConnectWithRetry(ctx, 5, 2*time.Second)
	}

	brokerClient := broker.NewClient(cfg.BrokerURL)
	brokerClient.StartReconnectLoop(ctx)

	healthHandler := handler.NewHealthHandler(esClient, brokerClient)
	searchHandler := handler.NewSearchHandler(esClient)

	mux := http.NewServeMux()
	mux.Handle("/api/health", healthHandler)
	mux.Handle("/api/search", searchHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Search Service listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Search Service gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	brokerClient.Close()
	log.Println("Search Service stopped cleanly")
}
