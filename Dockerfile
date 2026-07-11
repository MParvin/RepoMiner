# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /dataset-builder ./cmd/dataset-builder
RUN CGO_ENABLED=0 go build -o /dataset-api ./cmd/api
RUN CGO_ENABLED=0 go build -o /dataset-worker ./cmd/worker

# Runtime stage
FROM alpine:3.21
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY --from=builder /dataset-builder /dataset-api /dataset-worker /usr/local/bin/
COPY config/config.example.yaml /app/config.yaml
RUN mkdir -p /app/data /app/datasets /app/repos
EXPOSE 8080
CMD ["dataset-api", "--addr", ":8080"]
