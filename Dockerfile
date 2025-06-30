FROM golang:1.24.4-alpine AS builder  

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o auth-service ./cmd

FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/auth-service .

COPY .env .

EXPOSE 8080

CMD ["./auth-service"]
