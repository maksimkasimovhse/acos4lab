FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/worker_bin ./worker/main.go

FROM alpine:3.21

WORKDIR /root

COPY --from=builder /app/worker_bin .

ENV SERVER_ADDR=localhost:8080

CMD ["./worker_bin"]