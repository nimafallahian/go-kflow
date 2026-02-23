package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
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

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		svc.Start(ctx)
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Printf("service terminated with error: %v", err)
	}
}


