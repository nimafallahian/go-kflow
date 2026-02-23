## go-kflow – Kafka → Elasticsearch Indexer

`go-kflow` is a small Go service that consumes JSON messages from Kafka, applies domain rules, and indexes them into Elasticsearch. It is designed as a hexagonal architecture exercise to practice modern Go (1.26), testing, containers, and Kubernetes.

### Features

- **Contract-first domain model** generated from `contracts/message.json`.
- **Kafka adapter** using `segmentio/kafka-go` with manual offset commits.
- **Elasticsearch adapter** using the Bulk API.
- **Service layer** with a worker pool, status-based retry/skip logic.
- **HTTP health endpoint** on `:8080/health` for probes.
- **Docker + Kubernetes** ready.

---

### Prerequisites

- Go **1.26**+
- Docker (for integration tests and building the image)
- kubectl + a Kubernetes cluster (for deployment)

---

### Configuration (Environment Variables)

The service is configured via environment variables (parsed by `caarlos0/env`):

- **Required**
  - `KAFKA_BROKERS` – Comma-separated list of Kafka brokers, e.g. `kafka:9092` or `localhost:9092,other:9092`.
  - `ELASTIC_URLS` – Comma-separated list of Elasticsearch URLs, e.g. `http://elasticsearch:9200`.

- **Optional (with defaults)**
  - `KAFKA_TOPIC` – Kafka topic name, default: `messages`.
  - `KAFKA_GROUP_ID` – Consumer group ID, default: `indexer-group`.
  - `ELASTIC_INDEX` – Elasticsearch index name, default: `messages`.
  - `WORKER_COUNT` – Number of worker goroutines, default: `5`.
  - `LOG_LEVEL` – Log level string, default: `INFO`.

See `internal/config/config.go` for the authoritative list.

---

### `.env` example

You can use a simple `.env` file when running locally (e.g. with `direnv` or `source .env`):

```env
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=messages
KAFKA_GROUP_ID=indexer-group

ELASTIC_URLS=http://localhost:9200
ELASTIC_INDEX=messages

WORKER_COUNT=5
LOG_LEVEL=DEBUG
```

An example file is provided as `.env.example`; copy it to `.env` and adjust values as needed.

---

### How to run tests

#### Unit tests only

```bash
make test
```

`make test` runs `go test -v -race ./...` which includes:

- Domain and service unit tests.
- Integration tests (Kafka + Elasticsearch) **if Docker is available**.

When Docker is not available (e.g. some local setups), the integration tests are skipped automatically.

#### Linting

```bash
make lint
```

This uses `golangci-lint` with an increased timeout. In CI, the workflow builds `golangci-lint` from source with Go 1.26 to avoid Go-version mismatches.

---

### How to run the service locally

#### Directly with Go

1. Ensure Kafka and Elasticsearch are running and reachable (e.g. via Docker).
2. Export the necessary environment variables (or `source .env`).
3. Run:

```bash
go run ./cmd/indexer
```

The service will:

- Start consuming from Kafka.
- Index into Elasticsearch.
- Expose a health endpoint on `http://localhost:8080/health`.

#### With Docker

Build the image:

```bash
docker build -t go-kflow:local .
```

Run it (example, wiring host Kafka/ES):

```bash
docker run --rm \
  -p 8080:8080 \
  -e KAFKA_BROKERS=host.docker.internal:9092 \
  -e ELASTIC_URLS=http://host.docker.internal:9200 \
  -e KAFKA_TOPIC=messages \
  -e KAFKA_GROUP_ID=indexer-group \
  -e ELASTIC_INDEX=messages \
  -e WORKER_COUNT=5 \
  -e LOG_LEVEL=INFO \
  go-kflow:local
```

Check health:

```bash
curl -sf http://localhost:8080/health
```

---

### Kubernetes Deployment

A basic deployment manifest is provided at `deployments/k8s/deployment.yaml`.

It defines:

- A `Deployment` named `go-kflow` with **2 replicas**.
- Container image placeholder: `ghcr.io/your-org/go-kflow:latest`.
- Environment variable wiring for `KAFKA_*`, `ELASTIC_*`, `WORKER_COUNT`, and `LOG_LEVEL`.
- Resource requests/limits: `cpu: 100m/200m`, `memory: 128Mi/256Mi`.
- `livenessProbe` and `readinessProbe` hitting `GET /health` on port `8080`.

#### Apply to a cluster

1. Build and push the image to a registry (e.g. GHCR, Docker Hub) and update the `image:` field in `deployment.yaml`.
2. Apply the manifest:

```bash
kubectl apply -f deployments/k8s/deployment.yaml
```

3. Verify pods:

```bash
kubectl get pods -l app=go-kflow
```

4. Optionally port-forward a pod to inspect health:

```bash
kubectl port-forward deploy/go-kflow 8080:8080
curl -sf http://localhost:8080/health
```

---

### Project Layout (quick reference)

- `cmd/indexer/` – Application entrypoint and HTTP `/health` server.
- `internal/domain/` – Core domain model and business rules.
- `internal/ports/` – Interfaces for Kafka consumer and ES indexer.
- `internal/adapters/` – Kafka and Elasticsearch adapter implementations.
- `internal/service/` – Orchestration / worker pool logic.
- `internal/config/` – Env-based configuration loader.
- `tests/integration/` – Kafka + Elasticsearch integration tests using Testcontainers.
- `deployments/k8s/` – Kubernetes manifests.

