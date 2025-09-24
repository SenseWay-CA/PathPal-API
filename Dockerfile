# --- Build Stage ---
# Use an official Go image to build the application
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application, creating a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .


# --- Final Stage ---
# Use a minimal base image for the final container
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Expose the port the app runs on (Echo's default is 1323)
EXPOSE 1323

# Command to run the executable
CMD ["./main"]