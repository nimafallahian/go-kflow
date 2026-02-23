## Multi-stage build for go-kflow indexer

FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/indexer ./cmd/indexer


FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /bin/indexer /indexer

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/indexer"]

