# --- Stage 1: Build Golang Binary ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency file
COPY backend/go.mod ./

# Download dependencies (if any)
RUN go mod download

# Copy source code
COPY backend/main.go ./

# Compile binary statically
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# --- Stage 2: Final Light Container ---
FROM alpine:latest

WORKDIR /app

# Install ca-certificates just in case
RUN apk --no-cache add ca-certificates

# Copy compiled binary from builder
COPY --from=builder /app/server .

# Copy static frontend assets
COPY frontend/ ./frontend/

# Expose server port inside container
EXPOSE 8080

# Execute server
CMD ["./server"]
