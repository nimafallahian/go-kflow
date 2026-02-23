# Policy: CI/CD & Deployment

## GitHub Actions
- **Pipeline 1 (Test)**: Runs `golangci-lint` and `go test` (including Testcontainers).
- **Pipeline 2 (Build)**: Docker build with multi-stage. Base image `distroless/static`.
- **Pipeline 3 (Deploy)**: Update image tag in `deployments/k8s/deployment.yaml`.

## Kubernetes Spec
- Resource Limits: `memory: 256Mi`, `cpu: 200m`.
- Probes: 
  - Liveness: `GET /health` on port 8080.
  - Readiness: Check if Kafka broker is reachable.
