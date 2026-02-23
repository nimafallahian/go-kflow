# Policy: TDD & Test Quality

## 1. Unit Tests
- Location: Same package as code (`_test.go`).
- Requirement: Mock all interfaces using `testify/mock`. 
- Rule: 100% branch coverage for `internal/service`.

## 2. Integration Tests
- Location: `/tests/integration`.
- Tool: `testcontainers-go`.
- Setup: One `TestMain` that spins up Kafka and ES containers.
- Scenario: "Push real message to Kafka -> Wait -> Check ES for document."

## 3. Workflow
- Write the Test first.
- Run `go test ./...` (should fail).
- Write implementation until pass.
