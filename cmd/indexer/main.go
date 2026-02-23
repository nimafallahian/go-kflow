package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	elasticclient "github.com/elastic/go-elasticsearch/v8"
	"golang.org/x/sync/errgroup"

	esadapter "github.com/nimafallahian/go-workflow/internal/adapters/es"
	kafkaadapter "github.com/nimafallahian/go-workflow/internal/adapters/kafka"
	"github.com/nimafallahian/go-workflow/internal/config"
	"github.com/nimafallahian/go-workflow/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	kConsumer, err := kafkaadapter.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	if err != nil {
		logger.Error("failed to create kafka consumer", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := kConsumer.Close(); cerr != nil {
			logger.Error("failed to close kafka consumer", "error", cerr)
		}
	}()

	esClient, err := elasticclient.NewClient(elasticclient.Config{
		Addresses: cfg.ElasticURLs,
	})
	if err != nil {
		logger.Error("failed to create elasticsearch client", "error", err)
		os.Exit(1)
	}

	indexer, err := esadapter.NewIndexer(esClient, cfg.ElasticIndex)
	if err != nil {
		logger.Error("failed to create elasticsearch indexer", "error", err)
		os.Exit(1)
	}

	svc := service.NewIndexerService(kConsumer, indexer, cfg.WorkerCount)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	g, ctx := errgroup.WithContext(rootCtx)

	// Worker/service loop.
	g.Go(func() error {
		svc.Start(ctx)
		return nil
	})

	// HTTP health server.
	g.Go(func() error {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// Graceful shutdown for HTTP server on context cancellation.
	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown error", "error", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Printf("service terminated with error: %v", err)
	}
}


