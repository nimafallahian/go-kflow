# Project Blueprint: go-kflow

## 1. System Component Diagram


## 2. Directory Structure & Responsibilities
- `cmd/indexer/`: Entry point. Initializes Config, Logger, and the DI Container.
- `internal/domain/`: Core entities (e.g., `Message`, `ProcessingStatus`). Logic here is 100% side-effect free.
- `internal/ports/`: Interfaces for `MessageConsumer` (Kafka) and `DataIndexer` (ES).
- `internal/adapters/`:
    - `kafka/`: Consumer implementation. Handles offset commits and Kafka-specific error codes.
    - `es/`: Client implementation. Handles Bulk API logic and mapping.
- `internal/service/`: The "Brain." It consumes from the port, applies domain logic, and sends to the indexer port.
- `deployments/`: K8s manifests and Dockerfiles.
- `tests/integration/`: End-to-end tests using Testcontainers.

## 3. The "Life of a Message" (Execution Flow)
1. **Trigger**: Kafka Adapter's `FetchMessage` loop receives a byte array.
2. **Transform**: Adapter converts raw bytes into `domain.Message`.
3. **Orchestrate**: `service.Indexer` receives the message. 
4. **Logic**: Service checks `domain` rules (e.g., status-based filtering).
5. **Persist**: Service calls `ports.DataIndexer.Index()`.
6. **Finalize**: Once ES acknowledges (201 Created), Service signals Kafka Adapter to **Commit Offset**.
7. **Error**: If any step fails, the Error Policy in `02_KAFKA_SPEC.md` is invoked.
