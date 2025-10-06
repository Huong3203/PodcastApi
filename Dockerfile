# -----------------------
# Build stage
# -----------------------
FROM golang:1.24-alpine AS builder

# Cài git để go mod download hoạt động
RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod và go.sum trước để cache dependency
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ source code
COPY . .

# Build binary (file main.go nằm trong thư mục cmd)
RUN go build -o main ./cmd

# -----------------------
# Production stage
# -----------------------
FROM alpine:latest

# Cài chứng chỉ SSL + bash cho wait-for-it.sh
RUN apk --no-cache add ca-certificates bash

WORKDIR /root/

# Copy binary và script từ stage build
COPY --from=builder /app/main .
COPY --from=builder /app/wait-for-it.sh /wait-for-it.sh

# Gán quyền thực thi
RUN chmod +x /wait-for-it.sh ./main

# Mở cổng 8080 (phải trùng PORT trong .env)
EXPOSE 8080

# Chạy app sau khi DB sẵn sàng
CMD ["/wait-for-it.sh", "db:3306", "--", "./main"]
