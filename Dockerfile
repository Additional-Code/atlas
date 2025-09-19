# syntax=docker/dockerfile:1.6

##############################
# Builder stage
##############################
FROM golang:1.21-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache build-base git

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the atlas binary
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \ 
    go build -trimpath -ldflags "-s -w" -o /out/atlas ./main.go

##############################
# Runtime stage
##############################
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary
COPY --from=builder /out/atlas /usr/local/bin/atlas

# Copy database migrations (needed for runtime migrate commands)
COPY db/migrations/sql ./db/migrations/sql

# Expose default HTTP and metrics ports
EXPOSE 8080 9090

ENV PATH="/usr/local/bin:${PATH}"

# Entrypoint runs the HTTP server; override for CLI (e.g., `docker run ... atlas migrate up`)
ENTRYPOINT ["/usr/local/bin/atlas"]
CMD ["run"]
