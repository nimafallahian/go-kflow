package config

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("KAFKA_TOPIC", "orders")
	t.Setenv("KAFKA_GROUP_ID", "orders-consumer")
	t.Setenv("ELASTIC_URLS", "http://es1:9200,http://es2:9200")
	t.Setenv("ELASTIC_INDEX", "orders-index")
	t.Setenv("WORKER_COUNT", "10")
	t.Setenv("LOG_LEVEL", "DEBUG")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got, want := len(cfg.KafkaBrokers), 2; got != want {
		t.Fatalf("expected %d kafka brokers, got %d", want, got)
	}
	if cfg.KafkaBrokers[0] != "broker1:9092" || cfg.KafkaBrokers[1] != "broker2:9092" {
		t.Fatalf("unexpected kafka brokers: %#v", cfg.KafkaBrokers)
	}
	if cfg.KafkaTopic != "orders" {
		t.Fatalf("expected KafkaTopic=orders, got %s", cfg.KafkaTopic)
	}
	if cfg.KafkaGroupID != "orders-consumer" {
		t.Fatalf("expected KafkaGroupID=orders-consumer, got %s", cfg.KafkaGroupID)
	}
	if got, want := len(cfg.ElasticURLs), 2; got != want {
		t.Fatalf("expected %d elastic urls, got %d", want, got)
	}
	if cfg.ElasticURLs[0] != "http://es1:9200" || cfg.ElasticURLs[1] != "http://es2:9200" {
		t.Fatalf("unexpected elastic urls: %#v", cfg.ElasticURLs)
	}
	if cfg.ElasticIndex != "orders-index" {
		t.Fatalf("expected ElasticIndex=orders-index, got %s", cfg.ElasticIndex)
	}
	if cfg.WorkerCount != 10 {
		t.Fatalf("expected WorkerCount=10, got %d", cfg.WorkerCount)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Fatalf("expected LogLevel=DEBUG, got %s", cfg.LogLevel)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Ensure env is clear for these keys.
	_ = os.Unsetenv("KAFKA_TOPIC")
	_ = os.Unsetenv("KAFKA_GROUP_ID")
	_ = os.Unsetenv("ELASTIC_INDEX")
	_ = os.Unsetenv("WORKER_COUNT")
	_ = os.Unsetenv("LOG_LEVEL")

	t.Setenv("KAFKA_BROKERS", "broker1:9092")
	t.Setenv("ELASTIC_URLS", "http://es1:9200")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.KafkaTopic != "messages" {
		t.Fatalf("expected default KafkaTopic=messages, got %s", cfg.KafkaTopic)
	}
	if cfg.KafkaGroupID != "indexer-group" {
		t.Fatalf("expected default KafkaGroupID=indexer-group, got %s", cfg.KafkaGroupID)
	}
	if cfg.ElasticIndex != "messages" {
		t.Fatalf("expected default ElasticIndex=messages, got %s", cfg.ElasticIndex)
	}
	if cfg.WorkerCount != 5 {
		t.Fatalf("expected default WorkerCount=5, got %d", cfg.WorkerCount)
	}
	if cfg.LogLevel != "INFO" {
		t.Fatalf("expected default LogLevel=INFO, got %s", cfg.LogLevel)
	}
}

