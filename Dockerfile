# Build stage
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o deblock ./

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/deblock .

# Copy docs
COPY --from=builder /app/docs ./docs

# Set file permissions to ensure the binary is executable
RUN chmod +x /root/deblock

# Expose the application port
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release

# Command to run the executable
ENTRYPOINT ["/root/deblock"]

# Default command
CMD ["rest"]