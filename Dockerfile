# --- Build Stage ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# --- Final Stage ---
FROM alpine:latest

# ffmpeg is needed to decode the Pi's H264/MPEGTS UDP stream
RUN apk add --no-cache ffmpeg

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 1323
EXPOSE 8554/udp

CMD ["./main"]