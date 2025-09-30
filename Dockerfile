# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/main.go

# Production stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates bash

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/wait-for-it.sh /wait-for-it.sh

RUN chmod +x /wait-for-it.sh ./main

EXPOSE 8080

CMD ["/wait-for-it.sh", "db:3306", "--", "./main"]
