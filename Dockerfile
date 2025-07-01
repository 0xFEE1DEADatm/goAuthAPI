FROM golang:1.24.4-alpine AS builder  

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o auth-service ./cmd

FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/auth-service .

COPY wait-for-postgres.sh /wait-for-postgres.sh
RUN chmod +x /wait-for-postgres.sh

COPY .env .

EXPOSE 8080

RUN apt-get update && apt-get install -y netcat && rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/wait-for-postgres.sh", "./auth-service"]
