# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api/main.go

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
