FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server_bin ./server/main.go

FROM alpine:3.21

WORKDIR /root/

COPY --from=builder /app/server_bin .

RUN mkdir -p /root/images

EXPOSE 8080
CMD ["./server_bin"]