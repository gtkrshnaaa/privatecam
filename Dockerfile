# /Dockerfile
# =========================================================================
# Dockerfile Multi-stage untuk PrivateCam.
# Stage 1: Melakukan build binary Go secara statis menggunakan compiler Golang-alpine.
# Stage 2: Menyusun image final berbasis Alpine Linux yang sangat ringan, menyalin
#          binary server terkompilasi beserta aset statis frontend, dan mengekspos port 8080.
# =========================================================================
# --- Stage 1: Build Golang Binary ---
FROM golang:1.22-alpine AS builder

WORKDIR /app/backend

# Copy dependency files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY backend/ ./

# Compile binary statically
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server main.go

# --- Stage 2: Final Light Container ---
FROM alpine:latest

# Install ca-certificates and tzdata to support secure connections and correct timezones
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Create data directory for SQLite database persistence and setup non-root user
RUN mkdir -p /app/data && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /app

# Copy compiled binary from builder
COPY --from=builder --chown=appuser:appgroup /app/server .

# Copy static frontend assets
COPY --chown=appuser:appgroup frontend/ ./frontend/

# Use the non-root user
USER appuser

# Expose server port inside container
EXPOSE 8080

# Execute server
CMD ["./server"]
