# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api/main.go
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz \
    | tar xvz && mv migrate /usr/local/bin/migrate

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /usr/local/bin/migrate /usr/local/bin/migrate
EXPOSE 8080
CMD ["sh", "-c", "migrate -path ./migrations -database \"$DATABASE_URL\" up && ./server"]